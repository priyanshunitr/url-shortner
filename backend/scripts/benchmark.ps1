param([int]$Requests = 10000, [int]$Concurrency = 100)
$ErrorActionPreference = 'Stop'
$backendRoot = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
Push-Location $backendRoot
$originalCache = $env:CACHE_ENABLED
$originalLimit = $env:REDIRECT_RATE_LIMIT
function Wait-Api {
    for ($i = 0; $i -lt 100; $i++) {
        try { Invoke-RestMethod 'http://localhost:8080/readyz' | Out-Null; return } catch { Start-Sleep -Milliseconds 200 }
    }
    throw 'API did not become ready'
}
try {
    New-Item -ItemType Directory -Force -Path benchmarks | Out-Null
    $env:REDIRECT_RATE_LIMIT = [string]([Math]::Max(50000, 4 * $Requests))
    $env:CACHE_ENABLED = 'false'
    docker compose up -d --no-deps --force-recreate api
    if ($LASTEXITCODE -ne 0) { throw 'Could not start cache-off API' }
    Wait-Api
    $created = Invoke-RestMethod -Method Post -Uri 'http://localhost:8080/api/urls' -ContentType 'application/json' -Body '{"url":"https://example.com/benchmark"}'
    foreach ($mode in @('off', 'on')) {
        if ($mode -eq 'on') {
            $env:CACHE_ENABLED = 'true'
            docker compose up -d --no-deps --force-recreate api
            if ($LASTEXITCODE -ne 0) { throw 'Could not start cache-on API' }
            Wait-Api
        }
        # Warm the connection pool/cache outside the timed run.
        go run ./cmd/loadtest -url $created.short_url -n 1 -c 1 -label "warm-$mode"
        if ($LASTEXITCODE -ne 0) { throw 'Warm-up failed' }
        go run ./cmd/loadtest -url $created.short_url -n $Requests -c $Concurrency -label "cache-$mode" -out "benchmarks/cache-$mode.json"
        if ($LASTEXITCODE -ne 0) { throw "cache-$mode run contained failed or rate-limited requests" }
    }
} finally {
    if ($null -eq $originalCache) { Remove-Item Env:CACHE_ENABLED -ErrorAction SilentlyContinue } else { $env:CACHE_ENABLED = $originalCache }
    if ($null -eq $originalLimit) { Remove-Item Env:REDIRECT_RATE_LIMIT -ErrorAction SilentlyContinue } else { $env:REDIRECT_RATE_LIMIT = $originalLimit }
    docker compose up -d --no-deps --force-recreate api
    Pop-Location
}

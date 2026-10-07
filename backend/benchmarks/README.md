# Redirect benchmark

Measured on 8 October 2026 (02:37 IST), Windows amd64, Go 1.26.5 load client, 16 logical CPUs, Docker Desktop Linux containers. PostgreSQL 17, Redis 7 and the Go API ran locally; Prometheus scraped at 10-second intervals. No build or race-test job ran during this recorded comparison. The API used its release configuration, PostgreSQL pool defaults, Redis AOF every-second persistence and a five-second analytics flush interval.

Each mode received **10,000 requests at concurrency 100** against the same short URL after a one-request warm-up. Both modes retained sliding-window rate limiting, buffered analytics and metrics. Only URL caching changed. The redirect limit was raised to 50,000/minute during measurement and restored afterward. The client checks HTTP 302 and **never follows redirects** to the destination website.

| Mode | Successful 302s | Requests/sec | Mean latency | p95 | p99 |
| --- | ---: | ---: | ---: | ---: | ---: |
| Cache off | 10,000/10,000 | 7,118 | 14.02 ms | 18.16 ms | 39.06 ms |
| Cache on | 10,000/10,000 | 6,089 | 16.38 ms | 20.32 ms | 43.06 ms |

Raw observations: [cache off](cache-off.json), [cache on](cache-on.json).

The cache did **not** improve throughput in this local run. A nearby PostgreSQL lookup is inexpensive, and Redis cache scripts add their own latency. A cache hit still avoids the PostgreSQL URL lookup, reducing read load; this measurement does not establish a production speedup or capacity limit. These are single runs on one machine with a hot URL, not statistically controlled results. Repeat with realistic database latency, URL distributions, longer runs and alternating mode order before making performance claims.

From `backend`, with Docker Desktop and the Compose stack running:

```powershell
./scripts/benchmark.ps1 -Requests 10000 -Concurrency 100
```

The script recreates the API for each mode, overwrites the two JSON observations, and restores the original cache/rate settings in `finally`. Run it when no other command is rebuilding or recreating the API. It creates an anonymous benchmark URL and leaves it in the local development database.

For a custom run, raise the configured redirect limit before testing; its default is 100 requests/minute/IP:

```powershell
go run ./cmd/loadtest -url http://localhost:8080/YOUR_CODE -n 10000 -c 100 -expect 302 -out benchmarks/custom.json
```

Status `0` in the result denotes a client transport failure. Any unexpected status or transport failure makes the command exit unsuccessfully.

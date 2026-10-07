// A small HTTP load client that never follows redirects to the destination site.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

func main() {
	target := flag.String("url", "", "short URL to request")
	n := flag.Int("n", 10000, "requests")
	c := flag.Int("c", 100, "concurrency")
	expect := flag.Int("expect", http.StatusFound, "expected HTTP status")
	output := flag.String("out", "", "optional JSON result file")
	label := flag.String("label", "", "run label")
	flag.Parse()
	if *target == "" || *n <= 0 || *c <= 0 {
		fmt.Fprintln(os.Stderr, "provide -url, positive -n and -c")
		os.Exit(2)
	}
	transport := &http.Transport{MaxIdleConns: *c, MaxIdleConnsPerHost: *c, MaxConnsPerHost: *c}
	client := &http.Client{Transport: transport, Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	defer transport.CloseIdleConnections()
	latencies := make([]float64, *n)
	statuses := make([]int, *n)
	var next atomic.Int64
	var wg sync.WaitGroup
	started := time.Now()
	for i := 0; i < *c; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				index := int(next.Add(1) - 1)
				if index >= *n {
					return
				}
				t0 := time.Now()
				response, err := client.Get(*target)
				if err == nil {
					statuses[index] = response.StatusCode
					_, _ = io.Copy(io.Discard, response.Body)
					_ = response.Body.Close()
				}
				latencies[index] = float64(time.Since(t0)) / float64(time.Millisecond)
			}
		}()
	}
	wg.Wait()
	elapsed := time.Since(started).Seconds()
	counts := map[int]int{}
	successes := 0
	mean := 0.0
	for i, status := range statuses {
		counts[status]++
		mean += latencies[i]
		if status == *expect {
			successes++
		}
	}
	sort.Float64s(latencies)
	percentile := func(p float64) float64 { return latencies[int(math.Ceil(p*float64(*n)))-1] }
	result := map[string]any{
		"label": *label, "url": *target, "timestamp_utc": started.UTC().Format(time.RFC3339),
		"requests": *n, "concurrency": *c, "expected_status": *expect, "successful_requests": successes,
		"status_counts": counts, "elapsed_seconds": elapsed, "requests_per_second": float64(*n) / elapsed,
		"latency_ms":        map[string]float64{"mean": mean / float64(*n), "p50": percentile(.5), "p95": percentile(.95), "p99": percentile(.99)},
		"client":            map[string]any{"go": runtime.Version(), "os": runtime.GOOS, "arch": runtime.GOARCH, "logical_cpus": runtime.NumCPU()},
		"follows_redirects": false,
	}
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(data))
	if *output != "" {
		if err := os.WriteFile(*output, append(data, '\n'), 0644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	if successes != *n {
		os.Exit(1)
	}
}

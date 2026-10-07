// Package metrics owns a private Prometheus registry for each server instance.
package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	Registry        *prometheus.Registry
	Requests        *prometheus.CounterVec
	Duration        *prometheus.HistogramVec
	Redirects       prometheus.Counter
	CacheHits       prometheus.Counter
	CacheMisses     prometheus.Counter
	CacheErrors     prometheus.Counter
	RateRejections  prometheus.Counter
	RateErrors      prometheus.Counter
	AnalyticsErrors prometheus.Counter
	SamplesDropped  prometheus.Counter
	FlushErrors     prometheus.Counter
	FlushedBatches  prometheus.Counter
	PendingBatches  prometheus.Gauge
}

func New() *Metrics {
	counter := func(name, help string) prometheus.Counter {
		return prometheus.NewCounter(prometheus.CounterOpts{Name: name, Help: help})
	}
	m := &Metrics{
		Registry:        prometheus.NewRegistry(),
		Requests:        prometheus.NewCounterVec(prometheus.CounterOpts{Name: "http_requests_total", Help: "Completed HTTP requests by method, route template and status."}, []string{"method", "route", "status"}),
		Duration:        prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: "request_duration_seconds", Help: "HTTP request duration in seconds.", Buckets: prometheus.DefBuckets}, []string{"method", "route"}),
		Redirects:       counter("redirects_total", "Successful HTTP short-URL redirects."),
		CacheHits:       counter("cache_hits_total", "URL cache lookups served by Redis."),
		CacheMisses:     counter("cache_misses_total", "URL cache lookups requiring PostgreSQL."),
		CacheErrors:     counter("cache_errors_total", "URL cache read, fill or invalidation failures."),
		RateRejections:  counter("rate_limit_rejections_total", "Requests rejected by the sliding-window limiter."),
		RateErrors:      counter("rate_limit_errors_total", "Requests blocked because the rate limiter was unavailable."),
		AnalyticsErrors: counter("analytics_record_errors_total", "Redirect events not recorded in Redis."),
		SamplesDropped:  counter("analytics_samples_dropped_total", "Event details omitted due to the per-batch sample cap; counts remain recorded."),
		FlushErrors:     counter("analytics_flush_errors_total", "Failed batch persistence or acknowledgement attempts."),
		FlushedBatches:  counter("analytics_flushed_batches_total", "Acknowledged analytics batches, including safe retries."),
		PendingBatches:  prometheus.NewGauge(prometheus.GaugeOpts{Name: "analytics_pending_batches", Help: "Sealed analytics batches awaiting PostgreSQL persistence/acknowledgement."}),
	}
	m.Registry.MustRegister(m.Requests, m.Duration, m.Redirects, m.CacheHits, m.CacheMisses, m.CacheErrors, m.RateRejections, m.RateErrors, m.AnalyticsErrors, m.SamplesDropped, m.FlushErrors, m.FlushedBatches, m.PendingBatches, collectors.NewGoCollector(), collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	return m
}

func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.Registry, promhttp.HandlerOpts{})
}

func (m *Metrics) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		route := c.FullPath()
		if route == "/metrics" {
			return
		}
		if route == "" {
			route = "unmatched"
		}
		method := c.Request.Method
		switch method {
		case "GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS":
		default:
			method = "OTHER"
		}
		status := c.Writer.Status()
		m.Requests.WithLabelValues(method, route, strconv.Itoa(status)).Inc()
		m.Duration.WithLabelValues(method, route).Observe(time.Since(start).Seconds())
		if status == http.StatusFound && (route == "/:shortCode" || route == "/api/v1/:shortCode") {
			m.Redirects.Inc()
		}
		if value, ok := c.Get("rate_limit_rejected"); ok && value == true {
			m.RateRejections.Inc()
		}
		if value, ok := c.Get("rate_limit_error"); ok && value == true {
			m.RateErrors.Inc()
		}
	}
}

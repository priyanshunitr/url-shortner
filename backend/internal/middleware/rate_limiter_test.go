package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func TestSlidingWindowDoesNotResetAtMinuteBoundary(t *testing.T) {
	m := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: m.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	start := time.Date(2026, 10, 8, 2, 13, 59, 0, time.UTC)
	m.SetTime(start)
	for i := 0; i < 10; i++ {
		d, err := AllowRequest(context.Background(), client, "create:ip", 10, time.Minute)
		if err != nil || !d.Allowed {
			t.Fatalf("admission %d: %+v, %v", i, d, err)
		}
	}
	m.SetTime(start.Add(2 * time.Second))
	d, err := AllowRequest(context.Background(), client, "create:ip", 10, time.Minute)
	if err != nil || d.Allowed || d.Retry != 58*time.Second {
		t.Fatalf("window reset early: %+v, %v", d, err)
	}
	m.SetTime(start.Add(time.Minute))
	d, err = AllowRequest(context.Background(), client, "create:ip", 10, time.Minute)
	if err != nil || !d.Allowed {
		t.Fatalf("window did not release expired requests: %+v, %v", d, err)
	}
}

func TestConcurrentRequestsRespectSharedLimit(t *testing.T) {
	m := miniredis.RunT(t)
	m.SetTime(time.Date(2026, 10, 8, 0, 0, 0, 0, time.UTC))
	client := redis.NewClient(&redis.Options{Addr: m.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	var allowed atomic.Int64
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			d, err := AllowRequest(context.Background(), client, "POST:/api/urls:127.0.0.1", 10, time.Minute)
			if err != nil {
				t.Error(err)
				return
			}
			if d.Allowed {
				allowed.Add(1)
			}
		}()
	}
	wg.Wait()
	if allowed.Load() != 10 {
		t.Fatalf("admitted %d concurrent requests, want 10", allowed.Load())
	}
	for _, key := range m.Keys() {
		if m.TTL(key) <= 0 {
			t.Fatalf("limiter key has no expiry: %s", key)
		}
	}
}

func TestRateLimitHeadersAndUnavailableRedis(t *testing.T) {
	gin.SetMode(gin.TestMode)
	m := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: m.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	r := gin.New()
	_ = r.SetTrustedProxies(nil)
	r.POST("/api/urls", RateLimiter(client, 1, time.Minute), func(c *gin.Context) { c.Status(http.StatusCreated) })
	request := func() *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/urls", nil))
		return w
	}
	if w := request(); w.Code != http.StatusCreated {
		t.Fatalf("first request: %d", w.Code)
	}
	if w := request(); w.Code != http.StatusTooManyRequests || w.Header().Get("Retry-After") == "" {
		t.Fatalf("rejection: %d, %v", w.Code, w.Header())
	}
	_ = client.Close()
	if w := request(); w.Code != http.StatusServiceUnavailable {
		t.Fatalf("Redis failure: %d", w.Code)
	}
}

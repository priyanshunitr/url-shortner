//go:build integration

package routes

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gottatouchsomegrass/url/internal/cache"
	"github.com/gottatouchsomegrass/url/internal/config"
	"github.com/gottatouchsomegrass/url/internal/database"
	"github.com/gottatouchsomegrass/url/internal/metrics"
	"github.com/gottatouchsomegrass/url/internal/model"
	"github.com/gottatouchsomegrass/url/internal/repository"
	"github.com/gottatouchsomegrass/url/internal/service"
	"github.com/gottatouchsomegrass/url/internal/utils"
	"github.com/gottatouchsomegrass/url/migrations"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type loseFirstAck struct {
	services.AnalyticsBuffer
	lost bool
}

func (b *loseFirstAck) Ack(ctx context.Context, key string) error {
	if !b.lost {
		b.lost = true
		return errors.New("simulated lost Redis acknowledgement")
	}
	return b.AnalyticsBuffer.Ack(ctx, key)
}

func TestCoreIntegration(t *testing.T) {
	dbURL, redisURL := os.Getenv("INTEGRATION_DATABASE_URL"), os.Getenv("INTEGRATION_REDIS_URL")
	if dbURL == "" || redisURL == "" {
		t.Fatal("set INTEGRATION_DATABASE_URL and INTEGRATION_REDIS_URL to isolated test services")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(admin.Close)
	schema := fmt.Sprintf("urlshortner_test_%d", time.Now().UnixNano())
	quoted := pgx.Identifier{schema}.Sanitize()
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+quoted); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = admin.Exec(ctx, "DROP SCHEMA "+quoted+" CASCADE") })
	poolConfig, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		t.Fatal(err)
	}
	poolConfig.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	options, err := redis.ParseURL(redisURL)
	if err != nil || options.DB == 0 {
		t.Fatal("integration Redis must use an empty, dedicated nonzero database")
	}
	rdb := redis.NewClient(options)
	t.Cleanup(func() { _ = rdb.Close() })
	if size, err := rdb.DBSize(ctx).Result(); err != nil || size != 0 {
		t.Fatalf("integration Redis database must be empty: size %d, %v", size, err)
	}
	t.Cleanup(func() {
		if keys, err := rdb.Keys(ctx, "*").Result(); err == nil && len(keys) > 0 {
			_ = rdb.Del(ctx, keys...).Err()
		}
	})

	// Exercise an upgrade with real pre-existing data, rather than only a fresh DB.
	if err := migrations.Run(ctx, pool, "up", 5); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO urls(long_url, short_url, clicks, created_at) VALUES ('https://legacy.example', 'legacy01', 1872, '2026-10-08 00:00:00')`); err != nil {
		t.Fatal(err)
	}
	if err := migrations.Run(ctx, pool, "up", 0); err != nil {
		t.Fatal(err)
	}
	repo := &repositories.URLQuery{DB: pool}
	legacy, err := repo.GetByShortURL(ctx, "legacy01")
	if err != nil || legacy.Clicks != 1872 || legacy.LongURL != "https://legacy.example" {
		t.Fatalf("migration lost existing data: %+v, %v", legacy, err)
	}
	buffer := &cache.RedisAnalytics{Client: rdb}
	stats := metrics.New()
	urls := services.NewURLService(repo, cache.NewRedisURLCache(rdb, time.Hour))
	urls.BaseURL, urls.Recorder, urls.Metrics = "http://localhost:8080", buffer, stats
	worker := &services.AnalyticsWorker{Buffer: buffer, Repo: repo, Metrics: stats}
	db := &database.Queries{URLQuery: repo, AnalyticsQuery: &repositories.AnalyticsQuery{DB: pool}, UserQuery: &repositories.UserQuery{DB: pool, RDB: rdb}, Redis: rdb}
	gin.SetMode(gin.TestMode)
	utils.InitValidator()
	t.Setenv("JWT_SECRET", "01234567890123456789012345678901")
	cfg := configs.App{CreateLimit: 10, RedirectLimit: 100, StatsLimit: 100}
	router, err := NewRouter(db, cfg, urls, stats)
	if err != nil {
		t.Fatal(err)
	}
	request := func(method, path, body string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(method, path, bytes.NewBufferString(body))
		r.RemoteAddr = "192.0.2.19:10000"
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Referer", "https://news.example/article")
		router.ServeHTTP(w, r)
		return w
	}
	w := request("POST", "/api/urls", `{"url":"https://example.com/long/path"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", w.Code, w.Body)
	}
	var created models.CoreURLResponse
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`^[a-zA-Z0-9]{8}$`).MatchString(created.ShortCode) || created.ShortURL != urls.BaseURL+"/"+created.ShortCode {
		t.Fatalf("create response: %+v", created)
	}
	for i := 0; i < 10; i++ {
		if w := request("GET", "/"+created.ShortCode, ""); w.Code != 302 || w.Header().Get("Location") != "https://example.com/long/path" {
			t.Fatalf("redirect %d: %d %s", i, w.Code, w.Body)
		}
	}
	if err := worker.Flush(ctx); err != nil {
		t.Fatal(err)
	}
	// Retry a genuinely committed SQL batch whose Redis ACK was lost.
	if w := request("GET", "/"+created.ShortCode, ""); w.Code != 302 {
		t.Fatal(w.Body.String())
	}
	retryWorker := &services.AnalyticsWorker{Buffer: &loseFirstAck{AnalyticsBuffer: buffer}, Repo: repo}
	if err := retryWorker.Flush(ctx); err == nil {
		t.Fatal("lost ACK was not surfaced")
	}
	if err := retryWorker.Flush(ctx); err != nil {
		t.Fatal(err)
	}
	if err := worker.Flush(ctx); err != nil {
		t.Fatal(err)
	}
	w = request("GET", "/api/urls/"+created.ShortCode+"/stats", "")
	var result models.URLStats
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil || result.Clicks != 11 || result.LastAccessed == nil || len(result.TopReferrers) != 1 || result.TopReferrers[0].Clicks != 11 {
		t.Fatalf("persisted stats: %+v, %v, %s", result, err, w.Body)
	}
	for i := 0; i < 9; i++ {
		if w := request("POST", "/api/urls", `{"url":"https://example.com"}`); w.Code != 201 {
			t.Fatalf("admission: %d", w.Code)
		}
	}
	if w := request("POST", "/api/urls", `{"url":"https://example.com"}`); w.Code != 429 || w.Header().Get("Retry-After") == "" {
		t.Fatalf("create limit: %d %v", w.Code, w.Header())
	}
	if w := request("GET", "/missing", ""); w.Code != 404 {
		t.Fatalf("missing URL: %d", w.Code)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO urls(original_url, short_code, expires_at) VALUES ('https://expired.example', 'expired1', NOW()-INTERVAL '1 second')`); err != nil {
		t.Fatal(err)
	}
	if w := request("GET", "/expired1", ""); w.Code != 410 {
		t.Fatalf("expired URL: %d", w.Code)
	}
	var userID int64
	if err := pool.QueryRow(ctx, `INSERT INTO users(email, password_hash) VALUES ('cache@example.com', 'test') RETURNING id`).Scan(&userID); err != nil {
		t.Fatal(err)
	}
	u, err := urls.ShortenURL(ctx, userID, "admin", "https://before.example", "editme")
	if err != nil {
		t.Fatal(err)
	}
	legacyAnalytics := &repositories.AnalyticsQuery{DB: pool}
	if loaded, err := legacyAnalytics.GetURLByID(ctx, u.ID); err != nil || loaded == nil || loaded.ID != u.ID {
		t.Fatalf("legacy ownership lookup after migration: %+v, %v", loaded, err)
	}
	if overview, err := legacyAnalytics.GetAnalyticsOverview(ctx, u.ID); err != nil || overview.TotalClicks != 0 {
		t.Fatalf("empty legacy analytics: %+v, %v", overview, err)
	}
	publicURL, err := repo.GetByShortURL(ctx, created.ShortCode)
	if err != nil || publicURL == nil {
		t.Fatalf("public URL lookup: %v", err)
	}
	if daily, err := legacyAnalytics.GetDailyClicks(ctx, publicURL.ID); err != nil || len(daily) != 1 || daily[0].Clicks != 11 {
		t.Fatalf("UTC daily analytics: %+v, %v", daily, err)
	}
	if w := request("GET", "/editme", ""); w.Code != 302 {
		t.Fatal(w.Body.String())
	}
	if err := urls.UpdateUserURL(ctx, u.ID, userID, "https://after.example"); err != nil {
		t.Fatal(err)
	}
	if w := request("GET", "/editme", ""); w.Header().Get("Location") != "https://after.example" {
		t.Fatal("updated redirect stayed cached")
	}
	if w := request("GET", "/api/urls/editme/stats", ""); w.Code != 404 {
		t.Fatal("private URL stats leaked through public API")
	}
	if err := urls.DeleteUserURL(ctx, u.ID, userID); err != nil {
		t.Fatal(err)
	}
	if w := request("GET", "/editme", ""); w.Code != 404 {
		t.Fatal("deleted redirect stayed cached")
	}
	if err := worker.Flush(ctx); err != nil {
		t.Fatalf("deleted URL poisoned buffered analytics: %v", err)
	}
	w = request("GET", "/metrics", "")
	for _, metric := range []string{"cache_hits_total 10", "rate_limit_rejections_total 1", "analytics_flush_errors_total 0"} {
		if !strings.Contains(w.Body.String(), metric) {
			t.Fatalf("missing %s in metrics", metric)
		}
	}
	// Exercise reversibility on this test schema only, then restore the new schema.
	if err := migrations.Run(ctx, pool, "down", 2); err != nil {
		t.Fatal(err)
	}
	if err := migrations.Run(ctx, pool, "up", 0); err != nil {
		t.Fatal(err)
	}
}

package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/gottatouchsomegrass/url/docs"
	"github.com/gottatouchsomegrass/url/internal/cache"
	"github.com/gottatouchsomegrass/url/internal/config"
	"github.com/gottatouchsomegrass/url/internal/database"
	"github.com/gottatouchsomegrass/url/internal/metrics"
	"github.com/gottatouchsomegrass/url/internal/routes"
	"github.com/gottatouchsomegrass/url/internal/service"
	"github.com/gottatouchsomegrass/url/internal/utils"
	"github.com/joho/godotenv"
)

// @title URL Shortener
// @version 1.0
// @description URL shortener with Redis caching, sliding-window limits, buffered analytics and Prometheus metrics.
// @host localhost:8080
// @BasePath /
// @securityDefinitions.apikey Bearer
// @in header
// @name Authorization
func main() {
	envFile := os.Getenv("ENV_FILE")
	if envFile == "" {
		envFile = ".env"
	}
	_ = godotenv.Load(envFile)
	cfg, err := configs.Load()
	if err != nil {
		log.Fatal(err)
	}
	if mode := os.Getenv("GIN_MODE"); mode != "" {
		gin.SetMode(mode)
	}
	utils.InitValidator()
	db, err := database.OpenDBConnection()
	if err != nil {
		log.Fatal(err)
	}
	defer db.URLQuery.DB.Close()
	defer func() { _ = db.Redis.Close() }()

	stats := metrics.New()
	var urlCache cache.URLCache
	if cfg.CacheEnabled {
		urlCache = cache.NewRedisURLCache(db.Redis, cfg.CacheTTL)
	}
	urls := services.NewURLService(db.URLQuery, urlCache)
	urls.BaseURL, urls.Metrics = cfg.BaseURL, stats
	buffer := &cache.RedisAnalytics{Client: db.Redis}
	urls.Recorder = buffer
	worker := &services.AnalyticsWorker{Buffer: buffer, Repo: db.URLQuery, Interval: cfg.FlushInterval, Metrics: stats}
	router, err := routes.NewRouter(db, cfg, urls, stats)
	if err != nil {
		log.Fatal(err)
	}

	workerCtx, stopWorker := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { defer close(done); worker.Run(workerCtx) }()
	utils.StartSvrGracefulShutdown(configs.ConfigHTTPServer(router))
	stopWorker()
	<-done
	flushCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := worker.Flush(flushCtx); err != nil {
		log.Printf("final analytics flush retained for retry: %v", err)
	}
}

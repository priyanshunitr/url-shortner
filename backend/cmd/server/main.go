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
	"github.com/gottatouchsomegrass/url/internal/handler"
	"github.com/gottatouchsomegrass/url/internal/metrics"
	"github.com/gottatouchsomegrass/url/internal/middleware"
	"github.com/gottatouchsomegrass/url/internal/routes"
	"github.com/gottatouchsomegrass/url/internal/service"
	"github.com/gottatouchsomegrass/url/internal/utils"
	"github.com/joho/godotenv"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title           URL Shortener
// @version         0.67
// @description     High performance URL shortener API with cache-aside caching and analytics.
// @host            localhost:8080
// @BasePath        /api/v1
// @securityDefinitions.apikey Bearer
// @in              header
// @name            Authorization
func main() {
	utils.InitValidator()

	_ = godotenv.Load(".env.test")
	_ = godotenv.Load()

	r := gin.Default()
	stats := metrics.New()
	r.Use(stats.Middleware())
	r.GET("/metrics", gin.WrapH(stats.Handler()))
	if err := r.SetTrustedProxies(nil); err != nil {
		log.Fatal(err)
	}

	// Swagger
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// CORS
	r.Use(middleware.CORS())

	dbq, err := database.OpenDBConnection()
	if err != nil {
		log.Fatal(err)
	}

	//use global rate limiter
	// r.Use(
	//  middleware.RateLimiter(
	// 	dbq.RDB,
	// 	100,
	// 	time.Minute,
	//  ),
	// )

	urlService := services.NewURLService(dbq.URLQuery, cache.NewRedisURLCache(dbq.Redis, time.Hour))
	urlService.Metrics = stats
	buffer := &cache.RedisAnalytics{Client: dbq.Redis}
	urlService.Recorder = buffer
	worker := &services.AnalyticsWorker{Buffer: buffer, Repo: dbq.URLQuery, Interval: 5 * time.Second, Metrics: stats}
	workerCtx, stopWorker := context.WithCancel(context.Background())
	workerDone := make(chan struct{})
	go func() { defer close(workerDone); worker.Run(workerCtx) }()
	urlService.BaseURL = os.Getenv("BASE_URL")
	if urlService.BaseURL == "" {
		urlService.BaseURL = "http://localhost:8080"
	}
	urlController := controllers.NewURLController(
		urlService,
		dbq.Redis,
	)
	analyticsService := services.NewAnalyticsService(dbq.AnalyticsQuery)
	analyticsController := controllers.NewAnalyticsController(
		analyticsService,
	)
	authService := services.NewAuthService(
		dbq.UserQuery,
	)
	authController := controllers.NewAuthController(
		authService,
	)
	adminService := services.NewAdminService(dbq.UserQuery)
	adminController := controllers.NewAdminController(
		adminService,
		urlService,
	)

	api := r.Group("/api/v1")
	routes.PublicRoutes(api, urlController, analyticsController, authController)
	routes.PrivateRoutes(api, urlController, analyticsController, authController)
	routes.AdminRoutes(api, adminController)
	r.POST("/api/urls", middleware.RateLimiter(dbq.Redis, 10, time.Minute), urlController.CreatePublicURL)
	r.GET("/api/urls/:code/stats", middleware.RateLimiter(dbq.Redis, 100, time.Minute), urlController.GetPublicStats)
	r.GET("/:shortCode", middleware.RateLimiter(dbq.Redis, 100, time.Minute), urlController.RedirectURL)

	svr := configs.ConfigHTTPServer(r)
	utils.StartSvrGracefulShutdown(svr)
	stopWorker()
	<-workerDone
	flushCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := worker.Flush(flushCtx); err != nil {
		log.Printf("final analytics flush failed: %v", err)
	}
	dbq.URLQuery.DB.Close()
	_ = dbq.Redis.Close()
}

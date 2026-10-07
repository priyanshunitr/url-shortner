package routes

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gottatouchsomegrass/url/internal/config"
	"github.com/gottatouchsomegrass/url/internal/database"
	"github.com/gottatouchsomegrass/url/internal/handler"
	"github.com/gottatouchsomegrass/url/internal/metrics"
	"github.com/gottatouchsomegrass/url/internal/middleware"
	"github.com/gottatouchsomegrass/url/internal/service"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func NewRouter(db *database.Queries, cfg configs.App, urls *services.URLService, stats *metrics.Metrics) (*gin.Engine, error) {
	r := gin.New()
	r.Use(stats.Middleware(), gin.Logger(), gin.Recovery(), middleware.CORS())
	r.Use(func(c *gin.Context) { c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20); c.Next() })
	if err := r.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		return nil, err
	}
	r.GET("/metrics", gin.WrapH(stats.Handler()))
	r.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })
	r.GET("/readyz", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), time.Second)
		defer cancel()
		if err := db.URLQuery.DB.Ping(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "database unavailable"})
			return
		}
		if err := db.Redis.Ping(ctx).Err(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "Redis unavailable"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	uc := controllers.NewURLController(urls, db.Redis)
	ac := controllers.NewAnalyticsController(services.NewAnalyticsService(db.AnalyticsQuery))
	auc := controllers.NewAuthController(services.NewAuthService(db.UserQuery))
	admin := controllers.NewAdminController(services.NewAdminService(db.UserQuery), urls)
	api := r.Group("/api/v1")
	PublicRoutes(api, uc, ac, auc)
	PrivateRoutes(api, uc, ac, auc)
	AdminRoutes(api, admin)
	r.POST("/api/urls", middleware.RateLimiter(db.Redis, cfg.CreateLimit, time.Minute), uc.CreatePublicURL)
	r.GET("/api/urls/:code/stats", middleware.RateLimiter(db.Redis, cfg.StatsLimit, time.Minute), uc.GetPublicStats)
	r.GET("/:shortCode", middleware.RateLimiter(db.Redis, cfg.RedirectLimit, time.Minute), uc.RedirectURL)
	return r, nil
}

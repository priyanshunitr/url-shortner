package routes

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gottatouchsomegrass/url/app/controllers"
	"github.com/gottatouchsomegrass/url/pkg/middleware"
)

func PrivateRoutes(r *gin.RouterGroup,
	uc *controllers.URLController,
	ac *controllers.AnalyticsController,
	auc *controllers.AuthController) {
	//auth private endpoints
	r.Use(
		middleware.AuthMiddleware(auc.Service.UserRepo),
	)
	auth := r.Group("/auth")
	// Uncommented and added Rate Limit
	auth.POST("/logout", middleware.RateLimiter(uc.RDB, 20, time.Minute), auc.
		Logout)
	auth.GET("/me", auc.Me)
	// Added Rate Limit
	auth.GET("/sessions", middleware.RateLimiter(uc.RDB, 30, time.Minute), auc.
		GetSessions)
	// Added Rate Limit
	auth.DELETE("/sessions/:id", middleware.RateLimiter(uc.RDB, 30, time.Minute),
		auc.RevokeSession)
	// Added Rate Limit
	auth.DELETE("/sessions/others", middleware.RateLimiter(uc.RDB, 30, time.Minute),
		auc.RevokeAllOtherSessions)

	// analytics private endpoints
	analytics := r.Group("/analytics")
	analytics.Use(middleware.RequireRole("base", "premium", "admin"))
	analytics.GET(
		"/:id/overview",
		middleware.RateLimiter(
			uc.RDB,
			100,
			time.Minute,
		),
		ac.GetAnalyticsOverview,
	)
	analytics.GET(
		"/:id/daily",
		middleware.RateLimiter(
			uc.RDB,
			100,
			time.Minute,
		),
		ac.GetDailyClicks,
	)
	analytics.GET(
		"/:id/recent",
		middleware.RateLimiter(
			uc.RDB,
			100,
			time.Minute,
		),
		ac.GetRecentVisits,
	)
	analytics.GET(
		"/:id/browser",
		middleware.RateLimiter(
			uc.RDB,
			100,
			time.Minute,
		),
		ac.GetBrowserAnalytics,
	)
	analytics.GET(
		"/:id/device",
		middleware.RateLimiter(
			uc.RDB,
			100,
			time.Minute,
		),
		ac.GetDeviceAnalytics,
	)

	//url private endpoints
	r.POST("/shorten",
		middleware.RateLimiter(
			uc.RDB,
			40,
			time.Minute,
		),
		uc.ShortenURL)

	urls := r.Group("/urls")
	urls.GET("",
		middleware.RateLimiter(
			uc.RDB,
			100,
			time.Minute,
		),
		uc.GetUserURLs,
	)
	urls.PUT("/:id",
		middleware.RateLimiter(
			uc.RDB,
			40,
			time.Minute,
		),
		uc.UpdateUserURL,
	)
	urls.DELETE("/:id",
		middleware.RateLimiter(
			uc.RDB,
			40,
			time.Minute,
		),
		uc.DeleteUserURL,
	)
	urls.DELETE("/bulk",
		middleware.RateLimiter(
			uc.RDB,
			20,
			time.Minute,
		),
		uc.BulkDeleteUserURLs,
	)
}

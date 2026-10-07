// Package routes include all the routes
package routes

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gottatouchsomegrass/url/app/controllers"
	"github.com/gottatouchsomegrass/url/pkg/middleware"
)

func PublicRoutes(r *gin.RouterGroup,
	uc *controllers.URLController,
	ac *controllers.AnalyticsController,
	authController *controllers.AuthController) {

	//auth public routes
	auth := r.Group("/auth")
	// Max 5 registrations per minute per IP
	auth.POST("/register", middleware.RateLimiter(uc.RDB, 5, time.Minute),
		authController.Register)
	// Max 10 login attempts per minute per IP to prevent Bcrypt DoS
	auth.POST("/login", middleware.RateLimiter(uc.RDB, 10, time.Minute),
		authController.Login)
	// Max 10 refreshes per minute
	auth.POST("/refresh", middleware.RateLimiter(uc.RDB, 10, time.Minute),
		authController.RefreshToken)

	//url public routes
	r.GET("/:shortCode",
		middleware.RateLimiter(
			uc.RDB,
			100,
			time.Minute,
		),
		uc.RedirectURL)
}

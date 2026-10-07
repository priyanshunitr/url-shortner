package main

import (
	"log"

	"github.com/gin-gonic/gin"
	_ "github.com/gottatouchsomegrass/url/docs"
	"github.com/gottatouchsomegrass/url/internal/config"
	"github.com/gottatouchsomegrass/url/internal/database"
	"github.com/gottatouchsomegrass/url/internal/handler"
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

	urlService := services.NewURLService(dbq.URLQuery)
	urlController := controllers.NewURLController(
		urlService,
		dbq.URLQuery.RDB,
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

	svr := configs.ConfigHTTPServer(r)
	utils.StartSvrGracefulShutdown(svr)
}

package middleware

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"os"
	"strings"
)

func CORS() gin.HandlerFunc {
	value := os.Getenv("CORS_ORIGINS")
	if value == "" {
		value = "http://localhost:3000"
	}
	allowed := map[string]bool{}
	for _, origin := range strings.Split(value, ",") {
		allowed[strings.TrimSpace(origin)] = true
	}
	return func(c *gin.Context) {
		c.Header("Vary", "Origin")
		origin := c.GetHeader("Origin")
		if origin != "" && allowed[origin] {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
			c.Header("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")
		}
		if c.Request.Method == http.MethodOptions {
			if origin != "" && !allowed[origin] {
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

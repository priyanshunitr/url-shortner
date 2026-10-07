package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gottatouchsomegrass/url/app/repositories"
	"github.com/gottatouchsomegrass/url/pkg/utils"
)

func AuthMiddleware(q *repositories.UserQuery) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader(
			"Authorization",
		)

		if authHeader == "" {
			c.AbortWithStatusJSON(
				http.StatusUnauthorized,
				gin.H{
					"error": "missing authorization header",
				},
			)
			return
		}

		tokenString := strings.TrimPrefix(
			authHeader,
			"Bearer ",
		)

		claims, err := utils.ValidateJWT(
			tokenString,
		)

		if err != nil {
			c.AbortWithStatusJSON(
				http.StatusUnauthorized,
				gin.H{
					"error": "invalid token",
				},
			)
			return
		}

		// Check if token is blacklisted
		isBlacklisted, err := q.IsTokenBlacklisted(c.Request.Context(), tokenString)
		if err != nil || isBlacklisted {
			c.AbortWithStatusJSON(
				http.StatusUnauthorized,
				gin.H{
					"error": "token has been logged out",
				},
			)
			return
		}

		c.Set(
			"userID",
			claims.UserID,
		)
		c.Set(
			"userRole",
			claims.Role,
		)
		c.Set(
			"sessionID",
			claims.SessionID,
		)

		c.Next()
	}
}

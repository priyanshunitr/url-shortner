package middleware

import (
	"net/http"
	"slices"

	"github.com/gin-gonic/gin"
)

// RequireRole is a middleware that checks if the logged-in user has one of the allowed roles.
func RequireRole(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Get the user's role from the context (set by AuthMiddleware)
		role, exists := c.Get("userRole")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized, role not found"})
			return
		}

		userRole := role.(string)

		// 2. Check if the user's role is in the list of allowed roles
		roleAllowed := slices.Contains(allowedRoles, userRole)

		// 3. If not allowed, reject the request
		if !roleAllowed {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "forbidden: you do not have the required permissions or subscription plan",
			})
			return
		}

		// 4. If allowed, let them through
		c.Next()
	}
}

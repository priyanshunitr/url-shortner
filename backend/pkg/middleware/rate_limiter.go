// Package middleware manages middleware of the routes
package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RateLimiter : fixed window rate limiter
func RateLimiter(rdb *redis.Client, limit int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		ip := c.ClientIP()
		key := "rate_limit:" + ip + ":" + c.FullPath()

		counter, err := rdb.Incr(
			ctx,
			key,
		).Result()
		if err != nil {
			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"error": "rate limit error Incr",
				},
			)
			c.Abort()
			return
		}

		if counter == 1 {
			err = rdb.Expire(
				ctx,
				key,
				window,
			).Err()

			if err != nil {
				c.JSON(
					http.StatusInternalServerError,
					gin.H{
						"error": "rate limit error Expire",
					},
				)
				c.Abort()
				return
			}
		}

		if counter > int64(limit) {
			c.JSON(
				http.StatusTooManyRequests,
				gin.H{
					"error": "rate limit exceeded",
				},
			)
			c.Abort()
			return
		}
		c.Next()
	}
}

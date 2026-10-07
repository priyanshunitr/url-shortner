package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// Redis time and a single script make admission atomic across API instances.
// Only admitted requests enter the log, keeping each key bounded by its limit.
var slidingWindow = redis.NewScript(`
local t = redis.call('TIME')
local now = tonumber(t[1]) * 1000000 + tonumber(t[2])
local window = tonumber(ARGV[2])
local limit = tonumber(ARGV[1])
redis.call('ZREMRANGEBYSCORE', KEYS[1], '-inf', now - window)
local count = redis.call('ZCARD', KEYS[1])
local oldest = redis.call('ZRANGE', KEYS[1], 0, 0)
local retry = window
if #oldest > 0 then retry = math.max(1, tonumber(redis.call('ZSCORE', KEYS[1], oldest[1])) + window - now) end
if count >= limit then return {0, 0, retry} end
local seq = redis.call('INCR', KEYS[2])
redis.call('ZADD', KEYS[1], now, tostring(seq))
redis.call('PEXPIRE', KEYS[1], ARGV[3])
redis.call('PEXPIRE', KEYS[2], ARGV[3])
return {1, limit - count - 1, retry}
`)

type RateDecision struct {
	Allowed   bool
	Remaining int64
	Retry     time.Duration
}

func AllowRequest(ctx context.Context, client *redis.Client, scope string, limit int, window time.Duration) (RateDecision, error) {
	if client == nil || limit <= 0 || window < time.Millisecond {
		return RateDecision{}, errors.New("invalid rate limiter configuration")
	}
	digest := sha256.Sum256([]byte(scope))
	key := "rate:{" + hex.EncodeToString(digest[:]) + "}"
	values, err := slidingWindow.Run(ctx, client, []string{key, key + ":sequence"}, limit, window.Microseconds(), (window + time.Millisecond - 1).Milliseconds()).Int64Slice()
	if err != nil {
		return RateDecision{}, err
	}
	if len(values) != 3 {
		return RateDecision{}, errors.New("invalid limiter response")
	}
	return RateDecision{Allowed: values[0] == 1, Remaining: values[1], Retry: time.Duration(values[2]) * time.Microsecond}, nil
}

func RateLimiter(client *redis.Client, limit int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), time.Second)
		defer cancel()
		decision, err := AllowRequest(ctx, client, c.Request.Method+":"+c.FullPath()+":"+c.ClientIP(), limit, window)
		if err != nil {
			c.Set("rate_limit_error", true)
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "rate limiter unavailable"})
			return
		}
		retry := int64((decision.Retry + time.Second - 1) / time.Second)
		c.Header("X-RateLimit-Limit", strconv.Itoa(limit))
		c.Header("X-RateLimit-Remaining", strconv.FormatInt(decision.Remaining, 10))
		if !decision.Allowed {
			c.Set("rate_limit_rejected", true)
			c.Header("Retry-After", strconv.FormatInt(retry, 10))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			return
		}
		c.Next()
	}
}

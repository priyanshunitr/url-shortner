package utils

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

var ErrRateLimitExceeded = errors.New("rate limit exceeded: too many login attempts")

// CheckLoginRateLimit increments and checks the login rate limit for the given IP and email.
func CheckLoginRateLimit(ctx context.Context, rdb *redis.Client, ip, email string) error {
	if rdb == nil {
		return nil // Fail open if Redis is not configured
	}

	ipKey := "login_attempts:ip:" + ip
	emailKey := "login_attempts:email:" + email

	limit := int64(5)          // Max 5 attempts
	window := 15 * time.Minute // Per 15 minutes

	if err := incrementAndCheck(ctx, rdb, ipKey, limit, window); err != nil {
		return err
	}
	if err := incrementAndCheck(ctx, rdb, emailKey, limit, window); err != nil {
		return err
	}

	return nil
}

// incrementAndCheck safely increments the count for a key.
// It uses a pipeline to minimize network roundtrips and sets expiration on every attempt,
// effectively creating a sliding window that severely punishes aggressive brute-force bots.
func incrementAndCheck(ctx context.Context, rdb *redis.Client, key string, limit int64, window time.Duration) error {
	pipe := rdb.Pipeline()
	incr := pipe.Incr(ctx, key)
	// Setting expire on EVERY attempt ensures bots that keep hammering stay blocked
	pipe.Expire(ctx, key, window)

	_, err := pipe.Exec(ctx)
	if err != nil {
		// If Redis is down, we "fail open" (return nil) so legitimate users can still log in.
		// In a production app, you should log this error to your observability stack!
		return nil
	}

	if incr.Val() > limit {
		return ErrRateLimitExceeded
	}
	return nil
}

// ResetLoginAttempts deletes the rate limit keys upon a successful login.
func ResetLoginAttempts(ctx context.Context, rdb *redis.Client, ip, email string) {
	if rdb == nil {
		return
	}
	ipKey := "login_attempts:ip:" + ip
	emailKey := "login_attempts:email:" + email
	rdb.Del(ctx, ipKey, emailKey)
}

package database

import (
	"context"
	"log"
	"os"
	"strings"

	"github.com/redis/go-redis/v9"
)

func RedisConnection() (*redis.Client, error) {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = os.Getenv("REDIS_ADDR")
	}
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}

	var opt *redis.Options
	var err error

	// If it starts with redis:// or rediss://, parse as a URL
	if strings.HasPrefix(redisURL, "redis://") || strings.HasPrefix(redisURL, "rediss://") {
		opt, err = redis.ParseURL(redisURL)
		if err != nil {
			return nil, err
		}
	} else {
		// Fallback for simple "host:port" format
		opt = &redis.Options{
			Addr: redisURL,
		}
	}

	rdb := redis.NewClient(opt)

	_, err = rdb.Ping(context.Background()).Result()
	if err != nil {
		_ = rdb.Close()
		return nil, err
	}

	log.Println("redis connection successful...")

	return rdb, nil
}

// Package cache contains Redis adapters; URL repositories remain SQL-only.
package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/gottatouchsomegrass/url/internal/model"
	"github.com/redis/go-redis/v9"
)

type Lookup struct {
	URL     *models.URL
	Version string
}

type URLCache interface {
	Get(context.Context, string) (Lookup, error)
	Set(context.Context, *models.URL, string) error
	Invalidate(context.Context, ...string) error
}

type RedisURLCache struct {
	Client *redis.Client
	TTL    time.Duration
}

func NewRedisURLCache(client *redis.Client, ttl time.Duration) *RedisURLCache {
	return &RedisURLCache{Client: client, TTL: ttl}
}

var readURL = redis.NewScript(`
return {redis.call('GET', KEYS[1]) or '', redis.call('GET', KEYS[2]) or '0'}
`)

var fillURL = redis.NewScript(`
if (redis.call('GET', KEYS[2]) or '0') ~= ARGV[1] then return 0 end
redis.call('SET', KEYS[1], ARGV[2], 'PX', ARGV[3])
return 1
`)

var invalidateURL = redis.NewScript(`
redis.call('DEL', KEYS[1])
redis.call('INCR', KEYS[2])
redis.call('PEXPIRE', KEYS[2], ARGV[1])
return 1
`)

func urlKeys(code string) []string {
	// Matching hash tags allow each script to run on Redis Cluster too.
	return []string{"url:{" + code + "}", "url:{" + code + "}:version"}
}

func (c *RedisURLCache) Get(ctx context.Context, code string) (Lookup, error) {
	values, err := readURL.Run(ctx, c.Client, urlKeys(code)).Slice()
	if err != nil {
		return Lookup{}, err
	}
	if len(values) != 2 {
		return Lookup{}, errors.New("invalid cache response")
	}
	raw, ok := values[0].(string)
	version, vok := values[1].(string)
	if !ok || !vok {
		return Lookup{}, errors.New("invalid cache value")
	}
	lookup := Lookup{Version: version}
	if raw == "" {
		return lookup, nil
	}
	var url models.URL
	if err := json.Unmarshal([]byte(raw), &url); err != nil {
		return lookup, fmt.Errorf("decode cached URL: %w", err)
	}
	if url.ID <= 0 || url.ShortURL != code || url.LongURL == "" {
		return lookup, errors.New("incomplete cached URL")
	}
	lookup.URL = &url
	return lookup, nil
}

func (c *RedisURLCache) Set(ctx context.Context, url *models.URL, version string) error {
	ttl := c.TTL
	if url.Expiry != nil && time.Until(*url.Expiry) < ttl {
		ttl = time.Until(*url.Expiry)
	}
	if ttl < time.Millisecond {
		return nil
	}
	data, err := json.Marshal(url)
	if err != nil {
		return err
	}
	return fillURL.Run(ctx, c.Client, urlKeys(url.ShortURL), version, data, ttl.Milliseconds()).Err()
}

func (c *RedisURLCache) Invalidate(ctx context.Context, codes ...string) error {
	var result error
	for _, code := range codes {
		// Versions outlive a cached value and the bounded in-flight DB reads.
		if err := invalidateURL.Run(ctx, c.Client, urlKeys(code), (2 * c.TTL).Milliseconds()).Err(); err != nil {
			result = errors.Join(result, err)
		}
	}
	return result
}

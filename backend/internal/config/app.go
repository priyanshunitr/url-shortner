package configs

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type App struct {
	BaseURL        string
	CacheEnabled   bool
	CacheTTL       time.Duration
	FlushInterval  time.Duration
	CreateLimit    int
	RedirectLimit  int
	StatsLimit     int
	TrustedProxies []string
}

func Load() (App, error) {
	c := App{BaseURL: os.Getenv("BASE_URL"), CacheEnabled: true, CacheTTL: time.Hour, FlushInterval: 5 * time.Second, CreateLimit: 10, RedirectLimit: 100, StatsLimit: 100}
	if c.BaseURL == "" {
		c.BaseURL = "http://localhost:8080"
	}
	u, err := url.Parse(c.BaseURL)
	if err != nil || u.Hostname() == "" || (u.Scheme != "http" && u.Scheme != "https") || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") || u.User != nil {
		return c, fmt.Errorf("BASE_URL must be an HTTP(S) origin without a path, query or credentials")
	}
	if os.Getenv("DB_SERVER_URL") == "" {
		return c, fmt.Errorf("DB_SERVER_URL is required")
	}
	if len(os.Getenv("JWT_SECRET")) < 32 {
		return c, fmt.Errorf("JWT_SECRET must contain at least 32 bytes")
	}
	if value := os.Getenv("CACHE_ENABLED"); value != "" {
		c.CacheEnabled, err = strconv.ParseBool(value)
		if err != nil {
			return c, fmt.Errorf("CACHE_ENABLED: %w", err)
		}
	}
	for name, target := range map[string]*time.Duration{"CACHE_TTL": &c.CacheTTL, "ANALYTICS_FLUSH_INTERVAL": &c.FlushInterval} {
		if value := os.Getenv(name); value != "" {
			*target, err = time.ParseDuration(value)
			if err != nil || *target < time.Millisecond || *target > 24*time.Hour {
				return c, fmt.Errorf("%s must be a duration between 1ms and 24h", name)
			}
		}
	}
	for name, target := range map[string]*int{"CREATE_RATE_LIMIT": &c.CreateLimit, "REDIRECT_RATE_LIMIT": &c.RedirectLimit, "STATS_RATE_LIMIT": &c.StatsLimit} {
		if value := os.Getenv(name); value != "" {
			*target, err = strconv.Atoi(value)
			if err != nil || *target <= 0 {
				return c, fmt.Errorf("%s must be a positive integer", name)
			}
		}
	}
	if value := os.Getenv("TRUSTED_PROXIES"); value != "" {
		c.TrustedProxies = strings.Split(value, ",")
	}
	return c, nil
}

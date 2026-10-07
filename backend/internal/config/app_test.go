package configs

import "testing"

func TestConfigSupportsCacheComparisonAndRejectsInvalidDeployment(t *testing.T) {
	t.Setenv("DB_SERVER_URL", "postgres://localhost/test")
	t.Setenv("JWT_SECRET", "01234567890123456789012345678901")
	t.Setenv("BASE_URL", "https://sho.rt")
	t.Setenv("CACHE_ENABLED", "false")
	t.Setenv("CACHE_TTL", "10m")
	t.Setenv("ANALYTICS_FLUSH_INTERVAL", "2s")
	t.Setenv("REDIRECT_RATE_LIMIT", "50000")
	cfg, err := Load()
	if err != nil || cfg.CacheEnabled || cfg.RedirectLimit != 50000 {
		t.Fatalf("configuration: %+v, %v", cfg, err)
	}
	t.Setenv("BASE_URL", "https://sho.rt/private/path")
	if _, err := Load(); err == nil {
		t.Fatal("base URL containing a path was accepted")
	}
	t.Setenv("BASE_URL", "https://sho.rt")
	t.Setenv("JWT_SECRET", "short")
	if _, err := Load(); err == nil {
		t.Fatal("unsafe JWT configuration was accepted")
	}
}

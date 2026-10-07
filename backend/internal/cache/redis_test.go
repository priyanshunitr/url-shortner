package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gottatouchsomegrass/url/internal/model"
	"github.com/redis/go-redis/v9"
)

func TestCachePreservesMetadataAndBoundsExpiry(t *testing.T) {
	m := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: m.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	c := NewRedisURLCache(client, time.Hour)
	ctx := context.Background()
	expires := time.Now().Add(30 * time.Second)
	u := &models.URL{ID: 42, UserID: 7, ShortURL: "Ab91xK", LongURL: "https://example.com/path", Expiry: &expires}
	miss, err := c.Get(ctx, u.ShortURL)
	if err != nil || miss.URL != nil {
		t.Fatalf("miss: %+v, %v", miss, err)
	}
	if err := c.Set(ctx, u, miss.Version); err != nil {
		t.Fatal(err)
	}
	hit, err := c.Get(ctx, u.ShortURL)
	if err != nil || hit.URL == nil || hit.URL.ID != 42 || hit.URL.Expiry == nil || !hit.URL.Expiry.Equal(expires) {
		t.Fatalf("metadata was lost: %+v, %v", hit, err)
	}
	if ttl := m.TTL(urlKeys(u.ShortURL)[0]); ttl <= 0 || ttl > 30*time.Second {
		t.Fatalf("cache outlives URL expiry: %v", ttl)
	}
	m.FastForward(31 * time.Second)
	if value, _ := c.Get(ctx, u.ShortURL); value.URL != nil {
		t.Fatal("expired URL remained cached")
	}
}

func TestInvalidationRejectsInFlightStaleFill(t *testing.T) {
	m := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: m.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	c := NewRedisURLCache(client, time.Hour)
	ctx := context.Background()
	oldRead, err := c.Get(ctx, "abc")
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Invalidate(ctx, "abc"); err != nil {
		t.Fatal(err)
	}
	old := &models.URL{ID: 1, ShortURL: "abc", LongURL: "https://old.example"}
	if err := c.Set(ctx, old, oldRead.Version); err != nil {
		t.Fatal(err)
	}
	freshRead, _ := c.Get(ctx, "abc")
	if freshRead.URL != nil {
		t.Fatal("a read predating the mutation repopulated stale data")
	}
	old.LongURL = "https://new.example"
	if err := c.Set(ctx, old, freshRead.Version); err != nil {
		t.Fatal(err)
	}
	hit, _ := c.Get(ctx, "abc")
	if hit.URL == nil || hit.URL.LongURL != old.LongURL {
		t.Fatal("current generation could not fill cache")
	}
}

func TestShortCacheTTLStillProtectsInFlightReads(t *testing.T) {
	m := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: m.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	c := NewRedisURLCache(client, time.Millisecond)
	ctx := context.Background()
	old, err := c.Get(ctx, "shortttl")
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Invalidate(ctx, "shortttl"); err != nil {
		t.Fatal(err)
	}
	m.FastForward(time.Second)
	if err := c.Set(ctx, &models.URL{ID: 1, ShortURL: "shortttl", LongURL: "https://old.example"}, old.Version); err != nil {
		t.Fatal(err)
	}
	if result, err := c.Get(ctx, "shortttl"); err != nil || result.URL != nil {
		t.Fatalf("short TTL allowed stale fill: %+v, %v", result, err)
	}
}

package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gottatouchsomegrass/url/internal/cache"
	"github.com/gottatouchsomegrass/url/internal/model"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/redis/go-redis/v9"
)

type urlRepoStub struct {
	URLRepository
	read   func(string) (*models.URL, error)
	create func(*models.URL) error
	clicks chan int64
	events chan int64
}

func (r *urlRepoStub) GetByShortURL(_ context.Context, code string) (*models.URL, error) {
	return r.read(code)
}
func (r *urlRepoStub) CreateURL(_ context.Context, u *models.URL) error { return r.create(u) }
func (r *urlRepoStub) IncrementClicks(_ context.Context, id int64) error {
	r.clicks <- id
	return nil
}
func (r *urlRepoStub) CreateClickEvent(_ context.Context, event *models.ClickEvent) error {
	r.events <- event.URLID
	return nil
}

func TestCachedRedirectTracksRealURLIDAndExpiry(t *testing.T) {
	m := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: m.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	c := cache.NewRedisURLCache(client, time.Hour)
	u := &models.URL{ID: 52, ShortURL: "cachehit", LongURL: "https://example.com"}
	lookup, _ := c.Get(context.Background(), u.ShortURL)
	if err := c.Set(context.Background(), u, lookup.Version); err != nil {
		t.Fatal(err)
	}
	repo := &urlRepoStub{clicks: make(chan int64, 1), events: make(chan int64, 1), read: func(string) (*models.URL, error) {
		t.Error("cache hit unexpectedly queried PostgreSQL")
		return nil, errors.New("unexpected database read")
	}}
	s := NewURLService(repo, c)
	if target, err := s.HandleRedirect(context.Background(), u.ShortURL, "127.0.0.1", "", ""); err != nil || target != u.LongURL {
		t.Fatalf("redirect: %s, %v", target, err)
	}
	for _, ch := range []chan int64{repo.clicks, repo.events} {
		select {
		case id := <-ch:
			if id != u.ID {
				t.Fatalf("tracked invalid URL ID %d", id)
			}
		case <-time.After(time.Second):
			t.Fatal("analytics was not recorded")
		}
	}
	expired := time.Now().Add(-time.Second)
	repo.read = func(string) (*models.URL, error) {
		return &models.URL{ID: 53, ShortURL: "expired", LongURL: u.LongURL, Expiry: &expired}, nil
	}
	if _, err := s.HandleRedirect(context.Background(), "expired", "", "", ""); err == nil || err.Error() != "link expired" {
		t.Fatalf("expired URL should be rejected: %v", err)
	}
}

func TestGeneratedCollisionRetriesAndReturnsPersistedCode(t *testing.T) {
	calls := 0
	repo := &urlRepoStub{create: func(u *models.URL) error {
		calls++
		if calls == 1 {
			return &pgconn.PgError{Code: "23505"}
		}
		u.ID = 3
		return nil
	}}
	s := NewURLService(repo, nil)
	s.BaseURL = "https://sho.rt"
	result, err := s.CreatePublicURL(context.Background(), "https://example.com", nil)
	if err != nil || calls != 2 || result.ShortURL != "https://sho.rt/"+result.ShortCode {
		t.Fatalf("collision retry: %+v, attempts %d, %v", result, calls, err)
	}
}

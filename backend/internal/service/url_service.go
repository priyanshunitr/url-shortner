package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/gottatouchsomegrass/url/internal/cache"
	"github.com/gottatouchsomegrass/url/internal/metrics"
	"github.com/gottatouchsomegrass/url/internal/model"
	"github.com/gottatouchsomegrass/url/internal/utils"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/mileusna/useragent"
)

type URLService struct {
	Repo     URLRepository
	Cache    cache.URLCache
	BaseURL  string
	Recorder ClickRecorder
	Metrics  *metrics.Metrics
}

type ClickRecorder interface {
	Record(context.Context, *models.ClickEvent) (bool, error)
}

var ErrInvalidURL = errors.New("invalid URL request")

func NewURLService(repo URLRepository, urlCache cache.URLCache) *URLService {
	return &URLService{Repo: repo, Cache: urlCache}
}

func (s *URLService) ShortenURL(ctx context.Context, userID int64, userRole, longURL, customCode string) (*models.URL, error) {
	if customCode != "" && userRole != "premium" && userRole != "admin" {
		return nil, errors.New("custom aliases require a premium subscription")
	}

	totalURLs, err := s.Repo.CountUserURLs(ctx, userID)
	if err != nil {
		return nil, errors.New("failed to check url limits")
	}

	if userRole == "free" && totalURLs >= 10 {
		return nil, errors.New("free plan limit reached (10 URLs). please upgrade to base or premium")
	}

	if userRole == "base" && totalURLs >= 1000 {
		return nil, errors.New("base plan limit reached (1000 URLs). please upgrade to premium")
	}

	expiry := time.Now().Add(24 * time.Hour)
	return s.createURL(ctx, userID, longURL, customCode, &expiry)
}

func (s *URLService) createURL(ctx context.Context, userID int64, longURL, customCode string, expiry *time.Time) (*models.URL, error) {
	newURL := &models.URL{
		UserID:    userID,
		LongURL:   longURL,
		Expiry:    expiry,
		Clicks:    0,
		CreatedAt: time.Now(),
	}

	for attempt := 0; attempt < 5; attempt++ {
		newURL.ShortURL = customCode
		if customCode == "" {
			code, err := utils.GenerateShortCode()
			if err != nil {
				return nil, fmt.Errorf("generate short code: %w", err)
			}
			newURL.ShortURL = code
		}
		if err := s.Repo.CreateURL(ctx, newURL); err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				if customCode != "" {
					return nil, errors.New("custom code already exists")
				}
				continue
			}
			return nil, fmt.Errorf("create URL: %w", err)
		}
		return newURL, nil
	}
	return nil, errors.New("short code collision retry limit reached")
}

func (s *URLService) CreatePublicURL(ctx context.Context, longURL string, expiry *time.Time) (*models.CoreURLResponse, error) {
	parsed, err := url.ParseRequestURI(longURL)
	if err != nil || parsed.Hostname() == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") || len(longURL) > 8192 {
		return nil, fmt.Errorf("%w: url must be an absolute HTTP(S) URL, maximum 8192 bytes", ErrInvalidURL)
	}
	if expiry != nil && !expiry.After(time.Now()) {
		return nil, fmt.Errorf("%w: expires_at must be in the future", ErrInvalidURL)
	}
	url, err := s.createURL(ctx, 0, longURL, "", expiry)
	if err != nil {
		return nil, err
	}
	return &models.CoreURLResponse{ShortCode: url.ShortURL, ShortURL: strings.TrimRight(s.BaseURL, "/") + "/" + url.ShortURL}, nil
}

func (s *URLService) GetPublicStats(ctx context.Context, code string) (*models.URLStats, error) {
	stats, err := s.Repo.GetPublicStats(ctx, code)
	if err != nil {
		return nil, err
	}
	if stats == nil {
		return nil, errors.New("not found")
	}
	return stats, nil
}

func (s *URLService) HandleRedirect(ctx context.Context, code, ip, rawUserAgent, referer string) (string, error) {
	// The deadline also bounds how long a cache miss can hold an old version.
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	var lookup cache.Lookup
	if s.Cache != nil {
		var err error
		lookup, err = s.Cache.Get(ctx, code)
		if err != nil {
			log.Printf("URL cache read failed: %v", err)
			if s.Metrics != nil {
				s.Metrics.CacheErrors.Inc()
			}
		}
	}
	url := lookup.URL
	if s.Metrics != nil {
		if url != nil {
			s.Metrics.CacheHits.Inc()
		} else {
			s.Metrics.CacheMisses.Inc()
		}
	}
	if url == nil {
		var err error
		url, err = s.Repo.GetByShortURL(ctx, code)
		if err != nil {
			return "", errors.New("internal server error")
		}
		if url == nil {
			return "", errors.New("not found")
		}
		if s.Cache != nil && lookup.Version != "" {
			if err := s.Cache.Set(ctx, url, lookup.Version); err != nil {
				log.Printf("URL cache fill failed: %v", err)
				if s.Metrics != nil {
					s.Metrics.CacheErrors.Inc()
				}
			}
		}
	}

	if url.Expiry != nil && time.Now().After(*url.Expiry) {
		return "", errors.New("link expired")
	}

	ua := useragent.Parse(rawUserAgent)
	event := &models.ClickEvent{
		URLID:     url.ID,
		IPAddress: ip,
		UserAgent: rawUserAgent,
		Referer:   referer,
		Country:   "",
		Device:    ua.Device,
		Browser:   ua.Name,
		CreatedAt: time.Now().UTC(),
	}
	if s.Recorder != nil {
		stored, err := s.Recorder.Record(ctx, event)
		if err != nil {
			log.Printf("analytics record failed: %v", err)
			if s.Metrics != nil {
				s.Metrics.AnalyticsErrors.Inc()
			}
		} else if !stored && s.Metrics != nil {
			s.Metrics.SamplesDropped.Inc()
		}
	} else {
		// Retained for library callers; the server always uses the Redis buffer.
		go func() {
			writeCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			_ = s.Repo.IncrementClicks(writeCtx, url.ID)
			_ = s.Repo.CreateClickEvent(writeCtx, event)
		}()
	}

	return url.LongURL, nil
}

func (s *URLService) GetUserURLs(ctx context.Context, userID int64, limit, offset int) ([]models.URL, int, error) {
	urls, err := s.Repo.GetUserURLs(ctx, userID, limit, offset)
	if err != nil {
		return nil, 0, errors.New("internal server error")
	}

	total, err := s.Repo.CountUserURLs(ctx, userID)
	if err != nil {
		return nil, 0, errors.New("internal server error")
	}

	return urls, total, nil
}

func (s *URLService) UpdateUserURL(ctx context.Context, id, userID int64, longURL string) error {
	code, err := s.Repo.UpdateURL(ctx, id, userID, longURL)
	if err != nil {
		return err
	}
	s.invalidate(ctx, code)
	return nil
}

func (s *URLService) DeleteUserURL(ctx context.Context, id, userID int64) error {
	code, err := s.Repo.DeleteURL(ctx, id, userID)
	if err != nil {
		return err
	}
	s.invalidate(ctx, code)
	return nil
}

func (s *URLService) BulkDeleteUserURLs(ctx context.Context, ids []int64, userID int64) error {
	codes, err := s.Repo.BulkDeleteURLs(ctx, ids, userID)
	if err != nil {
		return err
	}
	s.invalidate(ctx, codes...)
	return nil
}

func (s *URLService) invalidate(ctx context.Context, codes ...string) {
	if s.Cache != nil {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Second)
		defer cancel()
		if err := s.Cache.Invalidate(ctx, codes...); err != nil {
			log.Printf("URL cache invalidation failed: %v", err)
			if s.Metrics != nil {
				s.Metrics.CacheErrors.Inc()
			}
		}
	}
}

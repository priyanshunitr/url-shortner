package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gottatouchsomegrass/url/internal/model"
	"github.com/gottatouchsomegrass/url/internal/repository"
	"github.com/gottatouchsomegrass/url/internal/utils"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/mileusna/useragent"
)

type URLService struct {
	Repo    *repositories.URLQuery
	BaseURL string
}

func NewURLService(repo *repositories.URLQuery) *URLService {
	return &URLService{Repo: repo}
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
	if expiry != nil && !expiry.After(time.Now()) {
		return nil, errors.New("expires_at must be in the future")
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
	url, err := s.Repo.GetByShortURL(ctx, code)
	if err != nil {
		return "", errors.New("internal server error")
	}
	if url == nil {
		return "", errors.New("not found")
	}

	if url.Expiry != nil && time.Now().After(*url.Expiry) {
		return "", errors.New("link expired")
	}

	// increment clicks asynchronously or handle errors quietly
	go func() {
		_ = s.Repo.IncrementClicks(context.Background(), url.ID)
	}()

	ua := useragent.Parse(rawUserAgent)
	event := &models.ClickEvent{
		URLID:     url.ID,
		IPAddress: ip,
		UserAgent: rawUserAgent,
		Referer:   referer,
		Country:   "",
		Device:    ua.Device,
		Browser:   ua.Name,
	}

	// Insert analytics asynchronously to ensure redirect is lightning fast
	go func() {
		_ = s.Repo.CreateClickEvent(context.Background(), event)
	}()

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
	err := s.Repo.UpdateURL(ctx, id, userID, longURL)
	if err != nil {
		return err
	}
	return nil
}

func (s *URLService) DeleteUserURL(ctx context.Context, id, userID int64) error {
	err := s.Repo.DeleteURL(ctx, id, userID)
	if err != nil {
		return err
	}
	return nil
}

func (s *URLService) BulkDeleteUserURLs(ctx context.Context, ids []int64, userID int64) error {
	err := s.Repo.BulkDeleteURLs(ctx, ids, userID)
	if err != nil {
		return err
	}
	return nil
}

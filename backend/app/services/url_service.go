package services

import (
	"context"
	"errors"
	"time"

	"github.com/gottatouchsomegrass/url/app/models"
	"github.com/gottatouchsomegrass/url/app/repositories"
	"github.com/gottatouchsomegrass/url/pkg/utils"
	"github.com/mileusna/useragent"
)

type URLService struct {
	Repo *repositories.URLQuery
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

	shortened := customCode
	if shortened == "" {
		shortened = utils.GenerateShortCode()
	}

	exist, err := s.Repo.CustomCodeExists(ctx, shortened)
	if err != nil {
		return nil, errors.New("failed checking shortcode")
	}
	if exist {
		return nil, errors.New("custom code already exists")
	}

	expiry := time.Now().Add(24 * time.Hour)
	newURL := &models.URL{
		UserID:    userID,
		LongURL:   longURL,
		ShortURL:  shortened,
		Expiry:    &expiry,
		Clicks:    0,
		CreatedAt: time.Now(),
	}

	err = s.Repo.CreateURL(ctx, newURL)
	if err != nil {
		return nil, errors.New("failed to create url")
	}

	return newURL, nil
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

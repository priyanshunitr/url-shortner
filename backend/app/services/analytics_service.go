package services

import (
	"context"
	"errors"

	"github.com/gottatouchsomegrass/url/app/models"
	"github.com/gottatouchsomegrass/url/app/repositories"
)

type AnalyticsService struct {
	Repo *repositories.AnalyticsQuery
}

func NewAnalyticsService(repo *repositories.AnalyticsQuery) *AnalyticsService {
	return &AnalyticsService{Repo: repo}
}

func (s *AnalyticsService) authorizeURL(ctx context.Context, urlID, userID int64) error {
	url, err := s.Repo.GetURLByID(ctx, urlID)
	if err != nil {
		return errors.New("internal server error")
	}
	if url == nil {
		return errors.New("url not found")
	}
	if url.UserID != userID {
		return errors.New("forbidden")
	}
	return nil
}

func (s *AnalyticsService) GetAnalyticsOverview(ctx context.Context, urlID, userID int64) (*models.AnalyticsOverview, error) {
	if err := s.authorizeURL(ctx, urlID, userID); err != nil {
		return nil, err
	}
	return s.Repo.GetAnalyticsOverview(ctx, urlID)
}

func (s *AnalyticsService) GetDailyClicks(ctx context.Context, urlID, userID int64) ([]models.DailyClick, error) {
	if err := s.authorizeURL(ctx, urlID, userID); err != nil {
		return nil, err
	}
	return s.Repo.GetDailyClicks(ctx, urlID)
}

func (s *AnalyticsService) GetRecentVisits(ctx context.Context, urlID, userID int64) ([]models.RecentVisit, error) {
	if err := s.authorizeURL(ctx, urlID, userID); err != nil {
		return nil, err
	}
	return s.Repo.GetRecentVisits(ctx, urlID)
}

func (s *AnalyticsService) GetBrowserAnalytics(ctx context.Context, urlID, userID int64) ([]models.BrowserAnalytics, error) {
	if err := s.authorizeURL(ctx, urlID, userID); err != nil {
		return nil, err
	}
	return s.Repo.GetBrowserAnalytics(ctx, urlID)
}

func (s *AnalyticsService) GetDeviceAnalytics(ctx context.Context, urlID, userID int64) ([]models.DeviceAnalytics, error) {
	if err := s.authorizeURL(ctx, urlID, userID); err != nil {
		return nil, err
	}
	return s.Repo.GetDeviceAnalytics(ctx, urlID)
}

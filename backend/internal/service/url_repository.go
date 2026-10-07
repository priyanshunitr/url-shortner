package services

import (
	"context"

	"github.com/gottatouchsomegrass/url/internal/model"
)

// URLRepository describes the SQL operations needed by the URL service.
type URLRepository interface {
	CreateURL(context.Context, *models.URL) error
	GetByShortURL(context.Context, string) (*models.URL, error)
	CountUserURLs(context.Context, int64) (int, error)
	GetUserURLs(context.Context, int64, int, int) ([]models.URL, error)
	IncrementClicks(context.Context, int64) error
	CreateClickEvent(context.Context, *models.ClickEvent) error
	UpdateURL(context.Context, int64, int64, string) (string, error)
	DeleteURL(context.Context, int64, int64) (string, error)
	BulkDeleteURLs(context.Context, []int64, int64) ([]string, error)
	GetPublicStats(context.Context, string) (*models.URLStats, error)
}

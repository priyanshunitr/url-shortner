package repositories

import (
	"context"
	"errors"

	"github.com/gottatouchsomegrass/url/app/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AnalyticsQuery struct {
	DB *pgxpool.Pool
}

// GetAnalyticsOverview returns the analytics overview for a given URL ID
func (q *AnalyticsQuery) GetAnalyticsOverview(
	ctx context.Context,
	urlID int64,
) (*models.AnalyticsOverview, error) {
	query := `
		SELECT
			SUM(CASE WHEN created_at >= NOW() - INTERVAL '1 day' THEN 1 ELSE 0 END) AS today,
			SUM(CASE WHEN created_at >= NOW() - INTERVAL '7 days' THEN 1 ELSE 0 END) AS this_week,
			SUM(CASE WHEN created_at >= NOW() - INTERVAL '30 days' THEN 1 ELSE 0 END) AS this_month,
			COUNT(*) AS total_clicks
		FROM click_events
		WHERE url_id = $1
	`
	var overview models.AnalyticsOverview
	err := q.DB.QueryRow(ctx, query, urlID).Scan(
		&overview.Today,
		&overview.ThisWeek,
		&overview.ThisMonth,
		&overview.TotalClicks,
	)
	return &overview, err
}

// GetDailyClicks returns the daily click counts for a given URL ID
func (q *AnalyticsQuery) GetDailyClicks(
	ctx context.Context,
	urlID int64,
) ([]models.DailyClick, error) {
	query := `
		SELECT DATE(created_at) AS date, COUNT(*) AS clicks
		FROM click_events
		WHERE url_id = $1
		GROUP BY DATE(created_at)
		ORDER BY DATE(created_at);
	`
	rows, err := q.DB.Query(ctx, query, urlID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dailyClicks []models.DailyClick
	for rows.Next() {
		var click models.DailyClick
		if err := rows.Scan(&click.Date, &click.Clicks); err != nil {
			return nil, err
		}
		dailyClicks = append(dailyClicks, click)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return dailyClicks, nil
}

// GetRecentVisits returns the recent visits for a given URL ID
func (q *AnalyticsQuery) GetRecentVisits(
	ctx context.Context,
	urlID int64,
) ([]models.RecentVisit, error) {
	query := `
		SELECT
			ip_address,
			referer,
			browser,
			device,
			created_at
		FROM click_events
		WHERE url_id = $1
		ORDER BY created_at DESC
		LIMIT 20
	`

	rows, err := q.DB.Query(ctx, query, urlID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var visits []models.RecentVisit

	for rows.Next() {
		var visit models.RecentVisit

		err := rows.Scan(
			&visit.IPAddress,
			&visit.Referer,
			&visit.Browser,
			&visit.Device,
			&visit.CreatedAt,
		)

		if err != nil {
			return nil, err
		}

		visits = append(visits, visit)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return visits, nil
}

func (q *AnalyticsQuery) GetBrowserAnalytics(
	ctx context.Context,
	urlID int64,
) ([]models.BrowserAnalytics, error) {
	query := `
		SELECT browser, COUNT(*)
		FROM click_events
		WHERE url_id = $1
		GROUP BY browser
		ORDER BY COUNT(*) DESC;
	`

	var analytics []models.BrowserAnalytics

	rows, err := q.DB.Query(
		ctx,
		query,
		urlID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var browserAnalytics models.BrowserAnalytics
		err := rows.Scan(
			&browserAnalytics.Browser,
			&browserAnalytics.Count,
		)
		if err != nil {
			return nil, err
		}
		analytics = append(analytics, browserAnalytics)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return analytics, nil
}

func (q *AnalyticsQuery) GetDeviceAnalytics(
	ctx context.Context,
	urlID int64,
) ([]models.DeviceAnalytics, error) {
	query := `
		SELECT device, COUNT(*)
		FROM click_events
		WHERE url_id = $1
		GROUP BY device
		ORDER BY COUNT(*) DESC;
	`

	var analytics []models.DeviceAnalytics

	rows, err := q.DB.Query(
		ctx,
		query,
		urlID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var deviceAnalytics models.DeviceAnalytics
		err := rows.Scan(
			&deviceAnalytics.Device,
			&deviceAnalytics.Count,
		)
		if err != nil {
			return nil, err
		}
		analytics = append(analytics, deviceAnalytics)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return analytics, nil
}

// GetURLByID returns the URL for a given URL ID
func (q *AnalyticsQuery) GetURLByID(
	ctx context.Context,
	urlID int64,
) (*models.URL, error) {
	query := `
		SELECT
			id,
			user_id,
			long_url,
			short_url,
			expiry,
			clicks,
			created_at
		FROM urls
		WHERE id = $1
	`

	var url models.URL

	err := q.DB.QueryRow(
		ctx,
		query,
		urlID,
	).Scan(
		&url.ID,
		&url.UserID,
		&url.LongURL,
		&url.ShortURL,
		&url.Expiry,
		&url.Clicks,
		&url.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}

		return nil, err
	}

	return &url, nil
}

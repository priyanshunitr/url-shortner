// Package queries have caching and db layer
package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/gottatouchsomegrass/url/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type URLQuery struct {
	DB *pgxpool.Pool
}

// CreateURL insert url to db
func (q *URLQuery) CreateURL(ctx context.Context, url *models.URL) error {
	query := `
		INSERT INTO urls (
			user_id,
			short_code,
			original_url,
			expires_at
		)
		VALUES (NULLIF($1, 0), $2, $3, $4)
		RETURNING id, created_at
	`

	err := q.DB.QueryRow(
		ctx,
		query,
		url.UserID,
		url.ShortURL,
		url.LongURL,
		url.Expiry,
	).Scan(&url.ID, &url.CreatedAt)

	return err
}

// GetPublicStats exposes statistics only for URLs created by the public API.
func (q *URLQuery) GetPublicStats(ctx context.Context, code string) (*models.URLStats, error) {
	stats := &models.URLStats{TopReferrers: []models.Referrer{}}
	var id int64
	err := q.DB.QueryRow(ctx, `SELECT id, short_code, click_count, created_at, last_accessed, expires_at
		FROM urls WHERE short_code = $1 AND user_id IS NULL`, code).Scan(
		&id, &stats.ShortCode, &stats.Clicks, &stats.CreatedAt, &stats.LastAccessed, &stats.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	rows, err := q.DB.Query(ctx, `SELECT COALESCE(NULLIF(referer, ''), 'Direct'), COUNT(*)
		FROM click_events WHERE url_id = $1 GROUP BY 1 ORDER BY 2 DESC, 1 LIMIT 10`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var ref models.Referrer
		if err := rows.Scan(&ref.Source, &ref.Clicks); err != nil {
			return nil, err
		}
		stats.TopReferrers = append(stats.TopReferrers, ref)
	}
	return stats, rows.Err()
}

// GetByShortURL get by short url from db
func (q *URLQuery) GetByShortURL(ctx context.Context, code string) (*models.URL, error) {
	query := `
		SELECT id, COALESCE(user_id, 0), short_code, original_url, expires_at, click_count, created_at
		FROM urls
		WHERE short_code = $1
	`

	var url models.URL

	err := q.DB.QueryRow(ctx, query, code).Scan(
		&url.ID,
		&url.UserID,
		&url.ShortURL,
		&url.LongURL,
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

// CustomCodeExists check whether customcode exists or not
func (q *URLQuery) CustomCodeExists(ctx context.Context, code string) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1 FROM urls WHERE short_code = $1
		)
	`
	var isexist bool

	err := q.DB.QueryRow(ctx, query, code).Scan(&isexist)

	return isexist, err
}

// IncrementClicks increments the click count for a given URL ID
func (q *URLQuery) IncrementClicks(ctx context.Context, id int64) error {
	query := `
		UPDATE urls
		SET click_count = click_count + 1
		WHERE id = $1
	`
	result, err := q.DB.Exec(ctx, query, id)
	if err != nil {
		fmt.Println("qery err in IncrementClicks:", err)
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("url not found")
	}
	return nil
}

// CreateClickEvent inserts a new click event into the database
func (q *URLQuery) CreateClickEvent(
	ctx context.Context,
	event *models.ClickEvent,
) error {
	query := `
		INSERT INTO click_events (url_id, ip_address, user_agent, referer, country, device, browser)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := q.DB.Exec(ctx, query,
		event.URLID,
		event.IPAddress,
		event.UserAgent,
		event.Referer,
		event.Country,
		event.Device,
		event.Browser,
	)
	return err
}

// GetUserURLs returns all URLs created by a specific user, with pagination
func (q *URLQuery) GetUserURLs(ctx context.Context, userID int64, limit int, offset int) ([]models.URL, error) {
	query := `
		SELECT id, COALESCE(user_id, 0), short_code, original_url, expires_at, click_count, created_at
		FROM urls
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := q.DB.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var urls []models.URL
	for rows.Next() {
		var url models.URL
		err := rows.Scan(
			&url.ID,
			&url.UserID,
			&url.ShortURL,
			&url.LongURL,
			&url.Expiry,
			&url.Clicks,
			&url.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		urls = append(urls, url)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return urls, nil
}

// CountUserURLs returns the total number of URLs created by a user
func (q *URLQuery) CountUserURLs(ctx context.Context, userID int64) (int, error) {
	query := `SELECT COUNT(*) FROM urls WHERE user_id = $1`
	var total int
	err := q.DB.QueryRow(ctx, query, userID).Scan(&total)
	return total, err
}

// DeleteURL returns the shortcode invalidated by this authorized mutation.
func (q *URLQuery) DeleteURL(ctx context.Context, id, userID int64) (string, error) {
	var code string
	err := q.DB.QueryRow(ctx, `DELETE FROM urls WHERE id = $1 AND user_id = $2 RETURNING short_code`, id, userID).Scan(&code)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", errors.New("url not found or unauthorized")
	}
	return code, err
}

func (q *URLQuery) UpdateURL(ctx context.Context, id, userID int64, longURL string) (string, error) {
	var code string
	err := q.DB.QueryRow(ctx, `UPDATE urls SET original_url = $3 WHERE id = $1 AND user_id = $2 RETURNING short_code`, id, userID, longURL).Scan(&code)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", errors.New("url not found or unauthorized")
	}
	return code, err
}

func (q *URLQuery) BulkDeleteURLs(ctx context.Context, ids []int64, userID int64) ([]string, error) {
	rows, err := q.DB.Query(ctx, `DELETE FROM urls WHERE id = ANY($1) AND user_id = $2 RETURNING short_code`, ids, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var codes []string
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, err
		}
		codes = append(codes, code)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(codes) == 0 {
		return nil, errors.New("no urls found or unauthorized")
	}
	return codes, nil
}

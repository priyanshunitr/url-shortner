// Package queries have caching and db layer
package repositories

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/gottatouchsomegrass/url/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type URLQuery struct {
	DB  *pgxpool.Pool
	RDB *redis.Client
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
	//cache-aside implementation
	cached, err := q.RDB.Get(ctx, code).Result()
	if err == nil {
		return &models.URL{
			ShortURL: code,
			LongURL:  cached,
		}, nil
	}
	if err != redis.Nil {
		log.Println("redis err:", err)
	}

	query := `
		SELECT id, COALESCE(user_id, 0), short_code, original_url, expires_at, click_count, created_at
		FROM urls
		WHERE short_code = $1
	`

	var url models.URL

	err = q.DB.QueryRow(ctx, query, code).Scan(
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

	ttl := time.Hour

	if url.Expiry != nil {
		remaining := time.Until(*url.Expiry)
		if remaining <= 0 {
			return nil, errors.New("link expired")
		}
		if remaining <= ttl {
			ttl = remaining
		}
	}
	err = q.RDB.Set(
		ctx,
		code,
		url.LongURL,
		ttl,
	).Err()

	if err != nil {
		log.Println("redis set err:", err)
	}

	return &url, nil
}

// CustomCodeExists check whether customcode exists or not
func (q *URLQuery) CustomCodeExists(ctx context.Context, code string) (bool, error) {
	// _, err := q.RDB.Get(ctx,code).Result()
	// if err==nil {
	// 	return true, nil
	// }
	// if err!=redis.Nil {
	// 	return false, err
	// }
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

// DeleteURL deletes a URL, ensuring it belongs to the user
func (q *URLQuery) DeleteURL(ctx context.Context, id int64, userID int64) error {
	query := `
		DELETE FROM urls WHERE id = $1 AND user_id = $2
	`
	res, err := q.DB.Exec(ctx, query, id, userID)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("url not found or unauthorized")
	}
	return nil
}

// UpdateURL updates the original_url of a shortcode, ensuring it belongs to the user
func (q *URLQuery) UpdateURL(ctx context.Context, id int64, userID int64, longURL string) error {
	query := `
		UPDATE urls SET original_url = $3 WHERE id = $1 AND user_id = $2
	`
	res, err := q.DB.Exec(ctx, query, id, userID, longURL)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("url not found or unauthorized")
	}
	return nil
}

// BulkDeleteURLs deletes multiple URLs, ensuring they belong to the user
func (q *URLQuery) BulkDeleteURLs(ctx context.Context, ids []int64, userID int64) error {
	query := `
		DELETE FROM urls WHERE id = ANY($1) AND user_id = $2
	`
	res, err := q.DB.Exec(ctx, query, ids, userID)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("no urls found or unauthorized")
	}
	return nil
}

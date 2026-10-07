// Package queries have caching and db layer
package repositories

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/gottatouchsomegrass/url/app/models"
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
			short_url,
			long_url,
			expiry
		)
		VALUES ($1, $2, $3, $4)
	`

	_, err := q.DB.Exec(
		ctx,
		query,
		url.UserID,
		url.ShortURL,
		url.LongURL,
		url.Expiry,
	)

	return err
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
		SELECT id, user_id, short_url, long_url, expiry, clicks, created_at
		FROM urls
		WHERE short_url = $1
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
			SELECT 1 FROM urls WHERE short_url = $1
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
		SET clicks = clicks + 1
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
		SELECT id, user_id, short_url, long_url, expiry, clicks, created_at
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

// UpdateURL updates the long_url of a shortcode, ensuring it belongs to the user
func (q *URLQuery) UpdateURL(ctx context.Context, id int64, userID int64, longURL string) error {
	query := `
		UPDATE urls SET long_url = $3 WHERE id = $1 AND user_id = $2
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

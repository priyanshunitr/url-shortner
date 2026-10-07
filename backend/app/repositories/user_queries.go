package repositories

import (
	"context"
	"errors"
	"time"

	"github.com/gottatouchsomegrass/url/app/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/redis/go-redis/v9"
)

type DBTX interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Begin(ctx context.Context) (pgx.Tx, error)
}

type UserQuery struct {
	DB  DBTX
	RDB *redis.Client
}

func NewUserQueries(DB DBTX) *UserQuery {
	return &UserQuery{
		DB: DB,
	}
}

func (q *UserQuery) WithTx(tx pgx.Tx) *UserQuery {
	return &UserQuery{
		DB:  tx,    // Replace the pool with the transaction
		RDB: q.RDB, // Keep the redis connection
	}
}

// CreateUser inserts a new user into the database.
func (q *UserQuery) CreateUser(
	ctx context.Context,
	user *models.User,
) error {
	query := `
		INSERT INTO users (
			email,
			password_hash
		)
		VALUES ($1, $2)
		RETURNING id, role, created_at
	`

	return q.DB.QueryRow(
		ctx,
		query,
		user.Email,
		user.PasswordHash,
	).Scan(
		&user.ID,
		&user.Role,
		&user.CreatedAt,
	)
}

// GetUserByEmail retrieves a user by their email address.
func (q *UserQuery) GetUserByEmail(
	ctx context.Context,
	email string,
) (*models.User, error) {
	query := `
		SELECT
			id,
			email,
			password_hash,
			role,
			created_at
		FROM users
		WHERE email = $1
	`

	var user models.User

	err := q.DB.QueryRow(
		ctx,
		query,
		email,
	).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

// GetUserByID retrieves a user by their ID.
func (q *UserQuery) GetUserByID(
	ctx context.Context,
	userID int64,
) (*models.User, error) {
	query := `
		SELECT
			id,
			email,
			role,
			created_at
		FROM users
		WHERE id = $1
	`

	var user models.User

	err := q.DB.QueryRow(
		ctx,
		query,
		userID,
	).Scan(
		&user.ID,
		&user.Email,
		&user.Role,
		&user.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

// GetAllUsers retrieves a paginated list of all users.
func (q *UserQuery) GetAllUsers(ctx context.Context, limit, offset int) ([]models.User, error) {
	query := `
		SELECT id, email, role, created_at
		FROM users
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := q.DB.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		if err := rows.Scan(&user.ID, &user.Email, &user.Role, &user.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}

// CountAllUsers returns the total number of registered users.
func (q *UserQuery) CountAllUsers(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM users`
	var total int
	err := q.DB.QueryRow(ctx, query).Scan(&total)
	return total, err
}

// DeleteUserAccount completely removes a user and all their associated data (URLs and click events).
func (q *UserQuery) DeleteUserAccount(ctx context.Context, userID int64) error {
	tx, err := q.DB.Begin(ctx)
	if err != nil {
		return err
	}
	// Defer a rollback in case anything fails. If tx.Commit() succeeds, rollback does nothing.
	defer func() { _ = tx.Rollback(ctx) }()

	// 1. Delete all click events for the user's URLs
	_, err = tx.Exec(ctx, `
		DELETE FROM click_events
		WHERE url_id IN (SELECT id FROM urls WHERE user_id = $1)
	`, userID)
	if err != nil {
		return err
	}

	// 2. Delete all URLs for the user
	_, err = tx.Exec(ctx, `
		DELETE FROM urls
		WHERE user_id = $1
	`, userID)
	if err != nil {
		return err
	}

	// 3. Delete the user
	_, err = tx.Exec(ctx, `
		DELETE FROM users
		WHERE id = $1
	`, userID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// InvalidateToken adds a JWT token to the Redis blacklist with the given expiration time.
func (q *UserQuery) InvalidateToken(ctx context.Context, tokenString string, expiration time.Duration) error {
	if q.RDB == nil {
		return nil // skip if redis is not configured
	}
	// Use the token string as the key and a simple boolean "1" as the value
	key := "blacklist:" + tokenString
	return q.RDB.Set(ctx, key, "1", expiration).Err()
}

// IsTokenBlacklisted checks if a JWT token is in the Redis blacklist.
func (q *UserQuery) IsTokenBlacklisted(ctx context.Context, tokenString string) (bool, error) {
	if q.RDB == nil {
		return false, nil
	}
	key := "blacklist:" + tokenString
	res, err := q.RDB.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, nil // Token is not blacklisted
	} else if err != nil {
		return false, err // Redis error
	}
	return res == "1", nil
}

// CreateRefreshToken creates a new refresh token.
func (q *UserQuery) CreateRefreshToken(ctx context.Context, rt *models.RefreshToken) error {
	query := `
		INSERT INTO refresh_tokens (user_id, refresh_hash, expires_at, ip, user_agent)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`
	return q.DB.QueryRow(
		ctx,
		query,
		rt.UserID,
		rt.RefreshHash,
		rt.ExpiresAt,
		rt.IP,
		rt.UserAgent,
	).Scan(&rt.ID, &rt.CreatedAt)
}

// EnforceMaxRefreshTokens enforces a maximum number of refresh token sessions per user.
func (q *UserQuery) EnforceMaxRefreshTokens(ctx context.Context, userID int64, max int) error {
	query := `
		DELETE FROM refresh_tokens
		WHERE id NOT IN (
			SELECT id FROM refresh_tokens
			WHERE user_id = $1
			ORDER BY last_used_at DESC NULLS LAST, created_at DESC
			LIMIT $2
		) AND user_id = $1
	`
	_, err := q.DB.Exec(ctx, query, userID, max)
	return err
}

// GetRefreshToken retrieves a refresh token by its hash.
func (q *UserQuery) GetRefreshToken(ctx context.Context, hash string) (*models.RefreshToken, error) {
	query := `
		SELECT id, user_id, refresh_hash, expires_at, last_used_at, ip, user_agent, created_at, revoked_at
		FROM refresh_tokens
		WHERE refresh_hash = $1
	`
	var rt models.RefreshToken
	err := q.DB.QueryRow(ctx, query, hash).Scan(
		&rt.ID, &rt.UserID, &rt.RefreshHash, &rt.ExpiresAt, &rt.LastUsedAt, &rt.IP, &rt.UserAgent, &rt.CreatedAt, &rt.RevokedAt,
	)
	if err != nil {
		return nil, err
	}
	return &rt, nil
}

// GetUserSessions retrieves all active refresh token sessions for a user.
func (q *UserQuery) GetUserSessions(ctx context.Context, userID int64) ([]models.RefreshToken, error) {
	query := `
		SELECT id, user_id, refresh_hash, expires_at, last_used_at, ip, user_agent, created_at, revoked_at
		FROM refresh_tokens
		WHERE user_id = $1 AND revoked_at IS NULL AND expires_at > NOW()
		ORDER BY last_used_at DESC NULLS LAST
	`
	rows, err := q.DB.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []models.RefreshToken
	for rows.Next() {
		var rt models.RefreshToken
		if err := rows.Scan(&rt.ID, &rt.UserID, &rt.RefreshHash, &rt.ExpiresAt, &rt.LastUsedAt, &rt.IP, &rt.UserAgent, &rt.CreatedAt, &rt.RevokedAt); err != nil {
			return nil, err
		}
		sessions = append(sessions, rt)
	}
	return sessions, rows.Err()
}

// UpdateRefreshTokenUsed updates the last_used_at field of a refresh token.
func (q *UserQuery) UpdateRefreshTokenUsed(ctx context.Context, id int64) error {
	query := `UPDATE refresh_tokens SET last_used_at = NOW() WHERE id = $1`
	_, err := q.DB.Exec(ctx, query, id)
	return err
}

// RevokeRefreshToken revokes a refresh token.
func (q *UserQuery) RevokeRefreshToken(ctx context.Context, hash string) error {
	query := `UPDATE refresh_tokens SET revoked_at = NOW() WHERE refresh_hash = $1`
	_, err := q.DB.Exec(ctx, query, hash)
	return err
}

// RevokeAllOtherSessions revokes all sessions for a user except the specified session.
func (q *UserQuery) RevokeAllOtherSessions(ctx context.Context, userID int64, currentSessionID int64) error {
	query := `UPDATE refresh_tokens SET revoked_at = NOW() WHERE user_id = $1 AND id != $2 AND revoked_at IS NULL`
	_, err := q.DB.Exec(ctx, query, userID, currentSessionID)
	return err
}

// DeleteRefreshToken deletes a refresh token.
func (q *UserQuery) DeleteRefreshToken(ctx context.Context, id int64) error {
	query := `DELETE FROM refresh_tokens WHERE id = $1`
	_, err := q.DB.Exec(ctx, query, id)
	return err
}

func (q *UserQuery) RotateRefreshToken(
	ctx context.Context,
	sessionID int64,
	newHash string,
	expiresAt time.Time,
	ip string,
	userAgent string,
) error {
	query := `UPDATE refresh_tokens
	SET
    refresh_hash = $1,
    expires_at = $2,
    last_used_at = NOW(),
    ip = $3,
    user_agent = $4
	WHERE id = $5;`

	cmd, err := q.DB.Exec(ctx, query, newHash, expiresAt, ip, userAgent, sessionID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() != 1 {
		return errors.New("refresh session not found")
	}
	return nil
}

// IncrementLoginAttempt increments the login attempt counter for a specific key.
// If the key is new, it sets the expiration time for the time window.
func (q *UserQuery) IncrementLoginAttempt(ctx context.Context, key string, window time.Duration) (int64, error) {
	if q.RDB == nil {
		return 0, nil // Skip if Redis is not configured
	}

	// Increment the key
	count, err := q.RDB.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}

	// If this is the first attempt, set the expiration window
	if count == 1 {
		q.RDB.Expire(ctx, key, window)
	}

	return count, nil
}

// ResetLoginAttempts deletes the rate limit keys upon a successful login.
func (q *UserQuery) ResetLoginAttempts(ctx context.Context, keys ...string) error {
	if q.RDB == nil {
		return nil
	}
	return q.RDB.Del(ctx, keys...).Err()
}

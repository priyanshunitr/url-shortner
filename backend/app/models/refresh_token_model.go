package models

import "time"

type RefreshToken struct {
	ID          int64      `json:"id" db:"id"`
	UserID      int64      `json:"user_id" db:"user_id"`
	RefreshHash string     `json:"-" db:"refresh_hash"`
	ExpiresAt   time.Time  `json:"expires_at" db:"expires_at"`
	LastUsedAt  *time.Time `json:"last_used_at" db:"last_used_at"`
	IP          string     `json:"ip" db:"ip"`
	UserAgent   string     `json:"user_agent" db:"user_agent"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	RevokedAt   *time.Time `json:"revoked_at" db:"revoked_at"`
}

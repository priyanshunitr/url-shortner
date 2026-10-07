package models

import "time"

// CoreURLResponse is the public create response, independent of legacy DTOs.
type CoreURLResponse struct {
	ShortCode string `json:"short_code"`
	ShortURL  string `json:"short_url"`
}

type URLStats struct {
	ShortCode    string     `json:"short_code"`
	Clicks       int64      `json:"clicks"`
	CreatedAt    time.Time  `json:"created_at"`
	LastAccessed *time.Time `json:"last_accessed"`
	ExpiresAt    *time.Time `json:"expires_at"`
	TopReferrers []Referrer `json:"top_referrers"`
}

type Referrer struct {
	Source string `json:"source"`
	Clicks int64  `json:"clicks"`
}

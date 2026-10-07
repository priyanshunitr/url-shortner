// Package models inclue all the models used
package models

import (
	"time"
	//"github.com/google/uuid"
)

type URL struct {
	ID        int64      `json:"id" db:"id"` //go for uuid later
	UserID    int64      `json:"user_id" db:"user_id"`
	LongURL   string     `json:"long_url" db:"long_url" binding:"required,url"`
	ShortURL  string     `json:"short_url" db:"short_url" binding:"required,shortcode"`
	Expiry    *time.Time `json:"expiry,omitempty" db:"expiry"`
	Clicks    int64      `json:"clicks" db:"clicks"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
}

type ClickEvent struct {
	ID        int64     `json:"id" db:"id"`
	URLID     int64     `json:"url_id" db:"url_id" binding:"required"`
	IPAddress string    `json:"ip_address,omitempty" db:"ip_address"`
	UserAgent string    `json:"user_agent,omitempty" db:"user_agent"`
	Referer   string    `json:"referer,omitempty" db:"referer"`
	Country   string    `json:"country,omitempty" db:"country"`
	Device    string    `json:"device,omitempty" db:"device"`
	Browser   string    `json:"browser,omitempty" db:"browser"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

//manual
// func (u URL) Value() (driver.Value, error) {
// 	return json.Marshal(u)
// }

// func (u *URL) Scan(value any) error {
// 	j, ok := value.([]byte)
// 	if !ok {
// 		return errors.New("type assertion to []byte failed")
// 	}

// 	return json.Unmarshal(j, &u)
// }

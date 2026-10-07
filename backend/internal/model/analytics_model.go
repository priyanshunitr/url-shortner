package models

import "time"

type AnalyticsOverview struct {
	TotalClicks int64 `json:"total_clicks"`
	Today       int64 `json:"today"`
	ThisWeek    int64 `json:"this_week"`
	ThisMonth   int64 `json:"this_month"`
}

type DailyClick struct {
	Date   string `json:"date"`
	Clicks int64  `json:"clicks"`
}

type BrowserAnalytics struct {
	Browser string `json:"browser"`
	Count   int64  `json:"count"`
}

type DeviceAnalytics struct {
	Device string `json:"device"`
	Count  int64  `json:"count"`
}

type RecentVisit struct {
	IPAddress string    `json:"ip_address"`
	Referer   string    `json:"referer"`
	Browser   string    `json:"browser"`
	Device    string    `json:"device"`
	CreatedAt time.Time `json:"created_at"`
}

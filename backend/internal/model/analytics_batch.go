package models

import "time"

type ClickAggregate struct {
	Count        int64
	LastAccessed time.Time
}

type AnalyticsBatch struct {
	ID     string
	Counts map[int64]ClickAggregate
	Events []ClickEvent
}

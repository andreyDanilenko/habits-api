package model

import "time"

type LogEntry struct {
	Timestamp  time.Time     `json:"timestamp"`
	StatusCode int           `json:"status_code"`
	Duration   time.Duration `json:"duration"`
	ClientIP   string        `json:"client_ip"`
	Method     string        `json:"method"`
	Path       string        `json:"path"`
	RawLog     string        `json:"raw_log"`
}

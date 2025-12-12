package models

import (
	"time"
)

type Proxy struct {
	ID           uint      `json:"id"`
	ProxyStr     string    `json:"proxy_str"`
	APIKey       string    `json:"api_key"`
	ServiceType  string    `json:"service_type"` // "tmproxy" or "kiotproxy"
	MinTimeReset int       `json:"min_time_reset"` // Seconds
	LastResetAt  time.Time `json:"last_reset_at"`
	CreatedAt    time.Time `json:"created_at"`
}

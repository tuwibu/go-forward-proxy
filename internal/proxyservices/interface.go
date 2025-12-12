package proxyservices

import "time"

// ProxyInfo holds the proxy information returned by external services
type ProxyInfo struct {
	ProxyStr       string    // Format: "ip:port:username:password" or "ip:port"
	ServiceType    string    // "tmproxy" or "kiotproxy"
	NextResetAfter int       // Seconds until next reset is allowed
	ExpiresAt      time.Time // When proxy expires
}

// ProxyService defines the interface for external proxy services
type ProxyService interface {
	GetCurrentProxy(apiKey string) (*ProxyInfo, error)
	GetNewProxy(apiKey string) (*ProxyInfo, error)
	GetServiceType() string
}

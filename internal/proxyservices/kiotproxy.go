package proxyservices

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const (
	kiotproxyGetNewURL     = "https://api.kiotproxy.com/api/v1/proxies/new"
	kiotproxyGetCurrentURL = "https://api.kiotproxy.com/api/v1/proxies/current"
)

type KiotProxyService struct {
	httpClient *http.Client
}

func NewKiotProxyService() *KiotProxyService {
	return &KiotProxyService{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (s *KiotProxyService) GetServiceType() string {
	return "kiotproxy"
}

type kiotproxyNewProxyResponse struct {
	Success bool   `json:"success"`
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		HTTP   string `json:"http"`   // Format: "ip:port"
		SOCKS5 string `json:"socks5"` // Format: "ip:port"
	} `json:"data"`
}

type kiotproxyCurrentProxyResponse struct {
	Success bool   `json:"success"`
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		NextRequestAt int64  `json:"nextRequestAt"` // Unix timestamp in milliseconds
		ExpirationAt  int64  `json:"expirationAt"`  // Unix timestamp in milliseconds
		HTTP          string `json:"http"`          // Format: "ip:port"
		TTC           int    `json:"ttc"`           // Time to change in seconds
	} `json:"data"`
}

func (s *KiotProxyService) GetCurrentProxy(apiKey string) (*ProxyInfo, error) {
	// Make HTTP request with API key as query parameter
	url := fmt.Sprintf("%s?key=%s", kiotproxyGetCurrentURL, apiKey)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var kiotResp kiotproxyCurrentProxyResponse
	if err := json.NewDecoder(resp.Body).Decode(&kiotResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !kiotResp.Success || kiotResp.Code != 200 {
		return nil, fmt.Errorf("kiotproxy API error: %s", kiotResp.Message)
	}

	// Convert unix milliseconds to time.Time
	expiresAt := time.Unix(0, kiotResp.Data.ExpirationAt*int64(time.Millisecond))

	return &ProxyInfo{
		ProxyStr:       kiotResp.Data.HTTP,
		ServiceType:    "kiotproxy",
		NextResetAfter: kiotResp.Data.TTC,
		ExpiresAt:      expiresAt,
	}, nil
}

func (s *KiotProxyService) GetNewProxy(apiKey string) (*ProxyInfo, error) {
	// Make HTTP request with API key as query parameter
	url := fmt.Sprintf("%s?key=%s", kiotproxyGetNewURL, apiKey)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var kiotResp kiotproxyNewProxyResponse
	if err := json.NewDecoder(resp.Body).Decode(&kiotResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !kiotResp.Success || kiotResp.Code != 200 {
		return nil, fmt.Errorf("kiotproxy API error: %s", kiotResp.Message)
	}

	// Format proxy string: "ip:port" (no separate credentials for KiotProxy)
	// Use HTTP proxy format (can also use SOCKS5)
	proxyStr := kiotResp.Data.HTTP

	// Note: GetNewProxy response doesn't include NextResetAfter and ExpiresAt
	// These fields should be populated by calling GetCurrentProxy after getting new proxy
	return &ProxyInfo{
		ProxyStr:    proxyStr,
		ServiceType: "kiotproxy",
	}, nil
}

package proxyservices

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const (
	tmproxyGetNewURL     = "https://tmproxy.com/api/proxy/get-new-proxy"
	tmproxyGetCurrentURL = "https://tmproxy.com/api/proxy/get-current-proxy"
)

type TMProxyService struct {
	httpClient *http.Client
}

func NewTMProxyService() *TMProxyService {
	return &TMProxyService{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (s *TMProxyService) GetServiceType() string {
	return "tmproxy"
}

type tmproxyRequest struct {
	APIKey     string `json:"api_key"`
	IDLocation int    `json:"id_location,omitempty"`
	IDISP      int    `json:"id_isp,omitempty"`
}

type tmproxyNewProxyResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Username string `json:"username"`
		Password string `json:"password"`
		HTTPS    string `json:"https"` // Format: "ip:port"
		SOCKS5   string `json:"socks5"` // Format: "ip:port"
	} `json:"data"`
}

type tmproxyCurrentProxyResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		NextRequest int    `json:"next_request"` // Seconds
		ExpiredAt   string `json:"expired_at"`   // ISO timestamp
		HTTPS       string `json:"https"`        // Format: "ip:port"
		Username    string `json:"username"`
		Password    string `json:"password"`
	} `json:"data"`
}

func (s *TMProxyService) GetCurrentProxy(apiKey string) (*ProxyInfo, error) {
	// Prepare request
	reqBody := tmproxyRequest{
		APIKey: apiKey,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Make HTTP request
	req, err := http.NewRequest("POST", tmproxyGetCurrentURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var tmResp tmproxyCurrentProxyResponse
	if err := json.NewDecoder(resp.Body).Decode(&tmResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if tmResp.Code != 200 {
		return nil, fmt.Errorf("tmproxy API error: %s", tmResp.Message)
	}

	// Parse expired_at timestamp
	expiresAt, err := time.Parse(time.RFC3339, tmResp.Data.ExpiredAt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse expired_at: %w", err)
	}

	// Format proxy string: "ip:port:username:password"
	proxyStr := fmt.Sprintf("%s:%s:%s",
		tmResp.Data.HTTPS,
		tmResp.Data.Username,
		tmResp.Data.Password,
	)

	return &ProxyInfo{
		ProxyStr:       proxyStr,
		ServiceType:    "tmproxy",
		NextResetAfter: tmResp.Data.NextRequest,
		ExpiresAt:      expiresAt,
	}, nil
}

func (s *TMProxyService) GetNewProxy(apiKey string) (*ProxyInfo, error) {
	// Prepare request
	reqBody := tmproxyRequest{
		APIKey:     apiKey,
		IDLocation: 1, // Default location
		IDISP:      1, // Default ISP
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Make HTTP request
	req, err := http.NewRequest("POST", tmproxyGetNewURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var tmResp tmproxyNewProxyResponse
	if err := json.NewDecoder(resp.Body).Decode(&tmResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if tmResp.Code != 200 {
		return nil, fmt.Errorf("tmproxy API error: %s", tmResp.Message)
	}

	// Format proxy string: "ip:port:username:password"
	// Use HTTPS proxy format (can also use SOCKS5)
	proxyStr := fmt.Sprintf("%s:%s:%s",
		tmResp.Data.HTTPS,
		tmResp.Data.Username,
		tmResp.Data.Password,
	)

	// Note: GetNewProxy response doesn't include NextResetAfter and ExpiresAt
	// These fields should be populated by calling GetCurrentProxy after getting new proxy
	return &ProxyInfo{
		ProxyStr:    proxyStr,
		ServiceType: "tmproxy",
	}, nil
}

package proxyservices

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

const (
	tmproxyGetNewURL     = "https://tmproxy.com/api/proxy/get-new-proxy"
	tmproxyGetCurrentURL = "https://tmproxy.com/api/proxy/get-current-proxy"
)

var (
	ErrNoCurrentProxy = errors.New("no current proxy available, need to call GetNewProxy")  // Exported (starts with uppercase)
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
		Username    string `json:"username"`
		Password    string `json:"password"`
		HTTPS       string `json:"https"`        // Format: "ip:port"
		SOCKS5      string `json:"socks5"`      // Format: "ip:port"
		NextRequest int    `json:"next_request"` // Seconds
		ExpiredAt   string `json:"expired_at"`   // Format: "HH:MM:SS MM/DD/YYYY"
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

	// Check for API error (code != 0 means error)
	if tmResp.Code != 0 {
		// Special handling for code=27 (no current proxy available)
		if tmResp.Code == 27 {
			return nil, ErrNoCurrentProxy
		}
		return nil, fmt.Errorf("tmproxy API error: code=%d, message=%s", tmResp.Code, tmResp.Message)
	}

	// Parse expired_at timestamp (format: "HH:MM:SS DD/MM/YYYY")
	expiresAt, err := time.Parse("15:04:05 02/01/2006", tmResp.Data.ExpiredAt)
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

	// Check for API error (code != 0 means error)
	if tmResp.Code != 0 {
		return nil, fmt.Errorf("tmproxy API error: code=%d, message=%s", tmResp.Code, tmResp.Message)
	}

	// Format proxy string: "ip:port:username:password"
	// Use HTTPS proxy format (can also use SOCKS5)
	proxyStr := fmt.Sprintf("%s:%s:%s",
		tmResp.Data.HTTPS,
		tmResp.Data.Username,
		tmResp.Data.Password,
	)

	// Parse expired_at timestamp (format: "HH:MM:SS DD/MM/YYYY")
	expiresAt, err := time.Parse("15:04:05 02/01/2006", tmResp.Data.ExpiredAt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse expired_at: %w", err)
	}

	return &ProxyInfo{
		ProxyStr:       proxyStr,
		ServiceType:    "tmproxy",
		NextResetAfter: tmResp.Data.NextRequest,
		ExpiresAt:      expiresAt,
	}, nil
}

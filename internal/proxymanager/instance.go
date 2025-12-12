package proxymanager

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"

	"go-forward-proxy/internal/config"
	"go-forward-proxy/pkg/dumbproxy/auth"
	"go-forward-proxy/pkg/dumbproxy/dialer"
	"go-forward-proxy/pkg/dumbproxy/forward"
	"go-forward-proxy/pkg/dumbproxy/handler"
	clog "go-forward-proxy/pkg/dumbproxy/log"
)

type ProxyInstance struct {
	ProxyID     uint
	Port        int
	ServiceType string
	Listener    net.Listener
	Handler     *handler.ProxyHandler
	Server      *http.Server
	Cancel      context.CancelFunc
	mu          sync.RWMutex
	logger      *clog.CondLogger
	auth        auth.Auth
	dialer      dialer.Dialer
}

func NewProxyInstance(proxyID uint, proxyStr, serviceType string, cfg *config.Config) (*ProxyInstance, error) {
	port := int(proxyID) + 10000

	// Create logger
	logger := clog.NewCondLogger(
		log.New(os.Stderr, fmt.Sprintf("[Proxy-%d] ", proxyID), log.LstdFlags),
		clog.INFO,
	)

	// Create auth provider
	authURL := fmt.Sprintf("static://?username=%s&password=%s", cfg.Username, cfg.Password)
	fmt.Println("authURL", authURL)
	authProvider, err := auth.NewAuth(authURL, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create auth: %w", err)
	}

	// Parse proxy string and create upstream dialer
	upstreamURL, err := parseProxyStr(proxyStr, serviceType)
	if err != nil {
		authProvider.Close()
		return nil, fmt.Errorf("failed to parse proxy string: %w", err)
	}

	upstreamDialer, err := dialer.ProxyDialerFromURL(upstreamURL, &net.Dialer{})
	if err != nil {
		authProvider.Close()
		return nil, fmt.Errorf("failed to create upstream dialer: %w", err)
	}

	// Create proxy handler
	proxyHandler := handler.NewProxyHandler(&handler.Config{
		Dialer:  upstreamDialer,
		Auth:    authProvider,
		Logger:  logger,
		Forward: forward.PairConnections,
	})

	instance := &ProxyInstance{
		ProxyID:     proxyID,
		Port:        port,
		ServiceType: serviceType,
		Handler:     proxyHandler,
		logger:      logger,
		auth:        authProvider,
		dialer:      upstreamDialer,
	}

	return instance, nil
}

func (pi *ProxyInstance) Start(ctx context.Context) error {
	pi.mu.Lock()
	defer pi.mu.Unlock()

	// Create listener
	addr := fmt.Sprintf(":%d", pi.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	pi.Listener = listener

	// Create HTTP server
	pi.Server = &http.Server{
		Handler: pi.Handler,
	}

	// Create cancellable context
	childCtx, cancel := context.WithCancel(ctx)
	pi.Cancel = cancel

	// Start server in goroutine
	go func() {
		pi.logger.Info("Starting proxy instance on port %d", pi.Port)
		if err := pi.Server.Serve(pi.Listener); err != nil && err != http.ErrServerClosed {
			pi.logger.Error("Server error: %v", err)
		}
	}()

	// Wait for context cancellation
	go func() {
		<-childCtx.Done()
		pi.Stop()
	}()

	return nil
}

func (pi *ProxyInstance) UpdateUpstream(proxyStr string) error {
	pi.mu.Lock()
	defer pi.mu.Unlock()

	pi.logger.Info("Updating upstream proxy")

	// Parse new proxy string
	upstreamURL, err := parseProxyStr(proxyStr, pi.ServiceType)
	if err != nil {
		return fmt.Errorf("failed to parse proxy string: %w", err)
	}

	// Create new upstream dialer
	newDialer, err := dialer.ProxyDialerFromURL(upstreamURL, &net.Dialer{})
	if err != nil {
		return fmt.Errorf("failed to create new upstream dialer: %w", err)
	}

	// Update handler's dialer
	// Note: We need to create a new handler with the new dialer
	newHandler := handler.NewProxyHandler(&handler.Config{
		Dialer:  newDialer,
		Auth:    pi.auth,
		Logger:  pi.logger,
		Forward: forward.PairConnections,
	})

	// Update server handler
	if pi.Server != nil {
		pi.Server.Handler = newHandler
	}

	pi.Handler = newHandler
	pi.dialer = newDialer

	pi.logger.Info("Upstream proxy updated successfully")
	return nil
}

func (pi *ProxyInstance) Stop() error {
	pi.mu.Lock()
	defer pi.mu.Unlock()

	pi.logger.Info("Stopping proxy instance")

	// Close auth provider
	if pi.auth != nil {
		pi.auth.Close()
	}

	// Shutdown HTTP server
	if pi.Server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5)
		defer cancel()
		if err := pi.Server.Shutdown(ctx); err != nil {
			pi.logger.Error("Failed to shutdown server: %v", err)
		}
	}

	// Close listener
	if pi.Listener != nil {
		pi.Listener.Close()
	}

	pi.logger.Info("Proxy instance stopped")
	return nil
}

// parseProxyStr converts proxy_str to upstream URL format for dumbproxy
func parseProxyStr(proxyStr, serviceType string) (string, error) {
	parts := strings.Split(proxyStr, ":")

	switch serviceType {
	case "tmproxy":
		// Format: "ip:port:username:password"
		if len(parts) != 4 {
			return "", fmt.Errorf("invalid tmproxy format, expected ip:port:username:password")
		}
		return fmt.Sprintf("http://%s:%s@%s:%s", parts[2], parts[3], parts[0], parts[1]), nil

	case "kiotproxy":
		// Format: "ip:port"
		if len(parts) != 2 {
			return "", fmt.Errorf("invalid kiotproxy format, expected ip:port")
		}
		return fmt.Sprintf("http://%s:%s", parts[0], parts[1]), nil

	default:
		return "", fmt.Errorf("unknown service type: %s", serviceType)
	}
}

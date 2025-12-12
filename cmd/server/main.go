package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-forward-proxy/internal/api"
	"go-forward-proxy/internal/config"
	"go-forward-proxy/internal/database"
	"go-forward-proxy/internal/proxymanager"
	"go-forward-proxy/internal/proxyservices"
)

func main() {
	// 1. Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Println("Configuration loaded successfully")

	// 2. Initialize database
	db, err := database.InitDB(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	log.Println("Database initialized successfully")

	// 3. Initialize proxy services
	tmService := proxyservices.NewTMProxyService()
	kiotService := proxyservices.NewKiotProxyService()

	services := map[string]proxyservices.ProxyService{
		"tmproxy":   tmService,
		"kiotproxy": kiotService,
	}

	log.Println("Proxy services initialized")

	// 4. Initialize proxy manager
	mgr := proxymanager.NewManager(db, cfg, services)

	// 5. Start all existing proxies
	if err := mgr.StartAll(); err != nil {
		log.Fatalf("Failed to start existing proxies: %v", err)
	}

	log.Println("All existing proxies started")

	// 6. Start auto-reset service
	autoReset := proxymanager.NewAutoResetService(mgr, db, services, cfg.AutoResetInterval)
	ctx, cancel := context.WithCancel(context.Background())

	go autoReset.Start(ctx)

	log.Println("Auto-reset service started")

	// 7. Setup API router
	router := api.SetupRouter(mgr, cfg)

	// 8. Start API server in goroutine
	go func() {
		addr := fmt.Sprintf(":%d", cfg.APIPort)
		log.Printf("Starting API server on %s", addr)
		if err := router.Start(addr); err != nil {
			log.Printf("API server error: %v", err)
		}
	}()

	// 9. Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan
	log.Println("\nShutdown signal received, gracefully shutting down...")

	// 10. Graceful shutdown
	// Stop auto-reset service
	cancel()

	// Stop all proxy instances
	if err := mgr.StopAll(); err != nil {
		log.Printf("Error stopping proxy instances: %v", err)
	}

	// Shutdown API server
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := router.Shutdown(shutdownCtx); err != nil {
		log.Printf("Error shutting down API server: %v", err)
	}

	log.Println("Shutdown complete")
}

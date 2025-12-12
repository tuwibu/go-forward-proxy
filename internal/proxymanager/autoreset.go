package proxymanager

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"go-forward-proxy/internal/database/models"
	"go-forward-proxy/internal/proxyservices"
)

type AutoResetService struct {
	manager       *Manager
	db            *sql.DB
	proxyServices map[string]proxyservices.ProxyService
	checkInterval time.Duration
}

func NewAutoResetService(mgr *Manager, db *sql.DB, services map[string]proxyservices.ProxyService, checkInterval int) *AutoResetService {
	return &AutoResetService{
		manager:       mgr,
		db:            db,
		proxyServices: services,
		checkInterval: time.Duration(checkInterval) * time.Second,
	}
}

func (ars *AutoResetService) Start(ctx context.Context) {
	ticker := time.NewTicker(ars.checkInterval)
	defer ticker.Stop()

	log.Printf("AutoResetService started with check interval: %v", ars.checkInterval)

	for {
		select {
		case <-ctx.Done():
			log.Println("AutoResetService stopped")
			return
		case <-ticker.C:
			ars.checkAndResetProxies()
		}
	}
}

func (ars *AutoResetService) checkAndResetProxies() {
	// Query all proxies
	rows, err := ars.db.Query(`
		SELECT id, proxy_str, api_key, service_type, min_time_reset, last_reset_at, created_at
		FROM proxies
	`)
	if err != nil {
		log.Printf("Failed to load proxies: %v", err)
		return
	}
	defer rows.Close()

	var proxies []models.Proxy
	for rows.Next() {
		var p models.Proxy
		if err := rows.Scan(&p.ID, &p.ProxyStr, &p.APIKey, &p.ServiceType, &p.MinTimeReset, &p.LastResetAt, &p.CreatedAt); err != nil {
			log.Printf("Failed to scan proxy: %v", err)
			continue
		}
		proxies = append(proxies, p)
	}

	now := time.Now()

	for _, proxy := range proxies {
		elapsed := now.Sub(proxy.LastResetAt).Seconds()

		if elapsed >= float64(proxy.MinTimeReset) {
			log.Printf("Proxy %d needs reset (elapsed: %.0fs, min: %ds)", proxy.ID, elapsed, proxy.MinTimeReset)

			if err := ars.resetProxy(&proxy, now); err != nil {
				log.Printf("Failed to reset proxy %d: %v", proxy.ID, err)
			} else {
				log.Printf("Proxy %d reset successfully", proxy.ID)
			}
		}
	}
}

func (ars *AutoResetService) resetProxy(proxy *models.Proxy, resetTime time.Time) error {
	// Get service
	service, ok := ars.proxyServices[proxy.ServiceType]
	if !ok {
		return fmt.Errorf("unknown service type: %s", proxy.ServiceType)
	}

	// Get new proxy
	proxyInfo, err := service.GetNewProxy(proxy.APIKey)
	if err != nil {
		return fmt.Errorf("failed to get new proxy: %w", err)
	}

	// Update database
	_, err = ars.db.Exec(`
		UPDATE proxies
		SET proxy_str = ?, last_reset_at = ?
		WHERE id = ?
	`, proxyInfo.ProxyStr, resetTime, proxy.ID)

	if err != nil {
		return fmt.Errorf("failed to update database: %w", err)
	}

	// Update running instance
	if err := ars.manager.UpdateInstance(proxy.ID, proxyInfo.ProxyStr); err != nil {
		return fmt.Errorf("failed to update instance: %w", err)
	}

	return nil
}

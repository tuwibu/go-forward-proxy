package proxymanager

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"go-forward-proxy/internal/config"
	"go-forward-proxy/internal/database/models"
	"go-forward-proxy/internal/proxyservices"
)

type Manager struct {
	instances     map[uint]*ProxyInstance
	db            *sql.DB
	config        *config.Config
	proxyServices map[string]proxyservices.ProxyService
	ctx           context.Context
	mu            sync.RWMutex
}

func NewManager(db *sql.DB, cfg *config.Config, services map[string]proxyservices.ProxyService) *Manager {
	return &Manager{
		instances:     make(map[uint]*ProxyInstance),
		db:            db,
		config:        cfg,
		proxyServices: services,
		ctx:           context.Background(),
	}
}

func (m *Manager) UpsertProxy(apiKey, serviceType string, minTimeReset int) (*models.Proxy, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Get proxy service
	service, ok := m.proxyServices[serviceType]
	if !ok {
		return nil, fmt.Errorf("unknown service type: %s", serviceType)
	}

	// Check if proxy exists by api_key
	var existingID uint
	err := m.db.QueryRow("SELECT id FROM proxies WHERE api_key = ? LIMIT 1", apiKey).Scan(&existingID)

	if err == sql.ErrNoRows {
		// INSERT flow: proxy does NOT exist
		return m.insertNewProxy(apiKey, serviceType, minTimeReset, service)
	} else if err != nil {
		return nil, fmt.Errorf("failed to check existing proxy: %w", err)
	}

	// UPDATE flow: proxy EXISTS
	return m.updateExistingProxy(existingID, apiKey, minTimeReset, service)
}

// insertNewProxy handles the INSERT flow when proxy doesn't exist
func (m *Manager) insertNewProxy(apiKey, serviceType string, minTimeReset int, service proxyservices.ProxyService) (*models.Proxy, error) {
	// Get current proxy info from service (NOT GetNewProxy)
	proxyInfo, err := service.GetCurrentProxy(apiKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get current proxy: %w", err)
	}

	// Calculate last_reset_at based on NextResetAfter
	// This ensures auto-reset will run at the right time
	now := time.Now()
	lastResetAt := now.Add(-time.Duration(proxyInfo.NextResetAfter) * time.Second)

	// Insert into database with calculated last_reset_at
	result, err := m.db.Exec(`
		INSERT INTO proxies (proxy_str, api_key, service_type, min_time_reset, last_reset_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, proxyInfo.ProxyStr, apiKey, serviceType, minTimeReset, lastResetAt, now)

	if err != nil {
		return nil, fmt.Errorf("failed to insert proxy in database: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert ID: %w", err)
	}

	proxy := &models.Proxy{
		ID:           uint(id),
		ProxyStr:     proxyInfo.ProxyStr,
		APIKey:       apiKey,
		ServiceType:  serviceType,
		MinTimeReset: minTimeReset,
		LastResetAt:  lastResetAt,
		CreatedAt:    now,
	}

	// Create and start proxy instance
	instance, err := NewProxyInstance(proxy.ID, proxy.ProxyStr, proxy.ServiceType, m.config)
	if err != nil {
		// Rollback database creation
		m.db.Exec("DELETE FROM proxies WHERE id = ?", id)
		return nil, fmt.Errorf("failed to create proxy instance: %w", err)
	}

	if err := instance.Start(m.ctx); err != nil {
		// Rollback
		m.db.Exec("DELETE FROM proxies WHERE id = ?", id)
		return nil, fmt.Errorf("failed to start proxy instance: %w", err)
	}

	// Store instance in map
	m.instances[proxy.ID] = instance

	return proxy, nil
}

// updateExistingProxy handles the UPDATE flow when proxy already exists
func (m *Manager) updateExistingProxy(proxyID uint, apiKey string, minTimeReset int, service proxyservices.ProxyService) (*models.Proxy, error) {
	// Get current proxy info from service
	proxyInfo, err := service.GetCurrentProxy(apiKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get current proxy: %w", err)
	}

	// Calculate last_reset_at based on NextResetAfter
	now := time.Now()
	lastResetAt := now.Add(-time.Duration(proxyInfo.NextResetAfter) * time.Second)

	// Update database: proxy_str, min_time_reset, last_reset_at
	_, err = m.db.Exec(`
		UPDATE proxies
		SET proxy_str = ?, min_time_reset = ?, last_reset_at = ?
		WHERE id = ?
	`, proxyInfo.ProxyStr, minTimeReset, lastResetAt, proxyID)

	if err != nil {
		return nil, fmt.Errorf("failed to update proxy in database: %w", err)
	}

	// Update running instance's upstream if instance is running
	if instance, ok := m.instances[proxyID]; ok {
		if err := instance.UpdateUpstream(proxyInfo.ProxyStr); err != nil {
			return nil, fmt.Errorf("failed to update proxy instance upstream: %w", err)
		}
	}

	// Get updated proxy from database to return
	proxy, err := m.GetProxyByID(proxyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated proxy: %w", err)
	}

	return proxy, nil
}

func (m *Manager) DeleteProxy(id uint) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Stop instance if running
	if instance, ok := m.instances[id]; ok {
		if err := instance.Stop(); err != nil {
			return fmt.Errorf("failed to stop proxy instance: %w", err)
		}
		delete(m.instances, id)
	}

	// Delete from database
	_, err := m.db.Exec("DELETE FROM proxies WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete proxy from database: %w", err)
	}

	return nil
}

func (m *Manager) GetAllProxies() ([]models.Proxy, error) {
	rows, err := m.db.Query(`
		SELECT id, proxy_str, api_key, service_type, min_time_reset, last_reset_at, created_at
		FROM proxies
		ORDER BY id
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query proxies: %w", err)
	}
	defer rows.Close()

	var proxies []models.Proxy
	for rows.Next() {
		var p models.Proxy
		if err := rows.Scan(&p.ID, &p.ProxyStr, &p.APIKey, &p.ServiceType, &p.MinTimeReset, &p.LastResetAt, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan proxy: %w", err)
		}
		proxies = append(proxies, p)
	}

	return proxies, nil
}

func (m *Manager) GetProxyByID(id uint) (*models.Proxy, error) {
	var p models.Proxy
	err := m.db.QueryRow(`
		SELECT id, proxy_str, api_key, service_type, min_time_reset, last_reset_at, created_at
		FROM proxies WHERE id = ?
	`, id).Scan(&p.ID, &p.ProxyStr, &p.APIKey, &p.ServiceType, &p.MinTimeReset, &p.LastResetAt, &p.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to get proxy: %w", err)
	}

	return &p, nil
}

func (m *Manager) StartAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Load all proxies from database
	proxies, err := m.GetAllProxies()
	if err != nil {
		return fmt.Errorf("failed to load proxies: %w", err)
	}

	// Start instance for each proxy
	for _, proxy := range proxies {
		instance, err := NewProxyInstance(proxy.ID, proxy.ProxyStr, proxy.ServiceType, m.config)
		if err != nil {
			fmt.Printf("Failed to create proxy instance %d: %v\n", proxy.ID, err)
			continue
		}

		if err := instance.Start(m.ctx); err != nil {
			fmt.Printf("Failed to start proxy instance %d: %v\n", proxy.ID, err)
			continue
		}

		m.instances[proxy.ID] = instance
		fmt.Printf("Started proxy instance %d on port %d\n", proxy.ID, instance.Port)
	}

	return nil
}

func (m *Manager) StopAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for id, instance := range m.instances {
		if err := instance.Stop(); err != nil {
			fmt.Printf("Failed to stop proxy instance %d: %v\n", id, err)
		}
	}

	// Clear instances map
	m.instances = make(map[uint]*ProxyInstance)

	return nil
}

func (m *Manager) UpdateInstance(proxyID uint, newProxyStr string) error {
	m.mu.RLock()
	instance, ok := m.instances[proxyID]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("proxy instance %d not found", proxyID)
	}

	return instance.UpdateUpstream(newProxyStr)
}

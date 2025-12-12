package api

import (
	"go-forward-proxy/internal/api/handlers"
	"go-forward-proxy/internal/config"
	"go-forward-proxy/internal/proxymanager"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func SetupRouter(mgr *proxymanager.Manager, cfg *config.Config) *echo.Echo {
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// API group with authentication
	api := e.Group("/api")
	// api.Use(authMiddleware.BasicAuthMiddleware(cfg.Username, cfg.Password))

	// Create handlers
	proxyHandler := handlers.NewProxyHandler(mgr)
	exportHandler := handlers.NewExportHandler(mgr, cfg)

	// Register routes
	api.POST("/proxies", proxyHandler.CreateProxy)
	api.DELETE("/proxies/:id", proxyHandler.DeleteProxy)
	api.GET("/proxies", proxyHandler.ListProxies)
	api.GET("/export", exportHandler.ExportText)

	return e
}

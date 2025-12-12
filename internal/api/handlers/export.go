package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"go-forward-proxy/internal/config"
	"go-forward-proxy/internal/proxymanager"

	"github.com/labstack/echo/v4"
)

type ExportHandler struct {
	manager *proxymanager.Manager
	config  *config.Config
}

func NewExportHandler(mgr *proxymanager.Manager, cfg *config.Config) *ExportHandler {
	return &ExportHandler{
		manager: mgr,
		config:  cfg,
	}
}

// GET /api/export
func (h *ExportHandler) ExportText(c echo.Context) error {
	proxies, err := h.manager.GetAllProxies()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	var lines []string
	for _, proxy := range proxies {
		port := proxy.ID + 10000
		line := fmt.Sprintf("%s:%d", h.config.ServerIP, port)
		lines = append(lines, line)
	}

	text := strings.Join(lines, "\n")
	if len(lines) > 0 {
		text += "\n" // Add trailing newline
	}

	return c.String(http.StatusOK, text)
}

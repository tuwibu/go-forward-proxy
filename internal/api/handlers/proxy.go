package handlers

import (
	"net/http"
	"strconv"

	"go-forward-proxy/internal/proxymanager"

	"github.com/labstack/echo/v4"
)

type ProxyHandler struct {
	manager *proxymanager.Manager
}

func NewProxyHandler(mgr *proxymanager.Manager) *ProxyHandler {
	return &ProxyHandler{
		manager: mgr,
	}
}

type UpsertProxyRequest struct {
	APIKey       string `json:"api_key" validate:"required"`
	ServiceType  string `json:"service_type" validate:"required"`
	MinTimeReset int    `json:"min_time_reset" validate:"required,min=1"`
}

// POST /api/proxies
func (h *ProxyHandler) CreateProxy(c echo.Context) error {
	var req UpsertProxyRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	// Validate service type
	if req.ServiceType != "tmproxy" && req.ServiceType != "kiotproxy" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "service_type must be 'tmproxy' or 'kiotproxy'",
		})
	}

	// Upsert proxy (insert or update based on api_key)
	proxy, err := h.manager.UpsertProxy(req.APIKey, req.ServiceType, req.MinTimeReset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, proxy)
}

// DELETE /api/proxies/:id
func (h *ProxyHandler) DeleteProxy(c echo.Context) error {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid proxy ID",
		})
	}

	if err := h.manager.DeleteProxy(uint(id)); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.NoContent(http.StatusNoContent)
}

// GET /api/proxies
func (h *ProxyHandler) ListProxies(c echo.Context) error {
	proxies, err := h.manager.GetAllProxies()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, proxies)
}

// Package handler contains HTTP handlers for the API gateway.
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// HealthResponse is the response for the health endpoint.
type HealthResponse struct {
	Status string `json:"status"`
}

// HealthHandler handles the health check endpoint.
type HealthHandler struct{}

// NewHealthHandler creates a new HealthHandler.
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// RegisterRoutes registers the health route.
func (h *HealthHandler) RegisterRoutes(r *gin.Engine) {
	r.GET("/health", h.Health)
}

// Health returns the health status.
func (h *HealthHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, HealthResponse{Status: "UP"})
}

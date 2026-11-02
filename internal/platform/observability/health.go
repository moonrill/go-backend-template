package observability

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type HealthHandler struct{}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

func (h *HealthHandler) Live(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    gin.H{"status": "live"},
	})
}

func (h *HealthHandler) Ready(c *gin.Context) {
	// Add checks only for dependencies required by the configured application.
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    gin.H{"status": "ready"},
	})
}

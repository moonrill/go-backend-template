package observability

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ReadinessCheck represents a named dependency health check evaluated by Ready.
type ReadinessCheck struct {
	Name  string
	Check func(ctx context.Context) error
}

type HealthHandler struct {
	checks []ReadinessCheck
}

func NewHealthHandler(checks ...ReadinessCheck) *HealthHandler {
	return &HealthHandler{
		checks: checks,
	}
}

func (h *HealthHandler) Live(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    gin.H{"status": "live"},
	})
}

func (h *HealthHandler) Ready(c *gin.Context) {
	if len(h.checks) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    gin.H{"status": "ready"},
		})
		return
	}

	results := make(map[string]string, len(h.checks))
	allReady := true
	for _, rc := range h.checks {
		if rc.Check == nil {
			continue
		}
		if err := rc.Check(c.Request.Context()); err != nil {
			results[rc.Name] = "unavailable"
			allReady = false
		} else {
			results[rc.Name] = "ready"
		}
	}

	if !allReady {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "SERVICE_UNAVAILABLE",
				"message": "One or more required dependencies are unavailable",
				"checks":  results,
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"status": "ready",
			"checks": results,
		},
	})
}

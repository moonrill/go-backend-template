package observability

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/moonrill/go-backend-template/internal/shared/httpx"
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

// Live godoc
// @Summary Liveness probe
// @Description Checks whether the application process is running and responding to requests.
// @Tags Health
// @Produce json
// @Success 200 {object} httpx.HealthResponse "Process is live"
// @Router /health/live [get]
func (h *HealthHandler) Live(c *gin.Context) {
	c.JSON(http.StatusOK, httpx.HealthResponse{
		Success: true,
		Data:    httpx.HealthData{Status: "live"},
	})
}

// Ready godoc
// @Summary Readiness probe
// @Description Evaluates readiness of required infrastructure dependencies (PostgreSQL, Redis, MinIO).
// @Tags Health
// @Produce json
// @Success 200 {object} httpx.HealthResponse "All required dependencies are ready"
// @Failure 503 {object} httpx.ErrorResponse "One or more required dependencies are unavailable"
// @Router /health/ready [get]
func (h *HealthHandler) Ready(c *gin.Context) {
	if len(h.checks) == 0 {
		c.JSON(http.StatusOK, httpx.HealthResponse{
			Success: true,
			Data:    httpx.HealthData{Status: "ready"},
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
				"code":       "SERVICE_UNAVAILABLE",
				"message":    "One or more required dependencies are unavailable",
				"request_id": c.GetString("request_id"),
				"checks":     results,
			},
		})
		return
	}

	c.JSON(http.StatusOK, httpx.HealthResponse{
		Success: true,
		Data: httpx.HealthData{
			Status: "ready",
			Checks: results,
		},
	})
}

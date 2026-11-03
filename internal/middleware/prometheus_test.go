package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
)

func TestPrometheusMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	reg := prometheus.NewRegistry()
	pm, err := NewPrometheusMiddleware(reg)
	if err != nil {
		t.Fatalf("unexpected error creating middleware: %v", err)
	}

	// Repeated creation should succeed via AlreadyRegisteredError handling
	pm2, err2 := NewPrometheusMiddleware(reg)
	if err2 != nil || pm2 == nil {
		t.Fatalf("expected repeated NewPrometheusMiddleware to succeed, got: %v", err2)
	}

	router := gin.New()
	router.Use(pm.Handler())

	router.GET("/hello/:name", func(c *gin.Context) {
		c.String(http.StatusOK, "hello "+c.Param("name"))
	})
	router.GET("/metrics", func(c *gin.Context) {
		c.String(http.StatusOK, "metrics")
	})

	// 1. Matched request
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest(http.MethodGet, "/hello/world", nil)
	router.ServeHTTP(w1, req1)
	if w1.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w1.Code)
	}

	// 2. Unmatched request (404)
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodGet, "/not-found", nil)
	router.ServeHTTP(w2, req2)
	if w2.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w2.Code)
	}

	// 3. /metrics endpoint (should be skipped)
	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest(http.MethodGet, "/metrics", nil)
	router.ServeHTTP(w3, req3)
	if w3.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w3.Code)
	}
}

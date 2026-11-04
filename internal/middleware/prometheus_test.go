package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func getCounterValue(t *testing.T, counter prometheus.Counter) float64 {
	t.Helper()
	var m dto.Metric
	if err := counter.Write(&m); err != nil {
		t.Fatalf("failed to write metric: %v", err)
	}
	return m.GetCounter().GetValue()
}

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

func TestPrometheusMiddleware_CardinalityProtection(t *testing.T) {
	gin.SetMode(gin.TestMode)

	pm, err := NewPrometheusMiddleware(nil)
	if err != nil {
		t.Fatalf("unexpected error creating middleware: %v", err)
	}

	router := gin.New()
	router.Use(pm.Handler())

	router.GET("/api/v1/users/:id", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// Send requests with distinct URL paths that match a single parameterized route
	initialCount := 0.0
	if c, err := pm.requestCount.GetMetricWithLabelValues("GET", "/api/v1/users/:id", "200"); err == nil {
		initialCount = getCounterValue(t, c)
	}

	for i := 0; i < 10; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/users/uuid-%d", i), nil)
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	}

	counter, err := pm.requestCount.GetMetricWithLabelValues("GET", "/api/v1/users/:id", "200")
	if err != nil {
		t.Fatalf("failed to retrieve metric with route label: %v", err)
	}
	if got := getCounterValue(t, counter) - initialCount; got != 10 {
		t.Errorf("expected 10 increments on parameterized route label, got %f", got)
	}

	// Send requests to arbitrary non-existent paths -> all must map to "UNMATCHED"
	initialUnmatched := 0.0
	if c, err := pm.requestCount.GetMetricWithLabelValues("GET", "UNMATCHED", "404"); err == nil {
		initialUnmatched = getCounterValue(t, c)
	}

	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/random-arbitrary-path-%d", i), nil)
		router.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	}

	unmatchedCounter, err := pm.requestCount.GetMetricWithLabelValues("GET", "UNMATCHED", "404")
	if err != nil {
		t.Fatalf("failed to retrieve metric with UNMATCHED label: %v", err)
	}
	if got := getCounterValue(t, unmatchedCounter) - initialUnmatched; got != 5 {
		t.Errorf("expected 5 increments on UNMATCHED route label, got %f", got)
	}
}

func TestPrometheusMiddleware_MethodNormalization(t *testing.T) {
	gin.SetMode(gin.TestMode)

	pm, err := NewPrometheusMiddleware(nil)
	if err != nil {
		t.Fatalf("unexpected error creating middleware: %v", err)
	}

	router := gin.New()
	router.Use(pm.Handler())
	router.Any("/test-method", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// 1. Arbitrary / malicious HTTP methods must normalize to "UNKNOWN"
	initialUnknown := 0.0
	if c, err := pm.requestCount.GetMetricWithLabelValues("UNKNOWN", "UNMATCHED", "404"); err == nil {
		initialUnknown = getCounterValue(t, c)
	}

	maliciousMethods := []string{"HACK_METHOD", "RANDOM_METHOD_123", "PROPFIND_CUSTOM", "EXPL0IT"}
	for _, m := range maliciousMethods {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(m, "/test-method", nil)
		router.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404 for unrouted method %s, got %d", m, w.Code)
		}
	}

	unknownCounter, err := pm.requestCount.GetMetricWithLabelValues("UNKNOWN", "UNMATCHED", "404")
	if err != nil {
		t.Fatalf("failed to get UNKNOWN method counter: %v", err)
	}
	if got := getCounterValue(t, unknownCounter) - initialUnknown; got != float64(len(maliciousMethods)) {
		t.Errorf("expected %d unknown method increments, got %f", len(maliciousMethods), got)
	}

	// 2. Standard methods with lowercase must normalize to uppercase
	initialPost := 0.0
	if c, err := pm.requestCount.GetMetricWithLabelValues("POST", "UNMATCHED", "404"); err == nil {
		initialPost = getCounterValue(t, c)
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("post", "/test-method", nil)
	router.ServeHTTP(w, req)

	postCounter, err := pm.requestCount.GetMetricWithLabelValues("POST", "UNMATCHED", "404")
	if err != nil {
		t.Fatalf("failed to get POST method counter: %v", err)
	}
	if got := getCounterValue(t, postCounter) - initialPost; got != 1 {
		t.Errorf("expected 1 POST increment, got %f", got)
	}

	// 3. Standard uppercase method matching route
	initialGet := 0.0
	if c, err := pm.requestCount.GetMetricWithLabelValues("GET", "/test-method", "200"); err == nil {
		initialGet = getCounterValue(t, c)
	}

	wGet := httptest.NewRecorder()
	reqGet, _ := http.NewRequest(http.MethodGet, "/test-method", nil)
	router.ServeHTTP(wGet, reqGet)
	if wGet.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", wGet.Code)
	}

	getCounter, err := pm.requestCount.GetMetricWithLabelValues("GET", "/test-method", "200")
	if err != nil {
		t.Fatalf("failed to get GET method counter: %v", err)
	}
	if got := getCounterValue(t, getCounter) - initialGet; got != 1 {
		t.Errorf("expected 1 GET increment, got %f", got)
	}
}

package middleware

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestRequestIDValidAndInvalid(t *testing.T) {
	tests := []struct {
		name      string
		incoming  string
		expectNew bool
	}{
		{"empty generates new", "", true},
		{"too short generates new", "abc1234", true},
		{"too long generates new", strings.Repeat("a", 65), true},
		{"invalid chars generates new", "valid-len-but-bad$char", true},
		{"valid request id preserved", "req_12345678-ABCD", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			router := gin.New()
			router.Use(RequestID())
			router.GET("/test", func(c *gin.Context) {
				c.String(http.StatusOK, c.GetString("request_id"))
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tc.incoming != "" {
				req.Header.Set("X-Request-ID", tc.incoming)
			}
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			gotHeader := rec.Header().Get("X-Request-ID")
			gotBody := rec.Body.String()
			if gotHeader == "" || gotHeader != gotBody {
				t.Fatalf("header=%q body=%q, expected matching non-empty ID", gotHeader, gotBody)
			}
			if tc.expectNew && gotHeader == tc.incoming {
				t.Fatalf("expected new request ID, got incoming %q", gotHeader)
			}
			if !tc.expectNew && gotHeader != tc.incoming {
				t.Fatalf("expected incoming %q, got %q", tc.incoming, gotHeader)
			}
		})
	}
}

func TestSecurityHeaders(t *testing.T) {
	router := gin.New()
	router.Use(SecurityHeaders())
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	expected := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"Referrer-Policy":        "no-referrer",
		"Cache-Control":          "no-store",
	}
	for k, want := range expected {
		if got := rec.Header().Get(k); got != want {
			t.Errorf("header %s = %q, want %q", k, got, want)
		}
	}
}

func TestRecoveryAndAccessLogOrder(t *testing.T) {
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuf, nil))

	router := gin.New()
	router.Use(
		RequestID(),
		SecurityHeaders(),
		AccessLog(logger),
		Recovery(logger),
	)
	router.GET("/panic", func(c *gin.Context) {
		panic("unexpected failure")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	req.Header.Set("X-Request-ID", "test-req-12345")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Errorf("missing security headers on recovered response")
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if body["success"] != false {
		t.Errorf("success = %v, want false", body["success"])
	}
	errObj, ok := body["error"].(map[string]any)
	if !ok || errObj["request_id"] != "test-req-12345" {
		t.Errorf("error.request_id = %v, want test-req-12345", errObj["request_id"])
	}

	logs := logBuf.String()
	if !strings.Contains(logs, `"msg":"panic recovered"`) {
		t.Errorf("expected panic recovery log entry, got: %s", logs)
	}
	if !strings.Contains(logs, `"msg":"http request"`) || !strings.Contains(logs, `"status":500`) {
		t.Errorf("expected access log entry with status 500, got: %s", logs)
	}
}

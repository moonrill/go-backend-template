package users_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/moonrill/go-backend-template/internal/config"
	"github.com/moonrill/go-backend-template/internal/modules/users"
	"github.com/moonrill/go-backend-template/internal/platform/postgres"
	"github.com/moonrill/go-backend-template/internal/server"
	"github.com/moonrill/go-backend-template/internal/shared/httpx"
)

func TestUsersModulePostgresIntegration(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("skipping Users PostgreSQL integration test: TEST_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	adminPool, err := postgres.NewPool(ctx, config.DatabaseConfig{
		Enabled:        true,
		URL:            dsn,
		MaxConns:       2,
		MinConns:       0,
		ConnectTimeout: 3 * time.Second,
	})
	if err != nil {
		t.Fatalf("admin NewPool() error = %v", err)
	}
	defer adminPool.Close()

	schemaName := fmt.Sprintf("test_users_%d", time.Now().UnixNano())
	if _, err := adminPool.Exec(ctx, "CREATE SCHEMA "+schemaName); err != nil {
		t.Fatalf("create test schema: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		_, _ = adminPool.Exec(cleanupCtx, "DROP SCHEMA IF EXISTS "+schemaName+" CASCADE")
	})

	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("parse dsn: %v", err)
	}
	q := u.Query()
	q.Set("search_path", schemaName+",public")
	u.RawQuery = q.Encode()

	pool, err := postgres.NewPool(ctx, config.DatabaseConfig{
		Enabled:        true,
		URL:            u.String(),
		MaxConns:       5,
		MinConns:       1,
		ConnectTimeout: 3 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewPool() error = %v", err)
	}
	defer pool.Close()

	if err := postgres.ApplyMigrations(ctx, pool); err != nil {
		t.Fatalf("ApplyMigrations() error = %v", err)
	}

	cfg := config.Config{
		Name:            "test-api",
		Environment:     "test",
		Port:            "8080",
		ReadTimeout:     5 * time.Second,
		WriteTimeout:    5 * time.Second,
		IdleTimeout:     30 * time.Second,
		ShutdownTimeout: 5 * time.Second,
		LogLevel:        "info",
		Database: config.DatabaseConfig{
			Enabled:        true,
			ConnectTimeout: 2 * time.Second,
		},
	}
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	srv, err := server.New(cfg, logger, server.Dependencies{DB: pool})
	if err != nil {
		t.Fatalf("server.New() error = %v", err)
	}

	// Seed 3 users via HTTP POST /api/v1/users
	seedPayloads := []string{
		`{"email":"charlie@example.com","full_name":"Charlie Admin","role":"admin","is_active":true}`,
		`{"email":"alice@example.com","full_name":"Alice Member","role":"member","is_active":true}`,
		`{"email":"bob@example.com","full_name":"Bob Viewer","role":"viewer","is_active":false}`,
	}
	var createdIDs []string
	for _, payload := range seedPayloads {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("POST /api/v1/users status = %d, body = %s", rec.Code, rec.Body.String())
		}
		var resp struct {
			Data users.UserResponse `json:"data"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal create: %v", err)
		}
		createdIDs = append(createdIDs, resp.Data.ID)
	}

	// Verify pagination and sorting by email ASC (page_size=2, page=1 -> alice, bob)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users?page=1&page_size=2&sort_by=email&sort_order=asc", nil)
	rec := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/users status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var listResp struct {
		Success bool                 `json:"success"`
		Data    []users.UserResponse `json:"data"`
		Meta    httpx.PageMeta       `json:"meta"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("unmarshal list: %v", err)
	}
	if listResp.Meta.TotalItems != 3 || listResp.Meta.TotalPages != 2 || len(listResp.Data) != 2 {
		t.Fatalf("unexpected pagination meta/data: meta=%+v len=%d", listResp.Meta, len(listResp.Data))
	}
	if listResp.Data[0].Email != "alice@example.com" || listResp.Data[1].Email != "bob@example.com" {
		t.Fatalf("unexpected sort order: got [%s, %s]", listResp.Data[0].Email, listResp.Data[1].Email)
	}

	// Verify filtering by role=admin and search
	req = httptest.NewRequest(http.MethodGet, "/api/v1/users?role=admin&search=charlie", nil)
	rec = httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, req)
	if err := json.Unmarshal(rec.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("unmarshal filtered list: %v", err)
	}
	if listResp.Meta.TotalItems != 1 || listResp.Data[0].Email != "charlie@example.com" {
		t.Fatalf("unexpected filtered result: %+v", listResp)
	}

	// Verify PATCH and DELETE against PostgreSQL
	targetID := createdIDs[0]
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/users/"+targetID, bytes.NewBufferString(`{"full_name":"Charlie Lead Admin"}`))
	rec = httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("PATCH status = %d, body = %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodDelete, "/api/v1/users/"+targetID, nil)
	rec = httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("DELETE status = %d, want 204", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/users/"+targetID, nil)
	rec = httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("GET after DELETE status = %d, want 404", rec.Code)
	}
}

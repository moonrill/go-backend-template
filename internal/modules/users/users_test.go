package users_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/moonrill/go-backend-template/internal/modules/users"
	"github.com/moonrill/go-backend-template/internal/platform/postgres/sqlc"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type fakeRepo struct {
	mu    sync.Mutex
	items map[string]sqlc.User
	seq   int
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		items: make(map[string]sqlc.User),
	}
}

func (f *fakeRepo) Create(_ context.Context, arg sqlc.CreateUserParams) (sqlc.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, existing := range f.items {
		if existing.Email == arg.Email {
			return sqlc.User{}, users.ErrEmailConflict
		}
	}

	f.seq++
	id := "11111111-2222-3333-4444-55555555555" + string(rune('0'+f.seq))
	now := time.Now().UTC()
	u := sqlc.User{
		ID:        id,
		Email:     arg.Email,
		FullName:  arg.FullName,
		Role:      arg.Role,
		IsActive:  arg.IsActive,
		CreatedAt: now,
		UpdatedAt: now,
	}
	f.items[id] = u
	return u, nil
}

func (f *fakeRepo) GetByID(_ context.Context, id string) (sqlc.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	u, ok := f.items[id]
	if !ok {
		return sqlc.User{}, users.ErrUserNotFound
	}
	return u, nil
}

func (f *fakeRepo) List(_ context.Context, _ users.ListUsersFilter) ([]sqlc.User, int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	out := make([]sqlc.User, 0, len(f.items))
	for _, u := range f.items {
		out = append(out, u)
	}
	return out, int64(len(out)), nil
}

func (f *fakeRepo) Update(_ context.Context, arg sqlc.UpdateUserParams) (sqlc.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	u, ok := f.items[arg.ID]
	if !ok {
		return sqlc.User{}, users.ErrUserNotFound
	}
	if arg.Email != nil {
		for id, existing := range f.items {
			if id != arg.ID && existing.Email == *arg.Email {
				return sqlc.User{}, users.ErrEmailConflict
			}
		}
		u.Email = *arg.Email
	}
	if arg.FullName != nil {
		u.FullName = *arg.FullName
	}
	if arg.Role != nil {
		u.Role = *arg.Role
	}
	if arg.IsActive != nil {
		u.IsActive = *arg.IsActive
	}
	u.UpdatedAt = time.Now().UTC()
	f.items[arg.ID] = u
	return u, nil
}

func (f *fakeRepo) Delete(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, ok := f.items[id]; !ok {
		return users.ErrUserNotFound
	}
	delete(f.items, id)
	return nil
}

func setupTestRouter() *gin.Engine {
	repo := newFakeRepo()
	svc := users.NewService(repo)
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	h := users.NewHandler(svc, logger)

	r := gin.New()
	h.RegisterRoutes(r.Group("/api/v1"))
	return r
}

func TestUsersHandlerCRUDAndValidation(t *testing.T) {
	r := setupTestRouter()

	// 1. Create valid user
	createBody := `{"email":"Alice@Example.com","full_name":"Alice Smith","role":"admin"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewBufferString(createBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /api/v1/users status = %d, want 201, body = %s", rec.Code, rec.Body.String())
	}

	var createResp struct {
		Success bool               `json:"success"`
		Data    users.UserResponse `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &createResp); err != nil {
		t.Fatalf("unmarshal create response: %v", err)
	}
	if !createResp.Success || createResp.Data.Email != "alice@example.com" || !createResp.Data.IsActive {
		t.Fatalf("unexpected created user: %+v", createResp.Data)
	}
	userID := createResp.Data.ID

	// 2. Duplicate email returns 409 Conflict
	req = httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewBufferString(createBody))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("duplicate POST status = %d, want 409", rec.Code)
	}

	// 3. Unknown JSON field rejected with 400
	req = httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewBufferString(`{"email":"b@example.com","full_name":"Bob","extra":123}`))
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unknown field POST status = %d, want 400", rec.Code)
	}

	// 4. Invalid fields rejected with 400 VALIDATION_ERROR
	req = httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewBufferString(`{"email":"not-an-email","full_name":"A","role":"superadmin"}`))
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid input POST status = %d, want 400", rec.Code)
	}

	// 5. GetByID with invalid UUID returns 400
	req = httptest.NewRequest(http.MethodGet, "/api/v1/users/not-a-uuid", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("GET invalid UUID status = %d, want 400", rec.Code)
	}

	// 6. GetByID with non-existent UUID returns 404
	req = httptest.NewRequest(http.MethodGet, "/api/v1/users/00000000-0000-0000-0000-000000000000", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("GET non-existent UUID status = %d, want 404", rec.Code)
	}

	// 7. GetByID succeeds
	req = httptest.NewRequest(http.MethodGet, "/api/v1/users/"+userID, nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET user status = %d, want 200", rec.Code)
	}

	// 8. PATCH with empty object returns 400
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/users/"+userID, bytes.NewBufferString(`{}`))
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("empty PATCH status = %d, want 400", rec.Code)
	}

	// 9. PATCH valid update succeeds
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/users/"+userID, bytes.NewBufferString(`{"full_name":"Alice Updated","is_active":false}`))
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("PATCH status = %d, want 200, body = %s", rec.Code, rec.Body.String())
	}

	// 10. List users with invalid sort_by rejected (allowlist enforcement)
	req = httptest.NewRequest(http.MethodGet, "/api/v1/users?sort_by=password%20DROP%20TABLE%20users", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid sort_by GET status = %d, want 400", rec.Code)
	}

	// 11. List users succeeds with pagination metadata
	req = httptest.NewRequest(http.MethodGet, "/api/v1/users?page=1&page_size=10&sort_by=email&sort_order=asc", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/users status = %d, want 200", rec.Code)
	}

	// 12. DELETE succeeds with 204 and subsequent DELETE returns 404
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/users/"+userID, nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("DELETE status = %d, want 204", rec.Code)
	}

	req = httptest.NewRequest(http.MethodDelete, "/api/v1/users/"+userID, nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("second DELETE status = %d, want 404", rec.Code)
	}
}

func TestParseListUsersFilterValidation(t *testing.T) {
	_, err := users.ParseListUsersFilter("0", "200", "invalid", "notbool", "", "badcol", "sideways")
	var valErr *users.ValidationError
	if !errors.As(err, &valErr) {
		t.Fatalf("expected ValidationError, got %v", err)
	}
	if len(valErr.Fields) != 6 {
		t.Fatalf("expected 6 field errors, got %d: %+v", len(valErr.Fields), valErr.Fields)
	}
}

package users

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/moonrill/go-backend-template/internal/shared/httpx"
)

const maxJSONBodyBytes = 64 * 1024 // 64 KiB

type Handler struct {
	service *Service
	logger  *slog.Logger
}

func NewHandler(service *Service, logger *slog.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	usersGroup := rg.Group("/users")
	usersGroup.GET("", h.List)
	usersGroup.GET("/:id", h.GetByID)
	usersGroup.POST("", h.Create)
	usersGroup.PATCH("/:id", h.Update)
	usersGroup.DELETE("/:id", h.Delete)
}

// List godoc
// @Summary List users
// @Description Retrieves a paginated list of users with optional filtering, searching, and sorting.
// @Tags Users
// @Produce json
// @Param page query int false "Page number" default(1) minimum(1)
// @Param page_size query int false "Page size" default(20) minimum(1) maximum(100)
// @Param role query string false "Filter by user role" Enums(admin, member, viewer)
// @Param is_active query bool false "Filter by active status"
// @Param search query string false "Search query matching full name or email (max 100 characters)"
// @Param sort_by query string false "Field to sort by" Enums(created_at, full_name, email) default(created_at)
// @Param sort_order query string false "Sort order direction" Enums(asc, desc) default(desc)
// @Success 200 {object} UserListResponse "Paginated list of users"
// @Failure 400 {object} httpx.ErrorResponse "Invalid query parameters"
// @Failure 500 {object} httpx.ErrorResponse "Internal server error"
// @Router /users [get]
func (h *Handler) List(c *gin.Context) {
	filter, err := ParseListUsersFilter(
		c.Query("page"),
		c.Query("page_size"),
		c.Query("role"),
		c.Query("is_active"),
		c.Query("search"),
		c.Query("sort_by"),
		c.Query("sort_order"),
	)
	if err != nil {
		h.writeError(c, err)
		return
	}

	items, meta, err := h.service.List(c.Request.Context(), filter)
	if err != nil {
		h.writeError(c, err)
		return
	}
	httpx.JSONPaginated(c, http.StatusOK, items, meta)
}

// GetByID godoc
// @Summary Get user by ID
// @Description Fetches a single user record by its canonical UUID.
// @Tags Users
// @Produce json
// @Param id path string true "User UUID" format(uuid)
// @Success 200 {object} UserSuccessResponse "User details"
// @Failure 400 {object} httpx.ErrorResponse "Invalid UUID format"
// @Failure 404 {object} httpx.ErrorResponse "User not found"
// @Failure 500 {object} httpx.ErrorResponse "Internal server error"
// @Router /users/{id} [get]
func (h *Handler) GetByID(c *gin.Context) {
	user, err := h.service.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	httpx.JSONSuccess(c, http.StatusOK, user)
}

// Create godoc
// @Summary Create a user
// @Description Creates a new user record.
// @Tags Users
// @Accept json
// @Produce json
// @Param user body CreateUserRequest true "User creation payload"
// @Success 201 {object} UserSuccessResponse "User created successfully"
// @Failure 400 {object} httpx.ErrorResponse "Validation failure or invalid JSON"
// @Failure 409 {object} httpx.ErrorResponse "Email address already exists"
// @Failure 413 {object} httpx.ErrorResponse "Request body exceeds maximum allowed size"
// @Failure 500 {object} httpx.ErrorResponse "Internal server error"
// @Router /users [post]
func (h *Handler) Create(c *gin.Context) {
	var req CreateUserRequest
	if err := decodeStrictJSON(c, &req); err != nil {
		return
	}

	user, err := h.service.Create(c.Request.Context(), req)
	if err != nil {
		h.writeError(c, err)
		return
	}
	httpx.JSONSuccess(c, http.StatusCreated, user)
}

// Update godoc
// @Summary Update a user
// @Description Partially updates an existing user record. At least one field must be provided.
// @Tags Users
// @Accept json
// @Produce json
// @Param id path string true "User UUID" format(uuid)
// @Param user body UpdateUserRequest true "Fields to update"
// @Success 200 {object} UserSuccessResponse "User updated successfully"
// @Failure 400 {object} httpx.ErrorResponse "Validation failure or invalid JSON"
// @Failure 404 {object} httpx.ErrorResponse "User not found"
// @Failure 409 {object} httpx.ErrorResponse "Email address already exists"
// @Failure 413 {object} httpx.ErrorResponse "Request body exceeds maximum allowed size"
// @Failure 500 {object} httpx.ErrorResponse "Internal server error"
// @Router /users/{id} [patch]
func (h *Handler) Update(c *gin.Context) {
	var req UpdateUserRequest
	if err := decodeStrictJSON(c, &req); err != nil {
		return
	}

	user, err := h.service.Update(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		h.writeError(c, err)
		return
	}
	httpx.JSONSuccess(c, http.StatusOK, user)
}

// Delete godoc
// @Summary Delete a user
// @Description Deletes a user record by UUID.
// @Tags Users
// @Produce json
// @Param id path string true "User UUID" format(uuid)
// @Success 204 "User deleted successfully"
// @Failure 400 {object} httpx.ErrorResponse "Invalid UUID format"
// @Failure 404 {object} httpx.ErrorResponse "User not found"
// @Failure 500 {object} httpx.ErrorResponse "Internal server error"
// @Router /users/{id} [delete]
func (h *Handler) Delete(c *gin.Context) {
	if err := h.service.Delete(c.Request.Context(), c.Param("id")); err != nil {
		h.writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func decodeStrictJSON(c *gin.Context, dst any) error {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxJSONBodyBytes)
	dec := json.NewDecoder(c.Request.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			httpx.JSONError(c, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "Request body exceeds maximum allowed size")
			return err
		}
		httpx.JSONError(c, http.StatusBadRequest, "INVALID_JSON", "Request body must be valid JSON without unknown fields")
		return err
	}

	if err := dec.Decode(&struct{}{}); err != io.EOF {
		httpx.JSONError(c, http.StatusBadRequest, "INVALID_JSON", "Request body must contain a single JSON object")
		return errors.New("multiple json values")
	}
	return nil
}

func (h *Handler) writeError(c *gin.Context, err error) {
	var valErr *ValidationError
	switch {
	case errors.As(err, &valErr):
		httpx.JSONError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Request validation failed", valErr.Fields...)
	case errors.Is(err, ErrUserNotFound):
		httpx.JSONError(c, http.StatusNotFound, "NOT_FOUND", "User not found")
	case errors.Is(err, ErrEmailConflict):
		httpx.JSONError(c, http.StatusConflict, "CONFLICT", "A user with this email already exists")
	default:
		if h.logger != nil {
			h.logger.Error("users handler error",
				"request_id", c.GetString("request_id"),
				"method", c.Request.Method,
				"path", c.Request.URL.Path,
				"error", err,
			)
		}
		httpx.JSONError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred")
	}
}

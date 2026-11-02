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

func (h *Handler) GetByID(c *gin.Context) {
	user, err := h.service.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	httpx.JSONSuccess(c, http.StatusOK, user)
}

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

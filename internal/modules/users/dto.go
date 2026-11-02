package users

import (
	"errors"
	"net/mail"
	"strconv"
	"strings"
	"time"

	"github.com/moonrill/go-backend-template/internal/shared/httpx"
)

const (
	DefaultPage     = 1
	DefaultPageSize = 20
	MaxPageSize     = 100
)

var (
	ErrUserNotFound  = errors.New("user not found")
	ErrEmailConflict = errors.New("user with this email already exists")
	ErrInvalidInput  = errors.New("invalid user input")
)

// ValidationError carries field-level validation failures.
type ValidationError struct {
	Fields []httpx.FieldError
}

func (e *ValidationError) Error() string {
	return ErrInvalidInput.Error()
}

func (e *ValidationError) Unwrap() error {
	return ErrInvalidInput
}

// UserResponse is the public API representation of a user.
type UserResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	FullName  string    `json:"full_name"`
	Role      string    `json:"role"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateUserRequest is the payload for POST /api/v1/users.
type CreateUserRequest struct {
	Email    string `json:"email"`
	FullName string `json:"full_name"`
	Role     string `json:"role"`
	IsActive *bool  `json:"is_active"`
}

// UpdateUserRequest is the payload for PATCH /api/v1/users/:id.
type UpdateUserRequest struct {
	Email    *string `json:"email"`
	FullName *string `json:"full_name"`
	Role     *string `json:"role"`
	IsActive *bool   `json:"is_active"`
}

// ListUsersFilter holds validated pagination, filtering, and sorting parameters.
type ListUsersFilter struct {
	Page      int
	PageSize  int
	Role      *string
	IsActive  *bool
	Search    *string
	SortBy    string
	SortOrder string
}

// NormalizeAndValidate validates CreateUserRequest and normalizes fields in-place.
func (r *CreateUserRequest) NormalizeAndValidate() error {
	var fields []httpx.FieldError

	r.Email = strings.ToLower(strings.TrimSpace(r.Email))
	if r.Email == "" {
		fields = append(fields, httpx.FieldError{Field: "email", Message: "email is required"})
	} else if !isValidEmail(r.Email) {
		fields = append(fields, httpx.FieldError{Field: "email", Message: "must be a valid email address (max 255 characters)"})
	}

	r.FullName = strings.TrimSpace(r.FullName)
	if r.FullName == "" {
		fields = append(fields, httpx.FieldError{Field: "full_name", Message: "full_name is required"})
	} else if len([]rune(r.FullName)) < 2 || len([]rune(r.FullName)) > 150 {
		fields = append(fields, httpx.FieldError{Field: "full_name", Message: "must be between 2 and 150 characters"})
	}

	r.Role = strings.ToLower(strings.TrimSpace(r.Role))
	if r.Role == "" {
		r.Role = "member"
	} else if !isAllowedRole(r.Role) {
		fields = append(fields, httpx.FieldError{Field: "role", Message: "must be one of: admin, member, viewer"})
	}

	if len(fields) > 0 {
		return &ValidationError{Fields: fields}
	}
	return nil
}

// NormalizeAndValidate validates UpdateUserRequest and normalizes fields in-place.
func (r *UpdateUserRequest) NormalizeAndValidate() error {
	var fields []httpx.FieldError

	if r.Email == nil && r.FullName == nil && r.Role == nil && r.IsActive == nil {
		fields = append(fields, httpx.FieldError{Field: "body", Message: "at least one field must be provided for update"})
		return &ValidationError{Fields: fields}
	}

	if r.Email != nil {
		trimmed := strings.ToLower(strings.TrimSpace(*r.Email))
		r.Email = &trimmed
		if trimmed == "" || !isValidEmail(trimmed) {
			fields = append(fields, httpx.FieldError{Field: "email", Message: "must be a valid email address (max 255 characters)"})
		}
	}

	if r.FullName != nil {
		trimmed := strings.TrimSpace(*r.FullName)
		r.FullName = &trimmed
		if len([]rune(trimmed)) < 2 || len([]rune(trimmed)) > 150 {
			fields = append(fields, httpx.FieldError{Field: "full_name", Message: "must be between 2 and 150 characters"})
		}
	}

	if r.Role != nil {
		trimmed := strings.ToLower(strings.TrimSpace(*r.Role))
		r.Role = &trimmed
		if !isAllowedRole(trimmed) {
			fields = append(fields, httpx.FieldError{Field: "role", Message: "must be one of: admin, member, viewer"})
		}
	}

	if len(fields) > 0 {
		return &ValidationError{Fields: fields}
	}
	return nil
}

// ParseListUsersFilter parses and validates query parameters for GET /api/v1/users.
func ParseListUsersFilter(pageStr, pageSizeStr, roleStr, isActiveStr, searchStr, sortByStr, sortOrderStr string) (ListUsersFilter, error) {
	var fields []httpx.FieldError
	filter := ListUsersFilter{
		Page:      DefaultPage,
		PageSize:  DefaultPageSize,
		SortBy:    "created_at",
		SortOrder: "desc",
	}

	if s := strings.TrimSpace(pageStr); s != "" {
		p, err := strconv.Atoi(s)
		if err != nil || p < 1 {
			fields = append(fields, httpx.FieldError{Field: "page", Message: "must be a positive integer"})
		} else {
			filter.Page = p
		}
	}

	if s := strings.TrimSpace(pageSizeStr); s != "" {
		ps, err := strconv.Atoi(s)
		if err != nil || ps < 1 || ps > MaxPageSize {
			fields = append(fields, httpx.FieldError{Field: "page_size", Message: "must be an integer between 1 and 100"})
		} else {
			filter.PageSize = ps
		}
	}

	if s := strings.ToLower(strings.TrimSpace(roleStr)); s != "" {
		if !isAllowedRole(s) {
			fields = append(fields, httpx.FieldError{Field: "role", Message: "must be one of: admin, member, viewer"})
		} else {
			filter.Role = &s
		}
	}

	if s := strings.TrimSpace(isActiveStr); s != "" {
		b, err := strconv.ParseBool(s)
		if err != nil {
			fields = append(fields, httpx.FieldError{Field: "is_active", Message: "must be true or false"})
		} else {
			filter.IsActive = &b
		}
	}

	if s := strings.TrimSpace(searchStr); s != "" {
		if len([]rune(s)) > 100 {
			fields = append(fields, httpx.FieldError{Field: "search", Message: "must not exceed 100 characters"})
		} else {
			filter.Search = &s
		}
	}

	if s := strings.ToLower(strings.TrimSpace(sortByStr)); s != "" {
		switch s {
		case "created_at", "full_name", "email":
			filter.SortBy = s
		default:
			fields = append(fields, httpx.FieldError{Field: "sort_by", Message: "must be one of: created_at, full_name, email"})
		}
	}

	if s := strings.ToLower(strings.TrimSpace(sortOrderStr)); s != "" {
		switch s {
		case "asc", "desc":
			filter.SortOrder = s
		default:
			fields = append(fields, httpx.FieldError{Field: "sort_order", Message: "must be asc or desc"})
		}
	}

	if len(fields) > 0 {
		return ListUsersFilter{}, &ValidationError{Fields: fields}
	}
	return filter, nil
}

func isAllowedRole(role string) bool {
	switch role {
	case "admin", "member", "viewer":
		return true
	default:
		return false
	}
}

func isValidEmail(email string) bool {
	if len(email) < 3 || len(email) > 255 {
		return false
	}
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
		return false
	}
	parts := strings.Split(email, "@")
	return len(parts) == 2 && strings.Contains(parts[1], ".") && !strings.HasPrefix(parts[1], ".") && !strings.HasSuffix(parts[1], ".")
}

// IsValidUUID checks whether id is a canonical 36-character UUID string.
func IsValidUUID(id string) bool {
	if len(id) != 36 {
		return false
	}
	for i, r := range id {
		switch i {
		case 8, 13, 18, 23:
			if r != '-' {
				return false
			}
		default:
			if !(r >= '0' && r <= '9') && !(r >= 'a' && r <= 'f') && !(r >= 'A' && r <= 'F') {
				return false
			}
		}
	}
	return true
}

package users

import (
	"context"

	"github.com/moonrill/go-backend-template/internal/platform/postgres/sqlc"
	"github.com/moonrill/go-backend-template/internal/shared/httpx"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, req CreateUserRequest) (UserResponse, error) {
	if err := req.NormalizeAndValidate(); err != nil {
		return UserResponse{}, err
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	u, err := s.repo.Create(ctx, sqlc.CreateUserParams{
		Email:    req.Email,
		FullName: req.FullName,
		Role:     req.Role,
		IsActive: isActive,
	})
	if err != nil {
		return UserResponse{}, err
	}
	return toUserResponse(u), nil
}

func (s *Service) GetByID(ctx context.Context, id string) (UserResponse, error) {
	if !IsValidUUID(id) {
		return UserResponse{}, &ValidationError{
			Fields: []httpx.FieldError{{Field: "id", Message: "must be a valid UUID"}},
		}
	}
	u, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return UserResponse{}, err
	}
	return toUserResponse(u), nil
}

func (s *Service) List(ctx context.Context, filter ListUsersFilter) ([]UserResponse, httpx.PageMeta, error) {
	rows, total, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, httpx.PageMeta{}, err
	}

	items := make([]UserResponse, 0, len(rows))
	for _, r := range rows {
		items = append(items, toUserResponse(r))
	}

	totalPages := 0
	if total > 0 {
		totalPages = int((total + int64(filter.PageSize) - 1) / int64(filter.PageSize))
	}

	meta := httpx.PageMeta{
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		TotalItems: total,
		TotalPages: totalPages,
	}
	return items, meta, nil
}

func (s *Service) Update(ctx context.Context, id string, req UpdateUserRequest) (UserResponse, error) {
	if !IsValidUUID(id) {
		return UserResponse{}, &ValidationError{
			Fields: []httpx.FieldError{{Field: "id", Message: "must be a valid UUID"}},
		}
	}
	if err := req.NormalizeAndValidate(); err != nil {
		return UserResponse{}, err
	}

	u, err := s.repo.Update(ctx, sqlc.UpdateUserParams{
		ID:       id,
		Email:    req.Email,
		FullName: req.FullName,
		Role:     req.Role,
		IsActive: req.IsActive,
	})
	if err != nil {
		return UserResponse{}, err
	}
	return toUserResponse(u), nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	if !IsValidUUID(id) {
		return &ValidationError{
			Fields: []httpx.FieldError{{Field: "id", Message: "must be a valid UUID"}},
		}
	}
	return s.repo.Delete(ctx, id)
}

func toUserResponse(u sqlc.User) UserResponse {
	return UserResponse{
		ID:        u.ID,
		Email:     u.Email,
		FullName:  u.FullName,
		Role:      u.Role,
		IsActive:  u.IsActive,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

package users

import (
	"context"

	"github.com/moonrill/go-backend-template/internal/platform/postgres"
	"github.com/moonrill/go-backend-template/internal/platform/postgres/sqlc"
)

// Repository defines persistence operations for the users module.
type Repository interface {
	Create(ctx context.Context, arg sqlc.CreateUserParams) (sqlc.User, error)
	GetByID(ctx context.Context, id string) (sqlc.User, error)
	List(ctx context.Context, filter ListUsersFilter) ([]sqlc.User, int64, error)
	Update(ctx context.Context, arg sqlc.UpdateUserParams) (sqlc.User, error)
	Delete(ctx context.Context, id string) error
}

type PostgresRepository struct {
	queries *sqlc.Queries
}

func NewRepository(db sqlc.DBTX) *PostgresRepository {
	return &PostgresRepository{
		queries: sqlc.New(db),
	}
}

func (r *PostgresRepository) Create(ctx context.Context, arg sqlc.CreateUserParams) (sqlc.User, error) {
	u, err := r.queries.CreateUser(ctx, arg)
	if err != nil {
		if postgres.IsUniqueViolation(err) {
			return sqlc.User{}, ErrEmailConflict
		}
		return sqlc.User{}, err
	}
	return u, nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, id string) (sqlc.User, error) {
	u, err := r.queries.GetUserByID(ctx, id)
	if err != nil {
		if postgres.IsNotFound(err) {
			return sqlc.User{}, ErrUserNotFound
		}
		return sqlc.User{}, err
	}
	return u, nil
}

func (r *PostgresRepository) List(ctx context.Context, filter ListUsersFilter) ([]sqlc.User, int64, error) {
	total, err := r.queries.CountUsers(ctx, sqlc.CountUsersParams{
		Role:     filter.Role,
		IsActive: filter.IsActive,
		Search:   filter.Search,
	})
	if err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []sqlc.User{}, 0, nil
	}

	offset := int32((filter.Page - 1) * filter.PageSize)
	limit := int32(filter.PageSize)
	items, err := r.queries.ListUsers(ctx, sqlc.ListUsersParams{
		Role:      filter.Role,
		IsActive:  filter.IsActive,
		Search:    filter.Search,
		SortBy:    filter.SortBy,
		SortOrder: filter.SortOrder,
		Offset:    offset,
		Limit:     limit,
	})
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *PostgresRepository) Update(ctx context.Context, arg sqlc.UpdateUserParams) (sqlc.User, error) {
	u, err := r.queries.UpdateUser(ctx, arg)
	if err != nil {
		if postgres.IsNotFound(err) {
			return sqlc.User{}, ErrUserNotFound
		}
		if postgres.IsUniqueViolation(err) {
			return sqlc.User{}, ErrEmailConflict
		}
		return sqlc.User{}, err
	}
	return u, nil
}

func (r *PostgresRepository) Delete(ctx context.Context, id string) error {
	rows, err := r.queries.DeleteUser(ctx, id)
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrUserNotFound
	}
	return nil
}

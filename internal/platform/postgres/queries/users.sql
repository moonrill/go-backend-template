-- name: CreateUser :one
INSERT INTO users (
    email,
    full_name,
    role,
    is_active
) VALUES (
    $1, $2, $3, $4
)
RETURNING id, email, full_name, role, is_active, created_at, updated_at, deleted_at;

-- name: GetUserByID :one
SELECT id, email, full_name, role, is_active, created_at, updated_at, deleted_at
FROM users
WHERE id = $1 AND deleted_at IS NULL;

-- name: UpdateUser :one
UPDATE users
SET
    email = COALESCE(sqlc.narg('email'), email),
    full_name = COALESCE(sqlc.narg('full_name'), full_name),
    role = COALESCE(sqlc.narg('role'), role),
    is_active = COALESCE(sqlc.narg('is_active'), is_active),
    updated_at = NOW()
WHERE id = sqlc.arg('id') AND deleted_at IS NULL
RETURNING id, email, full_name, role, is_active, created_at, updated_at, deleted_at;

-- name: DeleteUser :execrows
UPDATE users
SET
    deleted_at = NOW(),
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;

-- name: CountUsers :one
SELECT COUNT(*)
FROM users
WHERE
    deleted_at IS NULL
    AND (sqlc.narg('role')::varchar IS NULL OR role = sqlc.narg('role'))
    AND (sqlc.narg('is_active')::boolean IS NULL OR is_active = sqlc.narg('is_active'))
    AND (
        sqlc.narg('search')::text IS NULL
        OR email ILIKE '%' || sqlc.narg('search') || '%'
        OR full_name ILIKE '%' || sqlc.narg('search') || '%'
    );

-- name: ListUsers :many
SELECT id, email, full_name, role, is_active, created_at, updated_at, deleted_at
FROM users
WHERE
    deleted_at IS NULL
    AND (sqlc.narg('role')::varchar IS NULL OR role = sqlc.narg('role'))
    AND (sqlc.narg('is_active')::boolean IS NULL OR is_active = sqlc.narg('is_active'))
    AND (
        sqlc.narg('search')::text IS NULL
        OR email ILIKE '%' || sqlc.narg('search') || '%'
        OR full_name ILIKE '%' || sqlc.narg('search') || '%'
    )
ORDER BY
    CASE WHEN sqlc.arg('sort_by')::text = 'created_at' AND sqlc.arg('sort_order')::text = 'asc' THEN created_at END ASC,
    CASE WHEN sqlc.arg('sort_by')::text = 'created_at' AND sqlc.arg('sort_order')::text = 'desc' THEN created_at END DESC,
    CASE WHEN sqlc.arg('sort_by')::text = 'full_name' AND sqlc.arg('sort_order')::text = 'asc' THEN full_name END ASC,
    CASE WHEN sqlc.arg('sort_by')::text = 'full_name' AND sqlc.arg('sort_order')::text = 'desc' THEN full_name END DESC,
    CASE WHEN sqlc.arg('sort_by')::text = 'email' AND sqlc.arg('sort_order')::text = 'asc' THEN email END ASC,
    CASE WHEN sqlc.arg('sort_by')::text = 'email' AND sqlc.arg('sort_order')::text = 'desc' THEN email END DESC,
    id ASC
LIMIT sqlc.arg('limit')
OFFSET sqlc.arg('offset');


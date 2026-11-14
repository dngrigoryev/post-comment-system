-- name: GetUser :one
SELECT id, name
FROM users
WHERE id = sqlc.arg(user_id);

-- name: GetUsersBatch :many
SELECT id, name
FROM users
WHERE id = ANY(sqlc.arg(user_ids)::bigint[]);

-- name: EnsureUser :exec
INSERT INTO users (id, name, email)
VALUES (sqlc.arg(user_id), sqlc.arg(name), sqlc.arg(email))
ON CONFLICT (id) DO NOTHING;

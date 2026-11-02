-- name: ListMessages :many
SELECT id, body, created_at
FROM messages
ORDER BY created_at DESC
LIMIT 50;

-- name: CreateMessage :one
INSERT INTO messages (body)
VALUES ($1)
RETURNING id, body, created_at;

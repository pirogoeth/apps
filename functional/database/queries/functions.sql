-- name: CreateFunction :one
INSERT INTO functions (
    id, name, description, code_path, runtime, handler, 
    timeout_seconds, memory_mb, env_vars
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?, ?
) RETURNING *;

-- name: GetFunction :one
SELECT * FROM functions WHERE id = ?;

-- name: GetFunctionByName :one
SELECT * FROM functions WHERE name = ?;

-- name: ListFunctions :many
SELECT * FROM functions ORDER BY created_at DESC;

-- name: UpdateFunction :one
UPDATE functions 
SET 
    description = ?,
    code_path = ?,
    runtime = ?,
    handler = ?,
    timeout_seconds = ?,
    memory_mb = ?,
    env_vars = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- name: DeleteFunction :exec
DELETE FROM functions WHERE id = ?;
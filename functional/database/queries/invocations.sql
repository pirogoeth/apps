-- name: CreateInvocation :one
INSERT INTO invocations (
    id, function_id, deployment_id, status
) VALUES (
    ?, ?, ?, ?
) RETURNING *;

-- name: GetInvocation :one
SELECT * FROM invocations WHERE id = ?;

-- name: ListInvocations :many
SELECT * FROM invocations ORDER BY created_at DESC LIMIT ? OFFSET ?;

-- name: ListInvocationsByFunction :many
SELECT * FROM invocations 
WHERE function_id = ? 
ORDER BY created_at DESC 
LIMIT ? OFFSET ?;

-- name: UpdateInvocationComplete :one
UPDATE invocations 
SET 
    status = ?,
    duration_ms = ?,
    memory_used_mb = ?,
    response_size_bytes = ?,
    logs = ?,
    error = ?,
    completed_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- name: GetInvocationStats :one
SELECT 
    COUNT(*) as total_invocations,
    COUNT(CASE WHEN status = 'success' THEN 1 END) as successful_invocations,
    COUNT(CASE WHEN status = 'error' THEN 1 END) as failed_invocations,
    AVG(CASE WHEN duration_ms IS NOT NULL THEN duration_ms END) as avg_duration_ms,
    AVG(CASE WHEN memory_used_mb IS NOT NULL THEN memory_used_mb END) as avg_memory_mb
FROM invocations 
WHERE function_id = ? AND created_at >= ?;
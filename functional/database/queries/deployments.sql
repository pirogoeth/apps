-- name: CreateDeployment :one
INSERT INTO deployments (
    id, function_id, provider, resource_id, status, replicas, image_tag
) VALUES (
    ?, ?, ?, ?, ?, ?, ?
) RETURNING *;

-- name: GetDeployment :one
SELECT * FROM deployments WHERE id = ?;

-- name: GetDeploymentsByFunction :many
SELECT * FROM deployments WHERE function_id = ? ORDER BY created_at DESC;

-- name: GetActiveDeploymentByFunction :one
SELECT * FROM deployments 
WHERE function_id = ? AND status = 'active' 
ORDER BY created_at DESC 
LIMIT 1;

-- name: UpdateDeploymentStatus :one
UPDATE deployments 
SET 
    status = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- name: UpdateDeploymentReplicas :one
UPDATE deployments 
SET 
    replicas = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- name: DeleteDeployment :exec
DELETE FROM deployments WHERE id = ?;
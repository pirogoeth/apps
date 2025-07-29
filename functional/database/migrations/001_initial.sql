-- +goose Up
-- +goose StatementBegin
CREATE TABLE functions (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    code_path TEXT NOT NULL,
    runtime TEXT NOT NULL,
    handler TEXT NOT NULL,
    timeout_seconds INTEGER NOT NULL DEFAULT 30,
    memory_mb INTEGER NOT NULL DEFAULT 128,
    env_vars TEXT, -- JSON
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE deployments (
    id TEXT PRIMARY KEY,
    function_id TEXT NOT NULL,
    provider TEXT NOT NULL,
    resource_id TEXT NOT NULL,
    status TEXT NOT NULL,
    replicas INTEGER NOT NULL DEFAULT 1,
    image_tag TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (function_id) REFERENCES functions(id) ON DELETE CASCADE
);

CREATE TABLE invocations (
    id TEXT PRIMARY KEY,
    function_id TEXT NOT NULL,
    deployment_id TEXT,
    status TEXT NOT NULL,
    duration_ms INTEGER,
    memory_used_mb INTEGER,
    response_size_bytes INTEGER,
    logs TEXT,
    error TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    completed_at DATETIME,
    FOREIGN KEY (function_id) REFERENCES functions(id) ON DELETE CASCADE,
    FOREIGN KEY (deployment_id) REFERENCES deployments(id) ON DELETE SET NULL
);

CREATE INDEX idx_functions_name ON functions(name);
CREATE INDEX idx_deployments_function_id ON deployments(function_id);
CREATE INDEX idx_deployments_status ON deployments(status);
CREATE INDEX idx_invocations_function_id ON invocations(function_id);
CREATE INDEX idx_invocations_status ON invocations(status);
CREATE INDEX idx_invocations_created_at ON invocations(created_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_invocations_created_at;
DROP INDEX IF EXISTS idx_invocations_status;
DROP INDEX IF EXISTS idx_invocations_function_id;
DROP INDEX IF EXISTS idx_deployments_status;
DROP INDEX IF EXISTS idx_deployments_function_id;
DROP INDEX IF EXISTS idx_functions_name;
DROP TABLE IF EXISTS invocations;
DROP TABLE IF EXISTS deployments;
DROP TABLE IF EXISTS functions;
-- +goose StatementEnd
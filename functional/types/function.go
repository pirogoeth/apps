package types

import (
	"time"
)

type Function struct {
	ID             string            `json:"id" db:"id"`
	Name           string            `json:"name" db:"name"`
	Description    string            `json:"description" db:"description"`
	CodePath       string            `json:"code_path" db:"code_path"`
	Runtime        string            `json:"runtime" db:"runtime"`
	Handler        string            `json:"handler" db:"handler"`
	TimeoutSeconds int32             `json:"timeout_seconds" db:"timeout_seconds"`
	MemoryMB       int32             `json:"memory_mb" db:"memory_mb"`
	EnvVars        string            `json:"env_vars" db:"env_vars"` // JSON string
	CreatedAt      time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at" db:"updated_at"`
}

type CreateFunctionRequest struct {
	Name           string            `json:"name" binding:"required"`
	Description    string            `json:"description"`
	Runtime        string            `json:"runtime" binding:"required"`
	Handler        string            `json:"handler" binding:"required"`
	TimeoutSeconds int32             `json:"timeout_seconds"`
	MemoryMB       int32             `json:"memory_mb"`
	EnvVars        map[string]string `json:"env_vars"`
	Code           string            `json:"code" binding:"required"` // Base64 encoded ZIP
}

type UpdateFunctionRequest struct {
	Description    *string           `json:"description"`
	Runtime        *string           `json:"runtime"`
	Handler        *string           `json:"handler"`
	TimeoutSeconds *int32            `json:"timeout_seconds"`
	MemoryMB       *int32            `json:"memory_mb"`
	EnvVars        map[string]string `json:"env_vars"`
	Code           *string           `json:"code"` // Base64 encoded ZIP
}
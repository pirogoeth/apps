package types

import (
	"context"
	"time"
)

type ComputeProvider interface {
	Name() string
	Deploy(ctx context.Context, fn *Function, imageName string) (*DeployResult, error)
	Execute(ctx context.Context, deployment *Deployment, req *InvocationRequest) (*InvocationResult, error)
	Scale(ctx context.Context, deployment *Deployment, replicas int) error
	Remove(ctx context.Context, deployment *Deployment) error
	Health(ctx context.Context) error
}

type Deployment struct {
	ID         string            `json:"id" db:"id"`
	FunctionID string            `json:"function_id" db:"function_id"`
	Provider   string            `json:"provider" db:"provider"`
	ResourceID string            `json:"resource_id" db:"resource_id"`
	Status     DeploymentStatus  `json:"status" db:"status"`
	Replicas   int32             `json:"replicas" db:"replicas"`
	ImageTag   string            `json:"image_tag" db:"image_tag"`
	CreatedAt  time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at" db:"updated_at"`
}

type DeploymentStatus string

const (
	DeploymentStatusPending   DeploymentStatus = "pending"
	DeploymentStatusBuilding  DeploymentStatus = "building"
	DeploymentStatusActive    DeploymentStatus = "active"
	DeploymentStatusFailed    DeploymentStatus = "failed"
	DeploymentStatusStopped   DeploymentStatus = "stopped"
)

type DeployResult struct {
	DeploymentID string `json:"deployment_id"`
	ResourceID   string `json:"resource_id"`
	ImageTag     string `json:"image_tag"`
}

type InvocationRequest struct {
	FunctionID string            `json:"function_id"`
	Body       []byte            `json:"body"`
	Headers    map[string]string `json:"headers"`
	Method     string            `json:"method"`
	Path       string            `json:"path"`
	QueryArgs  map[string]string `json:"query_args"`
}

type InvocationResult struct {
	StatusCode    int               `json:"status_code"`
	Body          []byte            `json:"body"`
	Headers       map[string]string `json:"headers"`
	DurationMS    int64             `json:"duration_ms"`
	MemoryUsedMB  int32             `json:"memory_used_mb"`
	ResponseSize  int64             `json:"response_size"`
	Logs          string            `json:"logs"`
	Error         string            `json:"error,omitempty"`
}

type Invocation struct {
	ID                string     `json:"id" db:"id"`
	FunctionID        string     `json:"function_id" db:"function_id"`
	DeploymentID      *string    `json:"deployment_id" db:"deployment_id"`
	Status            string     `json:"status" db:"status"`
	DurationMS        *int64     `json:"duration_ms" db:"duration_ms"`
	MemoryUsedMB      *int32     `json:"memory_used_mb" db:"memory_used_mb"`
	ResponseSizeBytes *int64     `json:"response_size_bytes" db:"response_size_bytes"`
	Logs              *string    `json:"logs" db:"logs"`
	Error             *string    `json:"error" db:"error"`
	CreatedAt         time.Time  `json:"created_at" db:"created_at"`
	CompletedAt       *time.Time `json:"completed_at" db:"completed_at"`
}

type InvocationStatus string

const (
	InvocationStatusPending InvocationStatus = "pending"
	InvocationStatusRunning InvocationStatus = "running"
	InvocationStatusSuccess InvocationStatus = "success"
	InvocationStatusError   InvocationStatus = "error"
	InvocationStatusTimeout InvocationStatus = "timeout"
)
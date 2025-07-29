package interfaces

import (
	"context"
)

type ComputeProvider interface {
	Name() string
	Deploy(ctx context.Context, fn *Function, imageName string) (*DeployResult, error)
	Execute(ctx context.Context, deployment *Deployment, req *InvocationRequest) (*InvocationResult, error)
	Scale(ctx context.Context, deployment *Deployment, replicas int) error
	Remove(ctx context.Context, deployment *Deployment) error
	Health(ctx context.Context) error
}

// Forward declare types to avoid circular imports
type Function interface{}
type Deployment interface{}
type DeployResult interface{}
type InvocationRequest interface{}
type InvocationResult interface{}
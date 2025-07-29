package compute

import (
	"context"
	"fmt"
)

type ComputeProvider interface {
	Name() string
	Deploy(ctx context.Context, fn interface{}, imageName string) (interface{}, error)
	Execute(ctx context.Context, deployment interface{}, req interface{}) (interface{}, error)
	Scale(ctx context.Context, deployment interface{}, replicas int) error
	Remove(ctx context.Context, deployment interface{}) error
	Health(ctx context.Context) error
}

type Registry struct {
	providers map[string]ComputeProvider
}

func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]ComputeProvider),
	}
}

func (r *Registry) Register(provider ComputeProvider) {
	r.providers[provider.Name()] = provider
}

func (r *Registry) Get(name string) (ComputeProvider, error) {
	if provider, ok := r.providers[name]; ok {
		return provider, nil
	}
	return nil, fmt.Errorf("compute provider %q not found", name)
}

func (r *Registry) List() []string {
	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	return names
}
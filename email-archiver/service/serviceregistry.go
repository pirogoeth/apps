package service

import (
	"maps"
	"sync"

	"github.com/pirogoeth/apps/pkg/errors"
)

func GetAs[T any](sr *ServiceRegistry, name string) (T, bool) {
	service, found := sr.Get(name)
	if !found {
		return service.(T), false
	}

	casted, ok := service.(T)
	return casted, ok
}

// ServiceRegistry is a central place to register and retrieve services.
type ServiceRegistry struct {
	mu       sync.RWMutex
	services map[string]Service
}

// NewServiceRegistry creates a new ServiceRegistry.
func NewServiceRegistry() *ServiceRegistry {
	return &ServiceRegistry{
		services: make(map[string]Service),
	}
}

// Register registers a service with the registry.
func (sr *ServiceRegistry) Register(name string, service Service) {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	sr.services[name] = service
}

// Get retrieves a service from the registry.
func (sr *ServiceRegistry) Get(name string) (Service, bool) {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	service, found := sr.services[name]
	return service, found
}

func (sr *ServiceRegistry) Each(fn func(Service)) {
	for svc := range maps.Values(sr.services) {
		fn(svc)
	}
}

func (sr *ServiceRegistry) Close() error {
	err := new(errors.MultiError)

	sr.Each(func(svc Service) {
		err.Add(svc.Close())
	})

	return err
}

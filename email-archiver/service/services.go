package service

import (
	"github.com/pirogoeth/apps/email-archiver/config"
)

type Service interface {
	Close() error
}

func InitServices(cfg *config.Config) *ServiceRegistry {
	registry := NewServiceRegistry()

	mailhost := newMailhostService(cfg, registry)
	registry.Register("Mailhost", mailhost)

	message := newMessageService(cfg, registry)
	registry.Register("Message", message)

	storage := newStorageService(cfg, registry)
	registry.Register("Storage", storage)

	return registry
}

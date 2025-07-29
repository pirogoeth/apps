package service

import (
	"github.com/pirogoeth/apps/email-archiver/config"
)

var _ Service = (*StorageService)(nil)

type StorageService struct {
	cfg      *config.Config
	registry *ServiceRegistry
}

func newStorageService(cfg *config.Config, registry *ServiceRegistry) *StorageService {
	return &StorageService{
		cfg:      cfg,
		registry: registry,
	}
}

func (s *StorageService) Close() error {
	return nil
}

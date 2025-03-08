package worker

import (
	"context"

	"github.com/pirogoeth/apps/email-archiver/config"
	"github.com/pirogoeth/apps/email-archiver/search"
	"github.com/pirogoeth/apps/email-archiver/types"
)

type emailArchiveWorker struct {
	cfg *config.Config
}

func NewEmailArchiveWorker(cfg *config.Config) *emailArchiveWorker {
	return &emailArchiveWorker{
		cfg: cfg,
	}
}

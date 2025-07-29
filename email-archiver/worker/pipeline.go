package worker

import (
	"context"

	"github.com/pirogoeth/apps/email-archiver/config"
	"github.com/pirogoeth/apps/email-archiver/search"
	"github.com/pirogoeth/apps/email-archiver/service"
	"github.com/pirogoeth/apps/email-archiver/types"
	"github.com/pirogoeth/apps/pkg/pipeline"
)

type (
	PipelineData   = *types.ArchiverPipelineData
	PipelineInlet  = pipeline.Inlet[PipelineData]
	PipelineOutlet = pipeline.Outlet[PipelineData]
)

type Deps struct {
	Config          *config.Config
	Searcher        *search.Searcher
	ServiceRegistry *service.ServiceRegistry
}

func RunWorkerPipeline(ctx context.Context, deps *Deps) error {
	p := pipeline.NewPipeline(
		deps,
		NewEmailScannerWorker,
		NewEmailArchiveWorker,
		NewSearchIngestWorker,
	)
	return p.Run(ctx)
}

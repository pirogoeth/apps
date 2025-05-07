package worker

import (
	"context"

	"github.com/pirogoeth/apps/email-archiver/config"
	"github.com/pirogoeth/apps/email-archiver/search"
	"github.com/pirogoeth/apps/email-archiver/service"
	"github.com/pirogoeth/apps/pkg/pipeline"
	"github.com/sirupsen/logrus"
)

type searchIngestWorker struct {
	*pipeline.StageFitting[PipelineData]

	cfg      *config.Config
	searcher *search.Searcher
	services *service.Services
}

func NewSearchIngestWorker(deps *Deps, inlet PipelineInlet, outlet PipelineOutlet) pipeline.Stage {
	return &searchIngestWorker{
		StageFitting: pipeline.NewStageFitting(inlet, outlet),
		cfg:          deps.Config,
		searcher:     deps.Searcher,
		services:     deps.Services,
	}
}

// Run starts a loop that monitors for email envelopes and bodies to ingest into the searcher system
func (w *searchIngestWorker) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case searchData := <-w.Inlet():
			if err := w.ingestSearchData(ctx, searchData); err != nil {
				return err
			}

			w.Write(searchData)
		}
	}
}

func (w *searchIngestWorker) ingestSearchData(ctx context.Context, sd PipelineData) error {
	ingestHandle := w.searcher.ForTime(sd.Envelope.Date).IndexerHandle()
	defer ingestHandle.Close()

	// TODO: the thing
	data := make(map[string]any)
	data["envelope"] = sd.Envelope
	data["body"] = sd.Body
	data["mailbox"] = sd.Mailbox
	data["uid"] = sd.UID
	err := ingestHandle.Index().Index(sd.UID.String(), data)
	if err != nil {
		return err
	}

	logrus.Infof("Successfully indexed message %s", sd.UID)

	return nil
}

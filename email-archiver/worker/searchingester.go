package worker

import (
	"context"
	"fmt"

	"github.com/pirogoeth/apps/email-archiver/config"
	"github.com/pirogoeth/apps/email-archiver/search"
	"github.com/pirogoeth/apps/email-archiver/types"
)

type searchIngestWorker struct {
	cfg      *config.Config
	queue    chan *types.SearchData
	searcher *search.Searcher
}

func NewSearchIngestWorker(cfg *config.Config, searcher *search.Searcher) *searchIngestWorker {
	return &searchIngestWorker{
		cfg:      cfg,
		queue:    make(chan *types.SearchData, cfg.Worker.Indexer.QueueSize),
		searcher: searcher,
	}
}

// GetSender returns the sender end of the channel for usage in the emailscanner
func (w *searchIngestWorker) GetSender() types.SearchDataSender {
	return w.queue
}

// Run starts a loop that monitors for email envelopes and bodies to ingest into the searcher system
func (w *searchIngestWorker) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case searchData := <-w.queue:
			if err := w.ingestSearchData(ctx, searchData); err != nil {
				return err
			}
		}
	}
}

func (w *searchIngestWorker) ingestSearchData(ctx context.Context, sd *types.SearchData) error {
	ingestHandle := w.searcher.ForTime(sd.Envelope.Date).IndexerHandle()
	defer ingestHandle.Close()

	// TODO: the thing
	ingestHandle.Index()

	return fmt.Errorf("not implemented")
}

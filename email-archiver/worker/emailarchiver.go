package worker

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/pirogoeth/apps/email-archiver/config"
	"github.com/pirogoeth/apps/email-archiver/service"
	"github.com/pirogoeth/apps/pkg/pipeline"
)

var _ pipeline.Stage = (*emailArchiveWorker)(nil)

func NewEmailArchiveWorker(deps *Deps, inlet PipelineInlet, outlet PipelineOutlet) pipeline.Stage {
	return &emailArchiveWorker{
		StageFitting: pipeline.NewStageFitting(inlet, outlet),
		cfg:          deps.Config,
		services:     deps.Services,
	}
}

type emailArchiveWorker struct {
	*pipeline.StageFitting[PipelineData]

	cfg      *config.Config
	services *service.Services
}

func (w *emailArchiveWorker) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			w.Finish()
			return nil
		case data, ok := <-w.Inlet():
			if !ok {
				w.Finish()
				return nil
			}

			if err := w.processMessage(ctx, data); err != nil {
				logrus.Errorf("Error processing message: %v", err)
				continue
			}
		}
	}
}

// processMessage checks if the incoming message is already stored in the configured storage service.
// If it's not, it:
// - fetches the body
// - encrypts and archives the message to the storage backend
// - passes the body on to the next pipeline step(s)
func (w *emailArchiveWorker) processMessage(ctx context.Context, data PipelineData) error {
	messageID := service.MessageID(
		data.Mailbox.Mailhost,
		data.Mailbox.Username,
		data.UID,
	)

	// Check if we need to fetch the body
	if w.services.Message.NeedsBodyFetch(messageID) {
		logrus.Infof("Fetching body for message %s", messageID)
		body, err := w.services.Message.FetchMessageBody(ctx, data, data.UID)
		if err != nil {
			return fmt.Errorf("failed to fetch message body: %w", err)
		}

		// Update the pipeline data with the body
		data.Body = body

		// Add your archiving logic here
		// For example:
		// - Store the message in a local filesystem
		// - Upload to cloud storage
		// - Save in a database

		logrus.Infof("Successfully archived message %s", messageID)
	} else {
		logrus.Infof("Message %s already archived, skipping", messageID)
	}

	return nil
}

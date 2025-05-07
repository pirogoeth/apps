package worker

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	"github.com/pirogoeth/apps/email-archiver/config"
	"github.com/pirogoeth/apps/email-archiver/service"
	"github.com/pirogoeth/apps/email-archiver/types"
	"github.com/pirogoeth/apps/pkg/pipeline"
)

var _ pipeline.Stage = (*emailScannerWorker)(nil)

type emailScannerWorker struct {
	*pipeline.StageFitting[PipelineData]

	cfg             *config.Config
	serviceRegistry *service.ServiceRegistry
}

func NewEmailScannerWorker(deps *Deps, inlet PipelineInlet, outlet PipelineOutlet) pipeline.Stage {
	return &emailScannerWorker{
		StageFitting:    pipeline.NewStageFitting(inlet, outlet),
		cfg:             deps.Config,
		serviceRegistry: deps.ServiceRegistry,
	}
}

func (w *emailScannerWorker) Run(ctx context.Context) error {
	nextScanInterval, err := time.ParseDuration(w.cfg.Worker.ScanInterval)
	if err != nil {
		panic(fmt.Errorf("could not parse scan interval: %w", err))
	}
	scanInterval := 5 * time.Second

	eg, scanCtx := errgroup.WithContext(ctx)
	for {
		select {
		case <-scanCtx.Done():
			err := eg.Wait()
			if err != nil {
				logrus.Errorf("error while scanning inboxes: %s", err)
			}
		case <-ctx.Done():
			w.Finish()
			return nil
		case <-time.After(scanInterval):
			for _, mhCfg := range w.cfg.Inboxes {
				eg.Go(func() error {
					err := w.scanInbox(ctx, mhCfg)
					logrus.Infof("Scanner returned: %s", err)
					return err
				})
			}

			scanInterval = nextScanInterval
			logrus.Debugf("Setting next scan interval to %s", scanInterval)

			eg, scanCtx = errgroup.WithContext(ctx)
		}
	}
}

func (w *emailScannerWorker) scanInbox(ctx context.Context, mhCfg config.MailhostConfig) error {
	mhSvc, _ := service.GetAs[*service.MailhostService](w.serviceRegistry, "Mailhost")
	imapC, err := mhSvc.Connection(mhCfg)
	if err != nil {
		return fmt.Errorf("while scanning inboxes: %w", err)
	}

	targetMailboxes, err := w.collectChildMailboxes(ctx, imapC, mhCfg)
	if err != nil {
		return fmt.Errorf("could not collect mailboxes: %#v: %w", mhCfg, err)
	}

	logrus.Debugf("Collected mailboxes: %#v", targetMailboxes)

	// For each mailbox that exists, we need to be able to accept a series of child mailboxes,
	// but don't want to pre-allocate a channel that can accept all mailboxes. Instead, how about
	// a channel that the scanner sends channels through!
	for _, mailboxCfg := range targetMailboxes {
		if err := w.scanMailbox(ctx, imapC, mhCfg, mailboxCfg); err != nil {
			logrus.Errorf("error while scanning mailbox %s on %s: %s", mailboxCfg.Name, mhCfg.InboxAddr(), err)
		}
	}

	return err
}

func (w *emailScannerWorker) collectChildMailboxes(ctx context.Context, imapC *imapclient.Client, mhCfg config.MailhostConfig) ([]config.MailboxConfig, error) {
	mbList, err := imapC.List("", "*", nil).Collect()
	if err != nil {
		return nil, fmt.Errorf("could not list mailboxes: %w", err)
	}

	var targetMailboxes []config.MailboxConfig
	for _, mbCfg := range mhCfg.Mailboxes {
		for _, mb := range mbList {
			if !mbCfg.IncludeChildren && mb.Mailbox == mbCfg.Name {
				targetMailboxes = append(targetMailboxes, mbCfg)
			} else if mbCfg.IncludeChildren && strings.HasPrefix(mb.Mailbox, mbCfg.Name) {
				targetMailboxes = append(targetMailboxes, mbCfg.CloneChild(mb.Mailbox))
			}
		}
	}

	return targetMailboxes, nil
}

func (w *emailScannerWorker) scanMailbox(ctx context.Context, imapC *imapclient.Client, mhCfg config.MailhostConfig, mbCfg config.MailboxConfig) error {
	activeMailbox, err := imapC.Select(mbCfg.Name, nil).Wait()
	if err != nil {
		return fmt.Errorf("could not select mailbox: %s: %w", mbCfg.Name, err)
	}

	logrus.Debugf("Mailbox `%s` has %d messages", mbCfg.Name, activeMailbox.NumMessages)

	msgUids, err := imapC.Sort(&imapclient.SortOptions{
		SearchCriteria: &imap.SearchCriteria{
			Flag:    mbCfg.Flags,
			NotFlag: mbCfg.IgnoreFlags,
		},
		SortCriteria: []imapclient.SortCriterion{{
			Key:     imapclient.SortKeyDate,
			Reverse: true,
		}},
	}).Wait()
	if err != nil {
		return fmt.Errorf("could not sort messages: %w", err)
	}

	if len(msgUids) == 0 {
		logrus.Debugf("No messages remaining after applying search filters: flag=%v notFlag=%v",
			mbCfg.Flags, mbCfg.IgnoreFlags,
		)
		return nil
	}

	for chunk := range slices.Chunk(msgUids, mhCfg.FetchBatchSize) {
		seq := imap.SeqSetNum(chunk...)
		// TODO: Instead of always fetching, we can try to defer this!
		// The scanner should just gather all of the messages according
		// to the inboxes/mailboxes configuration, and pass that off, but
		// support a future pipeline step being able to "backfill" all of the message data.
		//
		// As probably icky as this may be, could we define a closured backfill function on the data
		// we pass along to backfill the message data?
		initialFetch := imapC.Fetch(seq, &imap.FetchOptions{
			Envelope:     true,
			InternalDate: true,
			Flags:        true,
			UID:          true,

			// BodyStructure: &imap.FetchItemBodyStructure{Extended: true},
		})
		defer initialFetch.Close()

		for msg := initialFetch.Next(); msg != nil; msg = initialFetch.Next() {
			msgBuf, err := msg.Collect()
			if err != nil {
				return fmt.Errorf("could not collect message: %w", err)
			}

			// At this point, feed the message in to the searcher for indexing
			logrus.Infof("Message: %#v", msgBuf)
			logrus.Infof("Envelope: %#v", msgBuf.Envelope)
			w.Write(PipelineData(&types.ArchiverPipelineData{
				Envelope: msgBuf.Envelope,
				UID:      types.NewImapUID(mbCfg.Name, activeMailbox.UIDValidity, msgBuf.UID),
				Mailbox: &types.MailboxData{
					InboxAddr: mhCfg.InboxAddr(),
					Mailhost:  mhCfg.Host,
					Username:  mhCfg.Username,
					Mailbox:   mbCfg.Name,
				},
			}))
		}
	}

	return err
}

package worker

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	"github.com/pirogoeth/apps/email-archiver/config"
)

type worker struct {
	cfg *config.Config
}

func New(cfg *config.Config) *worker {
	return &worker{
		cfg: cfg,
	}
}

func (w *worker) Run(ctx context.Context) {
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
			return
		case <-time.After(scanInterval):
			for _, inboxCfg := range w.cfg.Inboxes {
				eg.Go(func() error {
					err := w.scanInbox(ctx, inboxCfg)
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

func (w *worker) scanInbox(ctx context.Context, inboxCfg config.InboxConfig) error {
	var imapC *imapclient.Client
	var err error

	logrus.Infof("Opening connection to %s", inboxCfg.Host)

	clientOpts := &imapclient.Options{}
	if w.cfg.Worker.Debug.Imap {
		clientOpts.DebugWriter = logrus.StandardLogger().WriterLevel(logrus.DebugLevel)
	}

	inboxAddr := inboxCfg.InboxAddr()
	if inboxCfg.UseTLS {
		imapC, err = imapclient.DialTLS(inboxAddr, clientOpts)
	} else {
		imapC, err = imapclient.DialInsecure(inboxAddr, clientOpts)
	}
	if err != nil {
		return fmt.Errorf("could not dial imap for inbox: %s: %w", inboxAddr, err)
	}

	if imapC.Login(inboxCfg.Username, inboxCfg.Password).Wait() != nil {
		return fmt.Errorf("could not log in to inbox: %s: %w", inboxAddr, err)
	}
	defer imapC.Logout()

	needCaps := []imap.Cap{imap.CapSort, imap.CapESearch}
	for _, cap := range needCaps {
		if !imapC.Caps().Has(cap) {
			return fmt.Errorf("server %s does not support %s, can not continue", inboxAddr, cap)
		}
	}

	targetMailboxes, err := w.collectChildMailboxes(ctx, imapC, inboxCfg)
	if err != nil {
		return fmt.Errorf("could not collect mailboxes: %#v: %w", inboxCfg, err)
	}

	logrus.Debugf("Collected mailboxes: %#v", targetMailboxes)

	// For each mailbox that exists, we need to be able to accept a series of child mailboxes,
	// but don't want to pre-allocate a channel that can accept all mailboxes. Instead, how about
	// a channel that the scanner sends channels through!
	for _, mailboxCfg := range targetMailboxes {
		if err := w.scanMailbox(ctx, imapC, mailboxCfg); err != nil {
			logrus.Errorf("error while scanning mailbox %s on %s: %s", mailboxCfg.Name, inboxAddr, err)
		}
	}

	return err
}

func (w *worker) collectChildMailboxes(ctx context.Context, imapC *imapclient.Client, inboxCfg config.InboxConfig) ([]config.MailboxConfig, error) {
	mbList, err := imapC.List("", "*", nil).Collect()
	if err != nil {
		return nil, fmt.Errorf("could not list mailboxes: %w", err)
	}

	var targetMailboxes []config.MailboxConfig
	for _, mbCfg := range inboxCfg.Mailboxes {
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

func (w *worker) scanMailbox(ctx context.Context, imapC *imapclient.Client, mbCfg config.MailboxConfig) error {
	selection, err := imapC.Select(mbCfg.Name, nil).Wait()
	if err != nil {
		return fmt.Errorf("could not select mailbox: %s: %w", mbCfg.Name, err)
	}

	logrus.Debugf("Mailbox %s has %d messages", mbCfg.Name, selection.NumMessages)

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

	firstMsg := msgUids[0]
	lastMsg := msgUids[len(msgUids)-1]

	seq := imap.SeqSetNum(firstMsg, lastMsg)
	initialFetch := imapC.Fetch(seq, &imap.FetchOptions{
		Envelope:     true,
		InternalDate: true,
		Flags:        true,
	})
	defer initialFetch.Close()

	for msg := initialFetch.Next(); msg != nil; msg = initialFetch.Next() {
		msgBuf, err := msg.Collect()
		if err != nil {
			return fmt.Errorf("could not collect message: %w", err)
		}

		logrus.Infof("Message: %#v", msgBuf)
		logrus.Infof("Envelope: %#v", msgBuf.Envelope)
	}

	return err
}

package service

import (
	"errors"
	"fmt"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/pirogoeth/apps/email-archiver/config"
	"github.com/sirupsen/logrus"
)

var ErrMailhostConnNotFound = errors.New("mailhost connection not found")

var _ Service = (*MailhostService)(nil)

type MailhostService struct {
	cfg      *config.Config
	registry *ServiceRegistry
	conns    map[string]*imapclient.Client
}

func newMailhostService(cfg *config.Config, registry *ServiceRegistry) *MailhostService {
	return &MailhostService{
		cfg:      cfg,
		registry: registry,
		conns:    make(map[string]*imapclient.Client),
	}
}

func (s *MailhostService) Close() error {
	return nil
}

func (s *MailhostService) ConnectWithConfig(mhCfg config.MailhostConfig) (*imapclient.Client, error) {
	conn, err := s.getConn(mhCfg.InboxAddr())
	if err != nil {
		if err == ErrMailhostConnNotFound {
			return s.openConn(mhCfg.InboxAddr(), mhCfg.Username, mhCfg.Password, mhCfg.UseTLS)
		}

		return nil, err
	}

	return conn, nil
}

func (s *MailhostService) getConn(mailhostAddr string) (*imapclient.Client, error) {
	mhClient, ok := s.conns[mailhostAddr]
	if !ok {
		return nil, ErrMailhostConnNotFound
	}

	return mhClient, nil
}

func (s *MailhostService) openConn(mailhostAddr, username, password string, useTLS bool) (*imapclient.Client, error) {
	var imapC *imapclient.Client
	var err error

	logrus.Infof("Opening connection to %s", mailhostAddr)

	clientOpts := &imapclient.Options{}
	if s.cfg.Worker.Debug.Imap {
		clientOpts.DebugWriter = logrus.StandardLogger().WriterLevel(logrus.DebugLevel)
	}

	if useTLS {
		imapC, err = imapclient.DialTLS(mailhostAddr, clientOpts)
	} else {
		imapC, err = imapclient.DialInsecure(mailhostAddr, clientOpts)
	}
	if err != nil {
		return nil, fmt.Errorf("could not dial imap for inbox: %s: %w", mailhostAddr, err)
	}

	if err := imapC.Login(username, password).Wait(); err != nil {
		return nil, fmt.Errorf("could not log in to inbox: %s@%s: %w", username, mailhostAddr, err)
	}

	logrus.Debugf("Remote %s supports caps: %#v", mailhostAddr, imapC.Caps())

	needCaps := []imap.Cap{imap.CapSort, imap.CapESearch, imap.CapCondStore}
	for _, cap := range needCaps {
		if !imapC.Caps().Has(cap) {
			return nil, fmt.Errorf("server %s does not support %s, can not continue", mailhostAddr, cap)
		}
	}

	return imapC, nil
}

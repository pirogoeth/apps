package service

import (
	"context"
	"fmt"
	"sync"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/pirogoeth/apps/email-archiver/config"
	"github.com/pirogoeth/apps/email-archiver/types"
)

var _ Service = (*MessageService)(nil)

// MessageService handles message operations including metadata caching and body fetching
type MessageService struct {
	cfg      *config.Config
	registry *ServiceRegistry
	mu       sync.RWMutex

	// Track active connections to reuse them
	connections map[string]*imapclient.Client

	// Track which messages have had their bodies fetched
	fetchedBodies map[string]bool
}

func newMessageService(cfg *config.Config, registry *ServiceRegistry) *MessageService {
	return &MessageService{
		cfg:           cfg,
		registry:      registry,
		connections:   make(map[string]*imapclient.Client),
		fetchedBodies: make(map[string]bool),
	}
}

// MessageID creates a unique identifier for a message
func MessageID(mailhost, username string, uid types.ImapUID) string {
	return fmt.Sprintf("%s:%s:%s", mailhost, username, uid.String())
}

// RegisterMessage registers a new message with just envelope data
func (s *MessageService) RegisterMessage(ctx context.Context, data *types.ArchiverPipelineData) {
	// Store metadata if needed
	// This is where you'd update your envelope cache if necessary
}

// NeedsBodyFetch determines if a message needs its body fetched
func (s *MessageService) NeedsBodyFetch(messageID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Here you'd implement your logic to check if a message needs its body
	// e.g., check if it's already been archived
	return !s.fetchedBodies[messageID]
}

// FetchMessageBody fetches the full body for a message
func (s *MessageService) FetchMessageBody(ctx context.Context, data *types.ArchiverPipelineData, uid types.ImapUID) ([]byte, error) {
	mh := data.Mailbox

	// Get a connection from the Mailhost service
	mhSvc, _ := GetAs[*MailhostService](s.registry, "Mailhost")

	// Get or create IMAP connection
	// TODO: How do we manage/reference connections? Passing a config everywhere to get a connection would suck
	client, err := mhSvc.ConnectWithConfig(mh)
	if err != nil {
		return nil, err
	}

	// Select the mailbox
	_, err = client.Select(mh.Mailbox, nil).Wait()
	if err != nil {
		return nil, fmt.Errorf("failed to select mailbox %s: %w", mh.Mailbox, err)
	}

	// Fetch the message body
	set := imap.UIDSetNum(uid.MessageUID())
	fetch := client.Fetch(set, &imap.FetchOptions{
		BodySection: []*imap.FetchItemBodySection{
			{}, // Fetch entire message
		},
	})
	defer fetch.Close()

	msg := fetch.Next()
	if msg == nil {
		return nil, fmt.Errorf("message not found")
	}

	msgData, err := msg.Collect()
	if err != nil {
		return nil, err
	}

	// Get body content from the first (and only) body section
	if len(msgData.BodySection) == 0 {
		return nil, fmt.Errorf("body section not found")
	}

	// body := msgData.BodySection.

	// Mark as fetched
	messageID := MessageID(mh.Mailhost, mh.Username, uid)
	s.mu.Lock()
	s.fetchedBodies[messageID] = true
	s.mu.Unlock()

	return nil, nil
}

// Close closes all connections
func (s *MessageService) Close() error {
	return nil
}

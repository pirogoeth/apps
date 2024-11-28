package config

import (
	"fmt"

	"github.com/emersion/go-imap/v2"
	"github.com/pirogoeth/apps/pkg/config"
)

type InboxConfig struct {
	Username  string          `json:"username"`
	Password  string          `json:"password"`
	Host      string          `json:"host"`
	Port      int             `json:"port"`
	UseTLS    bool            `json:"use_tls"`
	Mailboxes []MailboxConfig `json:"mailboxes"`
}

func (i *InboxConfig) InboxAddr() string {
	return fmt.Sprintf("%s:%d", i.Host, i.Port)
}

type MailboxConfig struct {
	Name string `json:"name"`
	// IncludeChildren will include all children of the mailbox in the search
	IncludeChildren bool `json:"include_children"`
	// Flags are flags to include on the search
	Flags []imap.Flag `json:"flags"`
	// IgnoreFlags are flags to ignore on the search
	IgnoreFlags []imap.Flag `json:"ignore_flags"`
}

func (mb MailboxConfig) CloneChild(childMailbox string) MailboxConfig {
	return MailboxConfig{
		Name:            childMailbox,
		IncludeChildren: true,
		Flags:           mb.Flags,
		IgnoreFlags:     mb.IgnoreFlags,
	}
}

type SearchConfig struct {
	Index struct {
		BaseDir string `json:"base_dir" envconfig:"INDEX_BASE_DIR" default:"index"`
		Type    string `json:"type" envconfig:"INDEX_TYPE" default:"yearly"`
	} `json:"index"`
}

type WorkerConfig struct {
	// Debug controls debugging options for the worker
	Debug struct {
		// Imap enables debug logging of the imap chat
		Imap bool `json:"imap" envconfig:"WORKER_DEBUG_IMAP" default:"false"`
	} `json:"debug"`

	// ScanInterval is how frequently the worker will scan inboxes
	ScanInterval string `json:"scan_interval" envconfig:"SCAN_INTERVAL" default:"600s"`
}

type Config struct {
	config.CommonConfig

	Inboxes []InboxConfig `json:"inboxes"`
	Search  SearchConfig  `json:"search"`
	Worker  WorkerConfig  `json:"worker"`
}

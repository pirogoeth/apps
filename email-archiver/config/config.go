package config

import (
	"encoding/json"
	"fmt"

	"github.com/emersion/go-imap/v2"
	"github.com/pirogoeth/apps/pkg/config"
)

type InboxConfig struct {
	Username  string          `json:"username"`
	Password  string          `json:"password"`
	Host      string          `json:"host"`
	Mailboxes []MailboxConfig `json:"mailboxes"`
	Port      int             `json:"port"`
	UseTLS    bool            `json:"use_tls"`

	// Fetcher settings

	// FetchBatchSize is the size of the block of messages that
	// is pulled from the server at a time
	FetchBatchSize int `json:"fetch_batch_size" default:"20"`
}

func (i *InboxConfig) InboxAddr() string {
	return fmt.Sprintf("%s:%d", i.Host, i.Port)
}

type MailboxConfig struct {
	Name string `json:"name"`
	// Flags are flags to include on the search
	Flags []imap.Flag `json:"flags"`
	// IgnoreFlags are flags to ignore on the search
	IgnoreFlags []imap.Flag `json:"ignore_flags"`
	// IncludeChildren will include all children of the mailbox in the search
	IncludeChildren bool `json:"include_children"`
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
		BaseDir           string `json:"base_dir" envconfig:"INDEX_BASE_DIR" default:"index"`
		DatePartitionType string `json:"date_partition_type" envconfig:"INDEX_DATE_PARTITION_TYPE" default:"yearly"`
	} `json:"index"`
}

type WorkerConfig struct {
	// ScanInterval is how frequently the worker will scan inboxes
	ScanInterval string `json:"scan_interval" envconfig:"SCAN_INTERVAL" default:"600s"`
	// Debug controls debugging options for the worker
	Debug struct {
		// Imap enables debug logging of the imap chat
		Imap bool `json:"imap" envconfig:"WORKER_DEBUG_IMAP" default:"false"`
	} `json:"debug"`
	// Indexer controls settings for the searchingester worker
	Indexer struct {
		QueueSize int `json:"queue_size" envconfig:"WORKER_INDEXER_QUEUE_SIZE" default:"40"`
	} `json:"indexer"`
}

type ArchivalConfig struct {
	Encryption ArchivalEncryptionConfig `json:"encryption"`
	Storage    ArchivalStorageConfig    `json:"storage"`
}

type ArchivalStorageConfig struct {
	Type   string          `json:"type" envconfig:"ARCHIVAL_STORAGE_TYPE" default:"local"`
	Config json.RawMessage `json:"config" envconfig:"ARCHIVAL_STORAGE_CONFIG" default:"{}"`
}

// ArchivalStorageConfigLocal is the config used when `archival.storage.type` == "local"
type ArchivalStorageConfigLocal struct {
	Path string `json:"path" envconfig:"ARCHIVAL_STORAGE_LOCAL_PATH" default:"/var/lib/email-archiver"`
}

type ArchivalEncryptionConfig struct {
	// Key is the secret value that will be used to encrypt the archives
	Key string `json:"key" envconfig:"ARCHIVAL_ENCRYPTION_KEY"`

	// KdfAlgorithm is the hashing algorithm to use to derive a key suitable for the encryption algorithm
	// currently supported: "argon2"|"scrypt"
	KdfAlgorithm string `json:"kdf_algorithm" envconfig:"ARCHIVAL_ENCRYPTION_KDF_ALGORITHM" default:"argon2"`

	// EncryptionAlgorithm is the algorithm to use to encrypt the archives
	// currently supported: "chacha20poly1305"
	EncryptionAlgorithm string `json:"algorithm" envconfig:"ARCHIVAL_ENCRYPTION_ALGORITHM" default:"chacha20poly1305"`

	// SaltPath is the path where a randomly generated salt will be stored
	SaltPath string `json:"salt_path" envconfig:"ARCHIVAL_ENCRYPTION_SALT_PATH" default:"/var/lib/email-archiver/salt.bin"`
}

type Config struct {
	config.CommonConfig

	Archival ArchivalConfig `json:"archival"`
	Inboxes  []InboxConfig  `json:"inboxes"`
	Search   SearchConfig   `json:"search"`
	Worker   WorkerConfig   `json:"worker"`
}

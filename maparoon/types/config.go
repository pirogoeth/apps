package types

import (
	"time"

	"github.com/pirogoeth/apps/pkg/config"
)

type Config struct {
	config.CommonConfig

	Database struct {
		Path string `json:"path" envconfig:"DATABASE_PATH" default:":memory:"`
	}

	Search struct {
		IndexDir string `json:"index_dir" envconfig:"INDEX_DIR" default:"index"`
	}

	Worker struct {
		BaseURL                    string        `json:"base_url" envconfig:"BASE_URL" default:"http://localhost:8000"`
		ConcurrentIndexLimit       int           `json:"concurrent_index_limit" envconfig:"CONCURRENT_INDEX_LIMIT" default:"4"`
		ConcurrentNetworkScanLimit int           `json:"concurrent_scan_limit" envconfig:"CONCURRENT_SCAN_LIMIT" default:"2"`
		ReverseDNSResolvers        []string      `json:"reverse_dns_resolvers" envconfig:"REVERSE_DNS_RESOLVERS" default:""`
		ScanInterval               time.Duration `json:"scan_interval" envconfig:"SCAN_INTERVAL" default:"30m"`
		Token                      string        `json:"token" envconfig:"WORKER_TOKEN"`
	} `json:"worker"`
}

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

	Worker struct {
		BaseURL             string        `json:"base_url" envconfig:"BASE_URL" default:"http://localhost:8000"`
		ConcurrentScanLimit int           `json:"concurrent_scan_limit" envconfig:"CONCURRENT_SCAN_LIMIT" default:"2"`
		ScanInterval        time.Duration `json:"scan_interval" envconfig:"SCAN_INTERVAL" default:"30m"`
		Token               string        `json:"token" envconfig:"WORKER_TOKEN"`
	} `json:"worker"`
}

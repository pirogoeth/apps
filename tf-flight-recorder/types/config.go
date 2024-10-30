package types

import (
	"github.com/pirogoeth/apps/pkg/config"
)

type Config struct {
	config.CommonConfig

	Database struct {
		Path string `json:"path" envconfig:"DATABASE_PATH" default:":memory:"`
	}
}

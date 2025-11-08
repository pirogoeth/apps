package types

import (
	"github.com/pirogoeth/apps/pkg/config"

	"github.com/pirogoeth/apps/moneyman/database"
)

type Config struct {
	config.CommonConfig

	// Database is the configuration for the database connection
	Database *database.Config `json:"database"`
}

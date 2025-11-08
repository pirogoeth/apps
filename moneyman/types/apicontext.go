package types

import (
	"github.com/pirogoeth/apps/moneyman/database"
)

type ApiContext struct {
	// Config is the application configuration
	Config *Config

	// Querier is the database interface
	Querier *database.Queries
}

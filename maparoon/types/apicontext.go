package types

import (
	"github.com/pirogoeth/apps/maparoon/database"
	"github.com/pirogoeth/apps/maparoon/search"
)

type ApiContext struct {
	// Config is the application configuration
	Config *Config

	// Querier is the database interface
	Querier *database.Queries

	// Searcher is the search index interface
	Searcher *search.BleveSearcher
}

package types

import (
	"github.com/pirogoeth/apps/email-archiver/config"
	"github.com/pirogoeth/apps/email-archiver/search"
)

type ApiContext struct {
	// Config is the application configuration
	Config *config.Config

	// Searcher is the search index interface
	Searcher *search.Searcher
}

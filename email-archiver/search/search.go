package search

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/pirogoeth/apps/email-archiver/config"
	appsErrors "github.com/pirogoeth/apps/pkg/errors"
	"github.com/pirogoeth/apps/pkg/search"
)

var _ io.Closer = (*Searcher)(nil)

type Searcher struct {
	searchers     map[time.Time]*search.BleveSearcher
	searchCatalog map[time.Time]search.SearcherOpts
}

func (s *Searcher) Close() error {
	errs := new(appsErrors.MultiError)
	for timeRange, searcher := range s.searchers {
		if err := searcher.Close(); err != nil {
			errs.Add(fmt.Errorf("could not close searcher for time range %v: %w", timeRange, err))
		}
	}

	return fmt.Errorf("not implemented")
}

func New(cfg *config.SearchConfig) (*Searcher, error) {
	// Open catalog JSON file from index base dir
	catalogFilePath := filepath.Join(cfg.Index.BaseDir, "catalog.json")
	catalogFile, err := os.Open(catalogFilePath)
	catalogInit := false
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if catalogFile, err = os.Create(catalogFilePath); err != nil {
				return nil, fmt.Errorf("could not create search index catalog: %w", err)
			}
			catalogInit = true
		} else {
			return nil, fmt.Errorf("could not open search index catalog: %w", err)
		}
	}

	var searchCatalog map[time.Time]search.SearcherOpts
	if !catalogInit {
		buf := new(bytes.Buffer)
		if _, err := buf.ReadFrom(catalogFile); err != nil {
			return nil, fmt.Errorf("could not read search index catalog: %w", err)
		}

		if json.Unmarshal(buf.Bytes(), &searchCatalog); err != nil {
			return nil, fmt.Errorf("could not unmarshal search index catalog: %w", err)
		}
	}

	searchers := make(map[time.Time]*search.BleveSearcher)
	for timePeriod, searcherOpts := range searchCatalog {
		searchers[timePeriod], err = search.NewSearcher(searcherOpts)
		if err != nil {
			return nil, fmt.Errorf("could not open searcher for catalogued index: %s: %w", timePeriod, err)
		}

	}

	return &Searcher{
		searchers,
		searchCatalog,
	}, nil
}

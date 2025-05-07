package search

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/blevesearch/bleve"
	bleveMapping "github.com/blevesearch/bleve/mapping"
	"github.com/pirogoeth/apps/email-archiver/config"
	"github.com/pirogoeth/apps/email-archiver/util"
	appsErrors "github.com/pirogoeth/apps/pkg/errors"
	"github.com/pirogoeth/apps/pkg/search"
)

var _ io.Closer = (*Searcher)(nil)

type Searcher struct {
	searchCfg *config.SearchConfig

	searchers       map[time.Time]*search.BleveSearcher
	searchCatalog   map[time.Time]search.SearcherOpts
	catalogFilePath string
	datePartitioner util.DatePartitioner
}

func (s *Searcher) Close() error {
	errs := new(appsErrors.MultiError)
	for timePeriod, searcher := range s.searchers {
		if err := searcher.Close(); err != nil {
			errs.Add(fmt.Errorf("could not close searcher for time period %v: %w", timePeriod, err))
		}
	}

	if err := s.SaveCatalog(); err != nil {
		errs.Add(err)
	}

	return errs.ToError()
}

func (s *Searcher) SaveCatalog() error {
	// Ensure the dirpath exists
	if err := os.MkdirAll(filepath.Dir(s.catalogFilePath), 0o750); err != nil {
		return fmt.Errorf("could not create index directory: %w", err)
	}

	// Write the search catalog to file
	catalogFile, err := os.Create(s.catalogFilePath)
	if err != nil {
		return fmt.Errorf("could not create search index catalog file: %w", err)
	} else {
		defer catalogFile.Close()
		encoder := json.NewEncoder(catalogFile)
		if err := encoder.Encode(s.searchCatalog); err != nil {
			return fmt.Errorf("could not write search index catalog: %w", err)
		}
	}

	return nil
}

func New(searchCfg *config.SearchConfig) (*Searcher, error) {
	datePartitioner, err := util.GetDatePartitioner(searchCfg.Index.DatePartitionType)
	if err != nil {
		return nil, fmt.Errorf("could not load date partitioning func: %w", err)
	}

	// Open catalog JSON file from index base dir
	catalogFilePath := filepath.Join(searchCfg.Index.BaseDir, "catalog.json")
	// Ensure the dirpath exists
	if err := os.MkdirAll(filepath.Dir(catalogFilePath), 0o750); err != nil {
		return nil, fmt.Errorf("could not create index directory: %w", err)
	}

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

	searchCatalog := make(map[time.Time]search.SearcherOpts)
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
		searchCfg,
		searchers,
		searchCatalog,
		catalogFilePath,
		datePartitioner,
	}, nil
}

func (s *Searcher) pathForPeriodIndex(when time.Time) string {
	return filepath.Join(s.searchCfg.Index.BaseDir, strconv.FormatInt(when.Unix(), 10))
}

func (s *Searcher) createNewPeriodIndex(when time.Time) (*search.BleveSearcher, error) {
	opts := search.SearcherOpts{
		IndexDir:     s.pathForPeriodIndex(when),
		IndexMapping: createSearchIndexMapping(),
	}

	var err error
	s.searchers[when], err = search.NewSearcher(opts)
	if err != nil {
		return nil, err
	}

	s.searchCatalog[when] = opts
	if err := s.SaveCatalog(); err != nil {
		return nil, fmt.Errorf("could not create new period index for %v: %w", when, err)
	}

	return s.searchers[when], nil
}

func (s *Searcher) ForTime(when time.Time) *search.BleveSearcher {
	datePeriod := s.datePartitioner(when)

	var searcher *search.BleveSearcher
	var ok bool
	if searcher, ok = s.searchers[datePeriod]; !ok {
		var err error
		searcher, err = s.createNewPeriodIndex(datePeriod)
		if err != nil {
			panic(fmt.Errorf("unhandled error in Searcher.ForTime(when=%v): %w", when, err))
		}
	}

	return searcher
}

func createSearchIndexMapping() bleveMapping.IndexMapping {
	docMapping := bleve.NewDocumentMapping()
	indexMapping := bleve.NewIndexMapping()
	indexMapping.AddDocumentMapping("_default", docMapping)

	return indexMapping
}

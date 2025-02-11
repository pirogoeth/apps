package search

import (
	"fmt"
	"io"
	"os"
	"path"
	"sync"

	"github.com/blevesearch/bleve"
	bleveMapping "github.com/blevesearch/bleve/mapping"
	"github.com/sirupsen/logrus"
)

var _ io.Closer = (*BleveSearcher)(nil)

type SearcherOpts struct {
	IndexMapping bleveMapping.IndexMapping
	IndexDir     string
}

type BleveSearcher struct {
	index bleve.Index
	mu    sync.RWMutex
}

func NewSearcher(opts SearcherOpts) (*BleveSearcher, error) {
	// Ensure the IndexDir exists
	if err := os.MkdirAll(opts.IndexDir, 0o750); err != nil {
		logrus.Fatalf("could not create index directory: %s", err.Error())
	}

	// Check if the index exists
	var index bleve.Index
	indexPath := path.Join(opts.IndexDir, "index.bleve")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		// Create the index
		logrus.Debugf("Creating new bleve index at %s", indexPath)
		index, err = createBleveIndex(indexPath, opts.IndexMapping)
		if err != nil {
			return nil, fmt.Errorf("could not create bleve index: %w", err)
		}
	} else {
		// Open the index
		logrus.Debugf("Opening existing index at %s", indexPath)
		index, err = openBleveIndex(indexPath)
		if err != nil {
			return nil, fmt.Errorf("could not open bleve index: %w", err)
		}
	}

	searcher := &BleveSearcher{
		index: index,
	}

	return searcher, nil
}

func createBleveIndex(indexPath string, indexMapping bleveMapping.IndexMapping) (bleve.Index, error) {
	if indexMapping == nil {
		logrus.Debugf("Using default (blank) index mapping")
		indexMapping = bleve.NewIndexMapping()
	}

	index, err := bleve.New(indexPath, indexMapping)
	if err != nil {
		return nil, err
	}

	return index, nil
}

func openBleveIndex(indexPath string) (bleve.Index, error) {
	index, err := bleve.Open(indexPath)
	if err != nil {
		return nil, err
	}

	return index, nil
}

func (s *BleveSearcher) Close() error {
	return s.index.Close()
}

// Handle returns a read-locked handle for doing searches.
func (s *BleveSearcher) SearcherHandle() *searcherHandle {
	s.mu.RLock()
	return &searcherHandle{
		index: &s.index,
		mu:    &s.mu,
	}
}

// IndexerHandle returns a write-locked handle for doing data ingestion.
func (s *BleveSearcher) IndexerHandle() *indexerHandle {
	s.mu.Lock()
	return &indexerHandle{
		index: &s.index,
		mu:    &s.mu,
	}
}

type searcherHandle struct {
	index *bleve.Index
	mu    *sync.RWMutex
}

func (h *searcherHandle) Close() {
	h.index = nil
	h.mu.RUnlock()
}

func (h *searcherHandle) Index() bleve.Index {
	if h.index == nil {
		panic("operation on closed searcherHandle")
	}

	return *h.index
}

func (h *searcherHandle) PrepareSearchRequest(queryString string) *bleve.SearchRequest {
	return bleve.NewSearchRequest(bleve.NewQueryStringQuery(queryString))
}

func (h *searcherHandle) Search(searchReq *bleve.SearchRequest) (*bleve.SearchResult, error) {
	searchResults, err := (*h.index).Search(searchReq)
	if err != nil {
		return nil, err
	}

	return searchResults, nil
}

type indexerHandle struct {
	index *bleve.Index
	mu    *sync.RWMutex
}

func (h *indexerHandle) Close() {
	h.index = nil
	h.mu.Unlock()
}

func (h *indexerHandle) Index() bleve.Index {
	if h.index == nil {
		panic("operation on closed indexerHandle")
	}

	return *h.index
}

func (h *indexerHandle) Ingest(data any) error {
	return fmt.Errorf("not implemented")
}

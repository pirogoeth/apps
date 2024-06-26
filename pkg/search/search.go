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
	IndexDir     string
	IndexMapping bleveMapping.IndexMapping
}

type BleveSearcher struct {
	index bleve.Index
	mu    sync.Mutex
}

func NewSearcher(opts SearcherOpts) (*BleveSearcher, error) {
	// Ensure the IndexDir exists
	if err := os.MkdirAll(opts.IndexDir, 0750); err != nil {
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

func (s *BleveSearcher) Handle() *searcherHandle {
	s.mu.Lock()
	return &searcherHandle{
		index: &s.index,
		mu:    &s.mu,
	}
}

type searcherHandle struct {
	index *bleve.Index
	mu    *sync.Mutex
}

func (h *searcherHandle) Close() {
	h.index = nil
	h.mu.Unlock()
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

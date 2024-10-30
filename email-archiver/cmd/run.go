package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/blevesearch/bleve"
	bleveMapping "github.com/blevesearch/bleve/mapping"
	"github.com/gin-gonic/gin"
	"github.com/pirogoeth/apps/pkg/system"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/pirogoeth/apps/email-archiver/api"
	"github.com/pirogoeth/apps/email-archiver/config"
	"github.com/pirogoeth/apps/email-archiver/search"
	"github.com/pirogoeth/apps/email-archiver/types"
	"github.com/pirogoeth/apps/email-archiver/worker"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run email archiver",
	Run:   runFunc,
}

type App struct {
	cfg *config.Config
}

func runFunc(cmd *cobra.Command, args []string) {
	cfg := appStart(ComponentApi)
	gin.EnableJsonDecoderDisallowUnknownFields()

	app := &App{cfg}

	ctx, cancel := context.WithCancel(context.Background())
	searcher, err := search.New(&cfg.Search)
	if err != nil {
		panic(fmt.Errorf("could not create indexer: %w", err))
	}

	w := worker.New(cfg)

	router := system.DefaultRouter()
	api.MustRegister(router, &types.ApiContext{
		Config:   app.cfg,
		Searcher: searcher,
	})

	go router.Run(app.cfg.HTTP.ListenAddress)
	go w.Run(ctx)

	sw := system.NewSignalWaiter(os.Interrupt)
	sw.OnBeforeCancel(func(context.Context) error {
		if err := searcher.Close(); err != nil {
			panic(fmt.Errorf("could not safely close indexer: %w", err))
		}
		logrus.Infof("closed indexer")

		return nil
	})
	sw.Wait(ctx, cancel)
}

func createSearchIndexMapping() bleveMapping.IndexMapping {
	docMapping := bleve.NewDocumentMapping()
	indexMapping := bleve.NewIndexMapping()
	indexMapping.AddDocumentMapping("_default", docMapping)

	return indexMapping
}

package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/pirogoeth/apps/pkg/system"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/pirogoeth/apps/email-archiver/api"
	"github.com/pirogoeth/apps/email-archiver/config"
	"github.com/pirogoeth/apps/email-archiver/search"
	"github.com/pirogoeth/apps/email-archiver/service"
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

	// TODO: create search ingester first, pass channel reference to email scanner worker
	// ingestWorker := worker.NewSearchIngestWorker(cfg, searcher)
	// scannerWorker := worker.NewEmailScannerWorker(cfg, ingestWorker.GetSender())

	// Create message service
	registry := service.InitServices(app.cfg)

	router := system.DefaultRouter()
	api.MustRegister(router, &types.ApiContext{
		Config:   app.cfg,
		Searcher: searcher,
	})

	go router.Run(app.cfg.HTTP.ListenAddress)
	// go scannerWorker.Run(ctx)
	go worker.RunWorkerPipeline(ctx, &worker.Deps{
		Config:          app.cfg,
		Searcher:        searcher,
		ServiceRegistry: registry,
	})

	sw := system.NewSignalWaiter(os.Interrupt)
	sw.OnBeforeCancel(func(context.Context) error {
		if err := searcher.Close(); err != nil {
			panic(fmt.Errorf("could not safely close indexer: %w", err))
		}
		logrus.Infof("closed indexer")

		// Close all services
		if err := registry.Close(); err != nil {
			panic(fmt.Errorf("could not close services: %w", err))
		}
		logrus.Infof("closed all registered services")

		return nil
	})
	sw.Wait(ctx, cancel)
}

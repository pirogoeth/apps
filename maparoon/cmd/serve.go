package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/pirogoeth/apps/maparoon/api"
	"github.com/pirogoeth/apps/maparoon/database"
	"github.com/pirogoeth/apps/maparoon/search"
	"github.com/pirogoeth/apps/maparoon/types"
	"github.com/pirogoeth/apps/pkg/system"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Serve the maparoon API",
	Run:   serveFunc,
}

type App struct {
	cfg *types.Config
}

func serveFunc(cmd *cobra.Command, args []string) {
	cfg := appStart(ComponentApi)
	gin.EnableJsonDecoderDisallowUnknownFields()

	app := &App{cfg}

	ctx, cancel := context.WithCancel(context.Background())
	dbWrapper, err := database.Open(ctx, cfg.Database.Path)
	if err != nil {
		panic(fmt.Errorf("could not start (database): %w", err))
	}

	searcher, err := search.NewBleveSearcher(cfg.Search.IndexDir)
	if err != nil {
		panic(fmt.Errorf("could not start (indexer): %w", err))
	}

	router := system.DefaultRouter()
	api.MustRegister(router, &types.ApiContext{
		Config:   app.cfg,
		Querier:  dbWrapper.Querier(),
		Searcher: searcher,
	})

	go router.Run(app.cfg.HTTP.ListenAddress)

	sw := system.NewSignalWaiter(os.Interrupt)
	sw.OnBeforeCancel(func(context.Context) error {
		if err := searcher.Close(); err != nil {
			panic(fmt.Errorf("could not safely close indexer: %w", err))
		}
		logrus.Infof("closed indexer")

		if err := dbWrapper.Close(); err != nil {
			panic(fmt.Errorf("could not safely close database: %w", err))
		}
		logrus.Infof("closed database")

		return nil
	})
	sw.Wait(ctx, cancel)
}

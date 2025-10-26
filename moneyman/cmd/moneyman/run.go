package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/pirogoeth/apps/pkg/system"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/pirogoeth/apps/moneyman/api"
	"github.com/pirogoeth/apps/moneyman/database"
	"github.com/pirogoeth/apps/moneyman/types"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the moneyman API and clients",
	Run:   runFunc,
}

type App struct {
	cfg *types.Config
}

func runFunc(cmd *cobra.Command, args []string) {
	cfg := appStart(ComponentApi)
	gin.EnableJsonDecoderDisallowUnknownFields()
	app := &App{cfg}

	ctx, cancel := context.WithCancel(context.Background())
	dbWrapper, err := database.Open(ctx, cfg.Database)
	if err != nil {
		panic(fmt.Errorf("could not start (database): %w", err))
	}

	apiContext := &types.ApiContext{
		Config:  app.cfg,
		Querier: dbWrapper.Querier(),
	}

	router, err := system.DefaultRouterWithTracing(ctx, cfg.Tracing)
	if err != nil {
		panic(fmt.Errorf("could not start (tracing router): %w", err))
	}

	if err := api.MustRegister(router, apiContext); err != nil {
		panic(fmt.Errorf("could not start (api): %w", err))
	}

	go router.Run(app.cfg.HTTP.ListenAddress)

	sw := system.NewSignalWaiter(os.Interrupt)
	sw.OnBeforeCancel(func(context.Context) error {
		if err := dbWrapper.Close(); err != nil {
			panic(fmt.Errorf("could not safely close database: %w", err))
		}
		logrus.Infof("closed database")

		return nil
	})
	sw.Wait(ctx, cancel)
}

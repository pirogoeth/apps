package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/mark3labs/mcp-go/server"
	"github.com/pirogoeth/apps/pkg/system"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/pirogoeth/apps/orba/api"
	"github.com/pirogoeth/apps/orba/database"
	"github.com/pirogoeth/apps/orba/mcptools"
	"github.com/pirogoeth/apps/orba/seeder"
	"github.com/pirogoeth/apps/orba/types"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the orba API and clients",
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
	dbWrapper, err := database.Open(ctx, cfg.Database.Path)
	if err != nil {
		panic(fmt.Errorf("could not start (database): %w", err))
	}

	if err := seeder.SeedDatabase(ctx, cfg.Seeds, dbWrapper); err != nil {
		panic(fmt.Errorf("could not start (seeder): %w", err))
	}

	mcpServer := server.NewMCPServer(
		AppName,
		"0.0.1",
		server.WithToolCapabilities(true),
		server.WithRecovery(),
		server.WithLogging(),
	)

	apiContext := &types.ApiContext{
		Config:    app.cfg,
		Querier:   dbWrapper.Querier(),
		MCPServer: mcpServer,
	}

	if err := mcptools.MustRegister(apiContext); err != nil {
		panic(fmt.Errorf("could not start (mcptools): %w", err))
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

package cmd

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/pirogoeth/apps/maparoon/api"
	"github.com/pirogoeth/apps/maparoon/database"
	"github.com/pirogoeth/apps/maparoon/types"
	"github.com/pirogoeth/apps/pkg/system"
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
	cfg := appStart()
	gin.EnableJsonDecoderDisallowUnknownFields()

	app := &App{cfg}

	ctx, cancel := context.WithCancel(context.Background())
	querier, err := database.Open(ctx, cfg.Database.Path)
	if err != nil {
		panic(fmt.Errorf("could not start (database): %w", err))
	}

	router := system.DefaultRouter()
	api.MustRegister(router, &types.ApiContext{
		Config:  app.cfg,
		Querier: querier,
	})

	router.Run(app.cfg.HTTP.ListenAddress)

	cancel()
}

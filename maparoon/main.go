package main

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/pirogoeth/apps/maparoon/api"
	"github.com/pirogoeth/apps/maparoon/database"
	"github.com/pirogoeth/apps/maparoon/types"
	"github.com/pirogoeth/apps/pkg/config"
	"github.com/pirogoeth/apps/pkg/logging"
	"github.com/pirogoeth/apps/pkg/system"
)

type App struct {
	cfg *types.Config
}

func main() {
	logging.Setup()

	cfg, err := config.Load[types.Config]()
	if err != nil {
		panic(fmt.Errorf("could not start (config): %w", err))
	}

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

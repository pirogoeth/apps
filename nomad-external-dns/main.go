package main

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	nomadApi "github.com/hashicorp/nomad/api"
	"github.com/sirupsen/logrus"

	"github.com/pirogoeth/apps/nomad-deployer/types"
	"github.com/pirogoeth/apps/pkg/config"
	"github.com/pirogoeth/apps/pkg/logging"
	"github.com/pirogoeth/apps/pkg/middlewares"
	"github.com/pirogoeth/apps/pkg/system"
)

type App struct {
	cfg *types.Config
}

func main() {
	logging.Setup(logging.WithAppName("nomad-external-dns"))

	cfg, err := config.Load[types.Config]()
	if err != nil {
		panic(fmt.Errorf("could not start (config): %w", err))
	}

	app := &App{cfg}

	ctx, cancel := context.WithCancel(context.Background())

	// Create Nomad client
	nomadOpts := nomadApi.DefaultConfig()
	nomadClient, err := nomadApi.NewClient(nomadOpts)
	if err != nil {
		panic(fmt.Errorf("could not create nomad client: %w", err))
	}
	if err := system.NomadClientHealthCheck(ctx, nomadClient); err != nil {
		panic(fmt.Errorf("nomad client not healthy: %w", err))
	}
	system.RegisterNomadClientReadiness(nomadClient)
	defer nomadClient.Close()

	router := gin.New()
	router.Use(gin.LoggerWithConfig(gin.LoggerConfig{
		Output:    logrus.StandardLogger().Writer(),
		Formatter: logging.GinJsonLogFormatter,
	}))
	router.Use(gin.Recovery())
	router.Use(middlewares.PrettifyResponseJSON)
	system.RegisterSystemRoutesTo(router.Group("/sys"))
	router.Run(app.cfg.HTTP.ListenAddress)

	cancel()
}

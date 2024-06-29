package main

import (
	"context"
	"fmt"

	"github.com/google/go-github/v58/github"
	"github.com/gregjones/httpcache"
	nomadApi "github.com/hashicorp/nomad/api"

	"github.com/pirogoeth/apps/nomad-deployer/api"
	"github.com/pirogoeth/apps/nomad-deployer/types"
	"github.com/pirogoeth/apps/pkg/config"
	"github.com/pirogoeth/apps/pkg/logging"
	"github.com/pirogoeth/apps/pkg/system"
)

type App struct {
	cfg *types.Config
}

func main() {
	logging.Setup(logging.WithAppName("nomad-deployer"))

	cfg, err := config.Load[types.Config]()
	if err != nil {
		panic(fmt.Errorf("could not start (config): %w", err))
	}

	app := &App{cfg}

	ctx, cancel := context.WithCancel(context.Background())

	// Create Github client
	ghClient := github.NewClient(
		httpcache.NewMemoryCacheTransport().Client(),
	).WithAuthToken(app.cfg.Github.AuthToken)
	if err := system.GithubClientHealthCheck(ctx, ghClient); err != nil {
		panic(fmt.Errorf("github client not healthy: %w", err))
	}
	system.RegisterGithubClientReadiness(ghClient)

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

	router := system.DefaultRouter()
	api.MustRegister(router, &types.ApiContext{
		Config: app.cfg,
		Github: ghClient,
		Nomad:  nomadClient,
	})

	router.Run(app.cfg.HTTP.ListenAddress)

	cancel()
}

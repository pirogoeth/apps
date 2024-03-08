package main

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/google/go-github/v58/github"
	"github.com/gregjones/httpcache"
	nomadApi "github.com/hashicorp/nomad/api"
	"github.com/sirupsen/logrus"

	"github.com/pirogoeth/apps/nomad-deployer/api"
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
	logging.Setup()

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
	if err := githubClientHealthCheck(ctx, ghClient); err != nil {
		panic(fmt.Errorf("github client not healthy: %w", err))
	}
	registerGithubClientReadiness(ghClient)

	// Create Nomad client
	nomadOpts := nomadApi.DefaultConfig()
	nomadClient, err := nomadApi.NewClient(nomadOpts)
	if err != nil {
		panic(fmt.Errorf("could not create nomad client: %w", err))
	}
	if err := nomadClientHealthCheck(ctx, nomadClient); err != nil {
		panic(fmt.Errorf("nomad client not healthy: %w", err))
	}
	registerNomadClientReadiness(nomadClient)
	defer nomadClient.Close()

	router := gin.New()
	router.Use(gin.LoggerWithConfig(gin.LoggerConfig{
		Output:    logrus.StandardLogger().Writer(),
		Formatter: logging.GinJsonLogFormatter,
	}))
	router.Use(gin.Recovery())
	router.Use(middlewares.PrettifyResponseJSON)

	api.MustRegister(router, &types.ApiContext{
		Config: app.cfg,
		Github: ghClient,
		Nomad:  nomadClient,
	})

	router.Run(app.cfg.HTTP.ListenAddress)

	cancel()
}

func nomadClientHealthCheck(_ context.Context, nomadClient *nomadApi.Client) error {
	leader, err := nomadClient.Status().Leader()
	if err != nil {
		return fmt.Errorf("could not check nomad leader: %w", err)
	}

	logrus.Infof("nomad-client.ready: leader=%s", leader)
	return nil
}

func registerNomadClientReadiness(nomadClient *nomadApi.Client) {
	check := func(ctx context.Context) (string, error) {
		return "ready, leader found!", nomadClientHealthCheck(ctx, nomadClient)
	}

	system.AddReadinessCheck("nomad-client", check)
}

func githubClientHealthCheck(ctx context.Context, ghClient *github.Client) error {
	currentUser, _, err := ghClient.Users.Get(ctx, "")
	if err != nil {
		return fmt.Errorf("could not check github client user: %w", err)
	}

	logrus.Infof("github-client.ready: currentUser='%s'", *currentUser.Login)
	return nil
}

func registerGithubClientReadiness(ghClient *github.Client) {
	check := func(ctx context.Context) (string, error) {
		return "ready, logged in!", githubClientHealthCheck(ctx, ghClient)
	}

	system.AddReadinessCheck("github-client", check)
}

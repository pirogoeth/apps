package cmd

import (
	"context"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/pirogoeth/apps/pkg/system"
	"github.com/pirogoeth/apps/functional/api"
	"github.com/pirogoeth/apps/functional/compute"
	"github.com/pirogoeth/apps/functional/database"
	"github.com/pirogoeth/apps/functional/types"
)

const ComponentApi = "api"

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the FaaS API server",
	Run:   serveFunc,
}

func serveFunc(cmd *cobra.Command, args []string) {
	cfg := appStart(ComponentApi)

	// Setup database
	ctx, cancel := context.WithCancel(context.Background())
	db, err := database.Open(ctx, cfg.Database.Path)
	if err != nil {
		logrus.WithError(err).Fatal("failed to open database")
	}

	// Run migrations
	if err := db.RunMigrations(database.MigrationsFS); err != nil {
		logrus.WithError(err).Fatal("failed to run database migrations")
	}

	// Setup compute registry
	computeRegistry := compute.NewRegistry()
	
	// Register compute providers based on config
	switch cfg.Compute.Provider {
	case "docker":
		if cfg.Compute.Docker == nil {
			logrus.Fatal("docker provider selected but docker config is missing")
		}
		// Convert to local Docker config type
		dockerConfig := &compute.DockerConfig{
			Socket:   cfg.Compute.Docker.Socket,
			Network:  cfg.Compute.Docker.Network,
			Registry: cfg.Compute.Docker.Registry,
		}
		dockerProvider := compute.NewDockerProvider(dockerConfig)
		computeRegistry.Register(dockerProvider)
	case "firecracker":
		if cfg.Compute.Firecracker == nil {
			logrus.Fatal("firecracker provider selected but firecracker config is missing")
		}
		// Convert to local Firecracker config type
		firecrackerConfig := &compute.FirecrackerConfig{
			KernelImagePath: cfg.Compute.Firecracker.KernelImagePath,
			RootfsImagePath: cfg.Compute.Firecracker.RootfsImagePath,
			WorkDir:         cfg.Compute.Firecracker.WorkDir,
			NetworkDevice:   cfg.Compute.Firecracker.NetworkDevice,
		}
		firecrackerProvider := compute.NewFirecrackerProvider(firecrackerConfig)
		computeRegistry.Register(firecrackerProvider)
	default:
		logrus.WithField("provider", cfg.Compute.Provider).Fatal("unsupported compute provider")
	}

	// Create API context
	apiContext := &types.ApiContext{
		Config:  cfg,
		Querier: db.Queries,
		Compute: computeRegistry,
	}

	// Setup router
	router, err := system.DefaultRouterWithTracing(ctx, cfg.Tracing)
	if err != nil {
		logrus.WithError(err).Fatal("failed to setup router with tracing")
	}

	// Register API routes
	if err := api.MustRegister(router, apiContext); err != nil {
		logrus.WithError(err).Fatal("failed to register API routes")
	}

	// Start server in goroutine
	logrus.WithField("listen_address", cfg.HTTP.ListenAddress).Info("starting FaaS API server")
	go router.Run(cfg.HTTP.ListenAddress)

	// Setup graceful shutdown
	sw := system.NewSignalWaiter(os.Interrupt)
	sw.OnBeforeCancel(func(context.Context) error {
		if err := db.Close(); err != nil {
			logrus.WithError(err).Error("could not safely close database")
			return err
		}
		logrus.Info("closed database")
		return nil
	})
	sw.Wait(ctx, cancel)
}
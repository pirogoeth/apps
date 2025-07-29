package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/pirogoeth/apps/functional/database"
	"github.com/pirogoeth/apps/functional/proxy"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const ComponentProxy = "proxy"

var proxyCmd = &cobra.Command{
	Use:   "proxy",
	Short: "Start the function proxy service",
	Long: `Start the function proxy service that handles function invocations,
container pooling, and Traefik integration.`,
	RunE: runProxy,
}

func init() {
	rootCmd.AddCommand(proxyCmd)
}

func runProxy(cmd *cobra.Command, args []string) error {
	cfg := appStart(ComponentProxy)

	// Initialize database
	ctx := context.Background()
	db, err := database.Open(ctx, cfg.Database.Path)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to open database")
	}
	defer db.Close()

	// Run database migrations
	if err := db.RunMigrations(database.MigrationsFS); err != nil {
		logrus.WithError(err).Fatal("Failed to run database migrations")
	}

	// Set proxy defaults if not configured
	if cfg.Proxy.ListenAddress == "" {
		cfg.Proxy.ListenAddress = ":8080"
	}
	if cfg.Proxy.TraefikAPIURL == "" {
		cfg.Proxy.TraefikAPIURL = "http://traefik:8080/api"
	}
	if cfg.Proxy.MaxContainersPerFunction == 0 {
		cfg.Proxy.MaxContainersPerFunction = 5
	}
	if cfg.Proxy.ContainerIdleTimeout == 0 {
		cfg.Proxy.ContainerIdleTimeout = 300000000000 // 5 minutes in nanoseconds
	}

	// Create proxy service
	proxyService := proxy.NewProxyService(cfg, db)

	// Set up graceful shutdown
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		logrus.WithField("signal", sig).Info("Received shutdown signal")
		cancel()
	}()

	// Start proxy service
	logrus.WithFields(logrus.Fields{
		"listen_address":  cfg.Proxy.ListenAddress,
		"traefik_api_url": cfg.Proxy.TraefikAPIURL,
		"max_containers":  cfg.Proxy.MaxContainersPerFunction,
		"idle_timeout":    cfg.Proxy.ContainerIdleTimeout,
	}).Info("Starting function proxy service")

	if err := proxyService.Start(ctx); err != nil {
		if ctx.Err() != context.Canceled {
			logrus.WithError(err).Fatal("Proxy service failed")
		}
	}

	logrus.Info("Proxy service stopped")
	return nil
}


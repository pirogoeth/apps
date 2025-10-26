package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/pirogoeth/apps/pkg/system"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/pirogoeth/apps/docker-hermes/server"
	"github.com/pirogoeth/apps/docker-hermes/types"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Run the docker-hermes server",
	Long: `Run the docker-hermes server.

The server provides:
- Prometheus HTTP Service Discovery endpoint
- REST API for querying container information
- Central storage and coordination via Redis`,
	Run: runServer,
}

func runServer(cmd *cobra.Command, args []string) {
	cfg := appStart(ComponentServer)

	// Remove this or find a way to apply "dynamic" defaults
	types.ApplyDefaults(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create server
	srv, err := server.NewServer(ctx, cfg)
	if err != nil {
		panic(fmt.Errorf("failed to create server: %w", err))
	}
	defer srv.Close()

	// Run!
	logrus.Infof("Starting docker-hermes server on %s", cfg.HTTP.ListenAddress)
	go srv.Run(ctx)

	// Handle shutdown signals
	sw := system.NewSignalWaiter(os.Interrupt)
	sw.OnBeforeCancel(func(context.Context) error {
		return srv.Close()
	})
	sw.Wait(ctx, cancel)
}

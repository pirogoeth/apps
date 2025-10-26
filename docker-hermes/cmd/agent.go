package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/pirogoeth/apps/docker-hermes/agent"
	"github.com/pirogoeth/apps/docker-hermes/types"
	"github.com/pirogoeth/apps/pkg/system"
)

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Run the docker-hermes agent",
	Long: `Run the docker-hermes agent on a Docker host.

The agent monitors Docker containers and reports their labels and metadata
to the central Redis server. It performs periodic heartbeats and responds
to Docker events in real-time.`,
	Run: runAgent,
}

func runAgent(cmd *cobra.Command, args []string) {
	cfg := appStart(ComponentAgent)

	// Apply defaults
	types.ApplyDefaults(cfg)

	logrus.Infof("Starting docker-hermes agent on host %s", cfg.Agent.Hostname)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create agent
	agent, err := agent.NewAgent(cfg)
	if err != nil {
		panic(fmt.Errorf("failed to create agent: %w", err))
	}
	defer agent.Close()

	router, err := system.DefaultRouterWithTracing(ctx, cfg.Tracing)
	if err != nil {
		panic(fmt.Errorf("failed to create router: %w", err))
	}

	router.GET("/metrics", system.HttpHandlerToGinHandler(agent.GetMetricsHandler()))

	// Run agent
	go agent.Run(ctx)
	go router.Run(cfg.HTTP.ListenAddress)

	// Handle shutdown signals
	sw := system.NewSignalWaiter(os.Interrupt)
	sw.OnBeforeCancel(func(context.Context) error {
		return agent.Close()
	})
	sw.Wait(ctx, cancel)

	logrus.Info("Agent stopped")
}

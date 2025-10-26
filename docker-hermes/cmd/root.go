package cmd

import (
	"fmt"

	"github.com/pirogoeth/apps/docker-hermes/types"
	"github.com/pirogoeth/apps/pkg/config"
	"github.com/pirogoeth/apps/pkg/logging"
	"github.com/pirogoeth/apps/pkg/tracing"
	"github.com/spf13/cobra"
)

const (
	AppName         = "docker-hermes"
	ComponentAgent  = "agent"
	ComponentServer = "server"
	ComponentQuery  = "query"
)

var rootCmd = &cobra.Command{
	Use:   "docker-hermes",
	Short: "Docker service discovery system with Prometheus HTTP SD support",
	Long: `docker-hermes is a service registration system for Docker containers.

It consists of:
- Agent: Runs on each Docker host, monitors containers, reports to central server
- Server: Provides Prometheus HTTP SD endpoint and query APIs
- Query: CLI tool for interacting with the server API

The agent monitors Docker containers and reports their labels and metadata
to a central Redis server. The server provides a Prometheus HTTP Service
Discovery endpoint and REST APIs for querying container information.`,
}

func init() {
	rootCmd.AddCommand(agentCmd)
	rootCmd.AddCommand(serverCmd)
	rootCmd.AddCommand(queryCmd)
}

func appStart(component string) *types.Config {
	logging.Setup(
		logging.WithAppName(AppName),
		logging.WithComponentName(component),
	)

	cfg, err := config.Load[types.Config]()
	if err != nil {
		panic(fmt.Errorf("could not start (config): %w", err))
	}

	tracing.Setup(
		tracing.WithAppName(AppName),
		tracing.WithComponentName(component),
		tracing.WithConfig(cfg.CommonConfig.Tracing),
	)

	return cfg
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
	}
}

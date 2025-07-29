package cmd

import (
	"fmt"

	"github.com/pirogoeth/apps/orba/types"
	"github.com/pirogoeth/apps/pkg/config"
	"github.com/pirogoeth/apps/pkg/logging"
	"github.com/pirogoeth/apps/pkg/tracing"
	"github.com/spf13/cobra"
)

const (
	AppName         = "orba"
	ComponentApi    = "api"
	ComponentWorker = "worker"
)

var rootCmd = &cobra.Command{
	Use:   "orba",
	Short: "Operator for Responsibilities, Behaviors, and Activities",
}

func init() {
	rootCmd.AddCommand(runCmd)
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
	rootCmd.Execute()
}

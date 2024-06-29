package cmd

import (
	"fmt"

	"github.com/pirogoeth/apps/maparoon/types"
	"github.com/pirogoeth/apps/pkg/config"
	"github.com/pirogoeth/apps/pkg/logging"
	"github.com/spf13/cobra"
)

const (
	ComponentApi    = "api"
	ComponentWorker = "worker"
)

var rootCmd = &cobra.Command{
	Use:   "maparoon",
	Short: "Network mapping tool",
}

func init() {
	rootCmd.AddCommand(clientCmd, serveCmd, workerCmd)
}

func appStart(component string) *types.Config {
	logging.Setup(
		logging.WithAppName("maparoon"),
		logging.WithComponentName(component),
	)

	cfg, err := config.Load[types.Config]()
	if err != nil {
		panic(fmt.Errorf("could not start (config): %w", err))
	}

	return cfg
}

func Execute() {
	rootCmd.Execute()
}

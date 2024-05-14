package cmd

import (
	"fmt"

	"github.com/pirogoeth/apps/maparoon/types"
	"github.com/pirogoeth/apps/pkg/config"
	"github.com/pirogoeth/apps/pkg/logging"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "maparoon",
	Short: "Network mapping tool",
}

func init() {
	rootCmd.AddCommand(serveCmd, workerCmd)
}

func appStart() *types.Config {
	logging.Setup()

	cfg, err := config.Load[types.Config]()
	if err != nil {
		panic(fmt.Errorf("could not start (config): %w", err))
	}

	return cfg
}

func Execute() {
	rootCmd.Execute()
}

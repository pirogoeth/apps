package cmd

import (
	"fmt"

	"github.com/pirogoeth/apps/maparoon/types"
	"github.com/pirogoeth/apps/pkg/config"
	"github.com/pirogoeth/apps/pkg/logging"
	"github.com/spf13/cobra"

	"github.com/pirogoeth/apps/tf-flight-recorder/cmd/tools"
)

const (
	ComponentApi    = "api"
	ComponentWorker = "worker"
)

var rootCmd = &cobra.Command{
	Use:   "tf-flight-recorder",
	Short: "Terraform/Opentofu Flight Recorder",
}

func init() {
	rootCmd.AddCommand(serveCmd, tools.ToolsCmd)
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

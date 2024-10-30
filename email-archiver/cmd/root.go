package cmd

import (
	"fmt"

	"github.com/pirogoeth/apps/email-archiver/config"
	sysConfig "github.com/pirogoeth/apps/pkg/config"
	"github.com/pirogoeth/apps/pkg/logging"
	"github.com/spf13/cobra"
)

const (
	ComponentApi    = "api"
	ComponentWorker = "worker"
)

var rootCmd = &cobra.Command{
	Use:   "email-archiver",
	Short: "Email archival tool",
}

func init() {
	rootCmd.AddCommand(runCmd)
}

func appStart(component string) *config.Config {
	logging.Setup(
		logging.WithAppName("email-archiver"),
		logging.WithComponentName(component),
	)

	cfg, err := sysConfig.Load[config.Config]()
	if err != nil {
		panic(fmt.Errorf("could not start (config): %w", err))
	}

	return cfg
}

func Execute() {
	rootCmd.Execute()
}

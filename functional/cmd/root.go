package cmd

import (
	"os"

	"github.com/pirogoeth/apps/functional/types"
	"github.com/pirogoeth/apps/pkg/config"
	"github.com/pirogoeth/apps/pkg/logging"
	"github.com/pirogoeth/apps/pkg/tracing"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const AppName = "functional"

var rootCmd = &cobra.Command{
	Use:   AppName,
	Short: "Function-as-a-Service provider",
	Long:  "A modular FaaS provider with pluggable compute backends",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(serveCmd)
}

func appStart(component string) *types.Config {
	logging.Setup(logging.WithAppName(AppName), logging.WithComponentName(component))

	cfg, err := config.Load[types.Config]()
	if err != nil {
		logrus.WithError(err).Fatal("failed to load configuration")
	}

	tracing.Setup(
		tracing.WithAppName(AppName),
		tracing.WithComponentName(component),
		tracing.WithConfig(cfg.Tracing),
	)

	return cfg
}


package cmd

import (
	"fmt"

	"github.com/pirogoeth/apps/pkg/config"
	"github.com/pirogoeth/apps/pkg/logging"
	"github.com/pirogoeth/apps/voice-memos/types"
	"github.com/spf13/cobra"
)

const (
	ComponentApi = "api"
)

var rootCmd = &cobra.Command{
	Use:   "voice-memos",
	Short: "Voice memos integration for usememos/memos",
}

func init() {
	rootCmd.AddCommand(serveCmd)
}

func appStart(component string) *types.Config {
	logging.Setup(
		logging.WithAppName("voice-memos"),
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

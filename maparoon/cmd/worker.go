package cmd

import "github.com/spf13/cobra"

var workerCmd = &cobra.Command{
	Use:   "worker",
	Short: "Launch a maparoon worker",
	Run:   workerFunc,
}

func workerFunc(cmd *cobra.Command, args []string) {
	// cfg := appStart()
}

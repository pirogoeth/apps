package tools

import "github.com/spf13/cobra"

var ToolsCmd = &cobra.Command{
	Use: "tools",
	Short: "Various tools for working with TFFR",
}

func init() {
	ToolsCmd.AddCommand(planparseCmd)
}
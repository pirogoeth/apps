package tools

import "github.com/spf13/cobra"

var planparseCmd = &cobra.Command{
	Use: "plan-parse",
	Short: "Parse a plan file's JSON representation",
	Run: planparseFunc,
}

func planparseFunc(cmd *cobra.Command, args []string) {
	return
}
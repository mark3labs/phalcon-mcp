package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	// Version holds the current version of the CLI
	Version = "0.1.2"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Long:  `Display the current version of Phalcon MCP CLI tool.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Phalcon MCP v%s\n", Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

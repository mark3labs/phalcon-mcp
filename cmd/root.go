package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "phalcon-mcp",
	Short: "Phalcon MCP CLI tool",
	Long:  `Phalcon MCP is a command line interface tool.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Println("Welcome to Phalcon MCP!")
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().BoolP("version", "v", false, "Print the version number")
}

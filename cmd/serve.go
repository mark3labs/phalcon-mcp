package cmd

import (
	"fmt"

	"github.com/mark3labs/phalcon-mcp/server"
	"github.com/spf13/cobra"
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the MCP server in stdio mode",
	Long:  `Start the Model Context Protocol (MCP) server in stdio mode.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Starting MCP server in stdio mode...")

		// Create and start the server
		s := server.NewServer(Version)
		if err := s.ServeStdio(); err != nil {
			fmt.Printf("Server error: %v\n", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
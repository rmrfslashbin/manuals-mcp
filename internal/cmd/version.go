package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// versionCmd represents the version command.
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display version information",
	Long:  `Display detailed version information including build time and git commit.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Manuals MCP Server\n\n")
		fmt.Printf("Version:    %s\n", version)
		fmt.Printf("Git Commit: %s\n", gitCommit)
		fmt.Printf("Build Time: %s\n", buildTime)
		fmt.Printf("\nProject:    github.com/rmrfslashbin/manuals-mcp-server\n")
		fmt.Printf("License:    MIT\n")
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

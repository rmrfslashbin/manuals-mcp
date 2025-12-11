// manuals-mcp is an MCP server for hardware and software documentation.
//
// It provides GPIO pinout information, full-text search, and device
// specifications through the Model Context Protocol (MCP).
package main

import (
	"fmt"
	"os"

	"github.com/rmrfslashbin/manuals-mcp/internal/cmd"
)

// Build information (set via ldflags during build)
var (
	version   = "dev"
	gitCommit = "unknown"
	buildTime = "unknown"
)

func main() {
	// Set version info for commands to use
	cmd.SetVersionInfo(version, gitCommit, buildTime)

	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

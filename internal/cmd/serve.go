package cmd

import (
	"fmt"
	"log/slog"

	"github.com/rmrfslashbin/manuals-mcp/internal/client"
	"github.com/rmrfslashbin/manuals-mcp/internal/mcp"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	apiURL string
	apiKey string
)

// serveCmd represents the serve command.
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the MCP server",
	Long: `Start the MCP server and listen for requests via stdio.

The server connects to the Manuals REST API to serve documentation.

Environment Variables:
  MANUALS_API_URL    - URL of the Manuals REST API (required)
  MANUALS_API_KEY    - API key for authentication (optional, enables admin features)
  MANUALS_LOG_LEVEL  - Log level (debug, info, warn, error)
  MANUALS_LOG_FORMAT - Log format (json, text)
  MANUALS_LOG_OUTPUT - Log output (stderr, /path/to/file, /path/to/dir/)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := slog.Default()

		apiURL := viper.GetString("api.url")
		apiKey := viper.GetString("api.key")

		if apiURL == "" {
			return fmt.Errorf("MANUALS_API_URL is required")
		}

		// API key is now optional - allows anonymous read-only access
		anonymousMode := apiKey == ""
		if anonymousMode {
			logger.Info("running in anonymous mode (read-only access)")
		}

		logger.Info("starting MCP server",
			"version", version,
			"commit", gitCommit,
			"api_url", apiURL,
			"anonymous_mode", anonymousMode,
		)

		// Create API client
		apiClient := client.New(apiURL, apiKey)

		// Test connection by getting status
		status, err := apiClient.GetStatus()
		if err != nil {
			logger.Error("failed to connect to API", "error", err)
			return fmt.Errorf("failed to connect to API: %w", err)
		}

		logger.Info("connected to Manuals API",
			"api_version", status.APIVersion,
			"devices", status.Counts.Devices,
			"documents", status.Counts.Documents,
		)

		// Create MCP server
		mcpServer := mcp.NewServer(apiClient, version, gitCommit, buildTime, logger)

		logger.Info("MCP server ready, listening on stdio")

		// Serve (blocks until shutdown)
		return mcpServer.Serve(cmd.Context())
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)

	// Serve-specific flags
	serveCmd.Flags().StringVar(&apiURL, "api-url", "", "URL of the Manuals REST API")
	serveCmd.Flags().StringVar(&apiKey, "api-key", "", "API key for authentication")

	// Bind flags to viper
	viper.BindPFlag("api.url", serveCmd.Flags().Lookup("api-url"))
	viper.BindPFlag("api.key", serveCmd.Flags().Lookup("api-key"))
}

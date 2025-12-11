package cmd

import (
	"fmt"
	"log/slog"

	"github.com/rmrfslashbin/manuals-mcp-server/internal/db"
	"github.com/rmrfslashbin/manuals-mcp-server/internal/mcp"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	dbPath   string
	docsPath string
)

// serveCmd represents the serve command.
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the MCP server",
	Long: `Start the MCP server and listen for requests via stdio.

The server requires a SQLite database with indexed documentation. Use the
'index' command to create the database before running the server.

Environment Variables:
  MANUALS_DB_PATH    - Path to SQLite database (default: ./data/manuals.db)
  MANUALS_DOCS_PATH  - Path to documentation directory
  MANUALS_LOG_LEVEL  - Log level (debug, info, warn, error)
  MANUALS_LOG_FORMAT - Log format (json, text)
  MANUALS_LOG_OUTPUT - Log output (stderr, /path/to/file, /path/to/dir/)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := slog.Default()
		dbPath := viper.GetString("db.path")

		logger.Info("starting MCP server",
			"version", version,
			"commit", gitCommit,
			"db_path", dbPath,
		)

		// Initialize database
		database, err := db.InitDatabase(dbPath)
		if err != nil {
			logger.Error("failed to initialize database", "error", err)
			return fmt.Errorf("failed to initialize database: %w", err)
		}
		defer database.Close()

		// Get database stats
		stats, err := db.GetStats(database)
		if err != nil {
			logger.Warn("failed to get database stats", "error", err)
		} else {
			logger.Info("database loaded",
				"total_devices", stats.TotalDevices,
				"hardware", stats.HardwareCount,
				"software", stats.SoftwareCount,
				"protocol", stats.ProtocolCount,
				"pinouts", stats.TotalPinouts,
			)

			if stats.TotalDevices == 0 {
				logger.Warn("database is empty - run 'manuals-mcp index' to populate it")
			}
		}

		// Create MCP server
		docsPath := viper.GetString("docs.path")
		mcpServer := mcp.NewServer(database, docsPath, version, gitCommit, buildTime, logger)

		if docsPath != "" {
			logger.Info("MCP server ready with reindex capability", "docs_path", docsPath)
		} else {
			logger.Info("MCP server ready, listening on stdio")
		}

		// Serve (blocks until shutdown)
		return mcpServer.Serve(cmd.Context())
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)

	// Serve-specific flags
	serveCmd.Flags().StringVar(&dbPath, "db-path", "./data/manuals.db", "path to SQLite database")
	serveCmd.Flags().StringVar(&docsPath, "docs-path", "", "path to documentation directory")

	// Bind flags to viper
	viper.BindPFlag("db.path", serveCmd.Flags().Lookup("db-path"))
	viper.BindPFlag("docs.path", serveCmd.Flags().Lookup("docs-path"))
}

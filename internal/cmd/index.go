package cmd

import (
	"fmt"
	"log/slog"

	"github.com/rmrfslashbin/manuals-mcp-server/internal/db"
	"github.com/rmrfslashbin/manuals-mcp-server/internal/indexer"
	"github.com/rmrfslashbin/manuals-mcp-server/pkg/models"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// indexCmd represents the index command.
var indexCmd = &cobra.Command{
	Use:   "index",
	Short: "Index documentation into the database",
	Long: `Index documentation files into the SQLite database with FTS5 support.

This command scans the documentation directory for markdown files and
extracts device information, GPIO pinouts, and specifications.

The indexed database is required before running the server.

Example:
  manuals-mcp index --docs-path /path/to/docs --db-path ./data/manuals.db`,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := slog.Default()

		docsPath := viper.GetString("docs.path")
		dbPath := viper.GetString("db.path")

		if docsPath == "" {
			logger.Error("docs-path is required")
			cmd.Help()
			return fmt.Errorf("docs-path is required")
		}

		logger.Info("starting documentation indexing",
			"docs_path", docsPath,
			"db_path", dbPath,
		)

		// Initialize database
		database, err := db.InitDatabase(dbPath)
		if err != nil {
			logger.Error("failed to initialize database", "error", err)
			return fmt.Errorf("failed to initialize database: %w", err)
		}
		defer database.Close()

		// Run indexer
		opts := indexer.IndexOptions{
			DocsPath: docsPath,
			Clear:    true, // Always clear for now
			Verbose:  viper.GetString("log.level") == "debug",
		}

		result, err := indexer.IndexDocumentation(database, opts, logger)
		if err != nil {
			logger.Error("indexing failed", "error", err)
			return fmt.Errorf("indexing failed: %w", err)
		}

		// Print summary
		logger.Info("indexing completed successfully",
			"total_files", result.TotalFiles,
			"indexed", result.SuccessCount,
			"errors", result.ErrorCount,
			"hardware", result.DevicesByType[models.DomainHardware],
			"software", result.DevicesByType[models.DomainSoftware],
			"protocol", result.DevicesByType[models.DomainProtocol],
			"duration", result.Duration.String(),
		)

		if result.ErrorCount > 0 {
			logger.Warn("some files failed to index", "count", result.ErrorCount)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(indexCmd)

	// Index-specific flags
	indexCmd.Flags().StringVar(&dbPath, "db-path", "./data/manuals.db", "path to SQLite database")
	indexCmd.Flags().StringVar(&docsPath, "docs-path", "", "path to documentation directory (required)")
	indexCmd.MarkFlagRequired("docs-path")

	// Bind flags to viper
	viper.BindPFlag("db.path", indexCmd.Flags().Lookup("db-path"))
	viper.BindPFlag("docs.path", indexCmd.Flags().Lookup("docs-path"))
}

package indexer

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rmrfslashbin/manuals-mcp-server/internal/db"
	"github.com/rmrfslashbin/manuals-mcp-server/pkg/models"
)

// IndexOptions configures the indexing process.
type IndexOptions struct {
	DocsPath string
	Clear    bool
	Verbose  bool
}

// IndexResult holds statistics about the indexing process.
type IndexResult struct {
	TotalFiles    int
	SuccessCount  int
	ErrorCount    int
	Duration      time.Duration
	DevicesByType map[models.Domain]int
}

// IndexDocumentation indexes all markdown files in the documentation directory.
func IndexDocumentation(database *sql.DB, opts IndexOptions, logger *slog.Logger) (*IndexResult, error) {
	startTime := time.Now()

	result := &IndexResult{
		DevicesByType: make(map[models.Domain]int),
	}

	// Clear database if requested
	if opts.Clear {
		logger.Info("clearing existing database")
		if err := db.ClearDatabase(database); err != nil {
			return nil, fmt.Errorf("failed to clear database: %w", err)
		}
	}

	// Walk directory and find all markdown files
	logger.Info("scanning documentation directory", "path", opts.DocsPath)

	err := filepath.Walk(opts.DocsPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			logger.Warn("failed to access path", "path", path, "error", err)
			return nil // Continue walking
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Only process markdown files
		if !strings.HasSuffix(strings.ToLower(path), ".md") {
			return nil
		}

		result.TotalFiles++

		// Parse markdown file
		if opts.Verbose {
			logger.Debug("parsing file", "path", path)
		}

		doc, err := ParseMarkdownFile(path)
		if err != nil {
			logger.Warn("failed to parse file", "path", path, "error", err)
			result.ErrorCount++
			return nil // Continue walking
		}

		// Create device
		device := &models.Device{
			ID:        doc.ID,
			Domain:    doc.Domain,
			Type:      doc.Type,
			Name:      getDeviceName(doc.Metadata),
			Path:      path,
			Metadata:  doc.Metadata,
			IndexedAt: time.Now(),
			Content:   doc.Content,
		}

		// Insert device
		if err := db.InsertDevice(database, device); err != nil {
			logger.Warn("failed to insert device",
				"path", path,
				"id", device.ID,
				"error", err,
			)
			result.ErrorCount++
			return nil
		}

		// Insert pinouts (if any)
		if len(doc.Pinouts) > 0 {
			if err := db.InsertPinouts(database, device.ID, doc.Pinouts); err != nil {
				logger.Warn("failed to insert pinouts",
					"path", path,
					"id", device.ID,
					"error", err,
				)
				// Don't count as error - device was still inserted
			}
		}

		if opts.Verbose {
			logger.Info("indexed device",
				"id", device.ID,
				"name", device.Name,
				"domain", device.Domain,
				"type", device.Type,
				"pinouts", len(doc.Pinouts),
			)
		}

		result.SuccessCount++
		result.DevicesByType[doc.Domain]++

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	result.Duration = time.Since(startTime)

	// Log summary
	logger.Info("indexing complete",
		"total_files", result.TotalFiles,
		"success", result.SuccessCount,
		"errors", result.ErrorCount,
		"hardware", result.DevicesByType[models.DomainHardware],
		"software", result.DevicesByType[models.DomainSoftware],
		"protocol", result.DevicesByType[models.DomainProtocol],
		"duration_ms", result.Duration.Milliseconds(),
	)

	return result, nil
}

// getDeviceName extracts the device name from metadata.
func getDeviceName(metadata map[string]interface{}) string {
	// Try model first
	if model, ok := metadata["model"].(string); ok && model != "" {
		return model
	}

	// Try manufacturer + model
	if manufacturer, ok := metadata["manufacturer"].(string); ok {
		if model, ok := metadata["model"].(string); ok {
			return fmt.Sprintf("%s %s", manufacturer, model)
		}
	}

	// Fallback
	return "Unknown"
}

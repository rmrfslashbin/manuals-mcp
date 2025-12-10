package cmd

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// setupLogger creates and configures a structured logger based on viper settings.
func setupLogger() error {
	level := viper.GetString("log.level")
	format := viper.GetString("log.format")
	output := viper.GetString("log.output")

	// Parse log level
	var logLevel slog.Level
	switch strings.ToLower(level) {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		return fmt.Errorf("invalid log level: %s (must be debug, info, warn, or error)", level)
	}

	// Setup output writer
	var writer io.Writer
	switch {
	case output == "" || output == "stderr":
		writer = os.Stderr
	case strings.HasSuffix(output, "/"):
		// Directory - create dated log file
		if err := os.MkdirAll(output, 0755); err != nil {
			return fmt.Errorf("failed to create log directory: %w", err)
		}
		logFile := filepath.Join(output,
			fmt.Sprintf("manuals-mcp-%s.log", time.Now().Format("2006-01-02")))
		f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return fmt.Errorf("failed to open log file: %w", err)
		}
		// Write to both stderr and file
		writer = io.MultiWriter(os.Stderr, f)
	default:
		// Specific file path
		dir := filepath.Dir(output)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create log directory: %w", err)
		}
		f, err := os.OpenFile(output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return fmt.Errorf("failed to open log file: %w", err)
		}
		// Write to both stderr and file
		writer = io.MultiWriter(os.Stderr, f)
	}

	// Create handler based on format
	opts := &slog.HandlerOptions{Level: logLevel}
	var handler slog.Handler

	if format == "json" {
		handler = slog.NewJSONHandler(writer, opts)
	} else {
		handler = slog.NewTextHandler(writer, opts)
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)

	return nil
}

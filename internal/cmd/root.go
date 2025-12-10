// Package cmd provides the command-line interface for manuals-mcp.
package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// Version information (set by main)
	version   string
	gitCommit string
	buildTime string

	// Global flags
	cfgFile   string
	logLevel  string
	logFormat string
	logOutput string
)

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "manuals-mcp",
	Short: "MCP server for hardware and software documentation",
	Long: `Manuals MCP Server provides GPIO pinout information, full-text search,
and device specifications through the Model Context Protocol (MCP).

Features:
  - GPIO pinout information for hardware devices
  - Full-text search across documentation using SQLite FTS5
  - Device specifications and metadata
  - Cross-platform static binary
  - No external dependencies`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Setup logger for all commands
		return setupLogger()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	return rootCmd.Execute()
}

// SetVersionInfo sets the version information from main.
func SetVersionInfo(ver, commit, build string) {
	version = ver
	gitCommit = commit
	buildTime = build
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.manuals-mcp.yaml)")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().StringVar(&logFormat, "log-format", "text", "log format (json, text)")
	rootCmd.PersistentFlags().StringVar(&logOutput, "log-output", "stderr", "log output (stderr, /path/to/file, or /path/to/dir/)")

	// Bind flags to viper
	viper.BindPFlag("log.level", rootCmd.PersistentFlags().Lookup("log-level"))
	viper.BindPFlag("log.format", rootCmd.PersistentFlags().Lookup("log-format"))
	viper.BindPFlag("log.output", rootCmd.PersistentFlags().Lookup("log-output"))

	// Set environment variable prefix
	viper.SetEnvPrefix("MANUALS")
	viper.AutomaticEnv()
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error finding home directory: %v\n", err)
			os.Exit(1)
		}

		// Search config in home directory with name ".manuals-mcp" (without extension)
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".manuals-mcp")
	}

	// If a config file is found, read it in
	if err := viper.ReadInConfig(); err == nil {
		slog.Debug("using config file", "file", viper.ConfigFileUsed())
	}
}

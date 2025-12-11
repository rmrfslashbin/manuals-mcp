// Package mcp provides the MCP server implementation.
package mcp

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rmrfslashbin/manuals-mcp-server/internal/db"
	"github.com/rmrfslashbin/manuals-mcp-server/pkg/models"
)

// Server wraps the MCP server with our database.
type Server struct {
	mcp       *server.MCPServer
	db        *sql.DB
	docsPath  string
	logger    *slog.Logger
	version   string
	gitCommit string
	buildTime string
}

// NewServer creates a new MCP server instance.
func NewServer(database *sql.DB, docsPath, version, gitCommit, buildTime string, logger *slog.Logger) *Server {
	s := &Server{
		db:        database,
		docsPath:  docsPath,
		logger:    logger,
		version:   version,
		gitCommit: gitCommit,
		buildTime: buildTime,
	}

	// Create MCP server with all capabilities
	s.mcp = server.NewMCPServer(
		"manuals-mcp-server",
		version,
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(true, true), // subscribe and list
		server.WithPromptCapabilities(true),
		server.WithLogging(),
	)

	// Register tools
	s.registerTools()

	// Register resources
	s.registerResources()

	// Register prompts
	s.registerPrompts()

	return s
}

// registerTools registers all MCP tools.
func (s *Server) registerTools() {
	// Tool: get_pinout - Get GPIO pinout information
	s.mcp.AddTool(mcp.NewTool("get_pinout",
		mcp.WithDescription("Get GPIO pinout information for a hardware device"),
		mcp.WithString("device",
			mcp.Description("Device name or ID (e.g., 'raspberry-pi-4', 'esp32-s3')"),
			mcp.Required(),
		),
		mcp.WithString("interface",
			mcp.Description("Filter by interface type (optional): i2c, spi, uart, pwm, gpio, all"),
		),
	), s.handleGetPinout)

	// Tool: search - Full-text search
	s.mcp.AddTool(mcp.NewTool("search",
		mcp.WithDescription("Search across all documentation using full-text search"),
		mcp.WithString("query",
			mcp.Description("Search query (supports full-text search)"),
			mcp.Required(),
		),
		mcp.WithString("domain",
			mcp.Description("Filter by domain (optional): hardware, software, protocol, all"),
		),
		mcp.WithString("type",
			mcp.Description("Filter by device type (optional)"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum results to return"),
			mcp.DefaultNumber(10),
		),
	), s.handleSearch)

	// Tool: list_hardware - List all hardware devices
	s.mcp.AddTool(mcp.NewTool("list_hardware",
		mcp.WithDescription("List all available hardware devices organized by category"),
	), s.handleListHardware)

	// Tool: get_stats - Get database statistics
	s.mcp.AddTool(mcp.NewTool("get_stats",
		mcp.WithDescription("Get statistics about the indexed documentation database"),
	), s.handleGetStats)

	// Tool: get_info - Get server version and platform information
	s.mcp.AddTool(mcp.NewTool("get_info",
		mcp.WithDescription("Get MCP server version, build info, database statistics, and platform capabilities"),
	), s.handleGetInfo)

	// Tool: get_tags - List all unique tags
	s.mcp.AddTool(mcp.NewTool("get_tags",
		mcp.WithDescription("List all unique tags from device metadata"),
	), s.handleGetTags)

	// Tool: get_categories - List all categories
	s.mcp.AddTool(mcp.NewTool("get_categories",
		mcp.WithDescription("List all categories with device counts"),
	), s.handleGetCategories)

	// Tool: get_manufacturers - List all manufacturers
	s.mcp.AddTool(mcp.NewTool("get_manufacturers",
		mcp.WithDescription("List all manufacturers with device counts"),
	), s.handleGetManufacturers)

	// Tool: get_metadata_schema - Show metadata schema
	s.mcp.AddTool(mcp.NewTool("get_metadata_schema",
		mcp.WithDescription("Show available metadata fields and their types across all devices"),
	), s.handleGetMetadataSchema)

	// Tool: reindex - Rebuild documentation index (only if docs-path is configured)
	if s.docsPath != "" {
		s.mcp.AddTool(mcp.NewTool("reindex",
			mcp.WithDescription("Rebuild documentation index from source files (requires --docs-path)"),
		), s.handleReindex)
	}
}

// registerResources registers all MCP resources.
func (s *Server) registerResources() {
	// Resource template: Access device documentation by ID
	s.mcp.AddResourceTemplate(
		mcp.NewResourceTemplate(
			"manuals://device/{device_id}",
			"Device documentation and specifications",
		),
		s.handleDeviceResource,
	)

	// Resource template: Access device pinouts
	s.mcp.AddResourceTemplate(
		mcp.NewResourceTemplate(
			"manuals://device/{device_id}/pinout",
			"Device GPIO pinout information",
		),
		s.handlePinoutResource,
	)

	// Resource template: Access workflow guides
	s.mcp.AddResourceTemplate(
		mcp.NewResourceTemplate(
			"manuals://guide/{guide_id}",
			"Workflow and contribution guides (quickstart, workflow, overview, contributing)",
		),
		s.handleGuideResource,
	)
}

// registerPrompts registers all MCP prompts.
func (s *Server) registerPrompts() {
	// Prompt: wiring-guide - Help with GPIO wiring
	s.mcp.AddPrompt(mcp.NewPrompt("wiring-guide",
		mcp.WithPromptDescription("Guide for wiring a component to GPIO pins"),
		mcp.WithArgument("device",
			mcp.ArgumentDescription("Hardware device (e.g., 'raspberry-pi-4')"),
			mcp.RequiredArgument(),
		),
		mcp.WithArgument("component",
			mcp.ArgumentDescription("Component to wire (e.g., 'temperature sensor', 'LED')"),
			mcp.RequiredArgument(),
		),
		mcp.WithArgument("interface",
			mcp.ArgumentDescription("Communication interface (e.g., 'I2C', 'SPI', 'GPIO')"),
		),
	), s.handleWiringGuide)

	// Prompt: pinout-explain - Explain a device's pinout
	s.mcp.AddPrompt(mcp.NewPrompt("pinout-explain",
		mcp.WithPromptDescription("Get a detailed explanation of a device's pinout configuration"),
		mcp.WithArgument("device",
			mcp.ArgumentDescription("Hardware device (e.g., 'esp32-s3')"),
			mcp.RequiredArgument(),
		),
	), s.handlePinoutExplain)

	// Prompt: device-compare - Compare two devices
	s.mcp.AddPrompt(mcp.NewPrompt("device-compare",
		mcp.WithPromptDescription("Compare specifications and features of two devices"),
		mcp.WithArgument("device1",
			mcp.ArgumentDescription("First device name or ID"),
			mcp.RequiredArgument(),
		),
		mcp.WithArgument("device2",
			mcp.ArgumentDescription("Second device name or ID"),
			mcp.RequiredArgument(),
		),
	), s.handleDeviceCompare)

	// Prompt: protocol-guide - Guide for implementing a protocol
	s.mcp.AddPrompt(mcp.NewPrompt("protocol-guide",
		mcp.WithPromptDescription("Get implementation guidance for a communication protocol"),
		mcp.WithArgument("protocol",
			mcp.ArgumentDescription("Protocol name (e.g., 'MQTT', 'I2C', 'SPI')"),
			mcp.RequiredArgument(),
		),
		mcp.WithArgument("platform",
			mcp.ArgumentDescription("Target platform (optional)"),
		),
	), s.handleProtocolGuide)
}

// Serve starts the MCP server with stdio transport.
func (s *Server) Serve(ctx context.Context) error {
	s.logger.Info("starting MCP server with stdio transport")

	// Serve with stdio transport (default for MCP)
	return server.ServeStdio(s.mcp)
}

// findDeviceByNameOrID attempts to find a device by exact ID or partial name match.
func (s *Server) findDeviceByNameOrID(ctx context.Context, nameOrID string) (*models.Device, error) {
	// Try exact ID first
	device, err := db.GetDevice(s.db, nameOrID)
	if err != nil {
		return nil, fmt.Errorf("failed to get device: %w", err)
	}
	if device != nil {
		return device, nil
	}

	// Try searching by name
	opts := models.SearchOptions{
		Query:  nameOrID,
		Limit:  1,
		Offset: 0,
	}
	results, err := db.SearchDevices(s.db, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to search devices: %w", err)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("device not found: %s", nameOrID)
	}

	// Get full device info
	return db.GetDevice(s.db, results[0].ID)
}

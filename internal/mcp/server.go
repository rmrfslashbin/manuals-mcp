// Package mcp provides the MCP server implementation.
package mcp

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rmrfslashbin/manuals-mcp/internal/client"
)

// Server wraps the MCP server with our API client.
type Server struct {
	mcp       *server.MCPServer
	client    *client.Client
	logger    *slog.Logger
	version   string
	gitCommit string
	buildTime string
}

// NewServer creates a new MCP server instance.
func NewServer(apiClient *client.Client, version, gitCommit, buildTime string, logger *slog.Logger) *Server {
	s := &Server{
		client:    apiClient,
		logger:    logger,
		version:   version,
		gitCommit: gitCommit,
		buildTime: buildTime,
	}

	// Create MCP server
	s.mcp = server.NewMCPServer(
		"manuals-mcp",
		version,
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(true, true),
		server.WithLogging(),
	)

	// Register tools
	s.registerTools()

	// Register resources
	s.registerResources()

	return s
}

// registerTools registers all MCP tools.
func (s *Server) registerTools() {
	// Tool: search - Full-text search
	s.mcp.AddTool(mcp.NewTool("search_manuals",
		mcp.WithDescription("Search across hardware and software documentation using full-text search"),
		mcp.WithString("query",
			mcp.Description("Search query"),
			mcp.Required(),
		),
		mcp.WithString("domain",
			mcp.Description("Filter by domain: hardware, software, or protocol"),
		),
		mcp.WithString("type",
			mcp.Description("Filter by device type (e.g., sensors, dev-boards)"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum results to return (default: 10)"),
		),
	), s.handleSearch)

	// Tool: get_device - Get device details
	s.mcp.AddTool(mcp.NewTool("get_device",
		mcp.WithDescription("Get detailed information about a specific device including content and metadata"),
		mcp.WithString("device_id",
			mcp.Description("Device ID"),
			mcp.Required(),
		),
	), s.handleGetDevice)

	// Tool: list_devices - List all devices
	s.mcp.AddTool(mcp.NewTool("list_devices",
		mcp.WithDescription("List all devices with optional filtering"),
		mcp.WithString("domain",
			mcp.Description("Filter by domain: hardware, software, or protocol"),
		),
		mcp.WithString("type",
			mcp.Description("Filter by device type"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum results (default: 50)"),
		),
	), s.handleListDevices)

	// Tool: get_pinout - Get GPIO pinout
	s.mcp.AddTool(mcp.NewTool("get_pinout",
		mcp.WithDescription("Get GPIO pinout information for a hardware device"),
		mcp.WithString("device_id",
			mcp.Description("Device ID"),
			mcp.Required(),
		),
	), s.handleGetPinout)

	// Tool: get_specs - Get device specifications
	s.mcp.AddTool(mcp.NewTool("get_specs",
		mcp.WithDescription("Get technical specifications for a device"),
		mcp.WithString("device_id",
			mcp.Description("Device ID"),
			mcp.Required(),
		),
	), s.handleGetSpecs)

	// Tool: list_documents - List documents
	s.mcp.AddTool(mcp.NewTool("list_documents",
		mcp.WithDescription("List available documents (PDFs, datasheets)"),
		mcp.WithString("device_id",
			mcp.Description("Filter by device ID"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum results (default: 50)"),
		),
	), s.handleListDocuments)

	// Tool: get_status - Get API status
	s.mcp.AddTool(mcp.NewTool("get_status",
		mcp.WithDescription("Get Manuals API status and statistics"),
	), s.handleGetStatus)
}

// registerResources registers MCP resources.
func (s *Server) registerResources() {
	// Resource template: Device documentation
	s.mcp.AddResourceTemplate(
		mcp.NewResourceTemplate(
			"manuals://device/{device_id}",
			"Device documentation and content",
		),
		s.handleDeviceResource,
	)

	// Resource template: Device pinout
	s.mcp.AddResourceTemplate(
		mcp.NewResourceTemplate(
			"manuals://device/{device_id}/pinout",
			"Device GPIO pinout information",
		),
		s.handlePinoutResource,
	)
}

// Serve starts the MCP server with stdio transport.
func (s *Server) Serve(ctx context.Context) error {
	s.logger.Info("starting MCP server with stdio transport")
	return server.ServeStdio(s.mcp)
}

// Tool handlers

func (s *Server) handleSearch(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	query, _ := args["query"].(string)
	domain, _ := args["domain"].(string)
	deviceType, _ := args["type"].(string)
	limit := 10
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}

	results, err := s.client.Search(query, limit, domain, deviceType)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("search failed: %v", err)), nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d results for \"%s\":\n\n", results.Total, results.Query))

	for _, r := range results.Results {
		sb.WriteString(fmt.Sprintf("**%s** (ID: %s)\n", r.Name, r.DeviceID))
		sb.WriteString(fmt.Sprintf("  Domain: %s | Type: %s | Score: %.2f\n", r.Domain, r.Type, r.Score))
		if r.Snippet != "" {
			sb.WriteString(fmt.Sprintf("  %s\n", r.Snippet))
		}
		sb.WriteString("\n")
	}

	return mcp.NewToolResultText(sb.String()), nil
}

func (s *Server) handleGetDevice(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	deviceID, _ := args["device_id"].(string)

	device, err := s.client.GetDevice(deviceID, true)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get device: %v", err)), nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s\n\n", device.Name))
	sb.WriteString(fmt.Sprintf("**ID:** %s\n", device.ID))
	sb.WriteString(fmt.Sprintf("**Domain:** %s\n", device.Domain))
	sb.WriteString(fmt.Sprintf("**Type:** %s\n", device.Type))
	sb.WriteString(fmt.Sprintf("**Path:** %s\n", device.Path))
	sb.WriteString(fmt.Sprintf("**Indexed:** %s\n\n", device.IndexedAt))

	if device.Content != "" {
		sb.WriteString("## Content\n\n")
		sb.WriteString(device.Content)
	}

	return mcp.NewToolResultText(sb.String()), nil
}

func (s *Server) handleListDevices(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	domain, _ := args["domain"].(string)
	deviceType, _ := args["type"].(string)
	limit := 50
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}

	result, err := s.client.ListDevices(limit, 0, domain, deviceType)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list devices: %v", err)), nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Showing %d of %d devices:\n\n", len(result.Data), result.Total))

	for _, d := range result.Data {
		sb.WriteString(fmt.Sprintf("- **%s** (ID: %s) - %s/%s\n", d.Name, d.ID, d.Domain, d.Type))
	}

	return mcp.NewToolResultText(sb.String()), nil
}

func (s *Server) handleGetPinout(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	deviceID, _ := args["device_id"].(string)

	pinout, err := s.client.GetDevicePinout(deviceID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get pinout: %v", err)), nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Pinout for %s\n\n", pinout.Name))
	sb.WriteString("| Pin | GPIO | Name | Description |\n")
	sb.WriteString("|-----|------|------|-------------|\n")

	for _, pin := range pinout.Pins {
		gpio := "-"
		if pin.GPIONum != nil {
			gpio = fmt.Sprintf("%d", *pin.GPIONum)
		}
		sb.WriteString(fmt.Sprintf("| %d | %s | %s | %s |\n", pin.PhysicalPin, gpio, pin.Name, pin.Description))
	}

	return mcp.NewToolResultText(sb.String()), nil
}

func (s *Server) handleGetSpecs(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	deviceID, _ := args["device_id"].(string)

	specs, err := s.client.GetDeviceSpecs(deviceID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get specs: %v", err)), nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Specifications for %s\n\n", specs.Name))

	for key, value := range specs.Specs {
		sb.WriteString(fmt.Sprintf("- **%s:** %s\n", key, value))
	}

	return mcp.NewToolResultText(sb.String()), nil
}

func (s *Server) handleListDocuments(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	deviceID, _ := args["device_id"].(string)
	limit := 50
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}

	result, err := s.client.ListDocuments(limit, 0, deviceID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list documents: %v", err)), nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d documents:\n\n", result.Total))

	for _, d := range result.Data {
		size := float64(d.SizeBytes) / 1024
		sb.WriteString(fmt.Sprintf("- **%s** (ID: %s) - %.1f KB\n", d.Filename, d.ID, size))
	}

	return mcp.NewToolResultText(sb.String()), nil
}

func (s *Server) handleGetStatus(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	status, err := s.client.GetStatus()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get status: %v", err)), nil
	}

	var sb strings.Builder
	sb.WriteString("# Manuals API Status\n\n")
	sb.WriteString(fmt.Sprintf("- **Status:** %s\n", status.Status))
	sb.WriteString(fmt.Sprintf("- **API Version:** %s\n", status.APIVersion))
	sb.WriteString(fmt.Sprintf("- **Server Version:** %s\n", status.Version))
	sb.WriteString(fmt.Sprintf("- **Devices:** %d\n", status.Counts.Devices))
	sb.WriteString(fmt.Sprintf("- **Documents:** %d\n", status.Counts.Documents))
	if status.LastReindex != "" {
		sb.WriteString(fmt.Sprintf("- **Last Reindex:** %s\n", status.LastReindex))
	}

	return mcp.NewToolResultText(sb.String()), nil
}

// Resource handlers

func (s *Server) handleDeviceResource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	// Extract device_id from URI: manuals://device/{device_id}
	uri := request.Params.URI
	parts := strings.Split(uri, "/")
	if len(parts) < 4 {
		return nil, fmt.Errorf("invalid resource URI: %s", uri)
	}
	deviceID := parts[3]

	device, err := s.client.GetDevice(deviceID, true)
	if err != nil {
		return nil, fmt.Errorf("failed to get device: %w", err)
	}

	content := fmt.Sprintf("# %s\n\n%s", device.Name, device.Content)

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      uri,
			MIMEType: "text/markdown",
			Text:     content,
		},
	}, nil
}

func (s *Server) handlePinoutResource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	// Extract device_id from URI: manuals://device/{device_id}/pinout
	uri := request.Params.URI
	parts := strings.Split(uri, "/")
	if len(parts) < 4 {
		return nil, fmt.Errorf("invalid resource URI: %s", uri)
	}
	deviceID := parts[3]

	pinout, err := s.client.GetDevicePinout(deviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get pinout: %w", err)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Pinout for %s\n\n", pinout.Name))
	for _, pin := range pinout.Pins {
		gpio := "N/A"
		if pin.GPIONum != nil {
			gpio = fmt.Sprintf("GPIO%d", *pin.GPIONum)
		}
		sb.WriteString(fmt.Sprintf("Pin %d: %s (%s) - %s\n", pin.PhysicalPin, pin.Name, gpio, pin.Description))
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      uri,
			MIMEType: "text/markdown",
			Text:     sb.String(),
		},
	}, nil
}

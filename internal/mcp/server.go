// Package mcp provides the MCP server implementation.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

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
	// ===========================================
	// DISCOVERY & WORKFLOW TOOLS
	// These tools help AI assistants understand capabilities and workflows
	// ===========================================

	// Tool: my_capabilities - Show available actions based on role
	s.mcp.AddTool(mcp.NewTool("my_capabilities",
		mcp.WithDescription("Show available tools and capabilities based on your authentication role. Use this first to understand what actions you can perform. Returns a categorized list of available tools with usage examples."),
	), s.handleMyCapabilities)

	// Tool: ingest_workflow - Get document ingestion workflow guidance
	s.mcp.AddTool(mcp.NewTool("ingest_workflow",
		mcp.WithDescription("Get step-by-step guidance for ingesting new documentation into the platform. Explains the complete workflow from processing a source document (PDF/datasheet) to publishing it. Use this when you need to add new hardware or software documentation."),
		mcp.WithString("doc_type",
			mcp.Description("Type of documentation to ingest: 'hardware' (MCU, sensor, SBC), 'software' (applications, tools), or 'protocol' (I2C, SPI, UART). Defaults to 'hardware'."),
		),
	), s.handleIngestWorkflow)

	// ===========================================
	// READ-ONLY TOOLS (Available to all users including anonymous)
	// ===========================================

	// Tool: search - Full-text search
	s.mcp.AddTool(mcp.NewTool("search_manuals",
		mcp.WithDescription("Search across all hardware and software documentation using full-text search. Returns matching devices with relevance scores and text snippets. Use this to find devices by name, feature, interface (I2C, SPI, UART), or any keyword in the documentation."),
		mcp.WithString("query",
			mcp.Description("Search query - can be device name, feature, interface type, or any keyword"),
			mcp.Required(),
		),
		mcp.WithString("domain",
			mcp.Description("Filter by domain: 'hardware', 'software', or 'protocol'"),
		),
		mcp.WithString("type",
			mcp.Description("Filter by device type (e.g., 'sensors', 'mcu-boards', 'sbc', 'power', 'displays')"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum results to return (default: 10, max: 100)"),
		),
	), s.handleSearch)

	// Tool: get_device - Get device details
	s.mcp.AddTool(mcp.NewTool("get_device",
		mcp.WithDescription("Get complete documentation for a specific device including full markdown content, metadata, and specifications. Use the device_id from search_manuals or list_devices results."),
		mcp.WithString("device_id",
			mcp.Description("Device ID (e.g., 'sbc-raspberry-pi-raspberry-pi-5'). Use search_manuals to find device IDs."),
			mcp.Required(),
		),
	), s.handleGetDevice)

	// Tool: list_devices - List all devices
	s.mcp.AddTool(mcp.NewTool("list_devices",
		mcp.WithDescription("Browse all devices in the documentation library with optional filtering. Returns device names, IDs, domains, and types. Use this to explore available documentation or find devices by category."),
		mcp.WithString("domain",
			mcp.Description("Filter by domain: 'hardware', 'software', or 'protocol'"),
		),
		mcp.WithString("type",
			mcp.Description("Filter by device type: 'sensors', 'mcu-boards', 'sbc', 'power', 'displays', 'communication', etc."),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum results (default: 50, max: 200)"),
		),
	), s.handleListDevices)

	// Tool: get_pinout - Get GPIO pinout
	s.mcp.AddTool(mcp.NewTool("get_pinout",
		mcp.WithDescription("Get GPIO pinout table for a hardware device. Returns physical pin numbers, GPIO numbers, pin names, and descriptions. Essential for wiring diagrams and hardware connections."),
		mcp.WithString("device_id",
			mcp.Description("Device ID (e.g., 'sbc-raspberry-pi-raspberry-pi-5')"),
			mcp.Required(),
		),
	), s.handleGetPinout)

	// Tool: get_specs - Get device specifications
	s.mcp.AddTool(mcp.NewTool("get_specs",
		mcp.WithDescription("Get technical specifications for a device as key-value pairs. Useful for comparing devices or quick specification lookups without retrieving full documentation."),
		mcp.WithString("device_id",
			mcp.Description("Device ID (e.g., 'sensors-temperature-ds18b20')"),
			mcp.Required(),
		),
	), s.handleGetSpecs)

	// Tool: list_documents - List documents
	s.mcp.AddTool(mcp.NewTool("list_documents",
		mcp.WithDescription("List available PDF documents and datasheets. Returns document IDs, filenames, and sizes. Documents can be associated with specific devices or be standalone."),
		mcp.WithString("device_id",
			mcp.Description("Filter to show only documents for a specific device"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum results (default: 50)"),
		),
	), s.handleListDocuments)

	// Tool: get_status - Get API status
	s.mcp.AddTool(mcp.NewTool("get_status",
		mcp.WithDescription("Get Manuals API health status and database statistics. Shows total device count, document count, and last reindex time. Use to verify the API is operational."),
	), s.handleGetStatus)

	// Tool: info - Get MCP server information
	s.mcp.AddTool(mcp.NewTool("info",
		mcp.WithDescription("Get MCP server version, build info, API connection status, and current authentication details. Shows your user name, role, and what capabilities are available to you."),
	), s.handleInfo)

	// ===========================================
	// CONTENT MANAGEMENT TOOLS (Require RW or Admin role)
	// ===========================================

	// Tool: trigger_reindex - Trigger documentation reindex
	s.mcp.AddTool(mcp.NewTool("trigger_reindex",
		mcp.WithDescription("Trigger a background reindex of all documentation. The index is updated from files in the docs storage. Use after uploading new files. Requires RW or Admin role."),
	), s.handleTriggerReindex)

	// Tool: get_reindex_status - Get reindex status
	s.mcp.AddTool(mcp.NewTool("get_reindex_status",
		mcp.WithDescription("Check the status of the documentation reindex operation. Shows if reindex is running, last completion time, and statistics from the last run. Requires RW or Admin role."),
	), s.handleGetReindexStatus)

	// Tool: upload_file - Upload a file from local filesystem
	s.mcp.AddTool(mcp.NewTool("upload_file",
		mcp.WithDescription("Upload a file to the documentation storage. Can read directly from a local file path (preferred) or accept content as a string. Requires RW or Admin role."),
		mcp.WithString("dest_path",
			mcp.Description("Destination path in docs storage (e.g., 'sensors/environmental/bme680/BME680_Reference.md' or 'guides/QUICKSTART.md')"),
			mcp.Required(),
		),
		mcp.WithString("local_path",
			mcp.Description("Local filesystem path to read the file from (e.g., '/home/user/docs/README.md'). Preferred over content parameter."),
		),
		mcp.WithString("content",
			mcp.Description("File content as text. Only use if local_path is not available. For binary files, use local_path instead."),
		),
	), s.handleUploadFile)

	// Tool: publish - Upload file and trigger reindex in one operation
	s.mcp.AddTool(mcp.NewTool("publish",
		mcp.WithDescription("Upload a file and automatically trigger reindex. Combines upload_file + trigger_reindex in one operation. This is the preferred method for publishing new documentation. Requires RW or Admin role."),
		mcp.WithString("dest_path",
			mcp.Description("Destination path in docs storage (e.g., 'sensors/temperature/ds18b20/DS18B20_Reference.md')"),
			mcp.Required(),
		),
		mcp.WithString("local_path",
			mcp.Description("Local filesystem path to read the file from. Preferred method."),
		),
		mcp.WithString("content",
			mcp.Description("File content as text. Only use if local_path is not available."),
		),
		mcp.WithBoolean("wait_for_reindex",
			mcp.Description("If true, wait for reindex to complete before returning (default: false)"),
		),
	), s.handlePublish)

	// Tool: publish_batch - Upload multiple files and trigger single reindex
	s.mcp.AddTool(mcp.NewTool("publish_batch",
		mcp.WithDescription("Upload multiple files and trigger a single reindex after all uploads complete. More efficient than multiple publish calls. Requires RW or Admin role."),
		mcp.WithString("files",
			mcp.Description("JSON array of file objects: [{\"local_path\": \"/path/to/file\", \"dest_path\": \"sensors/temp/file.md\"}, ...]. Each object must have dest_path and either local_path or content."),
			mcp.Required(),
		),
		mcp.WithBoolean("wait_for_reindex",
			mcp.Description("If true, wait for reindex to complete before returning (default: false)"),
		),
	), s.handlePublishBatch)

	// Tool: sync_to_git - Sync documentation to git repository
	s.mcp.AddTool(mcp.NewTool("sync_to_git",
		mcp.WithDescription("Sync all documentation changes to the git repository. Commits and pushes any new or modified files to the remote repository. Use this after publishing new documentation to persist changes. Requires RW or Admin role."),
	), s.handleSyncToGit)

	// ===========================================
	// ADMIN TOOLS (Require Admin role)
	// ===========================================

	// Tool: list_users - List all users
	s.mcp.AddTool(mcp.NewTool("list_users",
		mcp.WithDescription("List all users with their roles, status, and creation dates. Use to audit user access. Requires Admin role."),
	), s.handleListUsers)

	// Tool: create_user - Create a new user
	s.mcp.AddTool(mcp.NewTool("create_user",
		mcp.WithDescription("Create a new user account and generate an API key. IMPORTANT: The API key is only shown once - save it immediately. Requires Admin role."),
		mcp.WithString("name",
			mcp.Description("User name (e.g., 'alice', 'ci-bot', 'readonly-viewer')"),
			mcp.Required(),
		),
		mcp.WithString("role",
			mcp.Description("User role: 'admin' (full access), 'rw' (read + publish docs), or 'ro' (read-only)"),
			mcp.Required(),
		),
	), s.handleCreateUser)

	// Tool: delete_user - Delete a user
	s.mcp.AddTool(mcp.NewTool("delete_user",
		mcp.WithDescription("Delete a user account and invalidate their API key. This action cannot be undone. Requires Admin role."),
		mcp.WithString("user_id",
			mcp.Description("User ID to delete (get from list_users)"),
			mcp.Required(),
		),
	), s.handleDeleteUser)
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

// ===========================================
// DISCOVERY & WORKFLOW HANDLERS
// ===========================================

func (s *Server) handleMyCapabilities(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var sb strings.Builder

	sb.WriteString("# Your Capabilities\n\n")

	// Check authentication status
	var role string
	var userName string
	if s.client.HasAPIKey() {
		user, err := s.client.GetMe()
		if err != nil {
			sb.WriteString("**Status:** Error checking authentication\n\n")
			role = "unknown"
		} else if user != nil {
			userName = user.Name
			role = user.Role
			sb.WriteString(fmt.Sprintf("**User:** %s\n", userName))
			sb.WriteString(fmt.Sprintf("**Role:** %s\n", role))
			sb.WriteString("**Status:** Authenticated\n\n")
		}
	} else {
		role = "anonymous"
		sb.WriteString("**Status:** Anonymous (read-only)\n\n")
	}

	// Read-only tools (available to everyone)
	sb.WriteString("## Read-Only Tools (Available)\n\n")
	sb.WriteString("| Tool | Description |\n")
	sb.WriteString("|------|-------------|\n")
	sb.WriteString("| `search_manuals` | Search documentation by keyword |\n")
	sb.WriteString("| `list_devices` | Browse all devices |\n")
	sb.WriteString("| `get_device` | Get full device documentation |\n")
	sb.WriteString("| `get_pinout` | Get GPIO pinout table |\n")
	sb.WriteString("| `get_specs` | Get device specifications |\n")
	sb.WriteString("| `list_documents` | List PDFs and datasheets |\n")
	sb.WriteString("| `get_status` | Check API health |\n")
	sb.WriteString("| `info` | Get server and auth info |\n")
	sb.WriteString("| `ingest_workflow` | Get document ingestion guidance |\n\n")

	// Content management tools (RW or Admin)
	if role == "rw" || role == "admin" {
		sb.WriteString("## Content Management Tools (Available)\n\n")
		sb.WriteString("| Tool | Description |\n")
		sb.WriteString("|------|-------------|\n")
		sb.WriteString("| `upload_file` | Upload a file to docs storage |\n")
		sb.WriteString("| `publish` | Upload + auto-reindex (recommended) |\n")
		sb.WriteString("| `publish_batch` | Upload multiple files + reindex |\n")
		sb.WriteString("| `trigger_reindex` | Manually trigger reindex |\n")
		sb.WriteString("| `get_reindex_status` | Check reindex progress |\n")
		sb.WriteString("| `sync_to_git` | Commit and push docs to git repo |\n\n")
	} else {
		sb.WriteString("## Content Management Tools (Requires RW Role)\n\n")
		sb.WriteString("*Not available with your current role. Contact admin for RW access.*\n\n")
	}

	// Admin tools
	if role == "admin" {
		sb.WriteString("## Admin Tools (Available)\n\n")
		sb.WriteString("| Tool | Description |\n")
		sb.WriteString("|------|-------------|\n")
		sb.WriteString("| `list_users` | List all user accounts |\n")
		sb.WriteString("| `create_user` | Create new user + API key |\n")
		sb.WriteString("| `delete_user` | Delete user account |\n\n")
	} else if role == "rw" {
		sb.WriteString("## Admin Tools (Requires Admin Role)\n\n")
		sb.WriteString("*Not available with your current role.*\n\n")
	}

	// Quick start guide
	sb.WriteString("## Quick Start\n\n")
	sb.WriteString("1. **Find devices:** `search_manuals(query: \"raspberry pi\")` or `list_devices()`\n")
	sb.WriteString("2. **Get details:** `get_device(device_id: \"...\")` using ID from search\n")
	sb.WriteString("3. **Get pinout:** `get_pinout(device_id: \"...\")` for wiring info\n")
	if role == "rw" || role == "admin" {
		sb.WriteString("4. **Add docs:** Use `ingest_workflow()` for guidance, then `publish()`\n")
	}

	return mcp.NewToolResultText(sb.String()), nil
}

func (s *Server) handleIngestWorkflow(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	docType, _ := args["doc_type"].(string)
	if docType == "" {
		docType = "hardware"
	}

	var sb strings.Builder

	sb.WriteString("# Document Ingestion Workflow\n\n")

	// Check if user has RW permissions
	if s.client.HasAPIKey() {
		user, err := s.client.GetMe()
		if err == nil && user != nil && (user.Role == "rw" || user.Role == "admin") {
			sb.WriteString("**Your Role:** " + user.Role + " ✓ (can publish)\n\n")
		} else {
			sb.WriteString("**⚠️ Note:** You need RW or Admin role to publish. Current workflow is read-only.\n\n")
		}
	} else {
		sb.WriteString("**⚠️ Note:** You're in anonymous mode. Publishing requires authentication.\n\n")
	}

	sb.WriteString("## Overview\n\n")
	sb.WriteString("This workflow guides you through adding new documentation to the Manuals platform.\n")
	sb.WriteString("The AI assistant (you) processes source documents and creates structured markdown.\n\n")

	sb.WriteString("## Step 1: Analyze Source Document\n\n")
	sb.WriteString("Read the source document (PDF, datasheet, manual) and extract:\n\n")

	switch docType {
	case "software":
		sb.WriteString("- Application name and version\n")
		sb.WriteString("- Supported platforms\n")
		sb.WriteString("- Key features and capabilities\n")
		sb.WriteString("- Installation instructions\n")
		sb.WriteString("- Configuration options\n")
		sb.WriteString("- Usage examples\n\n")
	case "protocol":
		sb.WriteString("- Protocol name and version\n")
		sb.WriteString("- Physical layer specifications\n")
		sb.WriteString("- Frame format and timing\n")
		sb.WriteString("- Command/register reference\n")
		sb.WriteString("- Example transactions\n\n")
	default: // hardware
		sb.WriteString("- Device name and model number\n")
		sb.WriteString("- Manufacturer\n")
		sb.WriteString("- Key specifications (voltage, current, temperature range)\n")
		sb.WriteString("- Complete pinout with GPIO mappings\n")
		sb.WriteString("- Communication interfaces (I2C, SPI, UART)\n")
		sb.WriteString("- Wiring diagrams (ASCII art)\n\n")
	}

	sb.WriteString("## Step 2: Create Markdown Documentation\n\n")
	sb.WriteString("Create a structured markdown file with:\n\n")
	sb.WriteString("```yaml\n")
	sb.WriteString("---\n")
	sb.WriteString("manufacturer: [Manufacturer Name]\n")
	sb.WriteString("model: [Model Number]\n")
	sb.WriteString("category: [category]/[subcategory]\n")
	sb.WriteString("version: v1.0\n")
	sb.WriteString("date: YYYY-MM-DD\n")
	sb.WriteString("tags: [tag1, tag2, tag3]\n")
	sb.WriteString("specs:\n")
	sb.WriteString("  key1: \"value1\"\n")
	sb.WriteString("  key2: \"value2\"\n")
	sb.WriteString("---\n")
	sb.WriteString("```\n\n")
	sb.WriteString("Follow with: Overview, Specifications, Pinout (table), Wiring, Examples.\n\n")

	sb.WriteString("## Step 3: Determine Destination Path\n\n")
	sb.WriteString("Path format: `{category}/{subcategory}/{device-name}/{filename}.md`\n\n")
	sb.WriteString("**Categories:**\n")
	sb.WriteString("- `mcu-boards/` - ESP32, STM32, Arduino boards\n")
	sb.WriteString("- `sensors/` - Temperature, radar, environmental\n")
	sb.WriteString("- `sbc/` - Raspberry Pi, Orange Pi\n")
	sb.WriteString("- `power/` - Regulators, battery management\n")
	sb.WriteString("- `displays/` - LCD, OLED, e-ink\n")
	sb.WriteString("- `communication/` - WiFi, LoRa, cellular\n")
	sb.WriteString("- `software/` - Applications, tools\n")
	sb.WriteString("- `protocols/` - I2C, SPI, UART specs\n")
	sb.WriteString("- `guides/` - Platform guides and tutorials\n\n")

	sb.WriteString("**Example paths:**\n")
	sb.WriteString("- `sensors/temperature/ds18b20/DS18B20_Reference.md`\n")
	sb.WriteString("- `mcu-boards/esp32/esp32-s3-devkitc-1/ESP32-S3-DevKitC-1_Reference.md`\n")
	sb.WriteString("- `guides/QUICKSTART.md`\n\n")

	sb.WriteString("## Step 4: Save & Publish\n\n")
	sb.WriteString("1. **Save the markdown file locally** (e.g., `/tmp/device_docs/DS18B20_Reference.md`)\n\n")
	sb.WriteString("2. **Publish using local path (recommended):**\n")
	sb.WriteString("   ```\n")
	sb.WriteString("   publish(\n")
	sb.WriteString("     local_path: \"/tmp/device_docs/DS18B20_Reference.md\",\n")
	sb.WriteString("     dest_path: \"sensors/temperature/ds18b20/DS18B20_Reference.md\"\n")
	sb.WriteString("   )\n")
	sb.WriteString("   ```\n\n")
	sb.WriteString("3. **For multiple files, use publish_batch:**\n")
	sb.WriteString("   ```\n")
	sb.WriteString("   publish_batch(\n")
	sb.WriteString("     files: '[{\"local_path\": \"/tmp/file1.md\", \"dest_path\": \"path/file1.md\"}, ...]'\n")
	sb.WriteString("   )\n")
	sb.WriteString("   ```\n\n")

	sb.WriteString("## Step 5: Verify\n\n")
	sb.WriteString("After publishing:\n")
	sb.WriteString("1. `get_reindex_status()` - Confirm reindex completed\n")
	sb.WriteString("2. `search_manuals(query: \"device name\")` - Verify document is searchable\n")
	sb.WriteString("3. `get_device(device_id: \"...\")` - Check content renders correctly\n\n")

	sb.WriteString("## Tips\n\n")
	sb.WriteString("- Use ASCII art for diagrams (not images)\n")
	sb.WriteString("- Use markdown tables for pinouts and specs\n")
	sb.WriteString("- Include code examples in fenced blocks with language\n")
	sb.WriteString("- Reference PDF pages for complex diagrams\n")

	return mcp.NewToolResultText(sb.String()), nil
}

// ===========================================
// READ-ONLY TOOL HANDLERS
// ===========================================

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

func (s *Server) handleInfo(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var sb strings.Builder

	// MCP Server info
	sb.WriteString("# Manuals MCP Server\n\n")
	sb.WriteString("## Server Info\n\n")
	sb.WriteString(fmt.Sprintf("- **Version:** %s\n", s.version))
	sb.WriteString(fmt.Sprintf("- **Git Commit:** %s\n", s.gitCommit))
	sb.WriteString(fmt.Sprintf("- **Build Time:** %s\n", s.buildTime))
	sb.WriteString("- **Project:** github.com/rmrfslashbin/manuals-mcp\n")
	sb.WriteString("- **License:** MIT\n\n")

	// API Connection info
	sb.WriteString("## API Connection\n\n")
	sb.WriteString(fmt.Sprintf("- **API URL:** %s\n", s.client.GetAPIURL()))

	// Get API status
	status, err := s.client.GetStatus()
	if err != nil {
		sb.WriteString(fmt.Sprintf("- **Status:** Error (%v)\n", err))
	} else {
		sb.WriteString(fmt.Sprintf("- **Status:** %s\n", status.Status))
		sb.WriteString(fmt.Sprintf("- **API Version:** %s\n", status.APIVersion))
		sb.WriteString(fmt.Sprintf("- **Devices:** %d\n", status.Counts.Devices))
		sb.WriteString(fmt.Sprintf("- **Documents:** %d\n", status.Counts.Documents))
	}
	sb.WriteString("\n")

	// Authentication info
	sb.WriteString("## Authentication\n\n")
	if s.client.HasAPIKey() {
		sb.WriteString("- **Mode:** Authenticated\n")
		user, err := s.client.GetMe()
		if err != nil {
			sb.WriteString(fmt.Sprintf("- **User:** Error fetching user info (%v)\n", err))
		} else if user != nil {
			sb.WriteString(fmt.Sprintf("- **User:** %s\n", user.Name))
			sb.WriteString(fmt.Sprintf("- **Role:** %s\n", user.Role))
			sb.WriteString(fmt.Sprintf("- **Active:** %t\n", user.IsActive))
		}
	} else {
		sb.WriteString("- **Mode:** Anonymous (read-only)\n")
		sb.WriteString("- **Access:** Read-only access to documentation\n")
		sb.WriteString("- **Note:** Admin features unavailable without API key\n")
	}

	return mcp.NewToolResultText(sb.String()), nil
}

// RW tool handlers

func (s *Server) handleTriggerReindex(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	resp, err := s.client.TriggerReindex()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to trigger reindex: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("# Reindex Triggered\n\n- **Status:** %s\n- **Message:** %s\n", resp.Status, resp.Message)), nil
}

func (s *Server) handleGetReindexStatus(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	resp, err := s.client.GetReindexStatus()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get reindex status: %v", err)), nil
	}

	var sb strings.Builder
	sb.WriteString("# Reindex Status\n\n")
	sb.WriteString(fmt.Sprintf("- **Status:** %s\n", resp.Status))

	if resp.StartedAt != "" {
		sb.WriteString(fmt.Sprintf("- **Started At:** %s\n", resp.StartedAt))
	}
	if resp.Elapsed != "" {
		sb.WriteString(fmt.Sprintf("- **Elapsed:** %s\n", resp.Elapsed))
	}
	if resp.LastCompleted != "" {
		sb.WriteString(fmt.Sprintf("- **Last Completed:** %s\n", resp.LastCompleted))
	}

	if resp.LastRun != nil {
		sb.WriteString("\n## Last Run Stats\n\n")
		sb.WriteString(fmt.Sprintf("- **Devices Indexed:** %d\n", resp.LastRun.DevicesIndexed))
		sb.WriteString(fmt.Sprintf("- **Documents Indexed:** %d\n", resp.LastRun.DocumentsIndexed))
		sb.WriteString(fmt.Sprintf("- **Errors:** %d\n", resp.LastRun.Errors))
		sb.WriteString(fmt.Sprintf("- **Duration:** %s\n", resp.LastRun.Duration))
	}

	return mcp.NewToolResultText(sb.String()), nil
}

func (s *Server) handleUploadFile(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	destPath, _ := args["dest_path"].(string)
	localPath, _ := args["local_path"].(string)
	content, _ := args["content"].(string)

	if destPath == "" {
		return mcp.NewToolResultError("dest_path is required"), nil
	}

	var fileContent []byte
	var filename string

	// Prefer local_path over content
	if localPath != "" {
		// Read file from local filesystem
		data, err := os.ReadFile(localPath)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to read local file '%s': %v", localPath, err)), nil
		}
		fileContent = data
		filename = filepath.Base(localPath)
	} else if content != "" {
		// Use provided content
		fileContent = []byte(content)
		filename = filepath.Base(destPath)
	} else {
		return mcp.NewToolResultError("either local_path or content must be provided"), nil
	}

	resp, err := s.client.UploadFile(destPath, filename, fileContent)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to upload file: %v", err)), nil
	}

	var sb strings.Builder
	sb.WriteString("# File Uploaded\n\n")
	sb.WriteString(fmt.Sprintf("- **Destination:** %s\n", resp.Path))
	sb.WriteString(fmt.Sprintf("- **Filename:** %s\n", resp.Filename))
	sb.WriteString(fmt.Sprintf("- **Size:** %d bytes\n", resp.Size))
	if localPath != "" {
		sb.WriteString(fmt.Sprintf("- **Source:** %s\n", localPath))
	}
	sb.WriteString("\n**Note:** Run `trigger_reindex()` or use `publish()` to make the file searchable.\n")

	return mcp.NewToolResultText(sb.String()), nil
}

func (s *Server) handlePublish(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	destPath, _ := args["dest_path"].(string)
	localPath, _ := args["local_path"].(string)
	content, _ := args["content"].(string)
	waitForReindex, _ := args["wait_for_reindex"].(bool)

	if destPath == "" {
		return mcp.NewToolResultError("dest_path is required"), nil
	}

	var fileContent []byte
	var filename string

	// Prefer local_path over content
	if localPath != "" {
		data, err := os.ReadFile(localPath)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to read local file '%s': %v", localPath, err)), nil
		}
		fileContent = data
		filename = filepath.Base(localPath)
	} else if content != "" {
		fileContent = []byte(content)
		filename = filepath.Base(destPath)
	} else {
		return mcp.NewToolResultError("either local_path or content must be provided"), nil
	}

	var sb strings.Builder
	sb.WriteString("# Publish Results\n\n")

	// Upload file
	uploadResp, err := s.client.UploadFile(destPath, filename, fileContent)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to upload file: %v", err)), nil
	}

	sb.WriteString("## Upload\n\n")
	sb.WriteString(fmt.Sprintf("- **Destination:** %s\n", uploadResp.Path))
	sb.WriteString(fmt.Sprintf("- **Filename:** %s\n", uploadResp.Filename))
	sb.WriteString(fmt.Sprintf("- **Size:** %d bytes\n", uploadResp.Size))
	if localPath != "" {
		sb.WriteString(fmt.Sprintf("- **Source:** %s\n", localPath))
	}

	// Trigger reindex
	reindexResp, err := s.client.TriggerReindex()
	if err != nil {
		sb.WriteString("\n## Reindex\n\n")
		sb.WriteString(fmt.Sprintf("**⚠️ Warning:** Reindex failed: %v\n", err))
		sb.WriteString("File was uploaded but may not be searchable. Try `trigger_reindex()` manually.\n")
		return mcp.NewToolResultText(sb.String()), nil
	}

	sb.WriteString("\n## Reindex\n\n")
	sb.WriteString(fmt.Sprintf("- **Status:** %s\n", reindexResp.Status))

	// Wait for reindex if requested
	if waitForReindex {
		sb.WriteString("- **Waiting:** Polling for completion...\n")

		// Poll for up to 60 seconds
		for i := 0; i < 30; i++ {
			time.Sleep(2 * time.Second)
			status, err := s.client.GetReindexStatus()
			if err != nil {
				sb.WriteString(fmt.Sprintf("- **Warning:** Error checking status: %v\n", err))
				break
			}
			if status.Status == "idle" {
				sb.WriteString(fmt.Sprintf("- **Completed:** Reindex finished in %s\n", status.LastRun.Duration))
				sb.WriteString(fmt.Sprintf("- **Devices:** %d indexed\n", status.LastRun.DevicesIndexed))
				sb.WriteString(fmt.Sprintf("- **Documents:** %d indexed\n", status.LastRun.DocumentsIndexed))
				break
			}
		}
	} else {
		sb.WriteString("- **Note:** Reindex running in background. Use `get_reindex_status()` to check progress.\n")
	}

	sb.WriteString("\n## Next Steps\n\n")
	sb.WriteString(fmt.Sprintf("1. Verify: `search_manuals(query: \"%s\")`\n", filename))
	sb.WriteString("2. Check content: `get_device(device_id: \"...\")` using ID from search\n")

	return mcp.NewToolResultText(sb.String()), nil
}

// BatchFile represents a file in a batch upload
type BatchFile struct {
	LocalPath string `json:"local_path"`
	DestPath  string `json:"dest_path"`
	Content   string `json:"content,omitempty"`
}

func (s *Server) handlePublishBatch(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	filesJSON, _ := args["files"].(string)
	waitForReindex, _ := args["wait_for_reindex"].(bool)

	if filesJSON == "" {
		return mcp.NewToolResultError("files parameter is required (JSON array)"), nil
	}

	// Parse files JSON
	var files []BatchFile
	if err := json.Unmarshal([]byte(filesJSON), &files); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to parse files JSON: %v", err)), nil
	}

	if len(files) == 0 {
		return mcp.NewToolResultError("files array is empty"), nil
	}

	var sb strings.Builder
	sb.WriteString("# Batch Publish Results\n\n")
	sb.WriteString(fmt.Sprintf("**Files to upload:** %d\n\n", len(files)))

	// Upload each file
	sb.WriteString("## Uploads\n\n")
	successCount := 0
	for i, f := range files {
		if f.DestPath == "" {
			sb.WriteString(fmt.Sprintf("%d. **Error:** Missing dest_path\n", i+1))
			continue
		}

		var fileContent []byte
		var filename string

		if f.LocalPath != "" {
			data, err := os.ReadFile(f.LocalPath)
			if err != nil {
				sb.WriteString(fmt.Sprintf("%d. **Error:** %s - failed to read: %v\n", i+1, f.LocalPath, err))
				continue
			}
			fileContent = data
			filename = filepath.Base(f.LocalPath)
		} else if f.Content != "" {
			fileContent = []byte(f.Content)
			filename = filepath.Base(f.DestPath)
		} else {
			sb.WriteString(fmt.Sprintf("%d. **Error:** %s - no local_path or content\n", i+1, f.DestPath))
			continue
		}

		resp, err := s.client.UploadFile(f.DestPath, filename, fileContent)
		if err != nil {
			sb.WriteString(fmt.Sprintf("%d. **Error:** %s - %v\n", i+1, f.DestPath, err))
			continue
		}

		sb.WriteString(fmt.Sprintf("%d. **✓** %s (%d bytes)\n", i+1, resp.Path, resp.Size))
		successCount++
	}

	sb.WriteString(fmt.Sprintf("\n**Uploaded:** %d/%d files\n", successCount, len(files)))

	if successCount == 0 {
		sb.WriteString("\n**⚠️ No files uploaded. Skipping reindex.**\n")
		return mcp.NewToolResultText(sb.String()), nil
	}

	// Trigger single reindex for all uploads
	sb.WriteString("\n## Reindex\n\n")
	reindexResp, err := s.client.TriggerReindex()
	if err != nil {
		sb.WriteString(fmt.Sprintf("**⚠️ Warning:** Reindex failed: %v\n", err))
		return mcp.NewToolResultText(sb.String()), nil
	}

	sb.WriteString(fmt.Sprintf("- **Status:** %s\n", reindexResp.Status))

	// Wait for reindex if requested
	if waitForReindex {
		sb.WriteString("- **Waiting:** Polling for completion...\n")

		for i := 0; i < 30; i++ {
			time.Sleep(2 * time.Second)
			status, err := s.client.GetReindexStatus()
			if err != nil {
				sb.WriteString(fmt.Sprintf("- **Warning:** Error checking status: %v\n", err))
				break
			}
			if status.Status == "idle" {
				sb.WriteString(fmt.Sprintf("- **Completed:** Reindex finished in %s\n", status.LastRun.Duration))
				sb.WriteString(fmt.Sprintf("- **Devices:** %d indexed\n", status.LastRun.DevicesIndexed))
				break
			}
		}
	}

	return mcp.NewToolResultText(sb.String()), nil
}

func (s *Server) handleSyncToGit(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	resp, err := s.client.TriggerSync()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to trigger sync: %v", err)), nil
	}

	var sb strings.Builder
	sb.WriteString("# Git Sync Results\n\n")

	switch resp.Status {
	case "success":
		sb.WriteString("**Status:** ✓ Success\n\n")
		sb.WriteString(fmt.Sprintf("- **Commit:** %s\n", resp.Commit))
		sb.WriteString(fmt.Sprintf("- **Files Changed:** %d\n", resp.FilesChanged))
		sb.WriteString(fmt.Sprintf("- **Branch:** %s\n", resp.Branch))
		sb.WriteString("\nDocumentation changes have been committed and pushed to the remote repository.\n")

	case "no_changes":
		sb.WriteString("**Status:** No Changes\n\n")
		sb.WriteString("No new or modified files to commit. The repository is already up to date.\n")

	case "error":
		sb.WriteString("**Status:** ⚠️ Error\n\n")
		if resp.Error != "" {
			sb.WriteString(fmt.Sprintf("**Error:** %s\n", resp.Error))
		}
		if resp.Message != "" {
			sb.WriteString(fmt.Sprintf("**Message:** %s\n", resp.Message))
		}

	default:
		sb.WriteString(fmt.Sprintf("**Status:** %s\n", resp.Status))
		if resp.Message != "" {
			sb.WriteString(fmt.Sprintf("**Message:** %s\n", resp.Message))
		}
	}

	return mcp.NewToolResultText(sb.String()), nil
}

// Admin tool handlers

func (s *Server) handleListUsers(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	resp, err := s.client.ListUsers()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list users: %v", err)), nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Users (%d)\n\n", resp.Count))
	sb.WriteString("| ID | Name | Role | Active | Created |\n")
	sb.WriteString("|-----|------|------|--------|--------|\n")

	for _, u := range resp.Users {
		sb.WriteString(fmt.Sprintf("| %s | %s | %s | %t | %s |\n",
			u.ID, u.Name, u.Role, u.IsActive, u.CreatedAt))
	}

	return mcp.NewToolResultText(sb.String()), nil
}

func (s *Server) handleCreateUser(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	name, _ := args["name"].(string)
	role, _ := args["role"].(string)

	resp, err := s.client.CreateUser(name, role)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to create user: %v", err)), nil
	}

	var sb strings.Builder
	sb.WriteString("# User Created\n\n")
	sb.WriteString(fmt.Sprintf("- **ID:** %s\n", resp.User.ID))
	sb.WriteString(fmt.Sprintf("- **Name:** %s\n", resp.User.Name))
	sb.WriteString(fmt.Sprintf("- **Role:** %s\n", resp.User.Role))
	sb.WriteString("\n## API Key\n\n")
	sb.WriteString(fmt.Sprintf("```\n%s\n```\n\n", resp.APIKey))
	sb.WriteString("**⚠️ Save this API key now - it will not be shown again!**\n")

	return mcp.NewToolResultText(sb.String()), nil
}

func (s *Server) handleDeleteUser(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	userID, _ := args["user_id"].(string)

	err := s.client.DeleteUser(userID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to delete user: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("# User Deleted\n\nUser `%s` has been deleted.", userID)), nil
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

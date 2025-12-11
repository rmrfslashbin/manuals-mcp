package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/rmrfslashbin/manuals-mcp-server/internal/db"
	"github.com/rmrfslashbin/manuals-mcp-server/pkg/models"
)

// handleGetPinout handles the get_pinout tool.
func (s *Server) handleGetPinout(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Parse arguments
	var args struct {
		Device    string `json:"device"`
		Interface string `json:"interface,omitempty"`
	}
	argsJSON, err := json.Marshal(request.Params.Arguments)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal arguments: %v", err)), nil
	}
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	// Find device
	device, err := s.findDeviceByNameOrID(ctx, args.Device)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Get pinouts
	var pinouts []models.Pinout
	if args.Interface != "" && args.Interface != "all" {
		pinouts, err = db.FindPinoutsByInterface(s.db, device.ID, args.Interface)
	} else {
		pinouts, err = db.GetPinouts(s.db, device.ID)
	}
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get pinouts: %v", err)), nil
	}

	if len(pinouts) == 0 {
		return mcp.NewToolResultText(fmt.Sprintf("No pinouts found for %s", device.Name)), nil
	}

	// Format as table
	var result strings.Builder
	result.WriteString(fmt.Sprintf("# %s Pinout\n\n", device.Name))

	// Add metadata
	if manufacturer, ok := device.Metadata["manufacturer"].(string); ok {
		result.WriteString(fmt.Sprintf("**Manufacturer:** %s\n", manufacturer))
	}
	if category, ok := device.Metadata["category"].(string); ok {
		result.WriteString(fmt.Sprintf("**Category:** %s\n", category))
	}
	result.WriteString("\n")

	// Create ASCII table
	result.WriteString("| Pin | GPIO | Name | Default Pull | Alt Functions |\n")
	result.WriteString("|-----|------|------|--------------|---------------|\n")

	for _, pin := range pinouts {
		gpio := "-"
		if pin.GPIO != nil {
			gpio = fmt.Sprintf("%d", *pin.GPIO)
		}

		defaultPull := "-"
		if pin.DefaultPull != nil {
			defaultPull = *pin.DefaultPull
		}

		altFuncs := "-"
		if len(pin.AltFunctions) > 0 {
			altFuncs = strings.Join(pin.AltFunctions, ", ")
		}

		result.WriteString(fmt.Sprintf("| %d | %s | %s | %s | %s |\n",
			pin.PhysicalPin, gpio, pin.Name, defaultPull, altFuncs))
	}

	return mcp.NewToolResultText(result.String()), nil
}

// handleSearch handles the search tool.
func (s *Server) handleSearch(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Parse arguments
	var args struct {
		Query  string  `json:"query"`
		Domain *string `json:"domain,omitempty"`
		Type   *string `json:"type,omitempty"`
		Limit  *int    `json:"limit,omitempty"`
	}
	argsJSON, err := json.Marshal(request.Params.Arguments)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal arguments: %v", err)), nil
	}
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	// Build search options
	opts := models.SearchOptions{
		Query:  args.Query,
		Limit:  10,
		Offset: 0,
	}

	if args.Limit != nil && *args.Limit > 0 {
		opts.Limit = *args.Limit
	}

	if args.Domain != nil && *args.Domain != "" && *args.Domain != "all" {
		domain := models.Domain(*args.Domain)
		opts.Domain = &domain
	}

	if args.Type != nil {
		opts.Type = args.Type
	}

	// Search
	results, err := db.SearchDevices(s.db, opts)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("search failed: %v", err)), nil
	}

	if len(results) == 0 {
		return mcp.NewToolResultText(fmt.Sprintf("No results found for '%s'", args.Query)), nil
	}

	// Format results
	var output strings.Builder
	output.WriteString(fmt.Sprintf("# Search Results for '%s'\n\n", args.Query))
	output.WriteString(fmt.Sprintf("Found %d result(s)\n\n", len(results)))

	for i, result := range results {
		output.WriteString(fmt.Sprintf("## %d. %s\n", i+1, result.Name))
		output.WriteString(fmt.Sprintf("- **Domain:** %s\n", result.Domain))
		output.WriteString(fmt.Sprintf("- **Type:** %s\n", result.Type))
		output.WriteString(fmt.Sprintf("- **ID:** %s\n", result.ID))

		if manufacturer, ok := result.Metadata["manufacturer"].(string); ok {
			output.WriteString(fmt.Sprintf("- **Manufacturer:** %s\n", manufacturer))
		}
		if category, ok := result.Metadata["category"].(string); ok {
			output.WriteString(fmt.Sprintf("- **Category:** %s\n", category))
		}
		if tags, ok := result.Metadata["tags"].([]interface{}); ok && len(tags) > 0 {
			tagStrs := make([]string, len(tags))
			for j, tag := range tags {
				tagStrs[j] = fmt.Sprint(tag)
			}
			output.WriteString(fmt.Sprintf("- **Tags:** %s\n", strings.Join(tagStrs, ", ")))
		}

		output.WriteString(fmt.Sprintf("- **Resource:** `manuals://device/%s`\n", result.ID))
		output.WriteString("\n")
	}

	return mcp.NewToolResultText(output.String()), nil
}

// handleListHardware handles the list_hardware tool.
func (s *Server) handleListHardware(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Get all hardware devices
	domain := models.DomainHardware
	devices, err := db.ListDevices(s.db, &domain)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list devices: %v", err)), nil
	}

	if len(devices) == 0 {
		return mcp.NewToolResultText("No hardware devices found in the database"), nil
	}

	// Group by category
	categories := make(map[string][]models.Device)
	for _, device := range devices {
		category := "Other"
		if cat, ok := device.Metadata["category"].(string); ok {
			category = cat
		}
		categories[category] = append(categories[category], device)
	}

	// Format output
	var output strings.Builder
	output.WriteString("# Hardware Devices\n\n")
	output.WriteString(fmt.Sprintf("Total: %d devices\n\n", len(devices)))

	// Sort categories and output
	for category, devs := range categories {
		output.WriteString(fmt.Sprintf("## %s (%d)\n\n", category, len(devs)))
		for _, dev := range devs {
			output.WriteString(fmt.Sprintf("- **%s** (`%s`)\n", dev.Name, dev.ID))
			if manufacturer, ok := dev.Metadata["manufacturer"].(string); ok {
				output.WriteString(fmt.Sprintf("  - Manufacturer: %s\n", manufacturer))
			}
			if tags, ok := dev.Metadata["tags"].([]interface{}); ok && len(tags) > 0 {
				tagStrs := make([]string, len(tags))
				for i, tag := range tags {
					tagStrs[i] = fmt.Sprint(tag)
				}
				output.WriteString(fmt.Sprintf("  - Tags: %s\n", strings.Join(tagStrs, ", ")))
			}
		}
		output.WriteString("\n")
	}

	return mcp.NewToolResultText(output.String()), nil
}

// handleGetStats handles the get_stats tool.
func (s *Server) handleGetStats(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	stats, err := db.GetStats(s.db)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get stats: %v", err)), nil
	}

	var output strings.Builder
	output.WriteString("# Documentation Database Statistics\n\n")
	output.WriteString(fmt.Sprintf("- **Total Devices:** %d\n", stats.TotalDevices))
	output.WriteString(fmt.Sprintf("  - Hardware: %d\n", stats.HardwareCount))
	output.WriteString(fmt.Sprintf("  - Software: %d\n", stats.SoftwareCount))
	output.WriteString(fmt.Sprintf("  - Protocols: %d\n", stats.ProtocolCount))
	output.WriteString(fmt.Sprintf("- **Total Pinouts:** %d\n", stats.TotalPinouts))
	output.WriteString(fmt.Sprintf("- **Total Specifications:** %d\n", stats.TotalSpecs))

	return mcp.NewToolResultText(output.String()), nil
}

// handleGetInfo handles the get_info tool.
func (s *Server) handleGetInfo(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Get database stats
	stats, err := db.GetStats(s.db)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get stats: %v", err)), nil
	}

	// Build info output
	var output strings.Builder
	output.WriteString("# Manuals MCP Server Information\n\n")

	// Version information
	output.WriteString("## Version\n\n")
	output.WriteString(fmt.Sprintf("- **Version:** %s\n", s.version))
	output.WriteString(fmt.Sprintf("- **Git Commit:** %s\n", s.gitCommit))
	output.WriteString(fmt.Sprintf("- **Build Time:** %s\n", s.buildTime))
	output.WriteString(fmt.Sprintf("- **Project:** github.com/rmrfslashbin/manuals-mcp-server\n"))
	output.WriteString(fmt.Sprintf("- **License:** MIT\n\n"))

	// Database statistics
	output.WriteString("## Database Statistics\n\n")
	output.WriteString(fmt.Sprintf("- **Total Devices:** %d\n", stats.TotalDevices))
	output.WriteString(fmt.Sprintf("  - Hardware: %d\n", stats.HardwareCount))
	output.WriteString(fmt.Sprintf("  - Software: %d\n", stats.SoftwareCount))
	output.WriteString(fmt.Sprintf("  - Protocols: %d\n", stats.ProtocolCount))
	output.WriteString(fmt.Sprintf("- **Total Pinouts:** %d\n", stats.TotalPinouts))
	output.WriteString(fmt.Sprintf("- **Total Specifications:** %d\n\n", stats.TotalSpecs))

	// Capabilities
	output.WriteString("## MCP Capabilities\n\n")
	output.WriteString("- **Tools:** 9 (search, get_pinout, list_hardware, get_stats, get_info, get_tags, get_categories, get_manufacturers, get_metadata_schema)\n")
	output.WriteString("- **Resources:** 3 templates (device documentation, pinout, guides)\n")
	output.WriteString("- **Prompts:** 4 templates (wiring-guide, pinout-explain, device-compare, protocol-guide)\n")
	output.WriteString("- **Transport:** stdio\n")

	return mcp.NewToolResultText(output.String()), nil
}

// handleGetTags handles the get_tags tool.
func (s *Server) handleGetTags(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tags, err := db.GetAllTags(s.db)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get tags: %v", err)), nil
	}

	var output strings.Builder
	output.WriteString("# Available Tags\n\n")
	output.WriteString(fmt.Sprintf("Total unique tags: %d\n\n", len(tags)))

	for _, tag := range tags {
		output.WriteString(fmt.Sprintf("- `%s`\n", tag))
	}

	return mcp.NewToolResultText(output.String()), nil
}

// handleGetCategories handles the get_categories tool.
func (s *Server) handleGetCategories(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	categories, err := db.GetAllCategories(s.db)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get categories: %v", err)), nil
	}

	var output strings.Builder
	output.WriteString("# Available Categories\n\n")
	output.WriteString(fmt.Sprintf("Total categories: %d\n\n", len(categories)))

	for category, count := range categories {
		output.WriteString(fmt.Sprintf("- **%s** (%d device%s)\n", category, count, pluralize(count)))
	}

	return mcp.NewToolResultText(output.String()), nil
}

// handleGetManufacturers handles the get_manufacturers tool.
func (s *Server) handleGetManufacturers(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	manufacturers, err := db.GetAllManufacturers(s.db)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get manufacturers: %v", err)), nil
	}

	var output strings.Builder
	output.WriteString("# Available Manufacturers\n\n")
	output.WriteString(fmt.Sprintf("Total manufacturers: %d\n\n", len(manufacturers)))

	for manufacturer, count := range manufacturers {
		output.WriteString(fmt.Sprintf("- **%s** (%d device%s)\n", manufacturer, count, pluralize(count)))
	}

	return mcp.NewToolResultText(output.String()), nil
}

// handleGetMetadataSchema handles the get_metadata_schema tool.
func (s *Server) handleGetMetadataSchema(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	schema, err := db.GetMetadataSchema(s.db)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get metadata schema: %v", err)), nil
	}

	var output strings.Builder
	output.WriteString("# Metadata Schema\n\n")
	output.WriteString("Available metadata fields across all devices:\n\n")

	for key, typeInfo := range schema {
		output.WriteString(fmt.Sprintf("- **%s**: %v\n", key, typeInfo))
	}

	return mcp.NewToolResultText(output.String()), nil
}

// pluralize returns "s" if count != 1, empty string otherwise.
func pluralize(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}

package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/rmrfslashbin/manuals-mcp-server/internal/db"
)

// handleDeviceResource handles reading device documentation.
func (s *Server) handleDeviceResource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	// Extract device_id from URI
	uri := request.Params.URI
	parts := strings.Split(uri, "/")
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid resource URI: %s", uri)
	}
	deviceID := parts[len(parts)-1]

	// Get device from database (includes full markdown content)
	device, err := db.GetDevice(s.db, deviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get device: %w", err)
	}
	if device == nil {
		return nil, fmt.Errorf("device not found: %s", deviceID)
	}

	// Use content from database (no file I/O needed)
	content := device.Content
	if content == "" {
		// Fallback: Generate minimal content from metadata if content is empty
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("# %s\n\n", device.Name))
		sb.WriteString(fmt.Sprintf("**Domain:** %s\n", device.Domain))
		sb.WriteString(fmt.Sprintf("**Type:** %s\n\n", device.Type))

		if manufacturer, ok := device.Metadata["manufacturer"].(string); ok {
			sb.WriteString(fmt.Sprintf("**Manufacturer:** %s\n", manufacturer))
		}
		if category, ok := device.Metadata["category"].(string); ok {
			sb.WriteString(fmt.Sprintf("**Category:** %s\n", category))
		}

		// Add metadata as JSON
		metadataJSON, _ := json.MarshalIndent(device.Metadata, "", "  ")
		sb.WriteString(fmt.Sprintf("\n## Metadata\n\n```json\n%s\n```\n", metadataJSON))

		content = sb.String()
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      uri,
			MIMEType: "text/markdown",
			Text:     content,
		},
	}, nil
}

// handlePinoutResource handles reading device pinout information.
func (s *Server) handlePinoutResource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	// Extract device_id from URI: manuals://device/{device_id}/pinout
	uri := request.Params.URI
	parts := strings.Split(uri, "/")
	if len(parts) < 4 {
		return nil, fmt.Errorf("invalid resource URI: %s", uri)
	}
	deviceID := parts[len(parts)-2]

	// Get device
	device, err := db.GetDevice(s.db, deviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get device: %w", err)
	}
	if device == nil {
		return nil, fmt.Errorf("device not found: %s", deviceID)
	}

	// Get pinouts
	pinouts, err := db.GetPinouts(s.db, device.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get pinouts: %w", err)
	}

	// Format as markdown table
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s Pinout\n\n", device.Name))

	if len(pinouts) == 0 {
		sb.WriteString("No pinout information available.\n")
	} else {
		sb.WriteString("| Pin | GPIO | Name | Default Pull | Alt Functions | Description |\n")
		sb.WriteString("|-----|------|------|--------------|---------------|-------------|\n")

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

			desc := "-"
			if pin.Description != nil {
				desc = *pin.Description
			}

			sb.WriteString(fmt.Sprintf("| %d | %s | %s | %s | %s | %s |\n",
				pin.PhysicalPin, gpio, pin.Name, defaultPull, altFuncs, desc))
		}
	}

	content := sb.String()
	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      uri,
			MIMEType: "text/markdown",
			Text:     content,
		},
	}, nil
}


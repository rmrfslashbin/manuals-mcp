package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/rmrfslashbin/manuals-mcp-server/internal/db"
)

// handleWiringGuide handles the wiring-guide prompt.
func (s *Server) handleWiringGuide(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	// Extract arguments
	deviceArg := request.Params.Arguments["device"]
	if deviceArg == "" {
		return nil, fmt.Errorf("device argument is required")
	}

	componentArg := request.Params.Arguments["component"]
	if componentArg == "" {
		return nil, fmt.Errorf("component argument is required")
	}

	interfaceArg := request.Params.Arguments["interface"]

	// Find device
	device, err := s.findDeviceByNameOrID(ctx, deviceArg)
	if err != nil {
		return nil, err
	}

	// Get pinouts
	pinouts, err := db.GetPinouts(s.db, device.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get pinouts: %w", err)
	}

	// Build prompt
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("You are helping a user wire a %s to a %s.\n\n", componentArg, device.Name))

	if interfaceArg != "" {
		sb.WriteString(fmt.Sprintf("The user wants to use the %s interface.\n\n", interfaceArg))
	}

	sb.WriteString("Available GPIO pins:\n\n")
	for _, pin := range pinouts {
		sb.WriteString(fmt.Sprintf("- Pin %d: %s", pin.PhysicalPin, pin.Name))
		if pin.GPIO != nil {
			sb.WriteString(fmt.Sprintf(" (GPIO %d)", *pin.GPIO))
		}
		if len(pin.AltFunctions) > 0 {
			sb.WriteString(fmt.Sprintf(" - Functions: %s", strings.Join(pin.AltFunctions, ", ")))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("\nPlease provide:\n")
	sb.WriteString("1. Recommended GPIO pins to use\n")
	sb.WriteString("2. Wiring diagram/instructions\n")
	sb.WriteString("3. Any important considerations or warnings\n")
	sb.WriteString("4. Example code if applicable\n")

	prompt := sb.String()

	return &mcp.GetPromptResult{
		Description: fmt.Sprintf("Wiring guide for connecting %s to %s", componentArg, device.Name),
		Messages: []mcp.PromptMessage{
			{
				Role: mcp.RoleUser,
				Content: mcp.TextContent{
					Type: "text",
					Text: prompt,
				},
			},
		},
	}, nil
}

// handlePinoutExplain handles the pinout-explain prompt.
func (s *Server) handlePinoutExplain(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	deviceArg := request.Params.Arguments["device"]
	if deviceArg == "" {
		return nil, fmt.Errorf("device argument is required")
	}

	// Find device
	device, err := s.findDeviceByNameOrID(ctx, deviceArg)
	if err != nil {
		return nil, err
	}

	// Get pinouts
	pinouts, err := db.GetPinouts(s.db, device.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get pinouts: %w", err)
	}

	// Build pinout table
	var pinoutTable strings.Builder
	pinoutTable.WriteString("| Pin | GPIO | Name | Functions |\n")
	pinoutTable.WriteString("|-----|------|------|----------|\n")
	for _, pin := range pinouts {
		gpio := "-"
		if pin.GPIO != nil {
			gpio = fmt.Sprintf("%d", *pin.GPIO)
		}
		funcs := "-"
		if len(pin.AltFunctions) > 0 {
			funcs = strings.Join(pin.AltFunctions, ", ")
		}
		pinoutTable.WriteString(fmt.Sprintf("| %d | %s | %s | %s |\n",
			pin.PhysicalPin, gpio, pin.Name, funcs))
	}

	prompt := fmt.Sprintf(`Please explain the pinout configuration for the %s in detail.

%s

Include:
1. Overview of the pin layout
2. Power pins (VCC, GND) and voltage levels
3. Communication interfaces available (I2C, SPI, UART, etc.)
4. Special function pins
5. Best practices for using the GPIO pins
6. Any warnings or considerations

Make the explanation clear and suitable for both beginners and experienced users.`,
		device.Name, pinoutTable.String())

	return &mcp.GetPromptResult{
		Description: fmt.Sprintf("Detailed explanation of %s pinout", device.Name),
		Messages: []mcp.PromptMessage{
			{
				Role: mcp.RoleUser,
				Content: mcp.TextContent{
					Type: "text",
					Text: prompt,
				},
			},
		},
	}, nil
}

// handleDeviceCompare handles the device-compare prompt.
func (s *Server) handleDeviceCompare(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	device1Arg := request.Params.Arguments["device1"]
	if device1Arg == "" {
		return nil, fmt.Errorf("device1 argument is required")
	}

	device2Arg := request.Params.Arguments["device2"]
	if device2Arg == "" {
		return nil, fmt.Errorf("device2 argument is required")
	}

	// Find both devices
	device1, err := s.findDeviceByNameOrID(ctx, device1Arg)
	if err != nil {
		return nil, fmt.Errorf("device1: %w", err)
	}

	device2, err := s.findDeviceByNameOrID(ctx, device2Arg)
	if err != nil {
		return nil, fmt.Errorf("device2: %w", err)
	}

	// Get pinout counts
	pinouts1, _ := db.GetPinouts(s.db, device1.ID)
	pinouts2, _ := db.GetPinouts(s.db, device2.ID)

	prompt := fmt.Sprintf(`Please compare these two devices in detail:

## Device 1: %s
- Type: %s
- GPIO Pins: %d
- Category: %v

## Device 2: %s
- Type: %s
- GPIO Pins: %d
- Category: %v

Provide a comprehensive comparison covering:
1. Key similarities and differences
2. Performance characteristics
3. Features and capabilities
4. Use cases where one is preferred over the other
5. Compatibility considerations
6. Price/availability (if known)
7. Overall recommendation based on different scenarios`,
		device1.Name, device1.Type, len(pinouts1), device1.Metadata["category"],
		device2.Name, device2.Type, len(pinouts2), device2.Metadata["category"])

	return &mcp.GetPromptResult{
		Description: fmt.Sprintf("Comparison of %s vs %s", device1.Name, device2.Name),
		Messages: []mcp.PromptMessage{
			{
				Role: mcp.RoleUser,
				Content: mcp.TextContent{
					Type: "text",
					Text: prompt,
				},
			},
		},
	}, nil
}

// handleProtocolGuide handles the protocol-guide prompt.
func (s *Server) handleProtocolGuide(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	protocolArg := request.Params.Arguments["protocol"]
	if protocolArg == "" {
		return nil, fmt.Errorf("protocol argument is required")
	}

	platformArg := request.Params.Arguments["platform"]

	var prompt string
	if platformArg != "" {
		prompt = fmt.Sprintf(`Please provide a comprehensive implementation guide for the %s protocol on %s.

Include:
1. Protocol overview and key concepts
2. Hardware requirements and pin connections
3. Initialization and configuration
4. Common operations (read, write, etc.)
5. Code examples with explanations
6. Error handling and debugging tips
7. Best practices and gotchas
8. Links to additional resources

Make this suitable for developers implementing %s communication.`,
			protocolArg, platformArg, protocolArg)
	} else {
		prompt = fmt.Sprintf(`Please provide a comprehensive guide for the %s protocol.

Include:
1. Protocol overview and history
2. How it works (technical details)
3. Common use cases and applications
4. Hardware requirements
5. Advantages and disadvantages
6. Comparison with alternative protocols
7. Implementation considerations
8. Resources for learning more

Make this clear and accessible.`, protocolArg)
	}

	return &mcp.GetPromptResult{
		Description: fmt.Sprintf("%s protocol implementation guide", protocolArg),
		Messages: []mcp.PromptMessage{
			{
				Role: mcp.RoleUser,
				Content: mcp.TextContent{
					Type: "text",
					Text: prompt,
				},
			},
		},
	}, nil
}

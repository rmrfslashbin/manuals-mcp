// Package indexer provides documentation parsing and indexing functionality.
package indexer

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/rmrfslashbin/manuals-mcp-server/pkg/models"
	"gopkg.in/yaml.v3"
)

// Frontmatter holds the YAML frontmatter from markdown files.
type Frontmatter struct {
	Manufacturer    string                 `yaml:"manufacturer"`
	Model           string                 `yaml:"model"`
	Category        string                 `yaml:"category"`
	Version         string                 `yaml:"version"`
	Date            string                 `yaml:"date"`
	Tags            []string               `yaml:"tags"`
	Datasheets      []string               `yaml:"datasheets"`
	RelatedHardware []string               `yaml:"related_hardware"`
	Specs           map[string]interface{} `yaml:"specs"`
}

// ParsedDocument represents a parsed markdown document.
type ParsedDocument struct {
	Metadata map[string]interface{}
	Content  string
	Pinouts  []models.Pinout
	Domain   models.Domain
	Type     string
	ID       string
}

// ParseMarkdownFile parses a markdown file and extracts metadata, content, and pinouts.
func ParseMarkdownFile(filePath string) (*ParsedDocument, error) {
	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	content := string(data)

	// Extract frontmatter
	frontmatter, body, err := extractFrontmatter(content)
	if err != nil {
		return nil, fmt.Errorf("failed to extract frontmatter: %w", err)
	}

	// Build metadata
	metadata := buildMetadata(frontmatter)

	// Determine domain and type
	domain := getDomainFromCategory(frontmatter.Category)
	deviceType := getTypeFromCategory(frontmatter.Category)

	// Generate ID
	id := generateDeviceID(frontmatter.Category, frontmatter.Model)

	// Extract pinouts (only for hardware)
	var pinouts []models.Pinout
	if domain == models.DomainHardware {
		pinouts = extractPinouts(body)
	}

	return &ParsedDocument{
		Metadata: metadata,
		Content:  body,
		Pinouts:  pinouts,
		Domain:   domain,
		Type:     deviceType,
		ID:       id,
	}, nil
}

// extractFrontmatter extracts YAML frontmatter from markdown content.
func extractFrontmatter(content string) (*Frontmatter, string, error) {
	// Look for YAML frontmatter: ---\n...\n---
	if !strings.HasPrefix(content, "---\n") {
		return &Frontmatter{}, content, nil
	}

	// Find end of frontmatter
	endIndex := strings.Index(content[4:], "\n---")
	if endIndex == -1 {
		return &Frontmatter{}, content, nil
	}

	// Extract frontmatter YAML
	yamlContent := content[4 : endIndex+4]
	body := strings.TrimSpace(content[endIndex+8:])

	// Parse YAML
	var fm Frontmatter
	if err := yaml.Unmarshal([]byte(yamlContent), &fm); err != nil {
		return nil, "", fmt.Errorf("failed to parse YAML frontmatter: %w", err)
	}

	return &fm, body, nil
}

// buildMetadata converts frontmatter to metadata map.
func buildMetadata(fm *Frontmatter) map[string]interface{} {
	metadata := make(map[string]interface{})

	if fm.Manufacturer != "" {
		metadata["manufacturer"] = fm.Manufacturer
	}
	if fm.Model != "" {
		metadata["model"] = fm.Model
	}
	if fm.Category != "" {
		metadata["category"] = fm.Category
	}
	if fm.Version != "" {
		metadata["version"] = fm.Version
	}
	if fm.Date != "" {
		metadata["date"] = fm.Date
	}
	if len(fm.Tags) > 0 {
		metadata["tags"] = fm.Tags
	}
	if len(fm.Datasheets) > 0 {
		metadata["datasheets"] = fm.Datasheets
	}
	if len(fm.RelatedHardware) > 0 {
		metadata["related_hardware"] = fm.RelatedHardware
	}
	if len(fm.Specs) > 0 {
		metadata["specs"] = fm.Specs
	}

	return metadata
}

// extractPinouts extracts pinout information from markdown tables.
func extractPinouts(markdown string) []models.Pinout {
	var pinouts []models.Pinout

	// Look for tables with "Physical Pin" or "Pin" header
	// Pattern: | Physical Pin | GPIO | Name | ... |
	scanner := bufio.NewScanner(strings.NewReader(markdown))
	var inTable bool
	var headerFound bool

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Check for table start (header with Physical Pin or just Pin)
		if !inTable && strings.Contains(trimmed, "|") {
			lowerLine := strings.ToLower(trimmed)
			if strings.Contains(lowerLine, "physical") && strings.Contains(lowerLine, "pin") {
				headerFound = true
				continue
			}
			if strings.Contains(lowerLine, "| pin |") || strings.HasPrefix(lowerLine, "| pin") {
				headerFound = true
				continue
			}
		}

		// Check for table separator
		if headerFound && strings.Contains(trimmed, "|") && strings.Contains(trimmed, "-") {
			inTable = true
			continue
		}

		// Parse table rows
		if inTable && strings.Contains(trimmed, "|") {
			// Empty line ends table
			if trimmed == "" {
				inTable = false
				headerFound = false
				continue
			}

			// Parse row
			if pin := parsePinoutRow(trimmed); pin != nil {
				pinouts = append(pinouts, *pin)
			}
		}

		// Non-table line ends table
		if inTable && !strings.Contains(trimmed, "|") {
			inTable = false
			headerFound = false
		}
	}

	return pinouts
}

// parsePinoutRow parses a single row from a pinout table.
func parsePinoutRow(row string) *models.Pinout {
	// Split by |
	cells := strings.Split(row, "|")
	var cleanCells []string
	for _, cell := range cells {
		trimmed := strings.TrimSpace(cell)
		if trimmed != "" {
			cleanCells = append(cleanCells, trimmed)
		}
	}

	if len(cleanCells) < 3 {
		return nil
	}

	// Parse physical pin (first column)
	physicalPin, err := strconv.Atoi(cleanCells[0])
	if err != nil {
		return nil
	}

	// Parse GPIO (second column)
	var gpio *int
	gpioStr := cleanCells[1]
	if gpioStr != "-" && gpioStr != "" {
		if gpioNum, err := strconv.Atoi(gpioStr); err == nil {
			gpio = &gpioNum
		}
	}

	// Parse name (third column)
	name := cleanCells[2]
	if name == "-" {
		name = ""
	}

	// Parse default pull (fourth column, optional)
	var defaultPull *string
	if len(cleanCells) > 3 {
		pullStr := strings.ToLower(cleanCells[3])
		if strings.Contains(pullStr, "high") {
			s := "high"
			defaultPull = &s
		} else if strings.Contains(pullStr, "low") {
			s := "low"
			defaultPull = &s
		} else if strings.Contains(pullStr, "none") {
			s := "none"
			defaultPull = &s
		}
	}

	// Parse alt functions (fifth column, optional)
	var altFunctions []string
	if len(cleanCells) > 4 {
		altStr := cleanCells[4]
		if altStr != "-" && altStr != "" {
			// Split by comma and clean
			funcs := strings.Split(altStr, ",")
			for _, f := range funcs {
				trimmed := strings.TrimSpace(f)
				if trimmed != "" {
					altFunctions = append(altFunctions, trimmed)
				}
			}
		}
	}

	// Parse description (sixth column, optional)
	var description *string
	if len(cleanCells) > 5 {
		desc := cleanCells[5]
		if desc != "-" && desc != "" {
			description = &desc
		}
	}

	return &models.Pinout{
		PhysicalPin:  physicalPin,
		GPIO:         gpio,
		Name:         name,
		DefaultPull:  defaultPull,
		AltFunctions: altFunctions,
		Description:  description,
	}
}

// getDomainFromCategory determines the domain from category.
func getDomainFromCategory(category string) models.Domain {
	lowerCat := strings.ToLower(category)

	// Software: libraries, frameworks, applications
	if strings.HasPrefix(lowerCat, "libraries/") ||
		strings.HasPrefix(lowerCat, "frameworks/") ||
		strings.HasPrefix(lowerCat, "applications/") ||
		strings.HasPrefix(lowerCat, "software/") {
		return models.DomainSoftware
	}

	// Protocol: communication protocols
	if strings.HasPrefix(lowerCat, "protocols/") ||
		strings.HasPrefix(lowerCat, "protocol/") {
		return models.DomainProtocol
	}

	// Default to hardware
	return models.DomainHardware
}

// getTypeFromCategory extracts the type from category.
func getTypeFromCategory(category string) string {
	parts := strings.Split(category, "/")
	if len(parts) == 0 {
		return "unknown"
	}
	return parts[len(parts)-1]
}

// generateDeviceID generates a unique ID for a device.
func generateDeviceID(category, model string) string {
	// Normalize category: replace / with -
	catNorm := strings.ReplaceAll(category, "/", "-")
	catNorm = regexp.MustCompile(`[^a-z0-9-]+`).ReplaceAllString(strings.ToLower(catNorm), "-")

	// Normalize model: lowercase, replace non-alphanumeric with -
	modelNorm := regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(strings.ToLower(model), "-")
	modelNorm = strings.Trim(modelNorm, "-")

	return fmt.Sprintf("%s-%s", catNorm, modelNorm)
}

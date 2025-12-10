// Package models defines the data structures used throughout the application.
package models

import "time"

// Domain represents the documentation domain type.
type Domain string

const (
	// DomainHardware represents hardware devices.
	DomainHardware Domain = "hardware"
	// DomainSoftware represents software libraries and frameworks.
	DomainSoftware Domain = "software"
	// DomainProtocol represents communication protocols.
	DomainProtocol Domain = "protocol"
)

// Device represents a hardware device, software library, or protocol.
type Device struct {
	ID         string                 `json:"id"`
	Domain     Domain                 `json:"domain"`
	Type       string                 `json:"type"`
	Name       string                 `json:"name"`
	Path       string                 `json:"path"`
	Metadata   map[string]interface{} `json:"metadata"`
	IndexedAt  time.Time              `json:"indexed_at"`
	Content    string                 `json:"content,omitempty"` // For indexing, not stored in DB
	Pinouts    []Pinout               `json:"pinouts,omitempty"` // For indexing, not stored directly
}

// Pinout represents a GPIO pin configuration.
type Pinout struct {
	PhysicalPin  int      `json:"physical_pin"`
	GPIO         *int     `json:"gpio,omitempty"`
	Name         string   `json:"name"`
	DefaultPull  *string  `json:"default_pull,omitempty"` // "high", "low", "none"
	AltFunctions []string `json:"alt_functions,omitempty"`
	Description  *string  `json:"description,omitempty"`
}

// SearchOptions configures search behavior.
type SearchOptions struct {
	Query  string
	Domain *Domain
	Type   *string
	Tags   []string
	Limit  int
	Offset int
}

// SearchResult represents a search result with relevance scoring.
type SearchResult struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Domain    Domain                 `json:"domain"`
	Type      string                 `json:"type"`
	Path      string                 `json:"path"`
	Relevance float64                `json:"relevance"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// DatabaseStats provides statistics about the indexed database.
type DatabaseStats struct {
	TotalDevices   int `json:"total_devices"`
	HardwareCount  int `json:"hardware_count"`
	SoftwareCount  int `json:"software_count"`
	ProtocolCount  int `json:"protocol_count"`
	TotalPinouts   int `json:"total_pinouts"`
	TotalSpecs     int `json:"total_specs"`
}

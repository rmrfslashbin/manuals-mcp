package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/rmrfslashbin/manuals-mcp-server/pkg/models"
)

// InsertDevice inserts or replaces a device in the database.
func InsertDevice(db *sql.DB, device *models.Device) error {
	// Marshal metadata to JSON
	metadataJSON, err := json.Marshal(device.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Insert device
	_, err = db.Exec(`
		INSERT OR REPLACE INTO devices (id, domain, type, name, path, content, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, device.ID, device.Domain, device.Type, device.Name, device.Path, device.Content, metadataJSON)
	if err != nil {
		return fmt.Errorf("failed to insert device: %w", err)
	}

	// Insert into FTS
	tags := ""
	if tagsVal, ok := device.Metadata["tags"].([]interface{}); ok {
		tagStrs := make([]string, len(tagsVal))
		for i, t := range tagsVal {
			tagStrs[i] = fmt.Sprint(t)
		}
		tags = strings.Join(tagStrs, " ")
	}

	_, err = db.Exec(`
		INSERT OR REPLACE INTO search_fts (device_id, name, content, tags)
		VALUES (?, ?, ?, ?)
	`, device.ID, device.Name, device.Content, tags)
	if err != nil {
		return fmt.Errorf("failed to insert into FTS: %w", err)
	}

	return nil
}

// InsertPinouts inserts pinouts for a device.
func InsertPinouts(db *sql.DB, deviceID string, pinouts []models.Pinout) error {
	stmt, err := db.Prepare(`
		INSERT OR REPLACE INTO pinouts
		(device_id, physical_pin, gpio_num, name, default_pull, alt_functions, description)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, pin := range pinouts {
		var altFuncsJSON []byte
		if len(pin.AltFunctions) > 0 {
			altFuncsJSON, err = json.Marshal(pin.AltFunctions)
			if err != nil {
				return fmt.Errorf("failed to marshal alt functions: %w", err)
			}
		}

		_, err = stmt.Exec(
			deviceID,
			pin.PhysicalPin,
			pin.GPIO,
			pin.Name,
			pin.DefaultPull,
			altFuncsJSON,
			pin.Description,
		)
		if err != nil {
			return fmt.Errorf("failed to insert pinout: %w", err)
		}
	}

	return nil
}

// SearchDevices performs a full-text search on devices.
func SearchDevices(db *sql.DB, opts models.SearchOptions) ([]models.SearchResult, error) {
	// Smart FTS5 query building:
	// - If query contains hyphens (like "raspberry-pi-4"), quote it as exact phrase
	// - Otherwise, allow multi-word boolean search (like "analog sensor")
	ftsQuery := buildFTSQuery(opts.Query)

	query := `
		SELECT
			d.id,
			d.name,
			d.domain,
			d.type,
			d.path,
			d.metadata,
			fts.rank as relevance
		FROM search_fts fts
		JOIN devices d ON d.id = fts.device_id
		WHERE search_fts MATCH ?
	`
	args := []interface{}{ftsQuery}

	if opts.Domain != nil {
		query += " AND d.domain = ?"
		args = append(args, *opts.Domain)
	}

	if opts.Type != nil {
		query += " AND d.type = ?"
		args = append(args, *opts.Type)
	}

	query += " ORDER BY fts.rank LIMIT ? OFFSET ?"
	args = append(args, opts.Limit, opts.Offset)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search devices: %w", err)
	}
	defer rows.Close()

	var results []models.SearchResult
	for rows.Next() {
		var result models.SearchResult
		var metadataJSON string

		err := rows.Scan(
			&result.ID,
			&result.Name,
			&result.Domain,
			&result.Type,
			&result.Path,
			&metadataJSON,
			&result.Relevance,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		if err := json.Unmarshal([]byte(metadataJSON), &result.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}

		results = append(results, result)
	}

	return results, rows.Err()
}

// GetDevice retrieves a device by ID.
func GetDevice(db *sql.DB, deviceID string) (*models.Device, error) {
	var device models.Device
	var metadataJSON string
	var indexedAtUnix int64

	err := db.QueryRow(`
		SELECT id, domain, type, name, path, content, metadata, indexed_at
		FROM devices
		WHERE id = ?
	`, deviceID).Scan(
		&device.ID,
		&device.Domain,
		&device.Type,
		&device.Name,
		&device.Path,
		&device.Content,
		&metadataJSON,
		&indexedAtUnix,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get device: %w", err)
	}

	if err := json.Unmarshal([]byte(metadataJSON), &device.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	// Convert Unix timestamp to time.Time
	device.IndexedAt = time.Unix(indexedAtUnix, 0)

	return &device, nil
}

// GetPinouts retrieves all pinouts for a device.
func GetPinouts(db *sql.DB, deviceID string) ([]models.Pinout, error) {
	rows, err := db.Query(`
		SELECT physical_pin, gpio_num, name, default_pull, alt_functions, description
		FROM pinouts
		WHERE device_id = ?
		ORDER BY physical_pin
	`, deviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to query pinouts: %w", err)
	}
	defer rows.Close()

	var pinouts []models.Pinout
	for rows.Next() {
		var pin models.Pinout
		var altFuncsJSON []byte

		err := rows.Scan(
			&pin.PhysicalPin,
			&pin.GPIO,
			&pin.Name,
			&pin.DefaultPull,
			&altFuncsJSON,
			&pin.Description,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan pinout: %w", err)
		}

		if len(altFuncsJSON) > 0 {
			if err := json.Unmarshal(altFuncsJSON, &pin.AltFunctions); err != nil {
				return nil, fmt.Errorf("failed to unmarshal alt functions: %w", err)
			}
		}

		pinouts = append(pinouts, pin)
	}

	return pinouts, rows.Err()
}

// buildFTSQuery constructs an FTS5 query string with smart quoting.
// Quotes terms with hyphens (device models) but allows boolean search for regular words.
func buildFTSQuery(query string) string {
	query = strings.TrimSpace(query)
	if query == "" {
		return ""
	}

	// Check if query contains hyphens between word characters (likely a model number)
	// Examples: "raspberry-pi-4", "esp32-s3", "sen0244" (no hyphens)
	hasModelHyphens := strings.Contains(query, "-") &&
		len(strings.FieldsFunc(query, func(r rune) bool { return r == '-' })) > 1

	// If it looks like a model number with hyphens, quote the whole thing
	if hasModelHyphens {
		// Escape any existing quotes
		escaped := strings.ReplaceAll(query, `"`, `""`)
		return `"` + escaped + `"`
	}

	// For multi-word queries without hyphens, use implicit AND (FTS5 default)
	// This allows "analog sensor" to match documents with both words
	// Escape any quotes in individual words
	words := strings.Fields(query)
	for i, word := range words {
		// If individual word contains hyphen, quote it
		if strings.Contains(word, "-") {
			words[i] = `"` + strings.ReplaceAll(word, `"`, `""`) + `"`
		}
	}

	return strings.Join(words, " ")
}

// ListDevices retrieves all devices, optionally filtered by domain.
func ListDevices(db *sql.DB, domain *models.Domain) ([]models.Device, error) {
	query := "SELECT id, domain, type, name, path, content, metadata, indexed_at FROM devices"
	args := []interface{}{}

	if domain != nil {
		query += " WHERE domain = ?"
		args = append(args, *domain)
	}

	query += " ORDER BY name"

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list devices: %w", err)
	}
	defer rows.Close()

	var devices []models.Device
	for rows.Next() {
		var device models.Device
		var metadataJSON string
		var indexedAtUnix int64

		err := rows.Scan(
			&device.ID,
			&device.Domain,
			&device.Type,
			&device.Name,
			&device.Path,
			&device.Content,
			&metadataJSON,
			&indexedAtUnix,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan device: %w", err)
		}

		if err := json.Unmarshal([]byte(metadataJSON), &device.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}

		// Convert Unix timestamp to time.Time
		device.IndexedAt = time.Unix(indexedAtUnix, 0)

		devices = append(devices, device)
	}

	return devices, rows.Err()
}

// GetAllTags retrieves all unique tags from all devices.
func GetAllTags(db *sql.DB) ([]string, error) {
	rows, err := db.Query(`
		SELECT DISTINCT json_each.value as tag
		FROM devices, json_each(json_extract(metadata, '$.tags'))
		WHERE json_extract(metadata, '$.tags') IS NOT NULL
		ORDER BY tag
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get tags: %w", err)
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, fmt.Errorf("failed to scan tag: %w", err)
		}
		tags = append(tags, tag)
	}

	return tags, rows.Err()
}

// GetAllCategories retrieves all unique categories with device counts.
func GetAllCategories(db *sql.DB) (map[string]int, error) {
	rows, err := db.Query(`
		SELECT json_extract(metadata, '$.category') as category, COUNT(*) as count
		FROM devices
		WHERE json_extract(metadata, '$.category') IS NOT NULL
		GROUP BY category
		ORDER BY category
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get categories: %w", err)
	}
	defer rows.Close()

	categories := make(map[string]int)
	for rows.Next() {
		var category string
		var count int
		if err := rows.Scan(&category, &count); err != nil {
			return nil, fmt.Errorf("failed to scan category: %w", err)
		}
		categories[category] = count
	}

	return categories, rows.Err()
}

// GetAllManufacturers retrieves all unique manufacturers with device counts.
func GetAllManufacturers(db *sql.DB) (map[string]int, error) {
	rows, err := db.Query(`
		SELECT json_extract(metadata, '$.manufacturer') as manufacturer, COUNT(*) as count
		FROM devices
		WHERE json_extract(metadata, '$.manufacturer') IS NOT NULL
		GROUP BY manufacturer
		ORDER BY manufacturer
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get manufacturers: %w", err)
	}
	defer rows.Close()

	manufacturers := make(map[string]int)
	for rows.Next() {
		var manufacturer string
		var count int
		if err := rows.Scan(&manufacturer, &count); err != nil {
			return nil, fmt.Errorf("failed to scan manufacturer: %w", err)
		}
		manufacturers[manufacturer] = count
	}

	return manufacturers, rows.Err()
}

// GetMetadataSchema retrieves all unique metadata keys across all devices.
func GetMetadataSchema(db *sql.DB) (map[string]interface{}, error) {
	// Get all metadata JSON documents
	rows, err := db.Query("SELECT metadata FROM devices")
	if err != nil {
		return nil, fmt.Errorf("failed to query metadata: %w", err)
	}
	defer rows.Close()

	// Collect all unique keys
	schema := make(map[string]map[string]bool) // key -> set of types
	for rows.Next() {
		var metadataJSON string
		if err := rows.Scan(&metadataJSON); err != nil {
			return nil, fmt.Errorf("failed to scan metadata: %w", err)
		}

		var metadata map[string]interface{}
		if err := json.Unmarshal([]byte(metadataJSON), &metadata); err != nil {
			continue // Skip invalid JSON
		}

		for key, value := range metadata {
			if schema[key] == nil {
				schema[key] = make(map[string]bool)
			}
			schema[key][fmt.Sprintf("%T", value)] = true
		}
	}

	// Convert to readable format
	result := make(map[string]interface{})
	for key, types := range schema {
		typeList := make([]string, 0, len(types))
		for t := range types {
			typeList = append(typeList, t)
		}
		if len(typeList) == 1 {
			result[key] = typeList[0]
		} else {
			result[key] = typeList
		}
	}

	return result, rows.Err()
}

// GetStats retrieves database statistics.
func GetStats(db *sql.DB) (*models.DatabaseStats, error) {
	var stats models.DatabaseStats

	err := db.QueryRow(`
		SELECT
			(SELECT COUNT(*) FROM devices) as total_devices,
			(SELECT COUNT(*) FROM devices WHERE domain = 'hardware') as hardware_count,
			(SELECT COUNT(*) FROM devices WHERE domain = 'software') as software_count,
			(SELECT COUNT(*) FROM devices WHERE domain = 'protocol') as protocol_count,
			(SELECT COUNT(*) FROM pinouts) as total_pinouts,
			(SELECT COUNT(*) FROM specifications) as total_specs
	`).Scan(
		&stats.TotalDevices,
		&stats.HardwareCount,
		&stats.SoftwareCount,
		&stats.ProtocolCount,
		&stats.TotalPinouts,
		&stats.TotalSpecs,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	return &stats, nil
}

// FindPinoutsByInterface finds pins that match an interface type (I2C, SPI, etc.).
func FindPinoutsByInterface(db *sql.DB, deviceID, interfaceType string) ([]models.Pinout, error) {
	pattern := "%" + interfaceType + "%"

	rows, err := db.Query(`
		SELECT physical_pin, gpio_num, name, default_pull, alt_functions, description
		FROM pinouts
		WHERE device_id = ?
		AND (LOWER(name) LIKE LOWER(?) OR LOWER(alt_functions) LIKE LOWER(?))
		ORDER BY physical_pin
	`, deviceID, pattern, pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to query pinouts by interface: %w", err)
	}
	defer rows.Close()

	var pinouts []models.Pinout
	for rows.Next() {
		var pin models.Pinout
		var altFuncsJSON []byte

		err := rows.Scan(
			&pin.PhysicalPin,
			&pin.GPIO,
			&pin.Name,
			&pin.DefaultPull,
			&altFuncsJSON,
			&pin.Description,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan pinout: %w", err)
		}

		if len(altFuncsJSON) > 0 {
			if err := json.Unmarshal(altFuncsJSON, &pin.AltFunctions); err != nil {
				return nil, fmt.Errorf("failed to unmarshal alt functions: %w", err)
			}
		}

		pinouts = append(pinouts, pin)
	}

	return pinouts, rows.Err()
}

// InsertGuide inserts or replaces a guide in the database.
func InsertGuide(db *sql.DB, id, title, content string) error {
	_, err := db.Exec(`
		INSERT OR REPLACE INTO guides (id, title, content)
		VALUES (?, ?, ?)
	`, id, title, content)
	if err != nil {
		return fmt.Errorf("failed to insert guide: %w", err)
	}
	return nil
}

// GetGuide retrieves a guide by ID.
func GetGuide(db *sql.DB, id string) (title, content string, err error) {
	err = db.QueryRow(`
		SELECT title, content
		FROM guides
		WHERE id = ?
	`, id).Scan(&title, &content)

	if err == sql.ErrNoRows {
		return "", "", fmt.Errorf("guide not found: %s", id)
	}
	if err != nil {
		return "", "", fmt.Errorf("failed to get guide: %w", err)
	}

	return title, content, nil
}

// ListGuides retrieves all guides.
func ListGuides(db *sql.DB) (map[string]string, error) {
	rows, err := db.Query(`
		SELECT id, title
		FROM guides
		ORDER BY id
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to list guides: %w", err)
	}
	defer rows.Close()

	guides := make(map[string]string)
	for rows.Next() {
		var id, title string
		if err := rows.Scan(&id, &title); err != nil {
			return nil, fmt.Errorf("failed to scan guide: %w", err)
		}
		guides[id] = title
	}

	return guides, rows.Err()
}

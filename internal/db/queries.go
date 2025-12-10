package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

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
		INSERT OR REPLACE INTO devices (id, domain, type, name, path, metadata)
		VALUES (?, ?, ?, ?, ?, ?)
	`, device.ID, device.Domain, device.Type, device.Name, device.Path, metadataJSON)
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
	args := []interface{}{opts.Query}

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

	err := db.QueryRow(`
		SELECT id, domain, type, name, path, metadata, indexed_at
		FROM devices
		WHERE id = ?
	`, deviceID).Scan(
		&device.ID,
		&device.Domain,
		&device.Type,
		&device.Name,
		&device.Path,
		&metadataJSON,
		&device.IndexedAt,
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

// ListDevices retrieves all devices, optionally filtered by domain.
func ListDevices(db *sql.DB, domain *models.Domain) ([]models.Device, error) {
	query := "SELECT id, domain, type, name, path, metadata, indexed_at FROM devices"
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

		err := rows.Scan(
			&device.ID,
			&device.Domain,
			&device.Type,
			&device.Name,
			&device.Path,
			&metadataJSON,
			&device.IndexedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan device: %w", err)
		}

		if err := json.Unmarshal([]byte(metadataJSON), &device.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}

		devices = append(devices, device)
	}

	return devices, rows.Err()
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

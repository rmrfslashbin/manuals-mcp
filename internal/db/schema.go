// Package db provides database schema and query operations.
package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite" // SQLite driver
)

const schema = `
-- Devices/Artifacts table
CREATE TABLE IF NOT EXISTS devices (
	id TEXT PRIMARY KEY,
	domain TEXT NOT NULL CHECK(domain IN ('hardware', 'software', 'protocol')),
	type TEXT NOT NULL,
	name TEXT NOT NULL,
	path TEXT NOT NULL,
	metadata TEXT NOT NULL, -- JSON
	indexed_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now'))
);

CREATE INDEX IF NOT EXISTS idx_devices_domain ON devices(domain);
CREATE INDEX IF NOT EXISTS idx_devices_type ON devices(type);
CREATE INDEX IF NOT EXISTS idx_devices_name ON devices(name);

-- Pinouts table
CREATE TABLE IF NOT EXISTS pinouts (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	device_id TEXT NOT NULL,
	physical_pin INTEGER NOT NULL,
	gpio_num INTEGER,
	name TEXT NOT NULL,
	default_pull TEXT CHECK(default_pull IN ('high', 'low', 'none', NULL)),
	alt_functions TEXT, -- JSON array
	description TEXT,
	FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE,
	UNIQUE(device_id, physical_pin)
);

CREATE INDEX IF NOT EXISTS idx_pinouts_device ON pinouts(device_id);
CREATE INDEX IF NOT EXISTS idx_pinouts_name ON pinouts(name);
CREATE INDEX IF NOT EXISTS idx_pinouts_gpio ON pinouts(gpio_num);

-- Specifications table
CREATE TABLE IF NOT EXISTS specifications (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	device_id TEXT NOT NULL,
	key TEXT NOT NULL,
	value TEXT NOT NULL,
	unit TEXT,
	FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE,
	UNIQUE(device_id, key)
);

CREATE INDEX IF NOT EXISTS idx_specs_device ON specifications(device_id);
CREATE INDEX IF NOT EXISTS idx_specs_key ON specifications(key);

-- Cross-references table
CREATE TABLE IF NOT EXISTS cross_references (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	from_device_id TEXT NOT NULL,
	to_device_id TEXT NOT NULL,
	relationship TEXT NOT NULL, -- 'requires', 'compatible', 'implements', 'example'
	FOREIGN KEY (from_device_id) REFERENCES devices(id) ON DELETE CASCADE,
	FOREIGN KEY (to_device_id) REFERENCES devices(id) ON DELETE CASCADE,
	UNIQUE(from_device_id, to_device_id, relationship)
);

CREATE INDEX IF NOT EXISTS idx_xref_from ON cross_references(from_device_id);
CREATE INDEX IF NOT EXISTS idx_xref_to ON cross_references(to_device_id);

-- Full-text search (FTS5)
CREATE VIRTUAL TABLE IF NOT EXISTS search_fts USING fts5(
	device_id UNINDEXED,
	name,
	content,
	tags,
	tokenize='porter unicode61'
);
`

// InitDatabase creates and initializes a SQLite database with the schema.
// It creates the directory structure if it doesn't exist.
func InitDatabase(dbPath string) (*sql.DB, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open database
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// Create schema
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	return db, nil
}

// ClearDatabase removes all data from the database.
func ClearDatabase(db *sql.DB) error {
	statements := []string{
		"DELETE FROM search_fts",
		"DELETE FROM cross_references",
		"DELETE FROM specifications",
		"DELETE FROM pinouts",
		"DELETE FROM devices",
	}

	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("failed to clear table: %w", err)
		}
	}

	return nil
}

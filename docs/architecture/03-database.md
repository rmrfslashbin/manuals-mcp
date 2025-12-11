# Database Schema

## Overview

The manuals platform uses SQLite with FTS5 (Full-Text Search) for documentation storage and retrieval. The schema is designed for read-heavy workloads with efficient full-text search capabilities.

## Database File

```
Location: /data/manuals.db
Size: ~15-20MB (typical)
Engine: SQLite 3.x with FTS5 extension
```

## Schema Design Principles

1. **Normalized**: Devices, pinouts, and documents in separate tables
2. **FTS5 Integration**: Separate FTS virtual table for full-text search
3. **Metadata First**: Core data in database, large binaries on filesystem
4. **Timestamps**: Track creation and modification times
5. **JSON Metadata**: Flexible metadata field for device-specific attributes

## Tables

### devices
Core device information and documentation content.

```sql
CREATE TABLE devices (
    id TEXT PRIMARY KEY,                      -- Unique device ID (e.g., 'ds18b20')
    name TEXT NOT NULL,                       -- Display name
    domain TEXT NOT NULL,                     -- hardware, software, protocol
    type TEXT,                                -- sensors, dev-boards, etc.
    category TEXT,                            -- Human-readable category
    manufacturer TEXT,                        -- Device manufacturer
    content TEXT NOT NULL,                    -- Markdown documentation
    metadata TEXT,                            -- JSON: {datasheet_url, tags, etc.}
    indexed_at INTEGER NOT NULL               -- Unix timestamp
);

CREATE INDEX idx_devices_domain ON devices(domain);
CREATE INDEX idx_devices_type ON devices(type);
CREATE INDEX idx_devices_category ON devices(category);
CREATE INDEX idx_devices_manufacturer ON devices(manufacturer);
```

**Example row:**
```json
{
  "id": "ds18b20",
  "name": "DS18B20 Digital Thermometer",
  "domain": "hardware",
  "type": "sensors",
  "category": "Temperature Sensors",
  "manufacturer": "Maxim Integrated",
  "content": "# DS18B20\n\n**Type:** Temperature Sensor...",
  "metadata": "{\"datasheet_url\":\"https://...\",\"tags\":[\"1-wire\",\"temperature\"]}",
  "indexed_at": 1702281600
}
```

### devices_fts
Full-text search virtual table for device content.

```sql
CREATE VIRTUAL TABLE devices_fts USING fts5(
    device_id UNINDEXED,                     -- Link to devices.id
    name,                                    -- Searchable device name
    content,                                 -- Searchable markdown content
    content=devices,                         -- External content table
    content_rowid=rowid                      -- Rowid mapping
);

-- Triggers to keep FTS in sync with devices table
CREATE TRIGGER devices_fts_insert AFTER INSERT ON devices BEGIN
    INSERT INTO devices_fts(rowid, device_id, name, content)
    VALUES (new.rowid, new.id, new.name, new.content);
END;

CREATE TRIGGER devices_fts_update AFTER UPDATE ON devices BEGIN
    UPDATE devices_fts
    SET name = new.name, content = new.content
    WHERE rowid = new.rowid;
END;

CREATE TRIGGER devices_fts_delete AFTER DELETE ON devices BEGIN
    DELETE FROM devices_fts WHERE rowid = old.rowid;
END;
```

### pinouts
GPIO pinout information for hardware devices.

```sql
CREATE TABLE pinouts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    device_id TEXT NOT NULL,                 -- Foreign key to devices.id
    physical INTEGER NOT NULL,               -- Physical pin number
    gpio INTEGER,                            -- GPIO number (nullable)
    name TEXT NOT NULL,                      -- Pin name
    function TEXT,                           -- power, ground, gpio, i2c, etc.
    interfaces TEXT,                         -- JSON array: ["i2c","gpio"]
    notes TEXT,                              -- Additional pin information
    FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE
);

CREATE INDEX idx_pinouts_device ON pinouts(device_id);
CREATE INDEX idx_pinouts_function ON pinouts(function);
CREATE INDEX idx_pinouts_gpio ON pinouts(gpio);
```

**Example rows:**
```json
[
  {
    "device_id": "raspberry-pi-4",
    "physical": 1,
    "gpio": null,
    "name": "3V3 Power",
    "function": "power",
    "interfaces": "[\"power\"]",
    "notes": "3.3V output, max 500mA total"
  },
  {
    "device_id": "raspberry-pi-4",
    "physical": 3,
    "gpio": 2,
    "name": "GPIO 2 / SDA1",
    "function": "i2c",
    "interfaces": "[\"i2c\",\"gpio\"]",
    "notes": "I2C data line with 1.8kΩ pull-up"
  }
]
```

### guides
Workflow and reference documentation.

```sql
CREATE TABLE guides (
    id TEXT PRIMARY KEY,                     -- Guide ID (e.g., 'quickstart')
    title TEXT NOT NULL,                     -- Display title
    content TEXT NOT NULL,                   -- Markdown content
    indexed_at INTEGER NOT NULL              -- Unix timestamp
);

CREATE INDEX idx_guides_title ON guides(title);
```

**Example rows:**
```json
[
  {
    "id": "quickstart",
    "title": "Quick Start Guide",
    "content": "# Quick Start\n\n...",
    "indexed_at": 1702281600
  },
  {
    "id": "api-reference",
    "title": "API Reference",
    "content": "# API Reference\n\n...",
    "indexed_at": 1702281600
  }
]
```

### documents
Metadata for source documents (PDFs, images, markdown files).

```sql
CREATE TABLE documents (
    id TEXT PRIMARY KEY,                     -- Unique document ID
    device_id TEXT NOT NULL,                 -- Foreign key to devices.id
    filename TEXT NOT NULL,                  -- Original filename
    type TEXT NOT NULL,                      -- pdf, markdown, image
    mime_type TEXT NOT NULL,                 -- application/pdf, image/png, etc.
    size_bytes INTEGER NOT NULL,             -- File size
    checksum TEXT,                           -- SHA256 hash
    description TEXT,                        -- User-provided description
    storage_type TEXT NOT NULL,              -- 'filesystem' or 'blob'
    storage_path TEXT,                       -- Path on filesystem (if not blob)
    content BLOB,                            -- Binary content (if blob storage)
    uploaded_at INTEGER NOT NULL,            -- Unix timestamp
    FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE
);

CREATE INDEX idx_documents_device ON documents(device_id);
CREATE INDEX idx_documents_type ON documents(type);
CREATE INDEX idx_documents_uploaded ON documents(uploaded_at);
```

**Example rows:**
```json
[
  {
    "id": "ds18b20-datasheet-pdf",
    "device_id": "ds18b20",
    "filename": "DS18B20.pdf",
    "type": "pdf",
    "mime_type": "application/pdf",
    "size_bytes": 392186,
    "checksum": "sha256:abc123...",
    "description": "Manufacturer datasheet rev 4.2",
    "storage_type": "filesystem",
    "storage_path": "hardware/sensors/ds18b20/DS18B20.pdf",
    "content": null,
    "uploaded_at": 1702281600
  },
  {
    "id": "ds18b20-pinout-png",
    "device_id": "ds18b20",
    "filename": "pinout.png",
    "type": "image",
    "mime_type": "image/png",
    "size_bytes": 45821,
    "checksum": "sha256:def456...",
    "description": "Pinout diagram",
    "storage_type": "blob",
    "storage_path": null,
    "content": <binary data>,
    "uploaded_at": 1702281600
  }
]
```

### specifications (Optional - Future)
Structured device specifications.

```sql
CREATE TABLE specifications (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    device_id TEXT NOT NULL,                 -- Foreign key to devices.id
    category TEXT NOT NULL,                  -- Electrical, Mechanical, etc.
    parameter TEXT NOT NULL,                 -- Voltage, Temperature Range, etc.
    value TEXT NOT NULL,                     -- Value with units
    min_value REAL,                          -- Numeric min (optional)
    max_value REAL,                          -- Numeric max (optional)
    unit TEXT,                               -- V, mA, °C, etc.
    notes TEXT,                              -- Additional context
    FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE
);

CREATE INDEX idx_specs_device ON specifications(device_id);
CREATE INDEX idx_specs_category ON specifications(category);
```

**Example rows:**
```json
[
  {
    "device_id": "ds18b20",
    "category": "Electrical",
    "parameter": "Operating Voltage",
    "value": "3.0V to 5.5V",
    "min_value": 3.0,
    "max_value": 5.5,
    "unit": "V",
    "notes": null
  },
  {
    "device_id": "ds18b20",
    "category": "Performance",
    "parameter": "Temperature Range",
    "value": "-55°C to +125°C",
    "min_value": -55,
    "max_value": 125,
    "unit": "°C",
    "notes": "±0.5°C accuracy from -10°C to +85°C"
  }
]
```

## Query Patterns

### Full-Text Search
```sql
-- Search devices by content
SELECT d.id, d.name, d.domain, d.type, d.manufacturer,
       bm25(devices_fts) as rank,
       snippet(devices_fts, 2, '<mark>', '</mark>', '...', 32) as snippet
FROM devices_fts
JOIN devices d ON d.id = devices_fts.device_id
WHERE devices_fts MATCH ?
ORDER BY rank
LIMIT ?;

-- Parameters: ("temperature AND i2c", 10)
```

### Get Device by ID
```sql
SELECT id, name, domain, type, category, manufacturer, content, metadata
FROM devices
WHERE id = ?;

-- Parameter: "ds18b20"
```

### List Devices by Category
```sql
SELECT id, name, domain, type, manufacturer
FROM devices
WHERE category = ?
ORDER BY name;

-- Parameter: "Temperature Sensors"
```

### Get Pinout
```sql
SELECT physical, gpio, name, function, interfaces, notes
FROM pinouts
WHERE device_id = ?
ORDER BY physical;

-- Optional interface filter:
WHERE device_id = ? AND function = ?
-- Parameters: ("raspberry-pi-4", "i2c")
```

### List Documents for Device
```sql
SELECT id, filename, type, mime_type, size_bytes, description, uploaded_at
FROM documents
WHERE device_id = ?
ORDER BY uploaded_at DESC;

-- Parameter: "ds18b20"
```

### Get Document Content
```sql
-- Filesystem storage:
SELECT storage_path FROM documents WHERE id = ?;
-- Then read from filesystem

-- Blob storage:
SELECT content, mime_type FROM documents WHERE id = ?;
```

### Statistics
```sql
-- Device counts by domain
SELECT domain, COUNT(*) as count
FROM devices
GROUP BY domain;

-- Device counts by type
SELECT type, COUNT(*) as count
FROM devices
GROUP BY type
ORDER BY count DESC;

-- Total documents by type
SELECT type, COUNT(*) as count, SUM(size_bytes) as total_bytes
FROM documents
GROUP BY type;
```

## Storage Strategy

### When to Use Database BLOB vs Filesystem

**Database BLOB** (content field):
- Small files (<1MB)
- Images that need atomic updates with metadata
- Files that should move with database backups

**Filesystem** (storage_path field):
- Large files (>1MB)
- PDFs, videos
- Files that don't change often
- Easier to browse/backup separately

### Filesystem Layout
```
/data/manuals-data/
├── hardware/
│   ├── sensors/
│   │   ├── ds18b20/
│   │   │   ├── device.md          # Indexed into devices.content
│   │   │   ├── DS18B20.pdf        # Referenced in documents table
│   │   │   └── pinout.png         # Referenced in documents table
│   │   └── bme280/
│   │       └── ...
│   └── dev-boards/
│       └── raspberry-pi-4/
│           ├── device.md
│           └── pinout.json        # Parsed into pinouts table
├── software/
│   └── ...
└── protocols/
    └── i2c/
        ├── device.md
        └── specification.pdf
```

## Indexes and Performance

### Query Performance Targets
- Full-text search: <10ms for typical queries
- Device lookup by ID: <1ms
- Pinout retrieval: <2ms
- Document metadata: <1ms

### Index Strategy
```sql
-- Covering indexes for common queries
CREATE INDEX idx_devices_listing ON devices(domain, type, name);

-- Composite index for filtered searches
CREATE INDEX idx_devices_domain_type ON devices(domain, type);

-- Index for pinout lookups
CREATE INDEX idx_pinouts_device_physical ON pinouts(device_id, physical);
```

### FTS5 Optimization
```sql
-- Rebuild FTS index for optimization
INSERT INTO devices_fts(devices_fts) VALUES('rebuild');

-- Optimize after bulk updates
INSERT INTO devices_fts(devices_fts) VALUES('optimize');
```

## Migrations

### Schema Versioning
```sql
CREATE TABLE schema_version (
    version INTEGER PRIMARY KEY,
    applied_at INTEGER NOT NULL,
    description TEXT
);

INSERT INTO schema_version VALUES (1, 1702281600, 'Initial schema');
```

### Example Migration
```sql
-- Migration: Add checksum field to documents
BEGIN TRANSACTION;

ALTER TABLE documents ADD COLUMN checksum TEXT;

UPDATE schema_version SET version = 2, applied_at = 1702285200,
  description = 'Add checksum to documents';

COMMIT;
```

## Backup Strategy

### Full Backup
```bash
# SQLite backup command
sqlite3 /data/manuals.db ".backup /backup/manuals-$(date +%Y%m%d).db"

# Or using VACUUM INTO:
sqlite3 /data/manuals.db "VACUUM INTO '/backup/manuals-$(date +%Y%m%d).db'"
```

### Incremental Backup (Litestream)
```yaml
# litestream.yml
dbs:
  - path: /data/manuals.db
    replicas:
      - url: s3://my-bucket/manuals-db
        sync-interval: 10s
```

### Filesystem Documents
```bash
# Rsync documents to backup location
rsync -av /data/manuals-data/ /backup/manuals-data/
```

## Database Maintenance

### Regular Tasks

**Daily:**
```sql
-- Analyze tables for query optimization
ANALYZE;
```

**Weekly:**
```sql
-- Optimize FTS index
INSERT INTO devices_fts(devices_fts) VALUES('optimize');

-- Check integrity
PRAGMA integrity_check;
```

**Monthly:**
```sql
-- Vacuum to reclaim space
VACUUM;
```

### Monitoring

```sql
-- Database size
SELECT page_count * page_size as size_bytes
FROM pragma_page_count(), pragma_page_size();

-- Table sizes
SELECT
    name,
    SUM(pgsize) as size_bytes
FROM dbstat
GROUP BY name
ORDER BY size_bytes DESC;

-- FTS index size
SELECT SUM(pgsize) as fts_size_bytes
FROM dbstat
WHERE name LIKE 'devices_fts%';
```

## Connection Management

### API Server Connection Pool
```go
db, err := sql.Open("sqlite", "/data/manuals.db?_journal_mode=WAL&_busy_timeout=5000")
db.SetMaxOpenConns(10)
db.SetMaxIdleConns(5)
db.SetConnMaxLifetime(time.Hour)
```

### WAL Mode (Write-Ahead Logging)
```sql
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;
PRAGMA cache_size = -64000;  -- 64MB cache
PRAGMA temp_store = MEMORY;
```

**Benefits:**
- Concurrent reads while writing
- Better performance
- Crash-safe

## Constraints and Data Integrity

### Enforced Constraints
```sql
-- Foreign keys
PRAGMA foreign_keys = ON;

-- Check constraints
ALTER TABLE devices ADD CONSTRAINT check_domain
  CHECK (domain IN ('hardware', 'software', 'protocol'));

-- Unique constraints
CREATE UNIQUE INDEX idx_documents_device_filename
  ON documents(device_id, filename);
```

### Cascading Deletes
```sql
-- When device is deleted, cascade to related tables
FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE
```

## Testing Data

### Seed Data Script
```sql
-- Create test device
INSERT INTO devices VALUES (
  'test-sensor',
  'Test Temperature Sensor',
  'hardware',
  'sensors',
  'Test Devices',
  'Test Manufacturer',
  '# Test Sensor\n\nThis is a test device.',
  '{"tags":["test"]}',
  1702281600
);

-- Create test pinout
INSERT INTO pinouts VALUES (
  NULL,
  'test-sensor',
  1,
  NULL,
  'VCC',
  'power',
  '["power"]',
  'Power supply'
);

-- Create test document
INSERT INTO documents VALUES (
  'test-sensor-datasheet',
  'test-sensor',
  'test-datasheet.pdf',
  'pdf',
  'application/pdf',
  12345,
  'sha256:test123',
  'Test datasheet',
  'filesystem',
  'hardware/sensors/test-sensor/datasheet.pdf',
  NULL,
  1702281600
);
```

## Future Enhancements

1. **Versioned Documents**: Track document revisions
2. **Audit Log**: Track all changes to devices/documents
3. **Tags Table**: Normalize tags instead of JSON array
4. **Relationships**: Link related devices (e.g., sensor + microcontroller)
5. **Read Replicas**: For high-traffic scenarios

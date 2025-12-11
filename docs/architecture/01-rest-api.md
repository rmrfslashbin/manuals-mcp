# REST API Design

## Overview

The manuals-api REST API provides HTTP access to the documentation platform, serving as the central backend for all clients (MCP, Web UI, CLI tools).

## Design Principles

1. **RESTful**: Follow REST conventions for resource naming and HTTP verbs
2. **Versioned**: `/api/v1/` prefix for future compatibility
3. **JSON**: All request/response bodies use JSON (except file uploads/downloads)
4. **Idempotent**: GET/PUT/DELETE operations are idempotent
5. **Error Consistent**: Uniform error response format
6. **OpenAPI**: Documented with OpenAPI 3.0 specification

## Base URL

```
http://manuals.local/api/v1
```

Production:
```
https://manuals.example.com/api/v1
```

## Authentication

**Phase 1 (MVP)**: No authentication
- Assumes trusted local network
- Rate limiting by IP

**Phase 2 (Future)**: API Key
```
Authorization: Bearer <api-key>
```

**Phase 3 (Future)**: JWT for web UI
```
Authorization: Bearer <jwt-token>
```

## Endpoints

### Query Endpoints

#### Search Documentation
```http
POST /api/v1/search
Content-Type: application/json

{
  "query": "temperature sensor i2c",
  "domain": "hardware",  // optional: hardware, software, protocol, all
  "type": "sensors",     // optional: device type filter
  "limit": 10            // optional: default 10, max 100
}

Response 200 OK:
{
  "results": [
    {
      "device_id": "ds18b20",
      "name": "DS18B20 Digital Thermometer",
      "domain": "hardware",
      "type": "sensors",
      "manufacturer": "Maxim Integrated",
      "rank": 0.95,
      "snippet": "...1-Wire digital temperature sensor..."
    }
  ],
  "count": 3,
  "query": "temperature sensor i2c",
  "duration_ms": 12
}
```

#### List All Devices
```http
GET /api/v1/devices
GET /api/v1/devices?category=sensors
GET /api/v1/devices?domain=hardware

Response 200 OK:
{
  "devices": [
    {
      "id": "ds18b20",
      "name": "DS18B20 Digital Thermometer",
      "domain": "hardware",
      "type": "sensors",
      "category": "Temperature Sensors",
      "manufacturer": "Maxim Integrated"
    }
  ],
  "count": 27
}
```

#### Get Device Details
```http
GET /api/v1/devices/:id
Accept: text/markdown

Response 200 OK:
Content-Type: text/markdown

# DS18B20 Digital Thermometer

**Type:** Temperature Sensor
**Interface:** 1-Wire
...
```

#### Get Device Pinout
```http
GET /api/v1/pinout/:device
GET /api/v1/pinout/:device?interface=i2c

Response 200 OK:
{
  "device_id": "raspberry-pi-4",
  "device_name": "Raspberry Pi 4 Model B",
  "total_pins": 40,
  "pins": [
    {
      "physical": 1,
      "gpio": null,
      "name": "3V3 Power",
      "function": "power",
      "interfaces": ["power"],
      "notes": "3.3V output, max 500mA total"
    },
    {
      "physical": 3,
      "gpio": 2,
      "name": "GPIO 2 / SDA1",
      "function": "i2c",
      "interfaces": ["i2c", "gpio"],
      "notes": "I2C data line with 1.8kΩ pull-up"
    }
  ]
}
```

#### Get Statistics
```http
GET /api/v1/stats

Response 200 OK:
{
  "database": {
    "total_devices": 27,
    "by_domain": {
      "hardware": 22,
      "software": 1,
      "protocol": 4
    },
    "by_type": {
      "sensors": 8,
      "dev-boards": 6,
      "displays": 4
    }
  },
  "pinouts": {
    "total_devices": 6,
    "total_pins": 287
  },
  "documents": {
    "total_files": 45,
    "total_size_bytes": 28473621,
    "by_type": {
      "pdf": 23,
      "markdown": 15,
      "image": 7
    }
  },
  "guides": {
    "total": 4
  },
  "indexed_at": "2025-12-11T03:00:00Z"
}
```

#### Get Server Info
```http
GET /api/v1/info

Response 200 OK:
{
  "version": "3.0.0",
  "git_commit": "a1b2c3d",
  "build_time": "2025-12-11T02:00:00Z",
  "api_version": "v1",
  "features": {
    "search": true,
    "pinout": true,
    "documents": true,
    "reindex": true
  },
  "docs_path": "/data/manuals-data",
  "database": {
    "path": "/data/manuals.db",
    "size_bytes": 15728640
  }
}
```

#### Get Tags
```http
GET /api/v1/tags

Response 200 OK:
{
  "tags": [
    {"name": "i2c", "count": 12},
    {"name": "spi", "count": 8},
    {"name": "temperature", "count": 5}
  ]
}
```

#### Get Categories
```http
GET /api/v1/categories

Response 200 OK:
{
  "categories": [
    {"name": "Temperature Sensors", "count": 8, "domain": "hardware"},
    {"name": "Development Boards", "count": 6, "domain": "hardware"}
  ]
}
```

#### Get Manufacturers
```http
GET /api/v1/manufacturers

Response 200 OK:
{
  "manufacturers": [
    {"name": "Raspberry Pi Foundation", "count": 5},
    {"name": "Espressif", "count": 3}
  ]
}
```

#### Get Metadata Schema
```http
GET /api/v1/metadata/schema

Response 200 OK:
{
  "fields": [
    {"name": "manufacturer", "type": "text", "description": "Device manufacturer"},
    {"name": "datasheet_url", "type": "text", "description": "URL to datasheet"}
  ]
}
```

### Resource Endpoints

#### Get Guide
```http
GET /api/v1/guides/:guide_id
Accept: text/markdown

Examples:
  /api/v1/guides/quickstart
  /api/v1/guides/workflow
  /api/v1/guides/api-reference

Response 200 OK:
Content-Type: text/markdown

# Quick Start Guide
...
```

#### List Guides
```http
GET /api/v1/guides

Response 200 OK:
{
  "guides": [
    {"id": "quickstart", "title": "Quick Start Guide"},
    {"id": "workflow", "title": "Add Hardware Workflow"},
    {"id": "api-reference", "title": "API Reference"}
  ]
}
```

### Document Management Endpoints

#### List Device Documents
```http
GET /api/v1/devices/:device_id/documents

Response 200 OK:
{
  "device_id": "ds18b20",
  "documents": [
    {
      "id": "ds18b20-datasheet-pdf",
      "filename": "DS18B20.pdf",
      "type": "pdf",
      "mime_type": "application/pdf",
      "size_bytes": 392186,
      "description": "Manufacturer datasheet",
      "uploaded_at": "2025-12-10T14:00:00Z",
      "checksum": "sha256:abc123..."
    },
    {
      "id": "ds18b20-device-md",
      "filename": "device.md",
      "type": "markdown",
      "mime_type": "text/markdown",
      "size_bytes": 4521,
      "uploaded_at": "2025-12-10T14:00:00Z"
    }
  ],
  "count": 2
}
```

#### Upload Document
```http
POST /api/v1/devices/:device_id/documents
Content-Type: multipart/form-data

Form Fields:
  file: <binary>              (required)
  type: pdf|markdown|image    (required)
  description: string         (optional)
  auto_reindex: boolean       (optional, default: true)

Response 201 Created:
{
  "id": "ds18b20-datasheet-pdf",
  "device_id": "ds18b20",
  "filename": "DS18B20.pdf",
  "type": "pdf",
  "size_bytes": 392186,
  "uploaded_at": "2025-12-11T03:00:00Z",
  "reindex_triggered": true
}
```

#### Get Document Metadata
```http
GET /api/v1/documents/:doc_id

Response 200 OK:
{
  "id": "ds18b20-datasheet-pdf",
  "device_id": "ds18b20",
  "filename": "DS18B20.pdf",
  "type": "pdf",
  "mime_type": "application/pdf",
  "size_bytes": 392186,
  "description": "Manufacturer datasheet",
  "uploaded_at": "2025-12-10T14:00:00Z",
  "checksum": "sha256:abc123...",
  "storage": {
    "type": "filesystem",
    "path": "hardware/sensors/ds18b20/DS18B20.pdf"
  }
}
```

#### Download Document
```http
GET /api/v1/documents/:doc_id/content

Response 200 OK:
Content-Type: application/pdf
Content-Disposition: attachment; filename="DS18B20.pdf"
Content-Length: 392186

<binary data>
```

#### Update Document
```http
PUT /api/v1/documents/:doc_id
Content-Type: multipart/form-data

Form Fields:
  file: <binary>              (required)
  auto_reindex: boolean       (optional, default: true)

Response 200 OK:
{
  "id": "ds18b20-datasheet-pdf",
  "filename": "DS18B20.pdf",
  "size_bytes": 395240,
  "updated_at": "2025-12-11T03:30:00Z",
  "reindex_triggered": true
}
```

#### Delete Document
```http
DELETE /api/v1/documents/:doc_id?reindex=true

Response 204 No Content
```

### Static File Serving

#### Get Static File
```http
GET /api/v1/files/*

Examples:
  /api/v1/files/hardware/sensors/ds18b20/DS18B20.pdf
  /api/v1/files/hardware/dev-boards/raspberry-pi-4/pinout.png

Response 200 OK:
Content-Type: application/pdf  (or image/png, etc.)
Content-Length: 392186
Cache-Control: public, max-age=3600

<binary data>
```

### Admin Endpoints

#### Trigger Reindex
```http
POST /api/v1/reindex
Content-Type: application/json

{
  "devices": ["ds18b20"],  // optional: specific devices, omit for full reindex
  "clear": false           // optional: clear database first, default: false
}

Response 202 Accepted:
{
  "status": "accepted",
  "job_id": "reindex-20251211-030000",
  "message": "Reindex started"
}

// Check status:
GET /api/v1/reindex/status/:job_id

Response 200 OK:
{
  "job_id": "reindex-20251211-030000",
  "status": "completed",  // or: running, failed
  "started_at": "2025-12-11T03:00:00Z",
  "completed_at": "2025-12-11T03:00:02Z",
  "duration_ms": 2145,
  "result": {
    "total_files": 27,
    "success_count": 27,
    "error_count": 0,
    "devices_by_type": {
      "hardware": 22,
      "software": 1,
      "protocol": 4
    }
  }
}
```

#### Health Check
```http
GET /api/v1/health

Response 200 OK:
{
  "status": "healthy",
  "timestamp": "2025-12-11T03:00:00Z",
  "checks": {
    "database": "ok",
    "filesystem": "ok",
    "memory_mb": 45
  }
}
```

## Error Responses

All errors follow consistent format:

```http
Response 4xx/5xx:
Content-Type: application/json

{
  "error": {
    "code": "NOT_FOUND",
    "message": "Device 'xyz' not found",
    "details": {
      "device_id": "xyz"
    },
    "timestamp": "2025-12-11T03:00:00Z"
  }
}
```

### Error Codes

| HTTP | Code | Description |
|------|------|-------------|
| 400 | INVALID_REQUEST | Malformed request body or parameters |
| 401 | UNAUTHORIZED | Missing or invalid authentication |
| 403 | FORBIDDEN | Insufficient permissions |
| 404 | NOT_FOUND | Resource does not exist |
| 409 | CONFLICT | Resource already exists |
| 413 | PAYLOAD_TOO_LARGE | File upload exceeds size limit |
| 422 | VALIDATION_ERROR | Request validation failed |
| 429 | RATE_LIMITED | Too many requests |
| 500 | INTERNAL_ERROR | Server error |
| 503 | SERVICE_UNAVAILABLE | Service temporarily unavailable |

## Rate Limiting

**Phase 1**: Simple IP-based rate limiting
- 100 requests per minute per IP
- 10 uploads per hour per IP

Headers:
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1702281600
```

Response when rate limited:
```http
Response 429 Too Many Requests:
{
  "error": {
    "code": "RATE_LIMITED",
    "message": "Rate limit exceeded, try again in 30 seconds",
    "retry_after": 30
  }
}
```

## CORS

**Development**: Allow all origins
```
Access-Control-Allow-Origin: *
```

**Production**: Restrict to known origins
```
Access-Control-Allow-Origin: https://manuals.example.com
Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS
Access-Control-Allow-Headers: Content-Type, Authorization
```

## Caching

### Static Files
```
Cache-Control: public, max-age=3600
ETag: "abc123..."
```

### Query Results
```
Cache-Control: public, max-age=60
```

### Device Documentation
```
Cache-Control: public, max-age=300
```

## Content Negotiation

### Markdown Resources
```http
GET /api/v1/devices/:id
Accept: text/markdown          → Returns raw markdown
Accept: application/json       → Returns JSON with metadata
Accept: text/html              → Returns rendered HTML (future)
```

## Pagination

For list endpoints with many results:

```http
GET /api/v1/devices?page=2&per_page=20

Response 200 OK:
{
  "devices": [...],
  "pagination": {
    "page": 2,
    "per_page": 20,
    "total_pages": 5,
    "total_count": 94
  },
  "links": {
    "first": "/api/v1/devices?page=1&per_page=20",
    "prev": "/api/v1/devices?page=1&per_page=20",
    "next": "/api/v1/devices?page=3&per_page=20",
    "last": "/api/v1/devices?page=5&per_page=20"
  }
}
```

## Webhook Notifications (Future)

```http
POST /api/v1/webhooks
Content-Type: application/json

{
  "url": "https://example.com/webhook",
  "events": ["document.uploaded", "device.updated", "reindex.completed"],
  "secret": "webhook-secret"
}

Webhook payload:
{
  "event": "document.uploaded",
  "timestamp": "2025-12-11T03:00:00Z",
  "data": {
    "document_id": "ds18b20-datasheet-pdf",
    "device_id": "ds18b20"
  }
}
```

## Implementation Notes

### File Upload Limits
- Maximum file size: 50MB
- Allowed types: PDF, Markdown, PNG, JPG, SVG
- Virus scanning: Optional (ClamAV integration)

### Search Performance
- FTS5 queries typically <10ms
- Index entire query string
- Support phrase queries: `"exact phrase"`
- Boolean operators: `temperature AND i2c`

### Database Connections
- Connection pool: 5-10 connections
- Max idle: 2 connections
- Connection timeout: 30s

### Static File Serving
- Streamed for large files (no memory buffering)
- Support Range requests for partial downloads
- ETag generation for caching

## OpenAPI Specification

Full OpenAPI 3.0 spec available at:
```
/api/v1/openapi.yaml
/api/v1/openapi.json
```

Interactive documentation:
```
/api/v1/docs  (Swagger UI)
```

## Testing

### Curl Examples

Search:
```bash
curl -X POST http://manuals.local/api/v1/search \
  -H "Content-Type: application/json" \
  -d '{"query": "temperature sensor", "limit": 5}'
```

Upload:
```bash
curl -X POST http://manuals.local/api/v1/devices/ds18b20/documents \
  -F "file=@DS18B20.pdf" \
  -F "type=pdf" \
  -F "description=Datasheet rev 4.2"
```

Download:
```bash
curl -o DS18B20.pdf \
  http://manuals.local/api/v1/documents/ds18b20-datasheet-pdf/content
```

### HTTP Client Libraries

Recommended clients:
- **Go**: `net/http`, `resty`
- **Python**: `requests`, `httpx`
- **JavaScript**: `fetch`, `axios`
- **Rust**: `reqwest`

# Architecture Overview

## Vision

Transform manuals-mcp from a single-binary documentation server into a **multi-client documentation platform** with centralized storage, REST API, and support for multiple access methods (MCP, Web UI, CLI tools).

## Design Principles

1. **Separation of Concerns**: API server, MCP client, and storage are independent components
2. **Centralized Data**: Single source of truth for documentation on Raspberry Pi
3. **Multiple Clients**: MCP, Web UI, CLI tools all access the same API
4. **Minimal Workflow Changes**: Existing indexing and documentation patterns remain valid
5. **Context-Efficient**: No binary data through MCP protocol; use direct API calls via curl
6. **Proper Deployment**: Designed for production with systemd, nginx, monitoring

## High-Level Architecture

```
┌──────────────────────────────────────────────────────────────┐
│ Raspberry Pi (Central Server)                                │
│                                                               │
│  ┌────────────────────────────────────────────────────────┐ │
│  │ Nginx (Reverse Proxy)                                  │ │
│  │ - SSL termination                                      │ │
│  │ - Rate limiting                                        │ │
│  │ - Static asset caching                                 │ │
│  │ - Request routing                                      │ │
│  └──────────────────┬─────────────────────────────────────┘ │
│                     │                                         │
│  ┌──────────────────▼─────────────────────────────────────┐ │
│  │ manuals-api (Go HTTP Server)                          │ │
│  │                                                        │ │
│  │ Endpoints:                                             │ │
│  │ - POST /api/v1/search                                  │ │
│  │ - GET  /api/v1/devices                                 │ │
│  │ - GET  /api/v1/devices/:id                             │ │
│  │ - GET  /api/v1/pinout/:device                          │ │
│  │ - POST /api/v1/devices/:id/documents (upload)          │ │
│  │ - GET  /api/v1/files/* (static files)                  │ │
│  │ - POST /api/v1/reindex                                 │ │
│  │                                                        │ │
│  │ Responsibilities:                                      │ │
│  │ - Query processing                                     │ │
│  │ - Document upload/management                           │ │
│  │ - Static file serving                                  │ │
│  │ - Reindex orchestration                                │ │
│  │ - FTS5 search                                          │ │
│  └──────────────────┬─────────────────────────────────────┘ │
│                     │                                         │
│  ┌──────────────────▼─────────────────────────────────────┐ │
│  │ SQLite Database (manuals.db)                          │ │
│  │ - FTS5 full-text search                                │ │
│  │ - Device metadata                                      │ │
│  │ - Markdown content (in database)                       │ │
│  │ - Pinout data                                          │ │
│  │ - Guides                                               │ │
│  │ - Document metadata                                    │ │
│  └────────────────────────────────────────────────────────┘ │
│                                                               │
│  ┌────────────────────────────────────────────────────────┐ │
│  │ Filesystem (docs-path)                                 │ │
│  │ /data/manuals-data/                                    │ │
│  │ ├── hardware/                                          │ │
│  │ │   └── sensors/ds18b20/                               │ │
│  │ │       ├── device.md                                  │ │
│  │ │       └── DS18B20.pdf                                │ │
│  │ └── protocols/i2c/                                     │ │
│  │     ├── device.md                                      │ │
│  │     └── specification.pdf                              │ │
│  └────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────┘
                           │
                           │ HTTP/REST (port 80/443)
            ───────────────┼────────────────────
            │              │                   │
            ▼              ▼                   ▼
    ┌──────────────┐ ┌──────────────┐  ┌──────────────┐
    │ MCP Client   │ │ Web UI       │  │ CLI Tools    │
    │              │ │              │  │              │
    │ Thin adapter │ │ React/Vue    │  │ curl         │
    │ stdio ↔ HTTP │ │ Browse/      │  │ scripts      │
    │              │ │ Search       │  │              │
    │ Provides:    │ │ Upload docs  │  │ Direct API   │
    │ - Query API  │ │ Visualize    │  │ access       │
    │ - Resources  │ │              │  │              │
    │ - Curl guide │ │              │  │              │
    └──────┬───────┘ └──────────────┘  └──────────────┘
           │
           │ stdio (MCP protocol)
           │
    ┌──────▼───────┐
    │ Claude Code  │
    │              │
    │ Uses:        │
    │ - MCP tools  │
    │ - Bash tool  │
    │   (for curl) │
    └──────────────┘
```

## Component Roles

### manuals-api (New - REST API Server)
**Location**: Raspberry Pi
**Language**: Go
**Purpose**: Central HTTP API for all clients

**Responsibilities**:
- Serve query endpoints (search, list, get device, pinout)
- Handle document uploads/deletes
- Serve static files (PDFs, images)
- Trigger reindexing
- Manage database connections
- Implement rate limiting, CORS, logging

**Does NOT**:
- Handle MCP protocol directly
- Maintain WebSocket connections
- Implement business logic beyond data access

### manuals-mcp-client (New - MCP Adapter)
**Location**: User's workstation
**Language**: Go
**Purpose**: Translate MCP protocol to REST API calls

**Responsibilities**:
- Implement MCP stdio protocol
- Translate MCP tool calls → HTTP requests
- Serve MCP resources from API
- Provide curl command templates for uploads
- Handle API errors gracefully
- Cache API responses (optional)

**Does NOT**:
- Store data locally
- Handle file uploads directly through MCP
- Implement search logic (delegates to API)

### manuals-mcp (Current - Local Binary)
**Status**: Maintained for single-user, offline use
**Purpose**: Self-contained local deployment

**Use Cases**:
- Offline documentation access
- Single-user scenarios
- Development/testing
- Air-gapped environments

### Web UI (Future)
**Location**: Static files served by nginx
**Purpose**: Human-friendly documentation browser

**Features**:
- Browse devices by category
- Full-text search interface
- View pinout diagrams
- Upload/manage documents
- View PDF datasheets inline

## Data Flow Examples

### Search Query
```
1. User asks Claude: "Find I2C temperature sensors"
2. Claude → manuals-mcp-client (MCP tool: search)
3. manuals-mcp-client → POST http://pi.local/api/v1/search
4. manuals-api → SQLite FTS5 query
5. manuals-api → JSON response
6. manuals-mcp-client → MCP result (formatted text)
7. Claude → User (formatted response)
```

### Document Upload
```
1. User asks Claude: "Upload DS18B20.pdf datasheet"
2. Claude → manuals-mcp-client (get_upload_command tool)
3. manuals-mcp-client → Returns curl command template
4. Claude → Bash tool (executes curl)
5. curl → POST http://pi.local/api/v1/devices/ds18b20/documents
6. manuals-api → Saves file to filesystem
7. manuals-api → Updates database metadata
8. manuals-api → Triggers reindex for device
9. manuals-api → Returns success JSON
10. Claude → User (confirms upload)
```

### View Device Documentation
```
1. User asks Claude: "Show me DS18B20 documentation"
2. Claude → manuals-mcp-client (MCP resource: manuals://device/ds18b20)
3. manuals-mcp-client → GET http://pi.local/api/v1/devices/ds18b20
4. manuals-api → SELECT content FROM devices WHERE id='ds18b20'
5. manuals-api → Returns markdown content
6. manuals-mcp-client → MCP resource response
7. Claude → User (displays formatted markdown)
```

## Key Design Decisions

### 1. No Binary Data Through MCP
**Decision**: Document uploads use curl via Bash tool, not MCP protocol
**Rationale**: Prevents context window bloat, more efficient, leverages existing tooling

### 2. Thin MCP Client
**Decision**: MCP client is just an adapter, no business logic
**Rationale**: Keep client simple, all logic in API for consistency across clients

### 3. Markdown in Database, PDFs on Filesystem
**Decision**: Continue storing markdown content in SQLite, reference PDFs from filesystem
**Rationale**: FTS5 indexes markdown efficiently, large binaries stay on disk

### 4. REST API Over GraphQL
**Decision**: Simple RESTful endpoints
**Rationale**: Easier to implement, curl-friendly, sufficient for use cases

### 5. Nginx Reverse Proxy
**Decision**: Nginx in front of Go API
**Rationale**: SSL termination, static caching, rate limiting, mature tooling

### 6. Keep Local Binary
**Decision**: Don't deprecate current manuals-mcp
**Rationale**: Valid use case for offline/single-user scenarios

## Migration Path

**Current State** (v2.3.0):
- Single binary: `manuals-mcp`
- Local SQLite database
- MCP server via stdio
- CLI indexing

**Future State** (v3.0.0):
- API server: `manuals-api` (on Pi)
- MCP adapter: `manuals-mcp-client` (on workstation)
- Web UI (optional)
- Local binary: `manuals-mcp` (still available)

**Migration Strategy**:
1. Build manuals-api, deploy to Pi
2. Build manuals-mcp-client, test against API
3. Update documentation
4. Users choose deployment model
5. Both options supported indefinitely

## Performance Considerations

**Local Binary** (current):
- Search latency: ~1-5ms (local SQLite)
- No network dependency
- Single user

**REST API** (new):
- Search latency: ~10-50ms (HTTP + SQLite)
- Network dependency
- Multiple concurrent users
- Cacheable responses

**Mitigation**:
- Nginx caching for static files
- Optional Redis caching layer
- Keep local binary for latency-sensitive use cases

## Security Considerations

**Phase 1** (MVP):
- No authentication
- Local network only (not exposed to internet)
- Basic rate limiting

**Phase 2** (Future):
- API key authentication
- JWT tokens for web UI
- HTTPS/SSL required
- Fine-grained permissions

## Success Metrics

1. **API Response Time**: <50ms p95 for queries
2. **Uptime**: >99.9% (Raspberry Pi reliability)
3. **Concurrent Users**: Support 5-10 simultaneous queries
4. **Document Upload**: <2s for 10MB PDF
5. **Reindex Time**: <5s for typical documentation set

## Next Steps

See detailed design documents:
- [REST API Design](./01-rest-api.md)
- [MCP Client Design](./02-mcp-client.md)
- [Database Schema](./03-database.md)
- [Document Management](./04-document-mgmt.md)
- [Deployment Architecture](./05-deployment.md)

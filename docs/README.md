# Manuals Platform Documentation

Comprehensive documentation for the manuals platform architecture, covering the transition from a single-binary MCP server to a multi-client REST API platform.

## Documentation Structure

### Architecture Documents

Located in `architecture/` - Technical design documentation for the v3.0 platform.

1. **[00-overview.md](architecture/00-overview.md)** - Start here
   - High-level architecture vision
   - Component diagram
   - Design principles
   - Data flow examples
   - Success metrics

2. **[01-rest-api.md](architecture/01-rest-api.md)** - REST API specification
   - Complete endpoint reference
   - Request/response formats
   - Authentication & authorization
   - Rate limiting & caching
   - Error handling
   - OpenAPI specification

3. **[02-mcp-client.md](architecture/02-mcp-client.md)** - MCP client adapter design
   - Client architecture
   - Tool implementations
   - Resource handlers
   - Curl integration pattern
   - Error handling & retries
   - Performance optimization

4. **[03-database.md](architecture/03-database.md)** - Database schema & storage
   - SQLite schema design
   - FTS5 full-text search
   - Storage strategies (BLOB vs filesystem)
   - Query patterns
   - Backup & maintenance
   - Performance tuning

5. **[04-document-mgmt.md](architecture/04-document-mgmt.md)** - Document lifecycle
   - Upload/download/delete flows
   - File validation
   - Reindexing strategies
   - Storage decision logic
   - Bulk operations
   - Monitoring & health checks

6. **[05-deployment.md](architecture/05-deployment.md)** - Raspberry Pi deployment
   - System architecture
   - Installation steps
   - Systemd service configuration
   - Nginx reverse proxy
   - SSL/TLS setup
   - Monitoring & logging
   - Backup strategies
   - Troubleshooting

### Guides

Located in `guides/` - Practical how-to guides.

1. **[migration.md](guides/migration.md)** - Migrating from v2.x to v3.x
   - Should you migrate?
   - Migration strategies
   - Step-by-step instructions
   - Rollback procedures
   - Common issues
   - Testing checklist

## Quick Reference

### For Users

**Getting Started:**
1. Read [Architecture Overview](architecture/00-overview.md) for high-level understanding
2. Follow [Migration Guide](guides/migration.md) if upgrading from v2.x
3. See [Deployment Guide](architecture/05-deployment.md) for Raspberry Pi setup

**Daily Use:**
- [REST API Reference](architecture/01-rest-api.md#endpoints) - Available endpoints
- [Document Management](architecture/04-document-mgmt.md#document-lifecycle) - Uploading files

### For Developers

**Contributing:**
1. [Architecture Overview](architecture/00-overview.md#component-roles) - Component responsibilities
2. [Database Schema](architecture/03-database.md#tables) - Data model
3. [REST API Design](architecture/01-rest-api.md#design-principles) - API conventions

**Building:**
- [MCP Client Design](architecture/02-mcp-client.md#project-structure) - Client implementation
- [Document Management](architecture/04-document-mgmt.md#implementation) - File handling

### For Operators

**Deployment:**
1. [Deployment Guide](architecture/05-deployment.md#installation-steps) - Full setup
2. [Monitoring](architecture/05-deployment.md#monitoring) - Health checks & logs
3. [Backup](architecture/05-deployment.md#backup-strategy) - Database backups

**Maintenance:**
- [Database Maintenance](architecture/03-database.md#database-maintenance) - Optimization tasks
- [Troubleshooting](architecture/05-deployment.md#troubleshooting) - Common issues

## Architecture at a Glance

### Current (v2.x)
```
Claude Code ↔ manuals-mcp ↔ Local SQLite
```
- Single binary, local-only, fast queries

### Future (v3.x)
```
┌──────────────────────────────────────┐
│ Raspberry Pi                          │
│  Nginx → manuals-api → SQLite        │
│            ↓                          │
│       Filesystem (docs)               │
└──────────────────────────────────────┘
           │
           │ HTTP REST API
           │
    ┌──────┴──────┬──────────┐
    │             │          │
MCP Client    Web UI    CLI Tools
    │
Claude Code
```
- Multi-client, remote access, web UI support

## Key Concepts

### Components

- **manuals-api**: REST API server (Go, runs on Raspberry Pi)
- **manuals-mcp-client**: Thin MCP adapter (translates MCP ↔ HTTP)
- **manuals-mcp**: Original local binary (still supported for offline use)

### Data Storage

- **SQLite Database**: Device metadata, markdown content, FTS5 search
- **Filesystem**: PDFs, images, large documents
- **Hybrid Model**: Small files in DB, large files on disk

### Document Upload Pattern

1. User asks Claude to upload document
2. MCP client provides curl command template
3. Claude executes curl via Bash tool
4. File uploads directly to API
5. API triggers reindex
6. Changes immediately searchable

## Version History

### v3.0.0 (Planned)
- REST API server (manuals-api)
- MCP client adapter (manuals-mcp-client)
- Document upload via API
- Web UI foundation

### v2.3.0 (Current)
- Reindex MCP tool
- Live documentation updates

### v2.2.0
- Workflow guides as MCP resources
- Self-contained database

### v2.1.0
- Environment variable support
- Metadata discovery tools

### v2.0.0
- MCP server implementation
- Full-text search

### v1.0.0
- Initial CLI indexer

## Design Principles

1. **Separation of Concerns**: API, client, and storage are independent
2. **Multiple Clients**: MCP, Web, CLI all use same API
3. **Context-Efficient**: No binary data through MCP
4. **Offline Support**: Local binary still available
5. **Production-Ready**: Systemd, nginx, monitoring from day 1

## Contributing to Documentation

### Adding New Documents

1. Follow existing structure and naming conventions
2. Use clear, concise headings
3. Include code examples and diagrams
4. Cross-reference related documents
5. Update this README with new document links

### Documentation Standards

- Use Markdown for all documentation
- Include table of contents for long documents
- Use code blocks with language hints
- Provide both conceptual and practical information
- Include troubleshooting sections

### Diagram Format

Use ASCII art for architecture diagrams:
```
┌─────────────┐
│ Component A │
└──────┬──────┘
       │ Protocol
       ▼
┌─────────────┐
│ Component B │
└─────────────┘
```

## Getting Help

### For Questions

- Check relevant architecture document first
- Review [Migration Guide](guides/migration.md) for v2.x→v3.x issues
- Open GitHub issue if documentation is unclear

### For Issues

- **API Issues**: See [REST API docs](architecture/01-rest-api.md#error-responses)
- **Client Issues**: See [MCP Client docs](architecture/02-mcp-client.md#error-handling)
- **Database Issues**: See [Database docs](architecture/03-database.md#troubleshooting)
- **Deployment Issues**: See [Deployment docs](architecture/05-deployment.md#troubleshooting)

## Roadmap

### Phase 1: API Foundation (v3.0)
- [x] Architecture design
- [ ] REST API implementation
- [ ] MCP client adapter
- [ ] Raspberry Pi deployment
- [ ] Migration tools

### Phase 2: Web UI (v3.1)
- [ ] Browse devices by category
- [ ] Search interface
- [ ] Document upload UI
- [ ] Pinout visualization

### Phase 3: Advanced Features (v3.2+)
- [ ] User authentication
- [ ] Permissions & roles
- [ ] Document versioning
- [ ] Webhook notifications
- [ ] Mobile app

## License

MIT License - See [LICENSE](../LICENSE) file for details

## Author

Robert Sigler (code@sigler.io)

## Acknowledgments

Built with:
- [MCP Go SDK](https://github.com/mark3labs/mcp-go)
- SQLite with FTS5
- Go standard library
- Nginx
- Raspberry Pi Foundation

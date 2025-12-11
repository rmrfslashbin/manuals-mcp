# Manuals MCP Server

A Model Context Protocol (MCP) server for hardware and software documentation, connecting to the Manuals REST API.

## Features

- **Search**: Full-text search across hardware and software documentation
- **Device Information**: Get device details, pinouts, and specifications
- **Document Listings**: Browse available datasheets and PDFs
- **REST API Backend**: All data served from the Manuals REST API

## Installation

### Build from Source

```bash
git clone https://github.com/rmrfslashbin/manuals-mcp.git
cd manuals-mcp
go build -o manuals-mcp ./cmd/manuals-mcp
```

## Configuration

### Environment Variables

```bash
export MANUALS_API_URL="http://manuals.local:8080"
export MANUALS_API_KEY="your-api-key"
```

### Config File

Create `~/.manuals-mcp.yaml`:

```yaml
api:
  url: http://manuals.local:8080
  key: your-api-key
log:
  level: info
  format: text
```

## Usage with Claude Code

Add to your Claude Code MCP configuration (`~/.claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "manuals": {
      "command": "/path/to/manuals-mcp",
      "args": ["serve"],
      "env": {
        "MANUALS_API_URL": "http://manuals.local:8080",
        "MANUALS_API_KEY": "your-api-key"
      }
    }
  }
}
```

## Available Tools

| Tool | Description |
|------|-------------|
| `search_manuals` | Full-text search across documentation |
| `get_device` | Get device details and content |
| `list_devices` | List all devices with optional filtering |
| `get_pinout` | Get GPIO pinout for a device |
| `get_specs` | Get device specifications |
| `list_documents` | List available documents |
| `get_status` | Get API status and statistics |

## Available Resources

| Resource | Description |
|----------|-------------|
| `manuals://device/{id}` | Device documentation |
| `manuals://device/{id}/pinout` | Device pinout information |

## Examples

Once configured with Claude Code, you can ask:

- "Search for ESP32 pinout information"
- "Get the specifications for the Raspberry Pi 4"
- "List all sensor devices"
- "What documents are available for the BME280?"

## Author

Robert Sigler (code@sigler.io)

## License

MIT License - see [LICENSE](LICENSE) file for details

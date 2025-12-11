# Manuals MCP Server

A Model Context Protocol (MCP) server for hardware and software documentation, providing GPIO pinout information, full-text search, and device specifications.

## Features

- **GPIO Pinout Information**: Query detailed pinout data for hardware devices (Raspberry Pi, ESP32, etc.)
- **Full-Text Search**: Search across hardware, software, and protocol documentation using SQLite FTS5
- **Device Specifications**: Access manufacturer details, specs, and categorized device information
- **Cross-Platform**: Single static binary for macOS (Intel/ARM), Linux, and Windows
- **No Dependencies**: Self-contained executable with embedded SQLite

## Installation

### Quick Install (Recommended)

The install script automatically detects your platform and downloads the correct binary:

```bash
# Using wget
wget -qO- https://raw.githubusercontent.com/rmrfslashbin/manuals-mcp/main/install.sh | sh

# Using curl
curl -fsSL https://raw.githubusercontent.com/rmrfslashbin/manuals-mcp/main/install.sh | sh
```

The binary will be installed in the current directory. Move it to your PATH:

```bash
sudo mv manuals-mcp /usr/local/bin/
# or
mv manuals-mcp ~/.local/bin/
```

### Via Binary Release

Download the appropriate binary for your platform from the [releases page](https://github.com/rmrfslashbin/manuals-mcp/releases):

```bash
# macOS ARM64 (Apple Silicon)
curl -L https://github.com/rmrfslashbin/manuals-mcp/releases/latest/download/manuals-mcp-darwin-arm64 -o manuals-mcp
chmod +x manuals-mcp

# macOS AMD64 (Intel)
curl -L https://github.com/rmrfslashbin/manuals-mcp/releases/latest/download/manuals-mcp-darwin-amd64 -o manuals-mcp
chmod +x manuals-mcp

# Linux AMD64
curl -L https://github.com/rmrfslashbin/manuals-mcp/releases/latest/download/manuals-mcp-linux-amd64 -o manuals-mcp
chmod +x manuals-mcp
```

### Build from Source

Requirements:
- Go 1.21 or later
- Git

```bash
git clone https://github.com/rmrfslashbin/manuals-mcp-server.git
cd manuals-mcp-server
make build
```

## Usage

### Configure MCP Client

Add to your MCP client configuration (e.g., Claude Desktop):

```json
{
  "mcpServers": {
    "manuals": {
      "command": "/path/to/manuals-mcp",
      "env": {
        "MANUALS_DB_PATH": "/path/to/manuals.db",
        "MANUALS_DOCS_PATH": "/path/to/documentation",
        "LOG_LEVEL": "info"
      }
    }
  }
}
```

### Index Documentation

Before using the server, index your documentation:

```bash
./manuals-mcp index \
  --docs-path /path/to/documentation \
  --db-path /path/to/manuals.db
```

### Available Tools

The server provides four MCP tools:

1. **get_pinout**: Get GPIO pinout information for a device
   ```json
   {
     "device": "raspberry-pi-4",
     "interface": "i2c"  // optional: i2c, spi, uart, pwm, gpio, all
   }
   ```

2. **search**: Full-text search across documentation
   ```json
   {
     "query": "temperature sensor",
     "domain": "hardware",  // optional: hardware, software, protocol, all
     "type": "sensors",     // optional
     "limit": 10            // optional, default: 10
   }
   ```

3. **list_hardware**: List all hardware devices organized by category

4. **get_stats**: Get documentation index statistics

## Configuration

### Environment Variables

- `MANUALS_DB_PATH`: Path to SQLite database (default: `./data/manuals.db`)
- `MANUALS_DOCS_PATH`: Path to documentation directory for indexing
- `LOG_LEVEL`: Logging level - `debug`, `info`, `warn`, `error` (default: `info`)
- `LOG_FORMAT`: Log format - `json` or `text` (default: `text`)
- `LOG_OUTPUT`: Log destination - `stderr`, `/path/to/file`, or `/path/to/dir/` (default: `stderr`)

### Command-Line Flags

```bash
./manuals-mcp --help
./manuals-mcp index --help
./manuals-mcp serve --help
```

## Development

### Requirements

- Go 1.21+
- golangci-lint (for linting)
- staticcheck (for static analysis)

### Build

```bash
make build          # Build for current platform
make build-all      # Build for all platforms
make test           # Run tests with coverage
make lint           # Run linters
make check          # Run all checks (vet, lint, test)
```

### Project Structure

```
manuals-mcp/
├── cmd/
│   └── manuals-mcp/     # Main application entry point
├── internal/
│   ├── db/              # SQLite schema and queries
│   ├── indexer/         # Documentation indexer
│   ├── mcp/             # MCP server implementation
│   └── tools/           # MCP tool handlers
├── pkg/
│   └── models/          # Shared data models
└── Makefile
```

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'feat: add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## Author

Robert Sigler (code@sigler.io)

## License

MIT License - see [LICENSE](LICENSE) file for details

## Acknowledgments

- Built with the [MCP Go SDK](https://github.com/mark3labs/mcp-go)
- Uses SQLite with FTS5 for full-text search

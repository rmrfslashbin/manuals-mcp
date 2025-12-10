# Configuration Guide

Manuals MCP Server supports multiple configuration sources with a clear priority order.

## Configuration Priority (Highest to Lowest)

1. **Command-line flags** (highest priority)
2. **Environment variables**
3. **.env file** (current directory or home directory)
4. **Config file** (~/.manuals-mcp.yaml or --config)
5. **Defaults** (lowest priority)

## Configuration Methods

### 1. Environment Variables

All configuration options can be set via environment variables with the `MANUALS_` prefix:

```bash
export MANUALS_DB_PATH=./data/manuals.db
export MANUALS_DOCS_PATH=../manuals-data
export MANUALS_LOG_LEVEL=debug
export MANUALS_LOG_FORMAT=json
export MANUALS_LOG_OUTPUT=/var/log/manuals/

manuals-mcp serve
```

**Available Environment Variables:**

| Variable | Description | Default |
|----------|-------------|---------|
| `MANUALS_DB_PATH` | Path to SQLite database | `./data/manuals.db` |
| `MANUALS_DOCS_PATH` | Path to documentation directory | (required for index) |
| `MANUALS_LOG_LEVEL` | Log level: debug, info, warn, error | `info` |
| `MANUALS_LOG_FORMAT` | Log format: text, json | `text` |
| `MANUALS_LOG_OUTPUT` | Log output: stderr, file, or directory | `stderr` |

### 2. .env File

Create a `.env` file in your project directory or home directory:

```bash
# .env
MANUALS_DB_PATH=./data/manuals.db
MANUALS_DOCS_PATH=../manuals-data
MANUALS_LOG_LEVEL=info
MANUALS_LOG_FORMAT=text
MANUALS_LOG_OUTPUT=stderr
```

**Search Path:**
1. `.env` in current directory (checked first)
2. `~/.env` in home directory (checked second)

**Example:**
```bash
# Copy example and customize
cp .env.example .env
nano .env

# Run server (uses .env automatically)
manuals-mcp serve
```

### 3. Config File

Create a YAML config file at `~/.manuals-mcp.yaml`:

```yaml
db:
  path: ./data/manuals.db
docs:
  path: ../manuals-data
log:
  level: info
  format: text
  output: stderr
```

Or specify a custom config file:

```bash
manuals-mcp serve --config=/path/to/config.yaml
```

### 4. Command-line Flags

Flags override all other configuration sources:

```bash
manuals-mcp serve \
  --db-path=./data/manuals.db \
  --docs-path=../manuals-data \
  --log-level=debug \
  --log-format=json \
  --log-output=/var/log/manuals/
```

## Common Configuration Scenarios

### Development

**.env:**
```bash
MANUALS_DB_PATH=./data/manuals.db
MANUALS_DOCS_PATH=../manuals-data
MANUALS_LOG_LEVEL=debug
MANUALS_LOG_FORMAT=text
```

### Production

**Environment variables (systemd, Docker, etc.):**
```bash
MANUALS_DB_PATH=/var/lib/manuals/manuals.db
MANUALS_DOCS_PATH=/usr/share/manuals-data
MANUALS_LOG_LEVEL=info
MANUALS_LOG_FORMAT=json
MANUALS_LOG_OUTPUT=/var/log/manuals/
```

### CI/CD

**Inline environment variables:**
```bash
MANUALS_DB_PATH=/tmp/test.db \
MANUALS_DOCS_PATH=./test-data \
MANUALS_LOG_LEVEL=warn \
  manuals-mcp serve
```

## Configuration Verification

View effective configuration:

```bash
# Show help with environment variable descriptions
manuals-mcp serve --help

# Run with debug logging to see loaded config
MANUALS_LOG_LEVEL=debug manuals-mcp serve
```

## Best Practices

1. **Development:** Use `.env` file for local development
2. **Production:** Use environment variables for deployments
3. **Secrets:** Never commit `.env` to version control
4. **Portability:** Use relative paths in `.env`, absolute paths in production
5. **Documentation:** Copy `.env.example` to `.env` and customize

## Troubleshooting

**Configuration not loading?**

1. Check file exists: `ls -la .env`
2. Verify permissions: `chmod 600 .env`
3. Enable debug logging: `MANUALS_LOG_LEVEL=debug manuals-mcp serve`
4. Check priority: CLI flags > env vars > .env > config file > defaults

**Environment variables not working?**

- Ensure `MANUALS_` prefix is present
- Use underscores: `MANUALS_DB_PATH` (not `MANUALS.DB.PATH`)
- Check for typos: `MANUALS_LOG_LEVEL` (not `MANUALS_LOGLEVEL`)

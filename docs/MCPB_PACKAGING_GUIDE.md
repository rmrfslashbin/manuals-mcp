# MCPB Packaging Guide for MCP Servers

This guide provides comprehensive instructions for packaging Model Context Protocol (MCP) servers using MCPB (Model Context Protocol Bundler) for one-click installation in Claude Desktop and other compatible clients.

## Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Understanding MCPB](#understanding-mcpb)
- [Project Setup](#project-setup)
- [Creating the Manifest](#creating-the-manifest)
- [Building Cross-Platform Binaries](#building-cross-platform-binaries)
- [Integrating with Build System](#integrating-with-build-system)
- [Testing and Validation](#testing-and-validation)
- [Distribution](#distribution)
- [Common Issues and Solutions](#common-issues-and-solutions)
- [Best Practices](#best-practices)

## Overview

MCPB packages (.mcpb files) are ZIP archives that bundle:
- MCP server binaries/code for multiple platforms
- Configuration metadata (manifest.json)
- Optional documentation and assets

Users can install these packages with a single click in Claude Desktop, which handles:
- Platform-specific binary selection
- User configuration (API keys, directories, etc.)
- Server registration and startup

## Prerequisites

### Required Tools

1. **MCPB CLI**: Install globally via npm
   ```bash
   npm install -g @anthropic-ai/mcpb
   ```

2. **Build tools** for your language:
   - **Go**: Go 1.20+ for cross-compilation
   - **Node.js**: Node.js 18+ and build tools
   - **Python**: Python 3.8+, virtualenv

3. **Version control**: Git for releases

### Knowledge Requirements

- Understanding of MCP server architecture
- Your server's tools and their parameters
- User configuration needs (API keys, file paths, etc.)
- Target platforms for distribution

## Understanding MCPB

### Package Structure

```
my-mcp-server.mcpb (ZIP archive)
├── manifest.json          # Required: Package metadata
├── server/                # Server files
│   ├── server-binary      # Unix/Linux/macOS executable (binary type)
│   ├── server-binary.exe  # Windows executable (binary type)
│   ├── main.js           # Entry point (node type)
│   ├── main.py           # Entry point (python type)
│   └── [dependencies]     # Any additional files
└── README.md              # Optional: Package documentation
```

### Manifest Schema Version

**Current version: 0.2** (as of this guide)

Always check the latest version:
```bash
mcpb validate --help
```

### Server Types

MCPB supports three server types:

1. **binary** - Native executables (Go, Rust, C++)
   - Best for: Performance-critical servers, minimal dependencies
   - Cross-compilation required for multiple platforms

2. **node** - Node.js/TypeScript servers
   - Best for: JavaScript ecosystem, maximum compatibility
   - Node.js ships with Claude Desktop (no user installation needed)

3. **python** - Python servers
   - Best for: Python ecosystem, rapid development
   - Requires Python installed on user's system

## Project Setup

### 1. Create MCPB Directory

Create a dedicated directory for MCPB package files:

```bash
mkdir -p mcpb/server
```

### 2. Update .gitignore

Exclude build artifacts but keep source files:

```gitignore
# MCPB build artifacts
mcpb/server/        # Built binaries
*.mcpb             # Or keep in bin/ directory if preferred
```

### 3. Directory Structure

Recommended project structure:

```
your-mcp-server/
├── mcpb/
│   ├── manifest.json      # Source-controlled
│   ├── README.md          # Source-controlled
│   └── server/            # .gitignored - built files
├── cmd/                   # Go example
├── internal/
├── src/                   # Node/Python example
├── Makefile              # Build automation
└── README.md             # Main documentation
```

## Creating the Manifest

### Manifest Template

Create `mcpb/manifest.json`:

```json
{
  "manifest_version": "0.2",
  "name": "your-mcp-server",
  "display_name": "Your MCP Server",
  "version": "1.0.0",
  "description": "Brief one-line description",
  "long_description": "Detailed description with features and capabilities.",

  "author": {
    "name": "Your Name",
    "email": "your@email.com"
  },

  "repository": {
    "type": "git",
    "url": "https://github.com/username/repo"
  },

  "homepage": "https://github.com/username/repo",
  "documentation": "https://github.com/username/repo#readme",
  "support": "https://github.com/username/repo/issues",
  "license": "MIT",

  "keywords": [
    "keyword1",
    "keyword2"
  ],

  "server": {
    "type": "binary",
    "entry_point": "server/your-server",
    "mcp_config": {
      "command": "${__dirname}/server/your-server",
      "args": ["server"],
      "env": {
        "YOUR_API_KEY": "${user_config.api_key}",
        "LOG_LEVEL": "${user_config.log_level}"
      }
    }
  },

  "user_config": {
    "api_key": {
      "type": "string",
      "title": "API Key",
      "description": "Your API key from https://example.com",
      "required": true
    },
    "log_level": {
      "type": "string",
      "title": "Log Level",
      "description": "Logging verbosity (debug, info, warn, error)",
      "required": false,
      "default": "info"
    }
  },

  "tools": [
    {
      "name": "tool_name",
      "description": "Tool description"
    }
  ],

  "tools_generated": false,
  "prompts_generated": false,

  "compatibility": {
    "platforms": ["darwin", "linux", "win32"]
  }
}
```

### Key Manifest Fields

#### Required Fields

- `manifest_version`: Always "0.2" (current version)
- `name`: Package identifier (lowercase, hyphens)
- `display_name`: Human-readable name
- `version`: Semantic version (e.g., "1.0.0")
- `description`: Brief description
- `server.type`: "binary", "node", or "python"
- `server.entry_point`: Path to executable/entry file

#### Server Configuration

**For Binary Servers:**
```json
{
  "server": {
    "type": "binary",
    "entry_point": "server/my-server",
    "mcp_config": {
      "command": "${__dirname}/server/my-server",
      "args": ["server"],
      "env": {
        "API_KEY": "${user_config.api_key}"
      }
    }
  }
}
```

**For Node.js Servers:**
```json
{
  "server": {
    "type": "node",
    "entry_point": "server/index.js",
    "mcp_config": {
      "command": "node",
      "args": ["${__dirname}/server/index.js"],
      "env": {
        "API_KEY": "${user_config.api_key}"
      }
    }
  }
}
```

**For Python Servers:**
```json
{
  "server": {
    "type": "python",
    "entry_point": "server/main.py",
    "mcp_config": {
      "command": "python",
      "args": ["${__dirname}/server/main.py"],
      "env": {
        "API_KEY": "${user_config.api_key}"
      }
    }
  }
}
```

#### User Configuration Types

MCPB supports several user configuration types:

```json
{
  "user_config": {
    "api_key": {
      "type": "string",
      "title": "API Key",
      "description": "Your API key",
      "required": true
    },
    "port": {
      "type": "number",
      "title": "Port",
      "description": "Server port (1024-65535)",
      "required": false,
      "default": 8080
    },
    "enabled": {
      "type": "boolean",
      "title": "Enable Feature",
      "description": "Enable optional feature",
      "required": false,
      "default": true
    },
    "workspace": {
      "type": "directory",
      "title": "Workspace Directory",
      "description": "Working directory for the server",
      "required": true
    },
    "config_file": {
      "type": "file",
      "title": "Configuration File",
      "description": "Path to config file",
      "required": false
    }
  }
}
```

**Important Notes:**
- Do NOT use `"secret": true` (not supported in manifest v0.2)
- Do NOT use `"enum"`, `"minimum"`, `"maximum"` (not supported)
- Use descriptions to indicate valid ranges/options
- All sensitive values (API keys) should use type "string"

#### Variable Substitution

MCPB provides these variables:

- `${__dirname}` - Bundle installation directory
- `${user_config.field_name}` - User-configured values
- `${HOME}` - User's home directory

#### Tools Declaration

**Simple declaration (recommended):**
```json
{
  "tools": [
    {
      "name": "tool_name",
      "description": "What the tool does"
    }
  ],
  "tools_generated": false
}
```

**If server generates tools dynamically:**
```json
{
  "tools": [],
  "tools_generated": true
}
```

Do NOT include `inputSchema` in the manifest - that's handled by your MCP server implementation.

#### Platform Compatibility

Use OS identifiers (not architecture-specific):

```json
{
  "compatibility": {
    "platforms": ["darwin", "linux", "win32"]
  }
}
```

**Platform values:**
- `"darwin"` - macOS (both Intel and ARM)
- `"linux"` - Linux (all architectures)
- `"win32"` - Windows

MCPB will automatically select the correct binary based on user's OS and architecture.

## Building Cross-Platform Binaries

### Go Servers

Go makes cross-compilation straightforward:

```bash
# Linux x64
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o mcpb/server/server-linux-amd64 .

# Linux ARM64
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o mcpb/server/server-linux-arm64 .

# macOS Intel
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o mcpb/server/server-darwin-amd64 .

# macOS Apple Silicon
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o mcpb/server/server-darwin-arm64 .

# Windows
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o mcpb/server/server.exe .

# Create unified Unix binary (choose one as base)
cp mcpb/server/server-darwin-arm64 mcpb/server/server
```

**Key points:**
- Use `CGO_ENABLED=0` for static linking (no external dependencies)
- Create a main binary (`server`) and platform-specific variants
- Windows binary MUST have `.exe` extension
- MCPB will auto-select correct binary for user's platform

### Node.js Servers

Node.js servers don't need platform-specific builds:

```bash
# Copy source files
cp -r src/* mcpb/server/
cp package.json mcpb/server/

# Install dependencies (production only)
cd mcpb/server
npm install --production
cd ../..
```

**Key points:**
- Include all source files and `package.json`
- Include `node_modules` (production dependencies only)
- No platform-specific builds needed
- Node.js runtime provided by Claude Desktop

### Python Servers

Python servers need dependencies bundled:

```bash
# Copy source files
cp -r src/* mcpb/server/
cp requirements.txt mcpb/server/

# Bundle dependencies
pip install -r requirements.txt -t mcpb/server/
```

**Key points:**
- Include all `.py` files
- Bundle dependencies in server directory
- User must have Python installed
- Consider using virtualenv approach for complex dependencies

## Integrating with Build System

### Makefile Example

Add MCPB targets to your Makefile:

```makefile
# MCPB configuration
MCPB_DIR := mcpb
MCPB_SERVER_DIR := $(MCPB_DIR)/server
MCPB_PACKAGE := bin/$(BINARY).mcpb

.PHONY: mcpb-build mcpb-validate mcpb-pack mcpb-info mcpb-clean mcpb-all

mcpb-validate: ## Validate MCPB manifest
	@echo "Validating manifest..."
	@which mcpb > /dev/null || (echo "Install: npm install -g @anthropic-ai/mcpb" && exit 1)
	mcpb validate $(MCPB_DIR)/manifest.json

mcpb-build: ## Build cross-platform binaries for MCPB
	@echo "Building MCPB binaries..."
	@mkdir -p $(MCPB_SERVER_DIR)
	# Add your build commands here
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $(MCPB_SERVER_DIR)/server-linux-amd64 .
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o $(MCPB_SERVER_DIR)/server-darwin-arm64 .
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o $(MCPB_SERVER_DIR)/server.exe .
	cp $(MCPB_SERVER_DIR)/server-darwin-arm64 $(MCPB_SERVER_DIR)/server

mcpb-pack: mcpb-validate mcpb-build ## Create MCPB package
	@echo "Creating package..."
	@mkdir -p bin
	mcpb pack $(MCPB_DIR) $(MCPB_PACKAGE)

mcpb-info: ## Display package info
	@test -f $(MCPB_PACKAGE) || (echo "Package not found. Run 'make mcpb-pack'" && exit 1)
	mcpb info $(MCPB_PACKAGE)

mcpb-clean: ## Clean MCPB artifacts
	rm -rf $(MCPB_SERVER_DIR)/*
	rm -f $(MCPB_PACKAGE)

mcpb-all: mcpb-clean mcpb-pack ## Build complete package
	@echo "Package ready!"
```

### GitHub Actions Example

Automate MCPB builds on release:

```yaml
name: Release

on:
  push:
    tags:
      - 'v*'

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Install MCPB
        run: npm install -g @anthropic-ai/mcpb

      - name: Build MCPB package
        run: make mcpb-all

      - name: Create Release
        uses: softprops/action-gh-release@v1
        with:
          files: |
            bin/*.mcpb
            bin/*-linux-amd64
            bin/*-darwin-amd64
            bin/*-darwin-arm64
            bin/*-windows-amd64.exe
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

## Testing and Validation

### 1. Validate Manifest

```bash
mcpb validate mcpb/manifest.json
```

**Common validation errors:**
- `Unrecognized key`: Using unsupported field (e.g., `"secret": true`)
- `Invalid enum value`: Wrong platform identifier (use "darwin", not "darwin-amd64")
- `Missing required field`: Check required fields

### 2. Build Package

```bash
mcpb pack mcpb output.mcpb
```

This creates a ZIP archive with all your files.

### 3. Inspect Package

```bash
mcpb info output.mcpb
```

Verify:
- Package size is reasonable
- All expected files are included
- Version matches your manifest

### 4. Manual Testing

```bash
# Extract and inspect
unzip -l output.mcpb

# Test binary execution
chmod +x mcpb/server/your-server
./mcpb/server/your-server --version
```

### 5. Test in Claude Desktop

1. Build the package
2. Open the `.mcpb` file in Claude Desktop
3. Verify configuration UI appears
4. Enter test values
5. Install and test all tools

## Distribution

### GitHub Releases

1. **Tag your release:**
   ```bash
   git tag -a v1.0.0 -m "Release v1.0.0"
   git push origin v1.0.0
   ```

2. **Build package:**
   ```bash
   make mcpb-all
   ```

3. **Create GitHub release:**
   ```bash
   gh release create v1.0.0 \
     bin/your-server.mcpb \
     bin/your-server-linux-amd64 \
     bin/your-server-darwin-amd64 \
     bin/your-server-darwin-arm64 \
     bin/your-server-windows-amd64.exe \
     --title "v1.0.0 - Your Server" \
     --notes "Release notes here"
   ```

### Release Checklist

- [ ] Manifest version updated
- [ ] Git tag created and pushed
- [ ] MCPB package built and tested
- [ ] Pre-compiled binaries for all platforms
- [ ] Release notes written
- [ ] GitHub release created with all assets
- [ ] README.md updated with installation instructions
- [ ] CHANGELOG.md updated (if applicable)

## Common Issues and Solutions

### Issue: "Manifest validation failed: Unrecognized key"

**Problem:** Using unsupported manifest fields

**Solution:** Remove unsupported fields:
- Remove `"secret": true` from user_config
- Remove `"enum"`, `"minimum"`, `"maximum"` from user_config
- Remove `"inputSchema"` from tools
- Put valid range info in `description` instead

### Issue: "Invalid enum value" for platforms

**Problem:** Using architecture-specific platform identifiers

**Solution:** Use OS-level identifiers:
```json
{
  "compatibility": {
    "platforms": ["darwin", "linux", "win32"]
  }
}
```

Not: `["darwin-amd64", "linux-amd64"]`

### Issue: Package size too large

**Problem:** Including unnecessary files

**Solutions:**
- Exclude test files and dev dependencies
- Use `.mcpbignore` file (like `.gitignore`)
- Strip debug symbols from binaries: `go build -ldflags="-s -w"`
- For Node.js: `npm install --production`

### Issue: Binary not found or permission denied

**Problem:** Incorrect paths or permissions

**Solutions:**
- Ensure binary is executable: `chmod +x server/your-server`
- Use correct path in manifest: `${__dirname}/server/your-server`
- Windows binary MUST have `.exe` extension
- Create unified binary for Unix: `cp server-darwin-arm64 server`

### Issue: User config not working

**Problem:** Environment variables not set correctly

**Solution:** Check variable substitution syntax:
```json
{
  "env": {
    "API_KEY": "${user_config.api_key}"
  }
}
```

Not: `"${user_config.api_key}"` as a string without env wrapper

### Issue: Tools not appearing

**Problem:** Tools not declared or incorrectly formatted

**Solution:**
```json
{
  "tools": [
    {
      "name": "tool_name",
      "description": "Description"
    }
  ],
  "tools_generated": false
}
```

Don't include `inputSchema` - that's in your server code, not manifest.

## Best Practices

### 1. Version Management

- Use semantic versioning (MAJOR.MINOR.PATCH)
- Update manifest version for every release
- Keep manifest version in sync with git tags
- Document breaking changes in CHANGELOG.md

### 2. Security

- **Never bundle API keys** in the package
- Always use `user_config` for sensitive values
- Document security best practices in README
- Use HTTPS URLs for all external resources
- Validate user inputs in your server code

### 3. User Experience

- Provide clear, helpful descriptions for all config fields
- Include sensible defaults where possible
- Document what API keys/credentials are needed
- Link to credential acquisition in descriptions
- Test the installation flow yourself

### 4. Documentation

Include in your package's README.md:
- Clear installation instructions (MCPB + manual)
- How to obtain required credentials
- List of all tools with examples
- Troubleshooting section
- Support/issue reporting links

### 5. Testing

- Test on multiple platforms if possible
- Validate manifest before every release
- Test with fresh Claude Desktop install
- Verify all tools work after MCPB installation
- Check error handling for missing configs

### 6. Binary Optimization

**For Go:**
```bash
# Strip symbols and reduce size
CGO_ENABLED=0 go build -ldflags="-s -w" -trimpath

# Add version info
go build -ldflags="-X main.version=1.0.0 -X main.commit=$(git rev-parse --short HEAD)"
```

**For Node.js:**
- Use `npm install --production`
- Consider bundling with webpack/esbuild for smaller size
- Exclude test files and documentation

**For Python:**
- Use `pip install --no-cache-dir`
- Consider PyInstaller for binary distribution
- Bundle only production requirements

### 7. Platform-Specific Considerations

**macOS:**
- Build for both Intel (amd64) and Apple Silicon (arm64)
- Test Gatekeeper compatibility
- Consider code signing for larger distributions

**Linux:**
- Static linking (`CGO_ENABLED=0`) for maximum compatibility
- Test on Ubuntu/Debian and RHEL/CentOS if possible
- Consider different libc versions (glibc vs musl)

**Windows:**
- Always use `.exe` extension
- Test on Windows 10/11
- Consider Windows Defender compatibility
- Handle path separators correctly

### 8. Continuous Integration

Set up automated builds:
- Validate manifest on every PR
- Build MCPB package on tags
- Run tests before packaging
- Automate GitHub release creation

### 9. Maintenance

- Monitor GitHub issues for installation problems
- Update dependencies regularly
- Keep MCPB CLI updated (`npm update -g @anthropic-ai/mcpb`)
- Test with new Claude Desktop versions
- Document known issues and workarounds

## Additional Resources

### Official Documentation

- **MCPB Repository**: https://github.com/anthropics/mcpb
- **Manifest Specification**: https://github.com/anthropics/mcpb/blob/main/MANIFEST.md
- **CLI Documentation**: https://github.com/anthropics/mcpb/blob/main/CLI.md
- **MCP Specification**: https://spec.modelcontextprotocol.io/

### Example Implementations

- **This project**: See `mcpb/` directory for working example
- **MCPB Examples**: https://github.com/anthropics/mcpb/tree/main/examples
- **MCP Servers**: https://github.com/modelcontextprotocol/servers

### Community

- **MCP Discussions**: GitHub Discussions in MCP repositories
- **Claude Help Center**: https://support.claude.com/
- **Issues**: Report MCPB issues at https://github.com/anthropics/mcpb/issues

## Quick Reference

### Essential Commands

```bash
# Validate manifest
mcpb validate mcpb/manifest.json

# Create package
mcpb pack mcpb output.mcpb

# Inspect package
mcpb info output.mcpb

# Extract package (for debugging)
unzip output.mcpb -d extracted/
```

### Manifest Checklist

- [ ] `manifest_version: "0.2"`
- [ ] `name` (lowercase, hyphens only)
- [ ] `display_name` (human-readable)
- [ ] `version` (semantic versioning)
- [ ] `description` (brief, one-line)
- [ ] `server.type` (binary/node/python)
- [ ] `server.entry_point` (correct path)
- [ ] `server.mcp_config.command` with `${__dirname}`
- [ ] `user_config` for API keys/settings
- [ ] `tools` array (name + description)
- [ ] `compatibility.platforms` (darwin/linux/win32)

### Build Checklist

- [ ] MCPB CLI installed (`npm install -g @anthropic-ai/mcpb`)
- [ ] Manifest validates successfully
- [ ] Binaries built for all target platforms
- [ ] Windows binary has `.exe` extension
- [ ] Unified binary created for Unix
- [ ] Dependencies included (for Node/Python)
- [ ] Package size reasonable (<100MB ideal)
- [ ] Package tested in Claude Desktop
- [ ] All tools work correctly
- [ ] Documentation updated

---

## Conclusion

MCPB packaging makes MCP server distribution seamless for end users. By following this guide, you can create professional, distributable packages that install with a single click.

Key takeaways:
1. Use manifest version 0.2
2. Keep manifest simple (avoid unsupported fields)
3. Build for multiple platforms
4. Test thoroughly before release
5. Document installation and usage
6. Automate with Makefile/CI

Good luck with your MCPB packaging! 🚀

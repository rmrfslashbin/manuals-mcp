# MCP Client Design

## Overview

The `manuals-mcp-client` is a thin adapter that translates Model Context Protocol (MCP) stdio communication into REST API calls. It runs on the user's workstation and connects to the remote `manuals-api` server.

## Purpose

- Provide MCP interface to remote REST API
- Enable Claude Code to access centralized documentation
- Avoid binary data through MCP protocol (use curl pattern)
- Maintain compatibility with existing MCP workflows

## Architecture

```
Claude Code (stdio)
       │
       │ MCP Protocol (stdin/stdout)
       │
       ▼
┌──────────────────────────────────┐
│  manuals-mcp-client              │
│                                  │
│  ┌────────────────────────────┐ │
│  │ MCP Server (stdio)         │ │
│  │ - Implements MCP protocol  │ │
│  │ - Handles tool calls       │ │
│  │ - Serves resources         │ │
│  │ - Returns prompts          │ │
│  └────────┬───────────────────┘ │
│           │                      │
│  ┌────────▼───────────────────┐ │
│  │ HTTP Client                │ │
│  │ - Makes REST API calls     │ │
│  │ - Handles errors/retries   │ │
│  │ - Optional caching         │ │
│  └────────┬───────────────────┘ │
│           │                      │
└───────────┼──────────────────────┘
            │
            │ HTTP/REST
            │
            ▼
    manuals-api (Raspberry Pi)
```

## Configuration

### Via CLI Flags
```bash
manuals-mcp-client serve \
  --api-url http://manuals.local:8080 \
  --api-key optional-api-key \
  --timeout 30s \
  --cache-ttl 5m \
  --log-level info
```

### Via Environment Variables
```bash
export MANUALS_API_URL="http://manuals.local:8080"
export MANUALS_API_KEY="optional-api-key"
export MANUALS_CLIENT_TIMEOUT="30s"
export MANUALS_CLIENT_CACHE_TTL="5m"
```

### Via .mcp.json (Claude Code)
```json
{
  "mcpServers": {
    "manuals": {
      "command": "/path/to/manuals-mcp-client",
      "args": ["serve"],
      "env": {
        "MANUALS_API_URL": "http://manuals.local:8080",
        "MANUALS_API_KEY": "",
        "LOG_LEVEL": "info"
      }
    }
  }
}
```

## MCP Tools

All tools map 1:1 to REST API endpoints:

### search
```go
MCP Tool: search(query, domain, type, limit)
→ POST /api/v1/search

func (c *Client) handleSearch(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    var params struct {
        Query  string `json:"query"`
        Domain string `json:"domain,omitempty"`
        Type   string `json:"type,omitempty"`
        Limit  int    `json:"limit,omitempty"`
    }
    json.Unmarshal(req.Params.Arguments, &params)

    // Make HTTP request
    resp, err := c.post("/api/v1/search", params)
    if err != nil {
        return mcp.NewToolResultError(err.Error()), nil
    }

    var results SearchResults
    json.NewDecoder(resp.Body).Decode(&results)

    // Format as markdown table
    output := formatSearchResults(results)
    return mcp.NewToolResultText(output), nil
}
```

### get_pinout
```go
MCP Tool: get_pinout(device, interface)
→ GET /api/v1/pinout/:device?interface=:interface

func (c *Client) handleGetPinout(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    device := req.Params.Arguments["device"]
    iface := req.Params.Arguments["interface"]

    url := fmt.Sprintf("/api/v1/pinout/%s", device)
    if iface != "" {
        url += "?interface=" + iface
    }

    resp, err := c.get(url)
    if err != nil {
        return mcp.NewToolResultError(err.Error()), nil
    }

    var pinout PinoutResponse
    json.NewDecoder(resp.Body).Decode(&pinout)

    output := formatPinoutTable(pinout)
    return mcp.NewToolResultText(output), nil
}
```

### list_hardware
```go
MCP Tool: list_hardware()
→ GET /api/v1/devices
```

### get_stats
```go
MCP Tool: get_stats()
→ GET /api/v1/stats
```

### get_info
```go
MCP Tool: get_info()
→ GET /api/v1/info

Note: Returns combined info (API server + MCP client versions)
```

### reindex
```go
MCP Tool: reindex()
→ POST /api/v1/reindex

Returns: Job ID and status URL for polling
```

### get_upload_command (NEW)
```go
MCP Tool: get_upload_command(device_id, file_path, file_type, description)

Returns: Formatted curl command for direct upload

func (c *Client) handleGetUploadCommand(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    deviceID := req.Params.Arguments["device_id"]
    filePath := req.Params.Arguments["file_path"]
    fileType := req.Params.Arguments["file_type"]
    desc := req.Params.Arguments["description"]

    // Generate curl command
    cmd := fmt.Sprintf(`curl -X POST %s/api/v1/devices/%s/documents \
  -F "file=@%s" \
  -F "type=%s"`, c.apiURL, deviceID, filePath, fileType)

    if desc != "" {
        cmd += fmt.Sprintf(` \
  -F "description=%s"`, desc)
    }

    cmd += ` \
  -F "auto_reindex=true"`

    output := fmt.Sprintf("# Upload Command\n\nExecute this command to upload the file:\n\n```bash\n%s\n```\n", cmd)
    return mcp.NewToolResultText(output), nil
}
```

## MCP Resources

### Device Documentation
```go
MCP Resource: manuals://device/:id
→ GET /api/v1/devices/:id
Accept: text/markdown

func (c *Client) handleDeviceResource(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
    deviceID := extractDeviceID(req.Params.URI)

    resp, err := c.get(fmt.Sprintf("/api/v1/devices/%s", deviceID))
    if err != nil {
        return nil, err
    }

    content, _ := io.ReadAll(resp.Body)

    return []mcp.ResourceContents{
        mcp.TextResourceContents{
            URI:      req.Params.URI,
            MIMEType: "text/markdown",
            Text:     string(content),
        },
    }, nil
}
```

### Guide Documentation
```go
MCP Resource: manuals://guide/:id
→ GET /api/v1/guides/:id

Examples:
  manuals://guide/quickstart
  manuals://guide/workflow
  manuals://guide/api-reference
```

### Pinout Diagram
```go
MCP Resource: manuals://pinout/:device
→ GET /api/v1/pinout/:device

Returns formatted pinout table as markdown
```

## MCP Prompts

### wiring-guide
```go
Prompt: wiring-guide(device, interface)

Returns: Template for creating wiring instructions
Uses: GET /api/v1/pinout/:device?interface=:interface
```

### device-compare
```go
Prompt: device-compare(device1, device2)

Returns: Template for comparing two devices
Uses: GET /api/v1/devices/:id for each device
```

### upload-document
```go
Prompt: upload-document(device_id, file_type)

Returns: Step-by-step instructions with curl command template
```

## Error Handling

### HTTP Errors
```go
func (c *Client) handleHTTPError(resp *http.Response) error {
    if resp.StatusCode >= 200 && resp.StatusCode < 300 {
        return nil
    }

    var apiError APIError
    json.NewDecoder(resp.Body).Decode(&apiError)

    switch resp.StatusCode {
    case 404:
        return fmt.Errorf("not found: %s", apiError.Error.Message)
    case 429:
        return fmt.Errorf("rate limited: retry in %ds", apiError.Error.RetryAfter)
    case 500:
        return fmt.Errorf("server error: %s", apiError.Error.Message)
    default:
        return fmt.Errorf("API error (%d): %s", resp.StatusCode, apiError.Error.Message)
    }
}
```

### Network Errors
```go
func (c *Client) get(path string) (*http.Response, error) {
    url := c.apiURL + path

    req, _ := http.NewRequest("GET", url, nil)
    req.Header.Set("User-Agent", fmt.Sprintf("manuals-mcp-client/%s", c.version))

    if c.apiKey != "" {
        req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
    }

    resp, err := c.httpClient.Do(req)
    if err != nil {
        // Check if it's a network error
        if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
            return nil, fmt.Errorf("request timeout: API server not responding")
        }
        return nil, fmt.Errorf("network error: %w", err)
    }

    return resp, c.handleHTTPError(resp)
}
```

### Graceful Degradation
```go
// If API is unreachable, inform user
func (c *Client) handleSearch(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    resp, err := c.post("/api/v1/search", params)
    if err != nil {
        // Provide helpful error message
        errMsg := fmt.Sprintf(`# API Unreachable

Could not connect to manuals-api at %s

Possible causes:
- Raspberry Pi is offline
- Network connectivity issue
- API server not running

Check:
1. Ping the server: ping manuals.local
2. Check API health: curl %s/api/v1/health
3. Verify API URL in configuration
`, c.apiURL, c.apiURL)

        return mcp.NewToolResultText(errMsg), nil
    }
    // ... continue normally
}
```

## Caching (Optional)

### In-Memory Cache
```go
type Cache struct {
    data sync.Map
    ttl  time.Duration
}

type CacheEntry struct {
    value     interface{}
    expiresAt time.Time
}

func (c *Client) cachedGet(path string) (*http.Response, error) {
    // Check cache
    if entry, ok := c.cache.data.Load(path); ok {
        cached := entry.(CacheEntry)
        if time.Now().Before(cached.expiresAt) {
            // Return cached response
            return cached.value.(*http.Response), nil
        }
    }

    // Make request
    resp, err := c.get(path)
    if err != nil {
        return nil, err
    }

    // Cache successful responses
    if resp.StatusCode == 200 {
        c.cache.data.Store(path, CacheEntry{
            value:     resp,
            expiresAt: time.Now().Add(c.cacheTTL),
        })
    }

    return resp, nil
}
```

### Cache Strategy
- **Device docs**: Cache for 5 minutes
- **Guides**: Cache for 10 minutes
- **Search results**: Don't cache (always fresh)
- **Pinout**: Cache for 5 minutes
- **Stats**: Cache for 1 minute

## Retry Logic

### Exponential Backoff
```go
func (c *Client) doWithRetry(req *http.Request) (*http.Response, error) {
    maxRetries := 3
    baseDelay := 100 * time.Millisecond

    for attempt := 0; attempt <= maxRetries; attempt++ {
        resp, err := c.httpClient.Do(req)

        if err == nil && resp.StatusCode < 500 {
            return resp, nil
        }

        if attempt == maxRetries {
            return resp, err
        }

        // Exponential backoff
        delay := baseDelay * time.Duration(1<<attempt)
        c.logger.Debug("retrying request", "attempt", attempt+1, "delay", delay)
        time.Sleep(delay)
    }

    return nil, fmt.Errorf("max retries exceeded")
}
```

## Logging

### Structured Logging
```go
c.logger.Info("making API request",
    "method", "POST",
    "path", "/api/v1/search",
    "query", params.Query,
)

c.logger.Error("API request failed",
    "method", "GET",
    "path", path,
    "status", resp.StatusCode,
    "error", err,
)
```

### Request ID Propagation
```go
func (c *Client) get(path string) (*http.Response, error) {
    req, _ := http.NewRequest("GET", c.apiURL+path, nil)

    // Generate request ID
    requestID := uuid.New().String()
    req.Header.Set("X-Request-ID", requestID)

    c.logger.Debug("API request", "request_id", requestID, "path", path)

    resp, err := c.httpClient.Do(req)
    // ...
}
```

## Testing

### Mock API Server
```go
func TestSearchTool(t *testing.T) {
    // Start mock API server
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        assert.Equal(t, "/api/v1/search", r.URL.Path)

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(SearchResults{
            Results: []Device{
                {ID: "ds18b20", Name: "DS18B20"},
            },
        })
    }))
    defer server.Close()

    // Create client
    client := NewClient(server.URL, "", nil)

    // Test tool
    result, err := client.handleSearch(context.Background(), mcp.CallToolRequest{
        Params: mcp.CallToolParams{
            Arguments: map[string]interface{}{
                "query": "temperature",
            },
        },
    })

    assert.NoError(t, err)
    assert.Contains(t, result.Content[0].Text, "DS18B20")
}
```

## Performance

### Target Metrics
- Tool call latency: <100ms (network + API)
- Resource fetch: <200ms
- Memory usage: <50MB
- Concurrent requests: Support 5-10 simultaneous

### Optimization
- HTTP/2 for connection reuse
- Request pipelining where possible
- Optional caching reduces API load
- Streaming for large responses

## Configuration File

### .manuals-mcp-client.yaml
```yaml
api:
  url: http://manuals.local:8080
  key: ""
  timeout: 30s

cache:
  enabled: true
  ttl: 5m

retry:
  max_attempts: 3
  base_delay: 100ms

logging:
  level: info
  format: text
  output: stderr
```

## Integration with Claude Code

### Example Session
```
User: Find I2C temperature sensors

Claude: [Calls search tool via MCP client]
MCP Client → POST http://manuals.local/api/v1/search
API → Returns results
MCP Client → Formats as markdown
Claude → Displays results to user

User: Upload the DS18B20 datasheet

Claude: [Calls get_upload_command tool]
MCP Client → Generates curl command
Claude → Executes via Bash tool:
  curl -X POST http://manuals.local/api/v1/devices/ds18b20/documents \
    -F "file=@~/Downloads/DS18B20.pdf" \
    -F "type=pdf" \
    -F "auto_reindex=true"

API → Uploads file, triggers reindex
Claude → Confirms success to user
```

## Project Structure

```
manuals-mcp-client/
├── cmd/
│   └── client/
│       └── main.go              # Entry point
├── internal/
│   ├── client/
│   │   ├── client.go            # HTTP client
│   │   ├── cache.go             # Optional caching
│   │   └── retry.go             # Retry logic
│   ├── mcp/
│   │   ├── server.go            # MCP server (stdio)
│   │   ├── tools.go             # Tool handlers
│   │   ├── resources.go         # Resource handlers
│   │   └── prompts.go           # Prompt templates
│   └── format/
│       └── markdown.go          # Response formatting
├── pkg/
│   └── api/
│       └── types.go             # API request/response types
└── go.mod
```

## Dependencies

```go
require (
    github.com/mark3labs/mcp-go v0.x.x    // MCP protocol
    github.com/google/uuid v1.x.x         // Request IDs
    github.com/spf13/cobra v1.x.x         // CLI
    github.com/spf13/viper v1.x.x         // Config
)
```

## Future Enhancements

1. **WebSocket Support**: For real-time reindex status updates
2. **Local Fallback**: Cache last known state for offline access
3. **Multi-Server**: Support multiple API endpoints (failover)
4. **Compression**: Gzip compression for large responses
5. **Metrics**: Prometheus metrics for monitoring

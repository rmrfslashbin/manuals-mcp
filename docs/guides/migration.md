# Migration Guide: v2.x → v3.x

## Overview

This guide helps you migrate from the current self-contained `manuals-mcp` (v2.x) to the new client-server architecture (v3.x) with REST API.

## Architecture Changes

### v2.x (Current)
```
Claude Code ↔ manuals-mcp (stdio) ↔ Local SQLite Database
```

- Single binary
- Local-only
- Stdio MCP protocol
- Fast (<5ms queries)
- One user at a time

### v3.x (New)
```
Claude Code ↔ manuals-mcp-client (stdio) ↔ HTTP ↔ manuals-api (Raspberry Pi) ↔ SQLite
```

- Client-server architecture
- Remote access
- REST API
- Network latency (10-50ms)
- Multiple concurrent users
- Web UI support

## Should You Migrate?

### Stick with v2.x if:
- ✅ Single user
- ✅ Need offline access
- ✅ Want fastest possible queries (<5ms)
- ✅ Happy with current workflow
- ✅ Don't need web UI
- ✅ Don't need remote access

### Migrate to v3.x if:
- ✅ Multiple users in lab
- ✅ Want centralized documentation
- ✅ Need web UI for browsing
- ✅ Want remote access
- ✅ Need to share database across machines
- ✅ Want API for integration with other tools

## Migration Strategies

### Strategy 1: Parallel Deployment (Recommended)

Run both v2.x and v3.x simultaneously during transition.

**Timeline**: 2-4 weeks

**Steps**:
1. Set up v3.x on Raspberry Pi
2. Copy database and documents to Pi
3. Test v3.x with MCP client on one machine
4. Gradually migrate users
5. Keep v2.x for offline scenarios

**Pros**:
- No downtime
- Easy rollback
- Users can transition at their pace

**Cons**:
- Need to sync changes between systems temporarily

### Strategy 2: Direct Migration

Replace v2.x with v3.x entirely.

**Timeline**: 1-2 days

**Steps**:
1. Set up v3.x on Raspberry Pi
2. Migrate database and documents
3. Update all .mcp.json configurations
4. Restart Claude Code sessions

**Pros**:
- Clean cutover
- Single source of truth immediately

**Cons**:
- Requires coordination across users
- No offline fallback during issues

### Strategy 3: Hybrid Deployment

Keep v2.x for laptops (offline), use v3.x when connected to lab network.

**Timeline**: Ongoing

**Steps**:
1. Set up v3.x on Raspberry Pi
2. Keep v2.x on laptops
3. Use different .mcp.json configs based on location
4. Sync changes periodically

**Pros**:
- Best of both worlds
- Offline capability maintained
- Remote access when available

**Cons**:
- More complex to manage
- Need sync strategy

## Step-by-Step Migration

### Phase 1: Prepare Infrastructure

#### 1.1 Set Up Raspberry Pi

```bash
# Update system
sudo apt update && sudo apt upgrade -y

# Install dependencies
sudo apt install -y nginx git sqlite3

# Create directories
sudo mkdir -p /data/{db,docs,logs}
sudo useradd -r -s /bin/false -m -d /opt/manuals manuals
sudo chown -R manuals:manuals /data
```

#### 1.2 Clone Documentation Repository

```bash
cd /data
sudo -u manuals git clone https://github.com/your-org/manuals-data.git docs
```

### Phase 2: Migrate Database

#### 2.1 Copy Current Database

From your current machine:
```bash
# Find current database location
grep db-path ~/.config/manuals/config.yaml

# Copy to Raspberry Pi
scp /path/to/manuals.db pi@raspberrypi.local:/tmp/
```

On Raspberry Pi:
```bash
# Move to data directory
sudo mv /tmp/manuals.db /data/db/manuals.db
sudo chown manuals:manuals /data/db/manuals.db

# Verify
sqlite3 /data/db/manuals.db "SELECT COUNT(*) FROM devices;"
```

#### 2.2 Migrate Documents

If you have local PDFs/images not in the manuals-data repository:

```bash
# From your machine
rsync -av /path/to/local/docs/ pi@raspberrypi.local:/data/docs/
```

On Raspberry Pi:
```bash
# Fix permissions
sudo chown -R manuals:manuals /data/docs

# Verify
ls -R /data/docs | grep -c ".pdf"
```

### Phase 3: Install and Configure API Server

#### 3.1 Install Binary

```bash
# Download latest release
wget https://github.com/rmrfslashbin/manuals-mcp/releases/latest/download/manuals-api-linux-arm64

# Install
sudo mv manuals-api-linux-arm64 /opt/manuals/manuals-api
sudo chmod +x /opt/manuals/manuals-api
sudo chown manuals:manuals /opt/manuals/manuals-api

# Verify
/opt/manuals/manuals-api version
```

#### 3.2 Create Systemd Service

Create `/etc/systemd/system/manuals-api.service`:

```ini
[Unit]
Description=Manuals API Server
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=manuals
Group=manuals

ExecStart=/opt/manuals/manuals-api serve \
    --db-path /data/db/manuals.db \
    --docs-path /data/docs \
    --port 8080 \
    --log-level info \
    --log-output /data/logs/manuals-api.log

Restart=always
RestartSec=5s

[Install]
WantedBy=multi-user.target
```

Start service:
```bash
sudo systemctl daemon-reload
sudo systemctl enable manuals-api
sudo systemctl start manuals-api
sudo systemctl status manuals-api
```

#### 3.3 Configure Nginx

Create `/etc/nginx/sites-available/manuals`:

```nginx
upstream manuals_api {
    server 127.0.0.1:8080;
}

server {
    listen 80;
    server_name manuals.local;

    location /api/ {
        proxy_pass http://manuals_api;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }

    location /api/v1/files/ {
        proxy_pass http://manuals_api;
        proxy_cache_valid 200 1h;
    }
}
```

Enable:
```bash
sudo ln -s /etc/nginx/sites-available/manuals /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

#### 3.4 Test API

```bash
# From Raspberry Pi
curl http://localhost/api/v1/health
curl http://localhost/api/v1/info
curl -X POST http://localhost/api/v1/search -H "Content-Type: application/json" -d '{"query": "temperature"}'
```

### Phase 4: Install MCP Client

#### 4.1 Download Client

On your workstation:
```bash
# Download appropriate version
# macOS ARM:
curl -L https://github.com/rmrfslashbin/manuals-mcp/releases/latest/download/manuals-mcp-client-darwin-arm64 -o manuals-mcp-client
chmod +x manuals-mcp-client

# Linux:
curl -L https://github.com/rmrfslashbin/manuals-mcp/releases/latest/download/manuals-mcp-client-linux-amd64 -o manuals-mcp-client
chmod +x manuals-mcp-client

# Move to PATH
sudo mv manuals-mcp-client /usr/local/bin/
```

#### 4.2 Test Client

```bash
# Test connection
manuals-mcp-client serve --api-url http://raspberrypi.local &

# In another terminal
echo '{"method":"tools/list"}' | manuals-mcp-client serve --api-url http://raspberrypi.local
```

### Phase 5: Update Claude Code Configuration

#### 5.1 Backup Current Config

```bash
cp ~/.config/claude/mcp.json ~/.config/claude/mcp.json.backup
```

#### 5.2 Update .mcp.json

Option A: Replace existing configuration
```json
{
  "mcpServers": {
    "manuals": {
      "command": "/usr/local/bin/manuals-mcp-client",
      "args": ["serve"],
      "env": {
        "MANUALS_API_URL": "http://raspberrypi.local",
        "LOG_LEVEL": "info"
      }
    }
  }
}
```

Option B: Add alongside existing (parallel deployment)
```json
{
  "mcpServers": {
    "manuals-local": {
      "command": "/path/to/manuals-mcp",
      "args": ["serve", "--db-path", "/path/to/manuals.db"]
    },
    "manuals-remote": {
      "command": "/usr/local/bin/manuals-mcp-client",
      "args": ["serve"],
      "env": {
        "MANUALS_API_URL": "http://raspberrypi.local"
      }
    }
  }
}
```

#### 5.3 Restart Claude Code

Close and reopen Claude Code to load new configuration.

### Phase 6: Verify Migration

#### 6.1 Test Basic Operations

In Claude Code:
```
User: Search for I2C temperature sensors
User: Show pinout for Raspberry Pi 4
User: List all hardware devices
```

#### 6.2 Test Document Upload

```
User: Upload this PDF datasheet for the DS18B20 sensor

Claude should:
1. Use get_upload_command tool
2. Execute curl via Bash tool
3. Confirm upload success
```

#### 6.3 Performance Check

```bash
# Measure query time
time curl -X POST http://raspberrypi.local/api/v1/search \
    -H "Content-Type: application/json" \
    -d '{"query": "temperature"}'

# Should be <100ms for typical queries
```

### Phase 7: Update Workflows

#### 7.1 Update Documentation

Add to your team wiki/docs:
- API URL: http://raspberrypi.local
- Web UI URL: (future)
- How to upload documents via curl
- Troubleshooting steps

#### 7.2 Create Upload Helper Script

Create `~/bin/upload-datasheet.sh`:
```bash
#!/bin/bash

DEVICE_ID="$1"
PDF_FILE="$2"

if [ -z "$DEVICE_ID" ] || [ -z "$PDF_FILE" ]; then
    echo "Usage: $0 <device-id> <pdf-file>"
    exit 1
fi

curl -X POST "http://raspberrypi.local/api/v1/devices/$DEVICE_ID/documents" \
    -F "file=@$PDF_FILE" \
    -F "type=pdf" \
    -F "auto_reindex=true"
```

Usage:
```bash
upload-datasheet.sh ds18b20 ~/Downloads/DS18B20.pdf
```

## Rollback Plan

### If Migration Fails

#### Quick Rollback (Minutes)

1. Revert .mcp.json:
```bash
cp ~/.config/claude/mcp.json.backup ~/.config/claude/mcp.json
```

2. Restart Claude Code

#### Partial Rollback (Use Local Binary)

Keep using v2.x while troubleshooting v3.x:
```json
{
  "mcpServers": {
    "manuals": {
      "command": "/usr/local/bin/manuals-mcp",
      "args": [
        "serve",
        "--db-path", "/path/to/manuals.db",
        "--docs-path", "/path/to/manuals-data"
      ]
    }
  }
}
```

## Common Issues and Solutions

### Issue 1: Cannot Connect to API

**Symptoms**: MCP client can't reach API server

**Solutions**:
```bash
# Check API is running
ssh pi@raspberrypi.local "systemctl status manuals-api"

# Check network connectivity
ping raspberrypi.local

# Check firewall
ssh pi@raspberrypi.local "sudo ufw status"

# Test directly
curl http://raspberrypi.local/api/v1/health
```

### Issue 2: Slow Queries

**Symptoms**: Queries take >1 second

**Solutions**:
```bash
# Check network latency
ping -c 10 raspberrypi.local

# Check API performance
curl -w "@curl-format.txt" http://raspberrypi.local/api/v1/search

# Enable caching in MCP client
export MANUALS_CLIENT_CACHE_TTL=5m
```

### Issue 3: Database Out of Sync

**Symptoms**: Search results don't match recent uploads

**Solutions**:
```bash
# Trigger manual reindex
curl -X POST http://raspberrypi.local/api/v1/reindex

# Check reindex status
curl http://raspberrypi.local/api/v1/reindex/status/<job-id>
```

### Issue 4: Upload Fails

**Symptoms**: Document upload returns error

**Solutions**:
```bash
# Check disk space
ssh pi@raspberrypi.local "df -h /data"

# Check file size limit
curl -I http://raspberrypi.local/api/v1/documents  # Check Content-Length header

# Check permissions
ssh pi@raspberrypi.local "ls -la /data/docs"
```

## Data Synchronization

### During Parallel Deployment

If running both v2.x and v3.x simultaneously, sync changes:

#### Daily Sync Script

```bash
#!/bin/bash

# Pull latest from Pi
rsync -av pi@raspberrypi.local:/data/docs/ ~/manuals-data/

# Reindex local database
manuals-mcp index --docs-path ~/manuals-data --db-path ~/manuals.db

echo "Sync complete: $(date)"
```

#### Auto-sync on Document Changes

Use `fswatch` or `inotify`:
```bash
# Install fswatch
brew install fswatch  # macOS

# Watch for changes
fswatch ~/manuals-data | while read file; do
    echo "Change detected: $file"
    rsync -av ~/manuals-data/ pi@raspberrypi.local:/data/docs/
    curl -X POST http://raspberrypi.local/api/v1/reindex
done
```

## Testing Checklist

Before completing migration, verify:

- [ ] API health endpoint returns 200
- [ ] Search returns expected results
- [ ] Device details load correctly
- [ ] Pinout data displays properly
- [ ] File uploads work
- [ ] File downloads work
- [ ] Reindex completes successfully
- [ ] Multiple users can query simultaneously
- [ ] Query performance is acceptable (<100ms)
- [ ] Backups are configured
- [ ] Monitoring is set up
- [ ] Team knows how to use new system

## Post-Migration Tasks

1. **Update Documentation**
   - Internal wiki with new URLs
   - Training materials
   - Troubleshooting guides

2. **Set Up Monitoring**
   - Configure health checks
   - Set up log rotation
   - Enable metrics collection

3. **Configure Backups**
   - Daily database backups
   - Weekly document backups
   - Test restore procedure

4. **Plan for v2.x Deprecation**
   - Set sunset date (if applicable)
   - Archive old databases
   - Update links/references

5. **Gather Feedback**
   - Survey team on new system
   - Identify pain points
   - Plan improvements

## Timeline Example

### Week 1: Preparation
- Monday: Set up Raspberry Pi hardware
- Tuesday: Install API server, configure systemd
- Wednesday: Configure nginx, test API
- Thursday: Migrate database and documents
- Friday: Verify API functionality

### Week 2: Client Deployment
- Monday: Install MCP client on test machine
- Tuesday: Test with Claude Code
- Wednesday: Document workflows
- Thursday: Train first users
- Friday: Gather feedback

### Week 3: Rollout
- Monday-Wednesday: Deploy to remaining users
- Thursday: Monitor and troubleshoot
- Friday: Review and optimize

### Week 4: Stabilization
- Monday: Address any issues
- Tuesday: Optimize performance
- Wednesday: Update documentation
- Thursday: Plan web UI development
- Friday: Retrospective

## Support

### Getting Help

- **GitHub Issues**: https://github.com/rmrfslashbin/manuals-mcp/issues
- **Documentation**: Check `/docs/architecture/` for technical details
- **Logs**: Review `/data/logs/manuals-api.log` on Pi

### Reporting Issues

Include:
1. API version (`curl http://raspberrypi.local/api/v1/info`)
2. Client version (`manuals-mcp-client version`)
3. Error messages from logs
4. Steps to reproduce
5. Expected vs actual behavior

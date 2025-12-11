# Deployment Architecture

## Overview

This document describes the deployment architecture for the manuals platform on a Raspberry Pi, including systemd services, nginx configuration, monitoring, and maintenance procedures.

## Target Hardware

### Raspberry Pi Specifications

**Recommended:**
- Raspberry Pi 4 Model B (4GB RAM minimum)
- Raspberry Pi 5 (8GB RAM)

**Storage:**
- microSD card: 64GB+ (for OS and applications)
- External SSD/USB drive (optional, recommended for database and documents)

**Network:**
- Ethernet connection (preferred for stability)
- WiFi acceptable for home lab

## System Architecture

```
Internet/LAN
     │
     ▼
┌─────────────────────────────────────────────────────┐
│ Raspberry Pi                                         │
│                                                      │
│  ┌────────────────────────────────────────────────┐ │
│  │ Nginx (Port 80/443)                            │ │
│  │ - Reverse proxy                                │ │
│  │ - SSL termination                              │ │
│  │ - Static file caching                          │ │
│  │ - Rate limiting                                │ │
│  │ - Access logging                               │ │
│  └──────────────────┬─────────────────────────────┘ │
│                     │                                │
│  ┌──────────────────▼─────────────────────────────┐ │
│  │ manuals-api (Port 8080)                        │ │
│  │ - Go HTTP server                               │ │
│  │ - Systemd service                              │ │
│  │ - Auto-restart on failure                      │ │
│  └──────────────────┬─────────────────────────────┘ │
│                     │                                │
│  ┌──────────────────▼─────────────────────────────┐ │
│  │ SQLite Database                                │ │
│  │ /data/manuals.db (WAL mode)                    │ │
│  └────────────────────────────────────────────────┘ │
│                                                      │
│  ┌────────────────────────────────────────────────┐ │
│  │ Filesystem (docs-path)                         │ │
│  │ /data/manuals-data/                            │ │
│  │ - Git repository (manuals-data)                │ │
│  │ - Source PDFs, markdown, images                │ │
│  └────────────────────────────────────────────────┘ │
│                                                      │
│  ┌────────────────────────────────────────────────┐ │
│  │ Monitoring & Logging                           │ │
│  │ - Journald (systemd logs)                      │ │
│  │ - Nginx access/error logs                      │ │
│  │ - API logs → /var/log/manuals-api/             │ │
│  └────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────┘
```

## Installation Steps

### 1. Prepare Raspberry Pi

```bash
# Update system
sudo apt update && sudo apt upgrade -y

# Install dependencies
sudo apt install -y nginx git sqlite3

# Create application user
sudo useradd -r -s /bin/false -m -d /opt/manuals manuals

# Create data directories
sudo mkdir -p /data/{db,docs,logs}
sudo chown -R manuals:manuals /data
```

### 2. Clone Documentation Repository

```bash
# Clone manuals-data repository
cd /data
sudo -u manuals git clone https://github.com/your-org/manuals-data.git docs
```

### 3. Install manuals-api Binary

```bash
# Download latest release
cd /tmp
wget https://github.com/rmrfslashbin/manuals-mcp/releases/latest/download/manuals-api-linux-arm64

# Install binary
sudo mv manuals-api-linux-arm64 /opt/manuals/manuals-api
sudo chmod +x /opt/manuals/manuals-api
sudo chown manuals:manuals /opt/manuals/manuals-api

# Verify
/opt/manuals/manuals-api version
```

### 4. Configure Systemd Service

Create `/etc/systemd/system/manuals-api.service`:

```ini
[Unit]
Description=Manuals API Server
Documentation=https://github.com/rmrfslashbin/manuals-mcp
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=manuals
Group=manuals

# Environment variables
Environment="MANUALS_DB_PATH=/data/db/manuals.db"
Environment="MANUALS_DOCS_PATH=/data/docs"
Environment="LOG_LEVEL=info"
Environment="LOG_FORMAT=json"
Environment="LOG_OUTPUT=/data/logs/manuals-api.log"

# Command
ExecStart=/opt/manuals/manuals-api serve \
    --db-path /data/db/manuals.db \
    --docs-path /data/docs \
    --port 8080 \
    --log-level info \
    --log-format json \
    --log-output /data/logs/manuals-api.log

# Restart policy
Restart=always
RestartSec=5s

# Security hardening
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/data

# Resource limits
LimitNOFILE=65536
MemoryLimit=512M

[Install]
WantedBy=multi-user.target
```

Enable and start:
```bash
sudo systemctl daemon-reload
sudo systemctl enable manuals-api
sudo systemctl start manuals-api
sudo systemctl status manuals-api
```

### 5. Configure Nginx

Create `/etc/nginx/sites-available/manuals`:

```nginx
# Upstream API server
upstream manuals_api {
    server 127.0.0.1:8080 fail_timeout=5s max_fails=3;
}

# Rate limiting zones
limit_req_zone $binary_remote_addr zone=api_limit:10m rate=10r/s;
limit_req_zone $binary_remote_addr zone=upload_limit:10m rate=5r/m;

# Logging format
log_format manuals_api '$remote_addr - $remote_user [$time_local] '
                       '"$request" $status $body_bytes_sent '
                       '"$http_referer" "$http_user_agent" '
                       'rt=$request_time uct="$upstream_connect_time" '
                       'uht="$upstream_header_time" urt="$upstream_response_time"';

server {
    listen 80;
    server_name manuals.local;

    # Logs
    access_log /var/log/nginx/manuals-access.log manuals_api;
    error_log /var/log/nginx/manuals-error.log warn;

    # Client settings
    client_max_body_size 50M;
    client_body_timeout 60s;

    # Compression
    gzip on;
    gzip_types text/plain text/css application/json application/javascript text/xml application/xml text/markdown;
    gzip_min_length 1000;

    # API endpoints
    location /api/ {
        # Rate limiting
        limit_req zone=api_limit burst=20 nodelay;

        # Proxy settings
        proxy_pass http://manuals_api;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # Timeouts
        proxy_connect_timeout 5s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;

        # Buffering
        proxy_buffering off;
    }

    # Upload endpoints (stricter rate limiting)
    location /api/v1/documents {
        limit_req zone=upload_limit burst=2 nodelay;

        proxy_pass http://manuals_api;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;

        # Longer timeouts for uploads
        proxy_send_timeout 300s;
        proxy_read_timeout 300s;

        # Disable buffering for uploads
        proxy_request_buffering off;
    }

    # Static files (cached)
    location /api/v1/files/ {
        proxy_pass http://manuals_api;
        proxy_http_version 1.1;
        proxy_set_header Host $host;

        # Caching
        proxy_cache static_cache;
        proxy_cache_valid 200 1h;
        proxy_cache_use_stale error timeout updating http_500 http_502 http_503 http_504;
        add_header X-Cache-Status $upstream_cache_status;

        # Timeouts
        proxy_connect_timeout 5s;
        proxy_read_timeout 30s;
    }

    # Health check (no rate limiting)
    location /api/v1/health {
        proxy_pass http://manuals_api;
        access_log off;
    }

    # Web UI (future)
    location / {
        root /var/www/manuals-web;
        try_files $uri $uri/ /index.html;

        # Caching for static assets
        location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg)$ {
            expires 1y;
            add_header Cache-Control "public, immutable";
        }
    }
}

# Cache configuration
proxy_cache_path /var/cache/nginx/manuals levels=1:2 keys_zone=static_cache:10m max_size=500m inactive=24h;
```

Enable site:
```bash
sudo ln -s /etc/nginx/sites-available/manuals /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

## SSL/TLS Configuration

### Option 1: Let's Encrypt (Public Domain)

```bash
# Install certbot
sudo apt install -y certbot python3-certbot-nginx

# Obtain certificate
sudo certbot --nginx -d manuals.example.com

# Auto-renewal
sudo systemctl enable certbot.timer
```

Update nginx config:
```nginx
server {
    listen 443 ssl http2;
    server_name manuals.example.com;

    ssl_certificate /etc/letsencrypt/live/manuals.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/manuals.example.com/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;

    # ... rest of config
}

# Redirect HTTP to HTTPS
server {
    listen 80;
    server_name manuals.example.com;
    return 301 https://$server_name$request_uri;
}
```

### Option 2: Self-Signed Certificate (Local Network)

```bash
# Generate certificate
sudo openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
    -keyout /etc/ssl/private/manuals-selfsigned.key \
    -out /etc/ssl/certs/manuals-selfsigned.crt \
    -subj "/CN=manuals.local"

# Update nginx
ssl_certificate /etc/ssl/certs/manuals-selfsigned.crt;
ssl_certificate_key /etc/ssl/private/manuals-selfsigned.key;
```

### Option 3: mDNS (Local Network Discovery)

```bash
# Install avahi
sudo apt install -y avahi-daemon

# Enable service
sudo systemctl enable avahi-daemon
sudo systemctl start avahi-daemon

# Access via http://raspberrypi.local/
```

## Monitoring

### Systemd Journal Logs

```bash
# View API logs
sudo journalctl -u manuals-api -f

# View recent errors
sudo journalctl -u manuals-api -p err -n 50

# View logs since boot
sudo journalctl -u manuals-api -b
```

### Application Logs

```bash
# Tail API logs
tail -f /data/logs/manuals-api.log

# View with jq (if JSON format)
tail -f /data/logs/manuals-api.log | jq .

# Search for errors
grep -i error /data/logs/manuals-api.log
```

### Nginx Logs

```bash
# Access log
tail -f /var/log/nginx/manuals-access.log

# Error log
tail -f /var/log/nginx/manuals-error.log

# Analyze traffic
cat /var/log/nginx/manuals-access.log | \
    awk '{print $1}' | sort | uniq -c | sort -rn | head -10
```

### Health Checks

```bash
# API health
curl http://localhost/api/v1/health

# Nginx status
sudo systemctl status nginx

# API service status
sudo systemctl status manuals-api

# Database size
ls -lh /data/db/manuals.db

# Disk usage
df -h /data
```

### Automated Monitoring Script

Create `/opt/manuals/monitor.sh`:

```bash
#!/bin/bash

LOG_FILE="/data/logs/monitor.log"

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" >> "$LOG_FILE"
}

# Check API health
if ! curl -sf http://localhost/api/v1/health > /dev/null; then
    log "ERROR: API health check failed"
    sudo systemctl restart manuals-api
    log "INFO: Restarted manuals-api service"
fi

# Check disk space
DISK_USAGE=$(df -h /data | tail -1 | awk '{print $5}' | sed 's/%//')
if [ "$DISK_USAGE" -gt 90 ]; then
    log "WARNING: Disk usage at ${DISK_USAGE}%"
fi

# Check database size
DB_SIZE=$(stat -f%z /data/db/manuals.db 2>/dev/null)
if [ "$DB_SIZE" -gt 1000000000 ]; then  # 1GB
    log "INFO: Database size: $(($DB_SIZE / 1024 / 1024))MB"
fi

log "INFO: Monitoring check complete"
```

Add to cron:
```bash
sudo chmod +x /opt/manuals/monitor.sh
sudo crontab -e

# Add line:
*/5 * * * * /opt/manuals/monitor.sh
```

## Backup Strategy

### Database Backup

Create `/opt/manuals/backup-db.sh`:

```bash
#!/bin/bash

BACKUP_DIR="/backup/manuals"
DB_PATH="/data/db/manuals.db"
TIMESTAMP=$(date +%Y%m%d-%H%M%S)
BACKUP_FILE="$BACKUP_DIR/manuals-$TIMESTAMP.db"

mkdir -p "$BACKUP_DIR"

# Backup using SQLite backup command
sqlite3 "$DB_PATH" ".backup '$BACKUP_FILE'"

# Compress
gzip "$BACKUP_FILE"

# Keep only last 7 days
find "$BACKUP_DIR" -name "manuals-*.db.gz" -mtime +7 -delete

echo "Backup complete: $BACKUP_FILE.gz"
```

Add to cron:
```bash
# Daily at 2 AM
0 2 * * * /opt/manuals/backup-db.sh
```

### Document Backup

```bash
#!/bin/bash

BACKUP_DIR="/backup/manuals-data"
DOCS_PATH="/data/docs"
TIMESTAMP=$(date +%Y%m%d)

mkdir -p "$BACKUP_DIR"

# Rsync to backup location
rsync -av --delete "$DOCS_PATH/" "$BACKUP_DIR/"

# Create tarball
tar -czf "$BACKUP_DIR-$TIMESTAMP.tar.gz" -C /backup manuals-data

# Keep only last 30 days
find /backup -name "manuals-data-*.tar.gz" -mtime +30 -delete
```

### Remote Backup (S3)

```bash
# Install rclone
sudo apt install -y rclone

# Configure S3
rclone config

# Sync to S3
rclone sync /backup/manuals s3:my-bucket/manuals-backup
```

### Litestream (Continuous Replication)

Install Litestream:
```bash
wget https://github.com/benbjohnson/litestream/releases/latest/download/litestream-linux-arm64.deb
sudo dpkg -i litestream-linux-arm64.deb
```

Configure `/etc/litestream.yml`:
```yaml
dbs:
  - path: /data/db/manuals.db
    replicas:
      - url: s3://my-bucket/manuals-db
        sync-interval: 10s
```

Enable service:
```bash
sudo systemctl enable litestream
sudo systemctl start litestream
```

## Maintenance

### Update API Binary

```bash
# Download new version
cd /tmp
wget https://github.com/rmrfslashbin/manuals-mcp/releases/download/v3.1.0/manuals-api-linux-arm64

# Stop service
sudo systemctl stop manuals-api

# Backup current binary
sudo cp /opt/manuals/manuals-api /opt/manuals/manuals-api.backup

# Install new binary
sudo mv manuals-api-linux-arm64 /opt/manuals/manuals-api
sudo chmod +x /opt/manuals/manuals-api
sudo chown manuals:manuals /opt/manuals/manuals-api

# Start service
sudo systemctl start manuals-api

# Verify
curl http://localhost/api/v1/info | jq .version
```

### Update Documentation

```bash
# Pull latest changes
cd /data/docs
sudo -u manuals git pull origin main

# Trigger reindex
curl -X POST http://localhost/api/v1/reindex
```

### Database Optimization

```bash
# Weekly maintenance
sqlite3 /data/db/manuals.db "PRAGMA optimize;"
sqlite3 /data/db/manuals.db "VACUUM;"

# Analyze query performance
sqlite3 /data/db/manuals.db ".eqp on" "SELECT * FROM devices WHERE domain='hardware';"
```

### Log Rotation

Configure `/etc/logrotate.d/manuals`:

```
/data/logs/manuals-api.log {
    daily
    rotate 14
    compress
    delaycompress
    notifempty
    create 0640 manuals manuals
    sharedscripts
    postrotate
        systemctl reload manuals-api
    endscript
}

/var/log/nginx/manuals-*.log {
    daily
    rotate 14
    compress
    delaycompress
    notifempty
    create 0640 www-data adm
    sharedscripts
    postrotate
        systemctl reload nginx
    endscript
}
```

## Troubleshooting

### API Not Responding

```bash
# Check service status
sudo systemctl status manuals-api

# Check logs
sudo journalctl -u manuals-api -n 50

# Check port binding
sudo netstat -tlnp | grep 8080

# Restart service
sudo systemctl restart manuals-api
```

### Database Corruption

```bash
# Check integrity
sqlite3 /data/db/manuals.db "PRAGMA integrity_check;"

# If corrupted, restore from backup
sudo systemctl stop manuals-api
cp /backup/manuals/manuals-latest.db.gz /tmp/
gunzip /tmp/manuals-latest.db.gz
sudo mv /tmp/manuals-latest.db /data/db/manuals.db
sudo chown manuals:manuals /data/db/manuals.db
sudo systemctl start manuals-api
```

### High Memory Usage

```bash
# Check memory usage
free -h

# Check API memory
sudo ps aux | grep manuals-api

# Restart to clear memory
sudo systemctl restart manuals-api
```

### Slow Queries

```bash
# Enable query logging in SQLite
sqlite3 /data/db/manuals.db "PRAGMA query_only = ON;"

# Analyze slow queries
sqlite3 /data/db/manuals.db ".eqp full" "SELECT * FROM devices_fts WHERE devices_fts MATCH 'temperature';"
```

## Performance Tuning

### SQLite Optimizations

```sql
-- WAL mode for better concurrency
PRAGMA journal_mode = WAL;

-- Increase cache size (64MB)
PRAGMA cache_size = -64000;

-- Memory for temporary tables
PRAGMA temp_store = MEMORY;

-- Normal synchronous (faster, still safe)
PRAGMA synchronous = NORMAL;
```

### Nginx Tuning

```nginx
# Worker processes (match CPU cores)
worker_processes auto;

# Worker connections
events {
    worker_connections 1024;
}

# Keepalive
keepalive_timeout 65;
keepalive_requests 100;

# Buffers
proxy_buffer_size 128k;
proxy_buffers 4 256k;
proxy_busy_buffers_size 256k;
```

### API Service Tuning

Adjust systemd service limits:
```ini
[Service]
LimitNOFILE=65536
MemoryLimit=1G
CPUQuota=200%  # 2 cores max
```

## Security Hardening

### Firewall Configuration

```bash
# Install ufw
sudo apt install -y ufw

# Allow SSH
sudo ufw allow 22/tcp

# Allow HTTP/HTTPS
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp

# Enable firewall
sudo ufw enable
```

### Fail2ban

```bash
# Install fail2ban
sudo apt install -y fail2ban

# Configure
sudo cat > /etc/fail2ban/jail.local <<EOF
[nginx-limit-req]
enabled = true
filter = nginx-limit-req
logpath = /var/log/nginx/manuals-error.log
maxretry = 5
findtime = 60
bantime = 3600
EOF

sudo systemctl restart fail2ban
```

## High Availability (Optional)

### Load Balancer Setup

Multiple Raspberry Pis behind load balancer:

```nginx
upstream manuals_cluster {
    least_conn;
    server pi1.local:80 max_fails=3 fail_timeout=30s;
    server pi2.local:80 max_fails=3 fail_timeout=30s;
}

server {
    listen 80;
    location / {
        proxy_pass http://manuals_cluster;
    }
}
```

### Shared Storage

Use NFS for shared document storage:

```bash
# On NFS server
sudo apt install -y nfs-kernel-server
echo "/data/docs *(rw,sync,no_subtree_check)" | sudo tee -a /etc/exports
sudo exportfs -ra

# On API servers
sudo apt install -y nfs-common
sudo mount nfs-server:/data/docs /data/docs
```

## Future Enhancements

1. **Containerization**: Docker/Podman deployment
2. **Kubernetes**: For larger deployments
3. **Prometheus Metrics**: Detailed monitoring
4. **Grafana Dashboards**: Visualization
5. **Auto-scaling**: Based on load
6. **CDN Integration**: For static files
7. **Database Replication**: Read replicas

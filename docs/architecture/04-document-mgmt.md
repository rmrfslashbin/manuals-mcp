# Document Management

## Overview

The document management system handles the lifecycle of documentation files (PDFs, markdown, images) including upload, storage, retrieval, deletion, and reindexing.

## Document Types

### Source Documents
Files added to the system by users:

- **PDF**: Datasheets, specifications, manuals
- **Markdown**: Device documentation, guides
- **Images**: Pinout diagrams, photos, schematics (PNG, JPG, SVG)

### Derivative Documents (Future)
Generated from source documents:

- **Thumbnails**: Preview images for PDFs/images
- **Extracted Text**: Full-text from PDFs for search
- **HTML**: Rendered markdown

## Storage Architecture

### Hybrid Storage Model

```
┌──────────────────────────────────────────┐
│ Document Metadata (SQLite)               │
│ - ID, filename, type, size               │
│ - Device association                     │
│ - Upload timestamp                       │
│ - Storage location reference             │
└──────────────────────────────────────────┘
                 │
                 ├─────────────────────────┐
                 ▼                         ▼
    ┌────────────────────┐    ┌────────────────────┐
    │ Small Files        │    │ Large Files        │
    │ (<1MB)             │    │ (>1MB)             │
    │                    │    │                    │
    │ SQLite BLOB        │    │ Filesystem         │
    │ - Images           │    │ - PDFs             │
    │ - Small markdown   │    │ - Videos           │
    └────────────────────┘    │ - Large images     │
                              └────────────────────┘
```

### Storage Decision Logic

```go
func determineStorage(file File) StorageType {
    if file.Size < 1*1024*1024 { // 1MB
        return StorageBlob
    }
    return StorageFilesystem
}
```

## Document Lifecycle

### 1. Upload

#### Flow
```
User/API → Upload Request
    ↓
Validate File (type, size, virus scan)
    ↓
Generate Document ID
    ↓
Calculate Checksum (SHA256)
    ↓
Determine Storage (blob vs filesystem)
    ↓
Store File
    ↓
Insert Metadata → Database
    ↓
Trigger Reindex (if auto_reindex=true)
    ↓
Return Document ID
```

#### API Request
```http
POST /api/v1/devices/:device_id/documents
Content-Type: multipart/form-data

file: <binary>
type: pdf|markdown|image
description: "Optional description"
auto_reindex: true
```

#### Implementation
```go
func (s *Server) handleUploadDocument(w http.ResponseWriter, r *http.Request) {
    deviceID := chi.URLParam(r, "device_id")

    // Parse multipart form
    r.ParseMultipartForm(50 << 20) // 50MB max
    file, header, err := r.FormFile("file")
    if err != nil {
        http.Error(w, "File required", 400)
        return
    }
    defer file.Close()

    // Validate device exists
    if !s.deviceExists(deviceID) {
        http.Error(w, "Device not found", 404)
        return
    }

    // Validate file type
    fileType := r.FormValue("type")
    if !validFileType(fileType) {
        http.Error(w, "Invalid file type", 400)
        return
    }

    // Read file content
    content, err := io.ReadAll(file)
    if err != nil {
        http.Error(w, "Failed to read file", 500)
        return
    }

    // Validate MIME type
    mimeType := http.DetectContentType(content)
    if !allowedMimeType(mimeType, fileType) {
        http.Error(w, "MIME type mismatch", 400)
        return
    }

    // Calculate checksum
    checksum := fmt.Sprintf("sha256:%x", sha256.Sum256(content))

    // Generate document ID
    docID := generateDocumentID(deviceID, header.Filename)

    // Determine storage
    var storagePath string
    var contentBlob []byte

    if len(content) < 1*1024*1024 {
        // Store in database
        contentBlob = content
    } else {
        // Store on filesystem
        storagePath = filepath.Join(
            s.docsPath,
            "hardware", // derive from device
            "sensors",  // derive from device
            deviceID,
            header.Filename,
        )

        os.MkdirAll(filepath.Dir(storagePath), 0755)
        if err := os.WriteFile(storagePath, content, 0644); err != nil {
            http.Error(w, "Failed to save file", 500)
            return
        }
    }

    // Insert metadata
    doc := Document{
        ID:          docID,
        DeviceID:    deviceID,
        Filename:    header.Filename,
        Type:        fileType,
        MIMEType:    mimeType,
        SizeBytes:   int64(len(content)),
        Checksum:    checksum,
        Description: r.FormValue("description"),
        StorageType: determineStorageType(len(content)),
        StoragePath: storagePath,
        Content:     contentBlob,
        UploadedAt:  time.Now().Unix(),
    }

    if err := s.db.InsertDocument(doc); err != nil {
        http.Error(w, "Failed to save metadata", 500)
        return
    }

    // Trigger reindex if requested
    autoReindex := r.FormValue("auto_reindex") == "true"
    if autoReindex {
        go s.reindexDevice(deviceID)
    }

    // Return response
    w.WriteHeader(201)
    json.NewEncoder(w).Encode(DocumentResponse{
        ID:              doc.ID,
        DeviceID:        doc.DeviceID,
        Filename:        doc.Filename,
        Type:            doc.Type,
        SizeBytes:       doc.SizeBytes,
        UploadedAt:      doc.UploadedAt,
        ReindexTriggered: autoReindex,
    })
}
```

### 2. Retrieval

#### Get Metadata
```http
GET /api/v1/documents/:doc_id

Response:
{
  "id": "ds18b20-datasheet-pdf",
  "device_id": "ds18b20",
  "filename": "DS18B20.pdf",
  "type": "pdf",
  "mime_type": "application/pdf",
  "size_bytes": 392186,
  "uploaded_at": "2025-12-10T14:00:00Z"
}
```

#### Download Content
```http
GET /api/v1/documents/:doc_id/content

Response: (binary stream)
Content-Type: application/pdf
Content-Disposition: attachment; filename="DS18B20.pdf"
Content-Length: 392186
```

#### Implementation
```go
func (s *Server) handleDownloadDocument(w http.ResponseWriter, r *http.Request) {
    docID := chi.URLParam(r, "doc_id")

    // Get document metadata
    doc, err := s.db.GetDocument(docID)
    if err != nil {
        http.Error(w, "Document not found", 404)
        return
    }

    // Set headers
    w.Header().Set("Content-Type", doc.MIMEType)
    w.Header().Set("Content-Disposition",
        fmt.Sprintf("attachment; filename=\"%s\"", doc.Filename))
    w.Header().Set("Content-Length", strconv.FormatInt(doc.SizeBytes, 10))

    // Serve content
    if doc.StorageType == "blob" {
        // Serve from database
        w.Write(doc.Content)
    } else {
        // Serve from filesystem
        fullPath := filepath.Join(s.docsPath, doc.StoragePath)
        http.ServeFile(w, r, fullPath)
    }
}
```

### 3. Update

#### Flow
```
User/API → Update Request
    ↓
Validate Document Exists
    ↓
Validate New File
    ↓
Calculate New Checksum
    ↓
Update Storage (replace file)
    ↓
Update Metadata → Database
    ↓
Trigger Reindex (if auto_reindex=true)
    ↓
Return Updated Metadata
```

#### API Request
```http
PUT /api/v1/documents/:doc_id
Content-Type: multipart/form-data

file: <binary>
auto_reindex: true
```

### 4. Delete

#### Flow
```
User/API → Delete Request
    ↓
Validate Document Exists
    ↓
Delete from Storage (blob or filesystem)
    ↓
Delete Metadata → Database
    ↓
Trigger Reindex (if reindex=true)
    ↓
Return 204 No Content
```

#### API Request
```http
DELETE /api/v1/documents/:doc_id?reindex=true

Response: 204 No Content
```

#### Implementation
```go
func (s *Server) handleDeleteDocument(w http.ResponseWriter, r *http.Request) {
    docID := chi.URLParam(r, "doc_id")

    // Get document metadata
    doc, err := s.db.GetDocument(docID)
    if err != nil {
        http.Error(w, "Document not found", 404)
        return
    }

    // Delete from storage
    if doc.StorageType == "filesystem" {
        fullPath := filepath.Join(s.docsPath, doc.StoragePath)
        if err := os.Remove(fullPath); err != nil {
            s.logger.Warn("failed to delete file", "path", fullPath, "error", err)
        }
    }

    // Delete metadata
    if err := s.db.DeleteDocument(docID); err != nil {
        http.Error(w, "Failed to delete", 500)
        return
    }

    // Trigger reindex if requested
    if r.URL.Query().Get("reindex") == "true" {
        go s.reindexDevice(doc.DeviceID)
    }

    w.WriteHeader(204)
}
```

## Reindexing

### Trigger Conditions

1. **Document Upload**: When `auto_reindex=true`
2. **Document Update**: When `auto_reindex=true`
3. **Document Delete**: When `reindex=true` query parameter
4. **Manual Trigger**: Via `POST /api/v1/reindex`
5. **Scheduled**: Daily full reindex (cron job)

### Reindex Strategies

#### Full Reindex
Reprocesses all devices and documents.

```http
POST /api/v1/reindex
{
  "clear": true,
  "devices": []  // empty = all devices
}
```

```go
func (s *Server) handleReindex(w http.ResponseWriter, r *http.Request) {
    var req ReindexRequest
    json.NewDecoder(r.Body).Decode(&req)

    // Start reindex in background
    jobID := uuid.New().String()
    go s.performReindex(jobID, req)

    // Return job ID immediately
    w.WriteHeader(202)
    json.NewEncoder(w).Encode(ReindexResponse{
        Status: "accepted",
        JobID:  jobID,
        Message: "Reindex started",
    })
}

func (s *Server) performReindex(jobID string, req ReindexRequest) {
    s.logger.Info("starting reindex", "job_id", jobID)

    startTime := time.Now()

    // Clear database if requested
    if req.Clear {
        s.db.ClearDatabase()
    }

    // Run indexer
    result, err := indexer.IndexDocumentation(s.db, indexer.IndexOptions{
        DocsPath: s.docsPath,
        Clear:    req.Clear,
        Devices:  req.Devices,
    }, s.logger)

    duration := time.Since(startTime)

    // Store job result
    s.storeJobResult(jobID, JobResult{
        Status:      "completed",
        StartedAt:   startTime,
        CompletedAt: time.Now(),
        Duration:    duration,
        Result:      result,
    })

    s.logger.Info("reindex completed",
        "job_id", jobID,
        "duration", duration,
        "files", result.TotalFiles,
    )
}
```

#### Partial Reindex
Reprocesses specific devices only.

```http
POST /api/v1/reindex
{
  "devices": ["ds18b20", "bme280"]
}
```

#### Incremental Reindex (Future)
Detects changed files and only reprocesses those.

```go
func (s *Server) incrementalReindex() {
    // Track file modification times
    lastIndexed := s.getLastIndexTime()

    changedFiles := s.findChangedFiles(s.docsPath, lastIndexed)

    for _, file := range changedFiles {
        s.reindexFile(file)
    }
}
```

## File Validation

### MIME Type Validation
```go
var allowedMimeTypes = map[string][]string{
    "pdf": {
        "application/pdf",
    },
    "markdown": {
        "text/markdown",
        "text/plain",
    },
    "image": {
        "image/png",
        "image/jpeg",
        "image/svg+xml",
    },
}

func validateMimeType(detected, declared string) bool {
    allowed, ok := allowedMimeTypes[declared]
    if !ok {
        return false
    }

    for _, mime := range allowed {
        if strings.HasPrefix(detected, mime) {
            return true
        }
    }

    return false
}
```

### Size Limits
```go
const (
    MaxPDFSize      = 50 * 1024 * 1024  // 50MB
    MaxImageSize    = 10 * 1024 * 1024  // 10MB
    MaxMarkdownSize = 1 * 1024 * 1024   // 1MB
)

func validateSize(fileType string, size int64) error {
    switch fileType {
    case "pdf":
        if size > MaxPDFSize {
            return fmt.Errorf("PDF exceeds 50MB limit")
        }
    case "image":
        if size > MaxImageSize {
            return fmt.Errorf("Image exceeds 10MB limit")
        }
    case "markdown":
        if size > MaxMarkdownSize {
            return fmt.Errorf("Markdown exceeds 1MB limit")
        }
    }
    return nil
}
```

### Virus Scanning (Optional)
```go
func scanForVirus(content []byte) error {
    // Integration with ClamAV or similar
    cmd := exec.Command("clamdscan", "-")
    cmd.Stdin = bytes.NewReader(content)

    if err := cmd.Run(); err != nil {
        return fmt.Errorf("virus detected or scan failed")
    }

    return nil
}
```

## Static File Serving

### Direct Filesystem Access
```http
GET /api/v1/files/hardware/sensors/ds18b20/DS18B20.pdf
```

```go
func (s *Server) handleStaticFile(w http.ResponseWriter, r *http.Request) {
    // Extract path from URL
    filePath := chi.URLParam(r, "*")

    // Security: prevent path traversal
    fullPath := filepath.Join(s.docsPath, filePath)
    if !strings.HasPrefix(fullPath, s.docsPath) {
        http.Error(w, "Invalid path", 400)
        return
    }

    // Check file exists
    info, err := os.Stat(fullPath)
    if err != nil {
        http.Error(w, "File not found", 404)
        return
    }

    // Don't serve directories
    if info.IsDir() {
        http.Error(w, "Cannot serve directory", 400)
        return
    }

    // Serve with caching headers
    w.Header().Set("Cache-Control", "public, max-age=3600")
    http.ServeFile(w, r, fullPath)
}
```

## Document Relationships

### Linking Documents to Devices
Documents are always associated with a device via `device_id` foreign key.

```sql
SELECT d.id, d.filename, d.type, d.size_bytes
FROM documents d
WHERE d.device_id = 'ds18b20'
ORDER BY d.uploaded_at DESC;
```

### Document Tags (Future)
```sql
CREATE TABLE document_tags (
    document_id TEXT NOT NULL,
    tag TEXT NOT NULL,
    PRIMARY KEY (document_id, tag),
    FOREIGN KEY (document_id) REFERENCES documents(id) ON DELETE CASCADE
);

-- Find all datasheets
SELECT DISTINCT d.*
FROM documents d
JOIN document_tags dt ON d.id = dt.document_id
WHERE dt.tag = 'datasheet';
```

## Bulk Operations

### Bulk Upload
```http
POST /api/v1/documents/bulk
Content-Type: multipart/form-data

files[]: <multiple files>
manifest: {
  "files": [
    {"filename": "DS18B20.pdf", "device_id": "ds18b20", "type": "pdf"},
    {"filename": "BME280.pdf", "device_id": "bme280", "type": "pdf"}
  ]
}
```

### Bulk Delete
```http
DELETE /api/v1/documents/bulk
Content-Type: application/json

{
  "document_ids": [
    "ds18b20-old-datasheet",
    "bme280-old-datasheet"
  ],
  "reindex": true
}
```

## Derivative Generation (Future)

### PDF Thumbnail
```go
func generatePDFThumbnail(pdfPath string) ([]byte, error) {
    // Use ghostscript or similar
    cmd := exec.Command("gs",
        "-dNOPAUSE", "-dBATCH", "-sDEVICE=pngalpha",
        "-r72", "-dFirstPage=1", "-dLastPage=1",
        "-sOutputFile=-", pdfPath,
    )

    return cmd.Output()
}
```

### PDF Text Extraction
```go
func extractPDFText(pdfPath string) (string, error) {
    // Use pdftotext or similar
    cmd := exec.Command("pdftotext", pdfPath, "-")
    output, err := cmd.Output()
    return string(output), err
}
```

## Monitoring

### Metrics to Track
```go
type DocumentMetrics struct {
    TotalDocuments     int64
    TotalSizeBytes     int64
    DocumentsByType    map[string]int64
    AverageUploadSize  int64
    UploadsLast24h     int64
    ReindexJobsRunning int
}

func (s *Server) getDocumentMetrics() DocumentMetrics {
    // Query database for metrics
    // ...
}
```

### Health Checks
```go
func (s *Server) checkDocumentHealth() HealthStatus {
    // Check filesystem access
    if _, err := os.Stat(s.docsPath); err != nil {
        return HealthStatus{Status: "unhealthy", Error: "docs path inaccessible"}
    }

    // Check database connection
    if err := s.db.Ping(); err != nil {
        return HealthStatus{Status: "unhealthy", Error: "database unreachable"}
    }

    // Check disk space
    var stat syscall.Statfs_t
    syscall.Statfs(s.docsPath, &stat)
    available := stat.Bavail * uint64(stat.Bsize)
    if available < 1*1024*1024*1024 { // Less than 1GB
        return HealthStatus{Status: "degraded", Warning: "low disk space"}
    }

    return HealthStatus{Status: "healthy"}
}
```

## Error Handling

### Common Error Scenarios

1. **File Too Large**: Return 413 Payload Too Large
2. **Invalid Type**: Return 400 Bad Request with details
3. **Device Not Found**: Return 404 Not Found
4. **Duplicate File**: Return 409 Conflict
5. **Storage Full**: Return 507 Insufficient Storage
6. **Virus Detected**: Return 422 Unprocessable Entity

### Error Response Format
```json
{
  "error": {
    "code": "FILE_TOO_LARGE",
    "message": "File exceeds maximum size of 50MB",
    "details": {
      "size_bytes": 52428800,
      "max_size_bytes": 52428800,
      "filename": "large-datasheet.pdf"
    }
  }
}
```

## Best Practices

1. **Always validate file types** by content (magic bytes), not just extension
2. **Calculate checksums** to detect duplicate files
3. **Use streaming** for large file uploads/downloads
4. **Implement rate limiting** on upload endpoints
5. **Set appropriate cache headers** for static files
6. **Log all document operations** for audit trail
7. **Handle partial uploads** gracefully
8. **Provide progress feedback** for long operations
9. **Clean up orphaned files** periodically
10. **Back up regularly** (both database and filesystem)

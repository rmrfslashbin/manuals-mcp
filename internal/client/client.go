// Package client provides an HTTP client for the Manuals REST API.
package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"time"
)

const (
	// APIVersion is the API version to use.
	APIVersion = "2025.12"
)

// Client is an HTTP client for the Manuals API.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// New creates a new API client.
func New(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SearchResult represents a search result.
type SearchResult struct {
	DeviceID string  `json:"device_id"`
	Name     string  `json:"name"`
	Domain   string  `json:"domain"`
	Type     string  `json:"type"`
	Path     string  `json:"path"`
	Score    float64 `json:"score"`
	Snippet  string  `json:"snippet"`
}

// SearchResponse is the response from the search endpoint.
type SearchResponse struct {
	Results []SearchResult `json:"results"`
	Total   int            `json:"total"`
	Query   string         `json:"query"`
}

// Device represents a device.
type Device struct {
	ID        string                 `json:"id"`
	Domain    string                 `json:"domain"`
	Type      string                 `json:"type"`
	Name      string                 `json:"name"`
	Path      string                 `json:"path"`
	Content   string                 `json:"content,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	IndexedAt string                 `json:"indexed_at"`
}

// DevicesResponse is the response from the devices list endpoint.
type DevicesResponse struct {
	Data   []Device `json:"data"`
	Total  int      `json:"total"`
	Limit  int      `json:"limit"`
	Offset int      `json:"offset"`
}

// Document represents a document.
type Document struct {
	ID        string `json:"id"`
	DeviceID  string `json:"device_id"`
	Path      string `json:"path"`
	Filename  string `json:"filename"`
	MimeType  string `json:"mime_type"`
	SizeBytes int64  `json:"size_bytes"`
	Checksum  string `json:"checksum"`
	IndexedAt string `json:"indexed_at"`
}

// DocumentsResponse is the response from the documents list endpoint.
type DocumentsResponse struct {
	Data   []Document `json:"data"`
	Total  int        `json:"total"`
	Limit  int        `json:"limit"`
	Offset int        `json:"offset"`
}

// PinoutPin represents a single pin in a pinout.
type PinoutPin struct {
	PhysicalPin  int      `json:"physical_pin"`
	GPIONum      *int     `json:"gpio_num,omitempty"`
	Name         string   `json:"name"`
	DefaultPull  string   `json:"default_pull,omitempty"`
	AltFunctions []string `json:"alt_functions,omitempty"`
	Description  string   `json:"description,omitempty"`
}

// PinoutResponse is the response from the pinout endpoint.
type PinoutResponse struct {
	DeviceID string      `json:"device_id"`
	Name     string      `json:"name"`
	Pins     []PinoutPin `json:"pins"`
}

// SpecsResponse is the response from the specs endpoint.
type SpecsResponse struct {
	DeviceID string            `json:"device_id"`
	Name     string            `json:"name"`
	Specs    map[string]string `json:"specs"`
}

// StatusResponse is the response from the status endpoint.
type StatusResponse struct {
	Status      string `json:"status"`
	APIVersion  string `json:"api_version"`
	Version     string `json:"version"`
	DBPath      string `json:"db_path"`
	LastReindex string `json:"last_reindex,omitempty"`
	Counts      struct {
		Devices   int `json:"devices"`
		Documents int `json:"documents"`
		Users     int `json:"users"`
	} `json:"counts"`
}

// ErrorResponse is an API error response.
type ErrorResponse struct {
	Error string `json:"error"`
}

// User represents the current authenticated user.
type User struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Role       string `json:"role"`
	CreatedAt  string `json:"created_at"`
	LastSeenAt string `json:"last_seen_at"`
	IsActive   bool   `json:"is_active"`
}

// MeResponse is the response from the /me endpoint.
type MeResponse struct {
	User User `json:"user"`
}

// ReindexResponse is the response from triggering a reindex.
type ReindexResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// ReindexStatusResponse is the response from the reindex status endpoint.
type ReindexStatusResponse struct {
	Status        string `json:"status"`
	StartedAt     string `json:"started_at,omitempty"`
	Elapsed       string `json:"elapsed,omitempty"`
	LastCompleted string `json:"last_completed,omitempty"`
	LastRun       *struct {
		DevicesIndexed   int    `json:"devices_indexed"`
		DocumentsIndexed int    `json:"documents_indexed"`
		Errors           int    `json:"errors"`
		Duration         string `json:"duration"`
	} `json:"last_run,omitempty"`
}

// UsersResponse is the response from the list users endpoint.
type UsersResponse struct {
	Users []User `json:"users"`
	Count int    `json:"count"`
}

// CreateUserRequest is the request to create a new user.
type CreateUserRequest struct {
	Name string `json:"name"`
	Role string `json:"role"`
}

// CreateUserResponse is the response from creating a user.
type CreateUserResponse struct {
	User    User   `json:"user"`
	APIKey  string `json:"api_key"`
	Message string `json:"message"`
}

// UploadResponse is the response from uploading a file.
type UploadResponse struct {
	Message  string `json:"message"`
	Path     string `json:"path"`
	Size     int64  `json:"size"`
	Filename string `json:"filename"`
}

// SyncResponse is the response from triggering a git sync.
type SyncResponse struct {
	Status       string `json:"status"`
	Message      string `json:"message,omitempty"`
	Commit       string `json:"commit,omitempty"`
	FilesChanged int    `json:"files_changed,omitempty"`
	Branch       string `json:"branch,omitempty"`
	Error        string `json:"error,omitempty"`
}

// RotateKeyResponse is the response from rotating an API key.
type RotateKeyResponse struct {
	APIKey  string `json:"api_key"`
	Message string `json:"message"`
}

// Setting represents a configuration setting.
type Setting struct {
	Key       string `json:"key"`
	Value     string `json:"value"`
	UpdatedAt string `json:"updated_at"`
}

// SettingsResponse is the response from the settings endpoint.
type SettingsResponse struct {
	Settings []Setting `json:"settings"`
}

// Reference represents a device reference (related device or external link).
type Reference struct {
	Type  string `json:"type"`
	Title string `json:"title"`
	URL   string `json:"url,omitempty"`
	ID    string `json:"id,omitempty"`
}

// RefsResponse is the response from the device refs endpoint.
type RefsResponse struct {
	DeviceID   string      `json:"device_id"`
	Name       string      `json:"name"`
	References []Reference `json:"references"`
}

// Guide represents a documentation guide.
type Guide struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Path      string `json:"path"`
	Content   string `json:"content,omitempty"`
	IndexedAt string `json:"indexed_at"`
}

// GuidesResponse is the response from the guides list endpoint.
type GuidesResponse struct {
	Data   []Guide `json:"data"`
	Total  int     `json:"total"`
	Limit  int     `json:"limit"`
	Offset int     `json:"offset"`
}

// SemanticSearchResult represents a result from semantic search.
type SemanticSearchResult struct {
	DeviceID string  `json:"device_id"`
	Name     string  `json:"name"`
	Domain   string  `json:"domain"`
	Type     string  `json:"type"`
	Heading  string  `json:"heading"`
	Content  string  `json:"content"`
	Score    float32 `json:"score"`
}

// SemanticSearchResponse is the response from the semantic search endpoint.
type SemanticSearchResponse struct {
	Query   string                 `json:"query"`
	Count   int                    `json:"count"`
	Results []SemanticSearchResult `json:"results"`
}

// Search searches for devices using keyword/FTS5 search.
func (c *Client) Search(query string, limit int, domain, deviceType string) (*SearchResponse, error) {
	params := url.Values{}
	params.Set("q", query)
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%d", limit))
	}
	if domain != "" {
		params.Set("domain", domain)
	}
	if deviceType != "" {
		params.Set("type", deviceType)
	}

	var resp SearchResponse
	if err := c.get("/search?"+params.Encode(), &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// SemanticSearch performs semantic/vector search using embeddings.
// Returns results ranked by semantic similarity to the query.
func (c *Client) SemanticSearch(query string, limit int, domain, deviceType string) (*SemanticSearchResponse, error) {
	params := url.Values{}
	params.Set("q", query)
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%d", limit))
	}
	if domain != "" {
		params.Set("domain", domain)
	}
	if deviceType != "" {
		params.Set("type", deviceType)
	}

	var resp SemanticSearchResponse
	if err := c.get("/search/semantic?"+params.Encode(), &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListDevices lists devices with pagination.
func (c *Client) ListDevices(limit, offset int, domain, deviceType string) (*DevicesResponse, error) {
	params := url.Values{}
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%d", limit))
	}
	if offset > 0 {
		params.Set("offset", fmt.Sprintf("%d", offset))
	}
	if domain != "" {
		params.Set("domain", domain)
	}
	if deviceType != "" {
		params.Set("type", deviceType)
	}

	path := "/devices"
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	var resp DevicesResponse
	if err := c.get(path, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetDevice gets a device by ID.
func (c *Client) GetDevice(id string, includeContent bool) (*Device, error) {
	path := "/devices/" + id
	if includeContent {
		path += "?content=true"
	}
	var resp Device
	if err := c.get(path, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetDevicePinout gets the pinout for a device.
func (c *Client) GetDevicePinout(id string) (*PinoutResponse, error) {
	var resp PinoutResponse
	if err := c.get("/devices/"+id+"/pinout", &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetDeviceSpecs gets the specifications for a device.
func (c *Client) GetDeviceSpecs(id string) (*SpecsResponse, error) {
	var resp SpecsResponse
	if err := c.get("/devices/"+id+"/specs", &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListDocuments lists documents with pagination.
func (c *Client) ListDocuments(limit, offset int, deviceID string) (*DocumentsResponse, error) {
	params := url.Values{}
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%d", limit))
	}
	if offset > 0 {
		params.Set("offset", fmt.Sprintf("%d", offset))
	}
	if deviceID != "" {
		params.Set("device_id", deviceID)
	}

	path := "/documents"
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	var resp DocumentsResponse
	if err := c.get(path, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetStatus gets the API status.
func (c *Client) GetStatus() (*StatusResponse, error) {
	var resp StatusResponse
	if err := c.get("/status", &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetMe gets the current authenticated user.
// Returns nil if not authenticated (anonymous mode).
func (c *Client) GetMe() (*User, error) {
	if c.apiKey == "" {
		return nil, nil
	}
	var resp MeResponse
	if err := c.get("/me", &resp); err != nil {
		return nil, err
	}
	return &resp.User, nil
}

// HasAPIKey returns true if an API key is configured.
func (c *Client) HasAPIKey() bool {
	return c.apiKey != ""
}

// GetAPIURL returns the configured API URL.
func (c *Client) GetAPIURL() string {
	return c.baseURL
}

// TriggerReindex triggers a reindex of the documentation.
// Requires RW or Admin role.
func (c *Client) TriggerReindex() (*ReindexResponse, error) {
	var resp ReindexResponse
	if err := c.post("/rw/reindex", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetReindexStatus gets the current reindex status.
// Requires RW or Admin role.
func (c *Client) GetReindexStatus() (*ReindexStatusResponse, error) {
	var resp ReindexStatusResponse
	if err := c.get("/rw/reindex/status", &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// TriggerSync triggers a git sync to push documentation changes.
// Requires RW or Admin role.
func (c *Client) TriggerSync() (*SyncResponse, error) {
	var resp SyncResponse
	if err := c.post("/rw/sync", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// UploadFile uploads a file to the documentation storage.
// Requires RW or Admin role.
func (c *Client) UploadFile(destPath string, filename string, content []byte) (*UploadResponse, error) {
	// Create multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add the path field
	if err := writer.WriteField("path", destPath); err != nil {
		return nil, fmt.Errorf("failed to write path field: %w", err)
	}

	// Add the file
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err := part.Write(content); err != nil {
		return nil, fmt.Errorf("failed to write file content: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// Create request
	req, err := http.NewRequest("POST", c.baseURL+"/api/"+APIVersion+"/rw/upload", &buf)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		var errResp ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
		}
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, errResp.Error)
	}

	var result UploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// ListUsers lists all users.
// Requires Admin role.
func (c *Client) ListUsers() (*UsersResponse, error) {
	var resp UsersResponse
	if err := c.get("/admin/users", &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// CreateUser creates a new user.
// Requires Admin role.
func (c *Client) CreateUser(name, role string) (*CreateUserResponse, error) {
	req := CreateUserRequest{Name: name, Role: role}
	var resp CreateUserResponse
	if err := c.post("/admin/users", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// DeleteUser deletes a user by ID.
// Requires Admin role.
func (c *Client) DeleteUser(id string) error {
	var resp map[string]string
	if err := c.delete("/admin/users/"+id, &resp); err != nil {
		return err
	}
	return nil
}

// UpdateUserRole updates a user's role.
// Requires Admin role.
func (c *Client) UpdateUserRole(id, role string) error {
	req := map[string]string{"role": role}
	return c.put("/admin/users/"+id+"/role", req)
}

// RotateAPIKey rotates a user's API key and returns the new key.
// Requires Admin role.
func (c *Client) RotateAPIKey(id string) (*RotateKeyResponse, error) {
	var resp RotateKeyResponse
	if err := c.post("/admin/users/"+id+"/rotate-key", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListSettings lists all settings.
// Requires Admin role.
func (c *Client) ListSettings() (*SettingsResponse, error) {
	var resp SettingsResponse
	if err := c.get("/admin/settings", &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// UpdateSetting updates a setting value.
// Requires Admin role.
func (c *Client) UpdateSetting(key, value string) error {
	req := map[string]string{"value": value}
	return c.put("/admin/settings/"+key, req)
}

// GetDeviceRefs gets the references for a device.
func (c *Client) GetDeviceRefs(id string) (*RefsResponse, error) {
	var resp RefsResponse
	if err := c.get("/devices/"+id+"/refs", &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListGuides lists guides with pagination.
func (c *Client) ListGuides(limit, offset int) (*GuidesResponse, error) {
	params := url.Values{}
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%d", limit))
	}
	if offset > 0 {
		params.Set("offset", fmt.Sprintf("%d", offset))
	}

	path := "/guides"
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	var resp GuidesResponse
	if err := c.get(path, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetGuide gets a guide by ID.
func (c *Client) GetGuide(id string) (*Guide, error) {
	var resp Guide
	if err := c.get("/guides/"+id, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetDocument gets a document by ID.
func (c *Client) GetDocument(id string) (*Document, error) {
	var resp Document
	if err := c.get("/documents/"+id, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// DownloadDocument downloads a document's content by ID.
func (c *Client) DownloadDocument(id string) ([]byte, string, error) {
	req, err := http.NewRequest("GET", c.baseURL+"/api/"+APIVersion+"/documents/"+id+"/download", nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %w", err)
	}
	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			body, _ := io.ReadAll(resp.Body)
			return nil, "", fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
		}
		return nil, "", fmt.Errorf("API error (%d): %s", resp.StatusCode, errResp.Error)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read response: %w", err)
	}

	contentType := resp.Header.Get("Content-Type")
	return content, contentType, nil
}

// get performs a GET request and decodes the JSON response.
func (c *Client) get(path string, result interface{}) error {
	req, err := http.NewRequest("GET", c.baseURL+"/api/"+APIVersion+path, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	// Only add API key header if configured (allows anonymous access)
	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
		}
		return fmt.Errorf("API error (%d): %s", resp.StatusCode, errResp.Error)
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	return nil
}

// post performs a POST request and decodes the JSON response.
func (c *Client) post(path string, body interface{}, result interface{}) error {
	return c.doJSON("POST", path, body, result)
}

// put performs a PUT request with JSON body.
func (c *Client) put(path string, body interface{}) error {
	return c.doJSON("PUT", path, body, nil)
}

// delete performs a DELETE request and decodes the JSON response.
func (c *Client) delete(path string, result interface{}) error {
	return c.doJSON("DELETE", path, nil, result)
}

// doJSON performs an HTTP request with JSON body and decodes the JSON response.
func (c *Client) doJSON(method, path string, body interface{}, result interface{}) error {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, c.baseURL+"/api/"+APIVersion+path, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Accept success status codes
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errResp ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			respBody, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("API error (%d): %s", resp.StatusCode, string(respBody))
		}
		return fmt.Errorf("API error (%d): %s", resp.StatusCode, errResp.Error)
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	client := New("http://example.com", "test-key")

	if client.baseURL != "http://example.com" {
		t.Errorf("New() baseURL = %s, want http://example.com", client.baseURL)
	}
	if client.apiKey != "test-key" {
		t.Errorf("New() apiKey = %s, want test-key", client.apiKey)
	}
}

func TestHasAPIKey(t *testing.T) {
	tests := []struct {
		name   string
		apiKey string
		want   bool
	}{
		{"with key", "test-key", true},
		{"empty key", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := New("http://example.com", tt.apiKey)
			if got := client.HasAPIKey(); got != tt.want {
				t.Errorf("HasAPIKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetAPIURL(t *testing.T) {
	client := New("http://example.com", "")
	if got := client.GetAPIURL(); got != "http://example.com" {
		t.Errorf("GetAPIURL() = %s, want http://example.com", got)
	}
}

func TestSearch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/"+APIVersion+"/search") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("q") != "arduino" {
			t.Errorf("query q = %s, want arduino", r.URL.Query().Get("q"))
		}

		json.NewEncoder(w).Encode(SearchResponse{
			Query: "arduino",
			Total: 1,
			Results: []SearchResult{
				{DeviceID: "arduino-uno", Name: "Arduino Uno", Domain: "hardware", Type: "mcu"},
			},
		})
	}))
	defer server.Close()

	client := New(server.URL, "test-key")
	resp, err := client.Search("arduino", 10, "", "")

	if err != nil {
		t.Errorf("Search() error = %v", err)
	}
	if resp == nil {
		t.Fatal("Search() returned nil")
	}
	if resp.Query != "arduino" {
		t.Errorf("Search() query = %s, want arduino", resp.Query)
	}
	if len(resp.Results) != 1 {
		t.Errorf("Search() results count = %d, want 1", len(resp.Results))
	}
}

func TestSearch_WithFilters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("domain") != "hardware" {
			t.Errorf("domain = %s, want hardware", r.URL.Query().Get("domain"))
		}
		if r.URL.Query().Get("type") != "sensors" {
			t.Errorf("type = %s, want sensors", r.URL.Query().Get("type"))
		}
		json.NewEncoder(w).Encode(SearchResponse{Results: []SearchResult{}})
	}))
	defer server.Close()

	client := New(server.URL, "")
	_, err := client.Search("test", 0, "hardware", "sensors")
	if err != nil {
		t.Errorf("Search() with filters error = %v", err)
	}
}

func TestSearch_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "query required"})
	}))
	defer server.Close()

	client := New(server.URL, "")
	_, err := client.Search("", 0, "", "")

	if err == nil {
		t.Error("Search() should return error on API error")
	}
}

func TestListDevices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/"+APIVersion+"/devices") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		json.NewEncoder(w).Encode(DevicesResponse{
			Data:  []Device{{ID: "test-1", Name: "Test Device"}},
			Total: 1,
		})
	}))
	defer server.Close()

	client := New(server.URL, "test-key")
	resp, err := client.ListDevices(10, 0, "", "")

	if err != nil {
		t.Errorf("ListDevices() error = %v", err)
	}
	if resp == nil {
		t.Fatal("ListDevices() returned nil")
	}
	if len(resp.Data) != 1 {
		t.Errorf("ListDevices() data count = %d, want 1", len(resp.Data))
	}
}

func TestListDevices_WithFilters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("limit") != "5" {
			t.Errorf("limit = %s, want 5", r.URL.Query().Get("limit"))
		}
		if r.URL.Query().Get("offset") != "10" {
			t.Errorf("offset = %s, want 10", r.URL.Query().Get("offset"))
		}
		if r.URL.Query().Get("domain") != "software" {
			t.Errorf("domain = %s, want software", r.URL.Query().Get("domain"))
		}
		if r.URL.Query().Get("type") != "tools" {
			t.Errorf("type = %s, want tools", r.URL.Query().Get("type"))
		}
		json.NewEncoder(w).Encode(DevicesResponse{Data: []Device{}})
	}))
	defer server.Close()

	client := New(server.URL, "")
	_, err := client.ListDevices(5, 10, "software", "tools")
	if err != nil {
		t.Errorf("ListDevices() with filters error = %v", err)
	}
}

func TestGetDevice(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/"+APIVersion+"/devices/test-device" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(Device{ID: "test-device", Name: "Test Device"})
	}))
	defer server.Close()

	client := New(server.URL, "test-key")
	device, err := client.GetDevice("test-device", false)

	if err != nil {
		t.Errorf("GetDevice() error = %v", err)
	}
	if device == nil {
		t.Fatal("GetDevice() returned nil")
	}
	if device.ID != "test-device" {
		t.Errorf("GetDevice() ID = %s, want test-device", device.ID)
	}
}

func TestGetDevice_WithContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("content") != "true" {
			t.Error("content query param should be true")
		}
		json.NewEncoder(w).Encode(Device{ID: "test-device", Content: "# Content"})
	}))
	defer server.Close()

	client := New(server.URL, "")
	device, err := client.GetDevice("test-device", true)
	if err != nil {
		t.Errorf("GetDevice() error = %v", err)
	}
	if device.Content == "" {
		t.Error("GetDevice() should include content")
	}
}

func TestGetDevicePinout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/pinout") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(PinoutResponse{
			DeviceID: "test-device",
			Pins:     []PinoutPin{{PhysicalPin: 1, Name: "GPIO1"}},
		})
	}))
	defer server.Close()

	client := New(server.URL, "")
	resp, err := client.GetDevicePinout("test-device")

	if err != nil {
		t.Errorf("GetDevicePinout() error = %v", err)
	}
	if resp == nil {
		t.Fatal("GetDevicePinout() returned nil")
	}
	if resp.DeviceID != "test-device" {
		t.Errorf("GetDevicePinout() DeviceID = %s, want test-device", resp.DeviceID)
	}
}

func TestGetDeviceSpecs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/specs") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(SpecsResponse{
			DeviceID: "test-device",
			Specs:    map[string]string{"voltage": "3.3V"},
		})
	}))
	defer server.Close()

	client := New(server.URL, "")
	resp, err := client.GetDeviceSpecs("test-device")

	if err != nil {
		t.Errorf("GetDeviceSpecs() error = %v", err)
	}
	if resp == nil {
		t.Fatal("GetDeviceSpecs() returned nil")
	}
	if resp.Specs["voltage"] != "3.3V" {
		t.Errorf("GetDeviceSpecs() voltage = %s, want 3.3V", resp.Specs["voltage"])
	}
}

func TestListDocuments(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/"+APIVersion+"/documents") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(DocumentsResponse{
			Data:  []Document{{ID: "doc-1", Filename: "test.pdf"}},
			Total: 1,
		})
	}))
	defer server.Close()

	client := New(server.URL, "")
	resp, err := client.ListDocuments(10, 0, "")

	if err != nil {
		t.Errorf("ListDocuments() error = %v", err)
	}
	if resp == nil {
		t.Fatal("ListDocuments() returned nil")
	}
	if len(resp.Data) != 1 {
		t.Errorf("ListDocuments() data count = %d, want 1", len(resp.Data))
	}
}

func TestListDocuments_WithFilters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("device_id") != "dev-1" {
			t.Errorf("device_id = %s, want dev-1", r.URL.Query().Get("device_id"))
		}
		json.NewEncoder(w).Encode(DocumentsResponse{Data: []Document{}})
	}))
	defer server.Close()

	client := New(server.URL, "")
	_, err := client.ListDocuments(0, 0, "dev-1")
	if err != nil {
		t.Errorf("ListDocuments() with device_id error = %v", err)
	}
}

func TestGetStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/"+APIVersion+"/status" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(StatusResponse{
			Status:     "ok",
			APIVersion: APIVersion,
		})
	}))
	defer server.Close()

	client := New(server.URL, "")
	resp, err := client.GetStatus()

	if err != nil {
		t.Errorf("GetStatus() error = %v", err)
	}
	if resp == nil {
		t.Fatal("GetStatus() returned nil")
	}
	if resp.Status != "ok" {
		t.Errorf("GetStatus() status = %s, want ok", resp.Status)
	}
}

func TestGetMe(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-Key") != "test-key" {
			t.Errorf("X-API-Key = %s, want test-key", r.Header.Get("X-API-Key"))
		}
		json.NewEncoder(w).Encode(MeResponse{
			User: User{ID: "user-1", Name: "testuser", Role: "admin"},
		})
	}))
	defer server.Close()

	client := New(server.URL, "test-key")
	user, err := client.GetMe()

	if err != nil {
		t.Errorf("GetMe() error = %v", err)
	}
	if user == nil {
		t.Fatal("GetMe() returned nil")
	}
	if user.Name != "testuser" {
		t.Errorf("GetMe() name = %s, want testuser", user.Name)
	}
}

func TestGetMe_Anonymous(t *testing.T) {
	// Should not make any request when no API key
	client := New("http://example.com", "")
	user, err := client.GetMe()

	if err != nil {
		t.Errorf("GetMe() error = %v", err)
	}
	if user != nil {
		t.Error("GetMe() should return nil for anonymous user")
	}
}

func TestTriggerReindex(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/api/"+APIVersion+"/rw/reindex" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(ReindexResponse{Status: "started"})
	}))
	defer server.Close()

	client := New(server.URL, "test-key")
	resp, err := client.TriggerReindex()

	if err != nil {
		t.Errorf("TriggerReindex() error = %v", err)
	}
	if resp == nil {
		t.Fatal("TriggerReindex() returned nil")
	}
	if resp.Status != "started" {
		t.Errorf("TriggerReindex() status = %s, want started", resp.Status)
	}
}

func TestGetReindexStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/"+APIVersion+"/rw/reindex/status" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(ReindexStatusResponse{Status: "idle"})
	}))
	defer server.Close()

	client := New(server.URL, "test-key")
	resp, err := client.GetReindexStatus()

	if err != nil {
		t.Errorf("GetReindexStatus() error = %v", err)
	}
	if resp == nil {
		t.Fatal("GetReindexStatus() returned nil")
	}
	if resp.Status != "idle" {
		t.Errorf("GetReindexStatus() status = %s, want idle", resp.Status)
	}
}

func TestTriggerSync(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/api/"+APIVersion+"/rw/sync" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(SyncResponse{Status: "success"})
	}))
	defer server.Close()

	client := New(server.URL, "test-key")
	resp, err := client.TriggerSync()

	if err != nil {
		t.Errorf("TriggerSync() error = %v", err)
	}
	if resp == nil {
		t.Fatal("TriggerSync() returned nil")
	}
	if resp.Status != "success" {
		t.Errorf("TriggerSync() status = %s, want success", resp.Status)
	}
}

func TestListUsers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/"+APIVersion+"/admin/users" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(UsersResponse{
			Users: []User{{ID: "user-1", Name: "admin"}},
			Count: 1,
		})
	}))
	defer server.Close()

	client := New(server.URL, "admin-key")
	resp, err := client.ListUsers()

	if err != nil {
		t.Errorf("ListUsers() error = %v", err)
	}
	if resp == nil {
		t.Fatal("ListUsers() returned nil")
	}
	if len(resp.Users) != 1 {
		t.Errorf("ListUsers() count = %d, want 1", len(resp.Users))
	}
}

func TestCreateUser(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type = %s, want application/json", r.Header.Get("Content-Type"))
		}

		var req CreateUserRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.Name != "newuser" || req.Role != "ro" {
			t.Errorf("request = %+v, want name=newuser role=ro", req)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(CreateUserResponse{
			User:    User{ID: "user-2", Name: "newuser", Role: "ro"},
			APIKey:  "mapi_test123",
			Message: "User created",
		})
	}))
	defer server.Close()

	client := New(server.URL, "admin-key")
	resp, err := client.CreateUser("newuser", "ro")

	if err != nil {
		t.Errorf("CreateUser() error = %v", err)
	}
	if resp == nil {
		t.Fatal("CreateUser() returned nil")
	}
	if resp.User.Name != "newuser" {
		t.Errorf("CreateUser() name = %s, want newuser", resp.User.Name)
	}
	if resp.APIKey == "" {
		t.Error("CreateUser() should return API key")
	}
}

func TestDeleteUser(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		if r.URL.Path != "/api/"+APIVersion+"/admin/users/user-123" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]string{"message": "deleted"})
	}))
	defer server.Close()

	client := New(server.URL, "admin-key")
	err := client.DeleteUser("user-123")

	if err != nil {
		t.Errorf("DeleteUser() error = %v", err)
	}
}

func TestUploadFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
			t.Errorf("Content-Type = %s, want multipart/form-data", r.Header.Get("Content-Type"))
		}

		r.ParseMultipartForm(10 << 20)
		if r.FormValue("path") != "test/upload.md" {
			t.Errorf("path = %s, want test/upload.md", r.FormValue("path"))
		}

		file, header, _ := r.FormFile("file")
		if header.Filename != "upload.md" {
			t.Errorf("filename = %s, want upload.md", header.Filename)
		}
		file.Close()

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(UploadResponse{
			Message:  "uploaded",
			Path:     "test/upload.md",
			Size:     100,
			Filename: "upload.md",
		})
	}))
	defer server.Close()

	client := New(server.URL, "rw-key")
	resp, err := client.UploadFile("test/upload.md", "upload.md", []byte("# Test Content"))

	if err != nil {
		t.Errorf("UploadFile() error = %v", err)
	}
	if resp == nil {
		t.Fatal("UploadFile() returned nil")
	}
	if resp.Path != "test/upload.md" {
		t.Errorf("UploadFile() path = %s, want test/upload.md", resp.Path)
	}
}

func TestUploadFile_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "path required"})
	}))
	defer server.Close()

	client := New(server.URL, "rw-key")
	_, err := client.UploadFile("", "test.md", []byte("content"))

	if err == nil {
		t.Error("UploadFile() should return error on API error")
	}
}

func TestGet_NetworkError(t *testing.T) {
	// Use invalid URL to trigger network error
	client := New("http://localhost:99999", "")
	_, err := client.GetStatus()

	if err == nil {
		t.Error("get() should return error on network failure")
	}
}

func TestGet_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer server.Close()

	client := New(server.URL, "")
	_, err := client.GetStatus()

	if err == nil {
		t.Error("get() should return error on invalid JSON")
	}
}

func TestPost_WithBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type = %s, want application/json", r.Header.Get("Content-Type"))
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	client := New(server.URL, "")
	var result map[string]string
	err := client.post("/test", map[string]string{"key": "value"}, &result)

	if err != nil {
		t.Errorf("post() error = %v", err)
	}
}

func TestDelete_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "forbidden"})
	}))
	defer server.Close()

	client := New(server.URL, "")
	err := client.DeleteUser("user-1")

	if err == nil {
		t.Error("delete() should return error on API error")
	}
	if !strings.Contains(err.Error(), "forbidden") {
		t.Errorf("error = %v, want to contain 'forbidden'", err)
	}
}

func TestUpdateUserRole(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("method = %s, want PUT", r.Method)
		}
		if r.URL.Path != "/api/"+APIVersion+"/admin/users/user-123/role" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var req map[string]string
		json.NewDecoder(r.Body).Decode(&req)
		if req["role"] != "rw" {
			t.Errorf("role = %s, want rw", req["role"])
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := New(server.URL, "admin-key")
	err := client.UpdateUserRole("user-123", "rw")

	if err != nil {
		t.Errorf("UpdateUserRole() error = %v", err)
	}
}

func TestRotateAPIKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/api/"+APIVersion+"/admin/users/user-123/rotate-key" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		json.NewEncoder(w).Encode(RotateKeyResponse{
			APIKey:  "mapi_newkey123",
			Message: "API key rotated",
		})
	}))
	defer server.Close()

	client := New(server.URL, "admin-key")
	resp, err := client.RotateAPIKey("user-123")

	if err != nil {
		t.Errorf("RotateAPIKey() error = %v", err)
	}
	if resp == nil {
		t.Fatal("RotateAPIKey() returned nil")
	}
	if resp.APIKey != "mapi_newkey123" {
		t.Errorf("RotateAPIKey() APIKey = %s, want mapi_newkey123", resp.APIKey)
	}
}

func TestListSettings(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/"+APIVersion+"/admin/settings" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(SettingsResponse{
			Settings: []Setting{{Key: "theme", Value: "dark"}},
		})
	}))
	defer server.Close()

	client := New(server.URL, "admin-key")
	resp, err := client.ListSettings()

	if err != nil {
		t.Errorf("ListSettings() error = %v", err)
	}
	if resp == nil {
		t.Fatal("ListSettings() returned nil")
	}
	if len(resp.Settings) != 1 {
		t.Errorf("ListSettings() count = %d, want 1", len(resp.Settings))
	}
	if resp.Settings[0].Key != "theme" {
		t.Errorf("ListSettings() key = %s, want theme", resp.Settings[0].Key)
	}
}

func TestUpdateSetting(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("method = %s, want PUT", r.Method)
		}
		if r.URL.Path != "/api/"+APIVersion+"/admin/settings/theme" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var req map[string]string
		json.NewDecoder(r.Body).Decode(&req)
		if req["value"] != "light" {
			t.Errorf("value = %s, want light", req["value"])
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := New(server.URL, "admin-key")
	err := client.UpdateSetting("theme", "light")

	if err != nil {
		t.Errorf("UpdateSetting() error = %v", err)
	}
}

func TestGetDeviceRefs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/"+APIVersion+"/devices/test-device/refs" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(RefsResponse{
			DeviceID: "test-device",
			Name:     "Test Device",
			References: []Reference{
				{Type: "datasheet", Title: "Datasheet", URL: "http://example.com/ds.pdf"},
			},
		})
	}))
	defer server.Close()

	client := New(server.URL, "")
	resp, err := client.GetDeviceRefs("test-device")

	if err != nil {
		t.Errorf("GetDeviceRefs() error = %v", err)
	}
	if resp == nil {
		t.Fatal("GetDeviceRefs() returned nil")
	}
	if resp.DeviceID != "test-device" {
		t.Errorf("GetDeviceRefs() DeviceID = %s, want test-device", resp.DeviceID)
	}
	if len(resp.References) != 1 {
		t.Errorf("GetDeviceRefs() references count = %d, want 1", len(resp.References))
	}
}

func TestListGuides(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/"+APIVersion+"/guides") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(GuidesResponse{
			Data:  []Guide{{ID: "guide-1", Title: "Getting Started"}},
			Total: 1,
		})
	}))
	defer server.Close()

	client := New(server.URL, "")
	resp, err := client.ListGuides(10, 0)

	if err != nil {
		t.Errorf("ListGuides() error = %v", err)
	}
	if resp == nil {
		t.Fatal("ListGuides() returned nil")
	}
	if len(resp.Data) != 1 {
		t.Errorf("ListGuides() data count = %d, want 1", len(resp.Data))
	}
}

func TestListGuides_WithPagination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("limit") != "5" {
			t.Errorf("limit = %s, want 5", r.URL.Query().Get("limit"))
		}
		if r.URL.Query().Get("offset") != "10" {
			t.Errorf("offset = %s, want 10", r.URL.Query().Get("offset"))
		}
		json.NewEncoder(w).Encode(GuidesResponse{Data: []Guide{}})
	}))
	defer server.Close()

	client := New(server.URL, "")
	_, err := client.ListGuides(5, 10)
	if err != nil {
		t.Errorf("ListGuides() with pagination error = %v", err)
	}
}

func TestGetGuide(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/"+APIVersion+"/guides/guide-123" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(Guide{
			ID:      "guide-123",
			Title:   "Test Guide",
			Content: "# Test Guide Content",
		})
	}))
	defer server.Close()

	client := New(server.URL, "")
	guide, err := client.GetGuide("guide-123")

	if err != nil {
		t.Errorf("GetGuide() error = %v", err)
	}
	if guide == nil {
		t.Fatal("GetGuide() returned nil")
	}
	if guide.ID != "guide-123" {
		t.Errorf("GetGuide() ID = %s, want guide-123", guide.ID)
	}
	if guide.Title != "Test Guide" {
		t.Errorf("GetGuide() Title = %s, want Test Guide", guide.Title)
	}
}

func TestGetDocument(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/"+APIVersion+"/documents/doc-123" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(Document{
			ID:       "doc-123",
			Filename: "test.pdf",
			MimeType: "application/pdf",
		})
	}))
	defer server.Close()

	client := New(server.URL, "")
	doc, err := client.GetDocument("doc-123")

	if err != nil {
		t.Errorf("GetDocument() error = %v", err)
	}
	if doc == nil {
		t.Fatal("GetDocument() returned nil")
	}
	if doc.ID != "doc-123" {
		t.Errorf("GetDocument() ID = %s, want doc-123", doc.ID)
	}
	if doc.Filename != "test.pdf" {
		t.Errorf("GetDocument() Filename = %s, want test.pdf", doc.Filename)
	}
}

func TestDownloadDocument(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/"+APIVersion+"/documents/doc-123/download" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/pdf")
		w.Write([]byte("PDF content here"))
	}))
	defer server.Close()

	client := New(server.URL, "")
	content, contentType, err := client.DownloadDocument("doc-123")

	if err != nil {
		t.Errorf("DownloadDocument() error = %v", err)
	}
	if string(content) != "PDF content here" {
		t.Errorf("DownloadDocument() content = %s, want 'PDF content here'", string(content))
	}
	if contentType != "application/pdf" {
		t.Errorf("DownloadDocument() contentType = %s, want application/pdf", contentType)
	}
}

func TestDownloadDocument_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "document not found"})
	}))
	defer server.Close()

	client := New(server.URL, "")
	_, _, err := client.DownloadDocument("nonexistent")

	if err == nil {
		t.Error("DownloadDocument() should return error on API error")
	}
	if !strings.Contains(err.Error(), "document not found") {
		t.Errorf("error = %v, want to contain 'document not found'", err)
	}
}

func TestDownloadDocument_NetworkError(t *testing.T) {
	client := New("http://localhost:99999", "")
	_, _, err := client.DownloadDocument("doc-123")

	if err == nil {
		t.Error("DownloadDocument() should return error on network failure")
	}
}

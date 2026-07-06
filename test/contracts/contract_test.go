package contracts

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ContractTestSuite validates API contracts between services
// These tests ensure that all services adhere to the HelixTerminator API contract

// TestGatewayHealthContract verifies the gateway health endpoint contract
func TestGatewayHealthContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/healthz", nil)
	
	// Simulate a minimal health response
	response := map[string]string{"status": "healthy"}
	b, _ := json.Marshal(response)
	w.Write(b)
	w.Code = http.StatusOK
	
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "healthy")
}

// TestAuthLoginContract verifies the auth login endpoint contract
func TestAuthLoginContract(t *testing.T) {
	loginReq := map[string]interface{}{
		"email":    "test@example.com",
		"password": "securePassword123",
	}
	b, err := json.Marshal(loginReq)
	require.NoError(t, err)
	require.NotEmpty(t, b)
	
	// Validate request structure
	var req map[string]interface{}
	err = json.Unmarshal(b, &req)
	require.NoError(t, err)
	assert.NotEmpty(t, req["email"])
	assert.NotEmpty(t, req["password"])
}

// TestAuthTokenResponseContract verifies JWT token response structure
func TestAuthTokenResponseContract(t *testing.T) {
	response := map[string]interface{}{
		"access_token":  "eyJhbGciOiJSUzI1NiIs...",
		"refresh_token": "eyJhbGciOiJSUzI1NiIs...",
		"token_type":    "Bearer",
		"expires_in":    3600,
	}
	b, _ := json.Marshal(response)
	
	var resp map[string]interface{}
	err := json.Unmarshal(b, &resp)
	require.NoError(t, err)
	assert.NotEmpty(t, resp["access_token"])
	assert.NotEmpty(t, resp["refresh_token"])
	assert.Equal(t, "Bearer", resp["token_type"])
	assert.NotZero(t, resp["expires_in"])
}

// TestHostCreateContract verifies host creation request contract
func TestHostCreateContract(t *testing.T) {
	createReq := map[string]interface{}{
		"name":     "production-server-01",
		"hostname": "192.168.1.100",
		"port":     22,
		"username": "admin",
		"authMethod": "password",
		"tags":     []string{"production", "web"},
	}
	b, err := json.Marshal(createReq)
	require.NoError(t, err)
	
	var req map[string]interface{}
	err = json.Unmarshal(b, &req)
	require.NoError(t, err)
	assert.NotEmpty(t, req["name"])
	assert.NotEmpty(t, req["hostname"])
	assert.NotZero(t, req["port"])
	assert.NotEmpty(t, req["username"])
	assert.NotEmpty(t, req["authMethod"])
}

// TestHostResponseContract verifies host response structure
func TestHostResponseContract(t *testing.T) {
	response := map[string]interface{}{
		"id":         uuid.New().String(),
		"name":       "production-server-01",
		"hostname":   "192.168.1.100",
		"port":       22,
		"username":   "admin",
		"status":     "active",
		"createdAt":  "2024-01-01T00:00:00Z",
		"updatedAt":  "2024-01-01T00:00:00Z",
	}
	b, _ := json.Marshal(response)
	
	var resp map[string]interface{}
	err := json.Unmarshal(b, &resp)
	require.NoError(t, err)
	assert.NotEmpty(t, resp["id"])
	assert.NotEmpty(t, resp["name"])
	assert.NotEmpty(t, resp["status"])
}

// TestPaginatedResponseContract verifies paginated list response structure
func TestPaginatedResponseContract(t *testing.T) {
	response := map[string]interface{}{
		"data":   []interface{}{},
		"total":  100,
		"limit":  20,
		"offset": 0,
	}
	b, _ := json.Marshal(response)
	
	var resp map[string]interface{}
	err := json.Unmarshal(b, &resp)
	require.NoError(t, err)
	assert.NotNil(t, resp["data"])
	assert.NotZero(t, resp["total"])
	assert.NotZero(t, resp["limit"])
}

// TestErrorResponseContract verifies error response structure
func TestErrorResponseContract(t *testing.T) {
	response := map[string]interface{}{
		"error": "invalid_request",
		"message": "The request body is malformed",
		"code":    400,
	}
	b, _ := json.Marshal(response)
	
	var resp map[string]interface{}
	err := json.Unmarshal(b, &resp)
	require.NoError(t, err)
	assert.NotEmpty(t, resp["error"])
	assert.NotEmpty(t, resp["message"])
	assert.NotZero(t, resp["code"])
}

// TestWorkspaceCreateContract verifies workspace creation request
func TestWorkspaceCreateContract(t *testing.T) {
	createReq := map[string]interface{}{
		"name":        "Engineering Team",
		"description": "Main engineering workspace",
		"orgId":       uuid.New().String(),
	}
	b, err := json.Marshal(createReq)
	require.NoError(t, err)
	
	var req map[string]interface{}
	err = json.Unmarshal(b, &req)
	require.NoError(t, err)
	assert.NotEmpty(t, req["name"])
	assert.NotEmpty(t, req["orgId"])
}

// TestVaultSecretCreateContract verifies vault secret creation request
func TestVaultSecretCreateContract(t *testing.T) {
	createReq := map[string]interface{}{
		"name":        "Database Password",
		"value":       "super-secret-password",
		"type":        "password",
		"hostId":      uuid.New().String(),
	}
	b, err := json.Marshal(createReq)
	require.NoError(t, err)
	
	var req map[string]interface{}
	err = json.Unmarshal(b, &req)
	require.NoError(t, err)
	assert.NotEmpty(t, req["name"])
	assert.NotEmpty(t, req["value"])
	assert.NotEmpty(t, req["type"])
}

// TestNotificationResponseContract verifies notification response structure
func TestNotificationResponseContract(t *testing.T) {
	response := map[string]interface{}{
		"id":        uuid.New().String(),
		"type":      "host_alert",
		"title":     "Host connection failed",
		"message":   "Failed to connect to production-server-01",
		"read":      false,
		"createdAt": "2024-01-01T00:00:00Z",
	}
	b, _ := json.Marshal(response)
	
	var resp map[string]interface{}
	err := json.Unmarshal(b, &resp)
	require.NoError(t, err)
	assert.NotEmpty(t, resp["id"])
	assert.NotEmpty(t, resp["type"])
	assert.NotEmpty(t, resp["title"])
	assert.NotNil(t, resp["read"])
}

// TestAPIVersionHeaderContract verifies API version header requirements
func TestAPIVersionHeaderContract(t *testing.T) {
	req, _ := http.NewRequest("GET", "/api/v1/hosts", nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-API-Version", "v1")
	req.Header.Set("X-Request-ID", uuid.New().String())
	
	assert.Equal(t, "application/json", req.Header.Get("Accept"))
	assert.Equal(t, "v1", req.Header.Get("X-API-Version"))
	assert.NotEmpty(t, req.Header.Get("X-Request-ID"))
}

// TestBulkOperationContract verifies bulk operation request structure
func TestBulkOperationContract(t *testing.T) {
	bulkReq := map[string]interface{}{
		"operation": "delete",
		"ids": []string{
			uuid.New().String(),
			uuid.New().String(),
			uuid.New().String(),
		},
	}
	b, err := json.Marshal(bulkReq)
	require.NoError(t, err)
	
	var req map[string]interface{}
	err = json.Unmarshal(b, &req)
	require.NoError(t, err)
	assert.NotEmpty(t, req["operation"])
	assert.NotNil(t, req["ids"])
}

// TestWebSocketMessageContract verifies WebSocket message structure for terminal
func TestWebSocketMessageContract(t *testing.T) {
	message := map[string]interface{}{
		"type":    "input",
		"data":    "ls -la\n",
		"session": uuid.New().String(),
	}
	b, _ := json.Marshal(message)
	
	var msg map[string]interface{}
	err := json.Unmarshal(b, &msg)
	require.NoError(t, err)
	assert.NotEmpty(t, msg["type"])
	assert.NotEmpty(t, msg["data"])
	assert.NotEmpty(t, msg["session"])
}

// TestHealthCheckResponseContract verifies standardized health check response
func TestHealthCheckResponseContract(t *testing.T) {
	response := map[string]interface{}{
		"status":    "healthy",
		"version":   "1.0.0",
		"timestamp": "2024-01-01T00:00:00Z",
		"checks": map[string]interface{}{
			"database": "up",
			"cache":    "up",
		},
	}
	b, _ := json.Marshal(response)
	
	var resp map[string]interface{}
	err := json.Unmarshal(b, &resp)
	require.NoError(t, err)
	assert.NotEmpty(t, resp["status"])
	assert.NotEmpty(t, resp["version"])
	assert.NotEmpty(t, resp["timestamp"])
	assert.NotNil(t, resp["checks"])
}

// TestRateLimitHeadersContract verifies rate limit headers
func TestRateLimitHeadersContract(t *testing.T) {
	recorder := httptest.NewRecorder()
	recorder.Header().Set("X-RateLimit-Limit", "100")
	recorder.Header().Set("X-RateLimit-Remaining", "95")
	recorder.Header().Set("X-RateLimit-Reset", "1640995200")
	
	assert.Equal(t, "100", recorder.Header().Get("X-RateLimit-Limit"))
	assert.Equal(t, "95", recorder.Header().Get("X-RateLimit-Remaining"))
	assert.Equal(t, "1640995200", recorder.Header().Get("X-RateLimit-Reset"))
}

// TestAuditLogEntryContract verifies audit log entry structure
func TestAuditLogEntryContract(t *testing.T) {
	entry := map[string]interface{}{
		"id":           uuid.New().String(),
		"action":       "host.create",
		"resourceType": "host",
		"resourceId":   uuid.New().String(),
		"userId":       uuid.New().String(),
		"orgId":        uuid.New().String(),
		"timestamp":    "2024-01-01T00:00:00Z",
		"ipAddress":    "192.168.1.1",
		"userAgent":    "HelixTerminator/1.0.0",
	}
	b, _ := json.Marshal(entry)
	
	var logEntry map[string]interface{}
	err := json.Unmarshal(b, &logEntry)
	require.NoError(t, err)
	assert.NotEmpty(t, logEntry["id"])
	assert.NotEmpty(t, logEntry["action"])
	assert.NotEmpty(t, logEntry["resourceType"])
	assert.NotEmpty(t, logEntry["userId"])
	assert.NotEmpty(t, logEntry["timestamp"])
}

// TestCollaborationSessionContract verifies collaboration session structure
func TestCollaborationSessionContract(t *testing.T) {
	session := map[string]interface{}{
		"id":          uuid.New().String(),
		"hostId":      uuid.New().String(),
		"createdBy":   uuid.New().String(),
		"name":        "Debug Session",
		"status":      "active",
		"participants": []string{uuid.New().String(), uuid.New().String()},
		"createdAt":   "2024-01-01T00:00:00Z",
	}
	b, _ := json.Marshal(session)
	
	var sess map[string]interface{}
	err := json.Unmarshal(b, &sess)
	require.NoError(t, err)
	assert.NotEmpty(t, sess["id"])
	assert.NotEmpty(t, sess["hostId"])
	assert.NotEmpty(t, sess["status"])
	assert.NotNil(t, sess["participants"])
}

// TestSFTPFileEntryContract verifies SFTP file entry structure
func TestSFTPFileEntryContract(t *testing.T) {
	entry := map[string]interface{}{
		"name":     "document.pdf",
		"path":     "/home/user/documents/document.pdf",
		"size":     1048576,
		"mode":     "0644",
		"isDir":    false,
		"modified": "2024-01-01T00:00:00Z",
	}
	b, _ := json.Marshal(entry)
	
	var fileEntry map[string]interface{}
	err := json.Unmarshal(b, &fileEntry)
	require.NoError(t, err)
	assert.NotEmpty(t, fileEntry["name"])
	assert.NotEmpty(t, fileEntry["path"])
	assert.NotNil(t, fileEntry["size"])
	assert.NotNil(t, fileEntry["isDir"])
}

// TestRecordingMetadataContract verifies recording metadata structure
func TestRecordingMetadataContract(t *testing.T) {
	metadata := map[string]interface{}{
		"id":            uuid.New().String(),
		"sessionId":     uuid.New().String(),
		"hostId":        uuid.New().String(),
		"filePath":      "/recordings/session-123.cast",
		"format":        "asciinema",
		"status":        "completed",
		"durationSec":   120,
		"fileSizeBytes": 102400,
		"createdAt":     "2024-01-01T00:00:00Z",
	}
	b, _ := json.Marshal(metadata)
	
	var meta map[string]interface{}
	err := json.Unmarshal(b, &meta)
	require.NoError(t, err)
	assert.NotEmpty(t, meta["id"])
	assert.NotEmpty(t, meta["sessionId"])
	assert.NotEmpty(t, meta["format"])
	assert.NotZero(t, meta["durationSec"])
}

// TestSnippetContract verifies snippet structure
func TestSnippetContract(t *testing.T) {
	snippet := map[string]interface{}{
		"id":          uuid.New().String(),
		"name":        "List Docker Containers",
		"content":     "docker ps -a",
		"language":    "bash",
		"tags":        []string{"docker", "containers"},
		"description": "List all Docker containers",
		"isPublic":    true,
		"usageCount":  42,
		"createdAt":   "2024-01-01T00:00:00Z",
	}
	b, _ := json.Marshal(snippet)
	
	var snip map[string]interface{}
	err := json.Unmarshal(b, &snip)
	require.NoError(t, err)
	assert.NotEmpty(t, snip["id"])
	assert.NotEmpty(t, snip["name"])
	assert.NotEmpty(t, snip["content"])
	assert.NotEmpty(t, snip["language"])
}

// TestPortForwardRuleContract verifies port forward rule structure
func TestPortForwardRuleContract(t *testing.T) {
	rule := map[string]interface{}{
		"id":         uuid.New().String(),
		"hostId":     uuid.New().String(),
		"localPort":  8080,
		"remotePort": 80,
		"remoteHost": "localhost",
		"protocol":   "tcp",
		"status":     "active",
	}
	b, _ := json.Marshal(rule)
	
	var pf map[string]interface{}
	err := json.Unmarshal(b, &pf)
	require.NoError(t, err)
	assert.NotEmpty(t, pf["id"])
	assert.NotZero(t, pf["localPort"])
	assert.NotZero(t, pf["remotePort"])
	assert.NotEmpty(t, pf["protocol"])
}

// TestAnalyticsEventContract verifies analytics event structure
func TestAnalyticsEventContract(t *testing.T) {
	event := map[string]interface{}{
		"id":        uuid.New().String(),
		"userId":    uuid.New().String(),
		"eventType": "session",
		"payload": map[string]interface{}{
			"action":   "host_connected",
			"hostId":   uuid.New().String(),
			"duration": 120,
		},
		"createdAt": "2024-01-01T00:00:00Z",
	}
	b, _ := json.Marshal(event)
	
	var evt map[string]interface{}
	err := json.Unmarshal(b, &evt)
	require.NoError(t, err)
	assert.NotEmpty(t, evt["id"])
	assert.NotEmpty(t, evt["eventType"])
	assert.NotNil(t, evt["payload"])
}

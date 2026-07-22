package model_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/helixdevelopment/user-service/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCreateUserRequest_JSONRoundTrip proves CreateUserRequest genuinely
// marshals/unmarshals every declared field through real encoding/json -
// not a tautology, a struct with a typo'd json tag would fail this.
func TestCreateUserRequest_JSONRoundTrip(t *testing.T) {
	orgID := "org-123"
	original := model.CreateUserRequest{
		Email:       "alice@example.com",
		DisplayName: "Alice Example",
		Role:        "admin",
		OrgID:       &orgID,
		Permissions: []string{"read", "write"},
	}

	raw, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded model.CreateUserRequest
	require.NoError(t, json.Unmarshal(raw, &decoded))

	assert.Equal(t, original.Email, decoded.Email)
	assert.Equal(t, original.DisplayName, decoded.DisplayName)
	assert.Equal(t, original.Role, decoded.Role)
	require.NotNil(t, decoded.OrgID)
	assert.Equal(t, *original.OrgID, *decoded.OrgID)
	assert.Equal(t, original.Permissions, decoded.Permissions)
}

// TestCreateUserRequest_BindingValidation exercises the REAL gin binding
// validators declared on CreateUserRequest's struct tags
// (`binding:"required,email"`, `binding:"required,oneof=user admin
// superadmin"`) via gin's actual ShouldBindJSON - not a hand-rolled
// check. A future edit that weakens or removes a validation tag makes
// this test fail.
func TestCreateUserRequest_BindingValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cases := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{
			name:    "valid request binds cleanly",
			body:    `{"email":"bob@example.com","displayName":"Bob","role":"user"}`,
			wantErr: false,
		},
		{
			name:    "missing email is rejected",
			body:    `{"displayName":"Bob","role":"user"}`,
			wantErr: true,
		},
		{
			name:    "malformed email is rejected",
			body:    `{"email":"not-an-email","displayName":"Bob","role":"user"}`,
			wantErr: true,
		},
		{
			name:    "missing displayName is rejected",
			body:    `{"email":"bob@example.com","role":"user"}`,
			wantErr: true,
		},
		{
			name:    "role outside the closed oneof set is rejected",
			body:    `{"email":"bob@example.com","displayName":"Bob","role":"superuser"}`,
			wantErr: true,
		},
		{
			name:    "role=admin (in the closed set) binds cleanly",
			body:    `{"email":"bob@example.com","displayName":"Bob","role":"admin"}`,
			wantErr: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/users", strings.NewReader(tc.body))
			c.Request.Header.Set("Content-Type", "application/json")

			var req model.CreateUserRequest
			err := c.ShouldBindJSON(&req)

			if tc.wantErr {
				assert.Error(t, err, "expected gin binding to reject: %s", tc.body)
			} else {
				assert.NoError(t, err, "expected gin binding to accept: %s", tc.body)
			}
		})
	}
}

// TestUpdateUserRequest_RoleBindingValidation proves UpdateUserRequest's
// `binding:"omitempty,oneof=user admin superadmin"` tag on Role really
// is omitempty (a nil Role must NOT be rejected) while a non-empty,
// out-of-set Role must be.
func TestUpdateUserRequest_RoleBindingValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("nil role is not required", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/users/1", strings.NewReader(`{"displayName":"New Name"}`))
		c.Request.Header.Set("Content-Type", "application/json")

		var req model.UpdateUserRequest
		require.NoError(t, c.ShouldBindJSON(&req))
		assert.Nil(t, req.Role)
	})

	t.Run("out-of-set role is rejected", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/users/1", strings.NewReader(`{"role":"root"}`))
		c.Request.Header.Set("Content-Type", "application/json")

		var req model.UpdateUserRequest
		assert.Error(t, c.ShouldBindJSON(&req))
	})
}

// TestUserResponse_OmitsEmptyOptionalFields proves the `omitempty` JSON
// tags on UserResponse's optional fields genuinely suppress those keys
// when zero-valued (AvatarURL="", OrgID=nil, LastLoginAt=nil) and
// genuinely include them once populated - a real serialization-shape
// contract, not just a compile-time tag check.
func TestUserResponse_OmitsEmptyOptionalFields(t *testing.T) {
	minimal := model.UserResponse{
		ID:          "u1",
		Email:       "min@example.com",
		DisplayName: "Min User",
		Role:        "user",
	}
	raw, err := json.Marshal(minimal)
	require.NoError(t, err)

	var asMap map[string]interface{}
	require.NoError(t, json.Unmarshal(raw, &asMap))

	for _, absentKey := range []string{"avatarUrl", "permissions", "orgId", "lastLoginAt"} {
		_, present := asMap[absentKey]
		assert.False(t, present, "expected %q to be omitted from JSON when zero-valued, got: %s", absentKey, raw)
	}

	orgID := "org-9"
	full := model.UserResponse{
		ID:          "u2",
		Email:       "full@example.com",
		DisplayName: "Full User",
		Role:        "admin",
		AvatarURL:   "https://example.com/avatar.png",
		Permissions: []string{"read"},
		OrgID:       &orgID,
	}
	raw, err = json.Marshal(full)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, &asMap))

	for _, presentKey := range []string{"avatarUrl", "permissions", "orgId"} {
		_, present := asMap[presentKey]
		assert.True(t, present, "expected %q to be present in JSON when populated, got: %s", presentKey, raw)
	}
}

// TestListUsersRequest_QueryBindingDefaults proves the `form:"limit,
// default=20"` / `form:"offset,default=0"` tags on ListUsersRequest are
// genuinely applied by gin's real ShouldBindQuery when the client omits
// those query parameters entirely - the exact request shape
// GET /api/v1/users sends with no pagination params.
func TestListUsersRequest_QueryBindingDefaults(t *testing.T) {
	gin.SetMode(gin.TestMode)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)

	var req model.ListUsersRequest
	require.NoError(t, c.ShouldBindQuery(&req))

	assert.Equal(t, 20, req.Limit, "default limit must be applied when the client omits ?limit=")
	assert.Equal(t, 0, req.Offset, "default offset must be applied when the client omits ?offset=")
	assert.Empty(t, req.OrgID)
	assert.Empty(t, req.Role)
	assert.Empty(t, req.Search)

	c2, _ := gin.CreateTestContext(httptest.NewRecorder())
	c2.Request = httptest.NewRequest(http.MethodGet, "/api/v1/users?limit=5&offset=15&orgId=org-1&role=admin&search=ali", nil)

	var req2 model.ListUsersRequest
	require.NoError(t, c2.ShouldBindQuery(&req2))
	assert.Equal(t, 5, req2.Limit)
	assert.Equal(t, 15, req2.Offset)
	assert.Equal(t, "org-1", req2.OrgID)
	assert.Equal(t, "admin", req2.Role)
	assert.Equal(t, "ali", req2.Search)
}

// TestUserProfileResponse_EmbedsUserResponse proves the embedded
// UserResponse fields are genuinely promoted and marshal flat (not
// nested under a "UserResponse" key) alongside the profile-specific
// fields - the actual wire shape API clients depend on.
func TestUserProfileResponse_EmbedsUserResponse(t *testing.T) {
	resp := model.UserProfileResponse{
		UserResponse: model.UserResponse{
			ID:          "u1",
			Email:       "p@example.com",
			DisplayName: "Profile User",
			Role:        "user",
		},
		Bio:      "hello",
		Timezone: "UTC",
	}
	raw, err := json.Marshal(resp)
	require.NoError(t, err)

	var asMap map[string]interface{}
	require.NoError(t, json.Unmarshal(raw, &asMap))

	assert.Equal(t, "u1", asMap["id"], "embedded UserResponse.ID must be promoted flat, got: %s", raw)
	assert.Equal(t, "p@example.com", asMap["email"])
	assert.Equal(t, "hello", asMap["bio"])
	assert.Equal(t, "UTC", asMap["timezone"])
	_, nested := asMap["UserResponse"]
	assert.False(t, nested, "UserResponse must not appear as a nested key, got: %s", raw)
}

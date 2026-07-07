package coreclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeCoreServer stands in for HelixTrack Core's unified /do endpoint at the
// HTTP-wire level only (real net/http server, real JSON encoding/decoding
// over a real loopback socket) — a unit-test-layer stand-in permitted by
// §11.4.27, NOT a substitute for the real-Core integration test in
// coreclient_live_test.go.
func fakeCoreServer(t *testing.T, wantUsername, wantPassword, token string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/do" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		var req doRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		assert.Equal(t, actionAuthenticate, req.Action)

		username, _ := req.Data["username"].(string)
		password, _ := req.Data["password"].(string)
		if username != wantUsername || password != wantPassword {
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(doResponse{
				ErrorCode:    1003,
				ErrorMessage: "Invalid username or password",
			})
			return
		}

		data, _ := json.Marshal(authenticateData{Token: token})
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(doResponse{
			ErrorCode: successErrorCode,
			Data:      data,
		})
	}))
}

func TestAuthenticate_Success(t *testing.T) {
	srv := fakeCoreServer(t, "admin_user", "Admin@123456", "fake.jwt.token")
	defer srv.Close()

	c := New(srv.URL, "admin_user", "Admin@123456")
	token, err := c.Authenticate(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "fake.jwt.token", token)

	cached, err := c.AccessToken()
	require.NoError(t, err)
	assert.Equal(t, "fake.jwt.token", cached, "Authenticate must cache the token via tokenmanager")
}

func TestAuthenticate_WrongPassword(t *testing.T) {
	srv := fakeCoreServer(t, "admin_user", "Admin@123456", "fake.jwt.token")
	defer srv.Close()

	c := New(srv.URL, "admin_user", "wrong-password")
	token, err := c.Authenticate(context.Background())
	assert.Error(t, err)
	assert.Empty(t, token)
	assert.Contains(t, err.Error(), "Invalid username or password")

	cached, cerr := c.AccessToken()
	require.NoError(t, cerr)
	assert.Empty(t, cached, "a rejected authenticate MUST NOT cache a token")
}

func TestAuthenticate_Unreachable(t *testing.T) {
	c := New("http://127.0.0.1:1", "admin_user", "Admin@123456")
	token, err := c.Authenticate(context.Background())
	assert.Error(t, err)
	assert.Empty(t, token)
}

func TestAuthenticate_EmptyTokenInResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := json.Marshal(authenticateData{Token: ""})
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(doResponse{ErrorCode: successErrorCode, Data: data})
	}))
	defer srv.Close()

	c := New(srv.URL, "admin_user", "Admin@123456")
	token, err := c.Authenticate(context.Background())
	assert.Error(t, err)
	assert.Empty(t, token)
	assert.Contains(t, err.Error(), "no token")
}

func TestEnsureAuthenticated_ReAuthenticatesWhenNoValidToken(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		data, _ := json.Marshal(authenticateData{Token: "token-from-call"})
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(doResponse{ErrorCode: successErrorCode, Data: data})
	}))
	defer srv.Close()

	c := New(srv.URL, "admin_user", "Admin@123456")

	require.NoError(t, c.EnsureAuthenticated(context.Background()))
	assert.Equal(t, 1, calls)

	// Second call: a valid cached token exists (24h TTL), so no re-authenticate.
	require.NoError(t, c.EnsureAuthenticated(context.Background()))
	assert.Equal(t, 1, calls, "EnsureAuthenticated must not re-authenticate while the cached token is valid")
}

func TestEnsureAuthenticated_PropagatesAuthFailure(t *testing.T) {
	srv := fakeCoreServer(t, "admin_user", "Admin@123456", "fake.jwt.token")
	defer srv.Close()

	c := New(srv.URL, "admin_user", "wrong-password")
	err := c.EnsureAuthenticated(context.Background())
	assert.Error(t, err)
}

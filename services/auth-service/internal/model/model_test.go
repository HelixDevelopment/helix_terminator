package model_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin/binding"
	"github.com/google/uuid"

	"github.com/helixdevelopment/auth-service/internal/model"
)

// These are real, non-mocked tests: they exercise the actual
// encoding/json standard library and the actual go-playground/validator
// instance gin wires up via binding.Validator - the exact same
// validation path Register/Login run at request time through
// c.ShouldBindJSON. Per §11.4.27 mocks are reserved for unit tests only;
// there is nothing to mock here, this validates real library behaviour
// against real struct tags.

func TestUser_JSONMarshal_RedactsSecrets(t *testing.T) {
	u := model.User{
		ID:           uuid.New(),
		Email:        "secret-holder@example.com",
		PasswordHash: "$argon2id$v=19$m=65536,t=3,p=4$somesalt$somehash",
		MFASecret:    "TOTPSECRETVALUE",
		DisplayName:  "Secret Holder",
		Role:         "user",
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	b, err := json.Marshal(u)
	if err != nil {
		t.Fatalf("json.Marshal(User) failed: %v", err)
	}
	out := string(b)

	if strings.Contains(out, u.PasswordHash) {
		t.Fatalf("marshaled User JSON leaks PasswordHash (json:\"-\" not honoured): %s", out)
	}
	if strings.Contains(out, u.MFASecret) {
		t.Fatalf("marshaled User JSON leaks MFASecret (json:\"-\" not honoured): %s", out)
	}
	if !strings.Contains(out, u.Email) {
		t.Fatalf("marshaled User JSON unexpectedly dropped Email: %s", out)
	}

	var round model.User
	if err := json.Unmarshal(b, &round); err != nil {
		t.Fatalf("json.Unmarshal(User) failed: %v", err)
	}
	if round.PasswordHash != "" {
		t.Fatalf("PasswordHash round-tripped through public JSON, got %q, want empty", round.PasswordHash)
	}
	if round.Email != u.Email {
		t.Fatalf("Email did not round-trip: got %q, want %q", round.Email, u.Email)
	}
}

// TestRegisterRequest_RealValidatorRejectsInvalidInput drives the exact
// go-playground/validator instance gin's binding package uses at
// runtime (binding.Validator.ValidateStruct) - not a hand-rolled
// stand-in - against RegisterRequest's `binding:"..."` tags.
func TestRegisterRequest_RealValidatorRejectsInvalidInput(t *testing.T) {
	cases := []struct {
		name    string
		req     model.RegisterRequest
		wantErr bool
	}{
		{
			name: "valid request passes",
			req: model.RegisterRequest{
				Email:       "valid@example.com",
				Password:    "a-genuinely-long-password-123",
				DisplayName: "Valid User",
			},
			wantErr: false,
		},
		{
			name: "malformed email rejected",
			req: model.RegisterRequest{
				Email:       "not-an-email",
				Password:    "a-genuinely-long-password-123",
				DisplayName: "Valid User",
			},
			wantErr: true,
		},
		{
			name: "short password rejected (min=12)",
			req: model.RegisterRequest{
				Email:       "valid@example.com",
				Password:    "short1",
				DisplayName: "Valid User",
			},
			wantErr: true,
		},
		{
			name: "missing display name rejected",
			req: model.RegisterRequest{
				Email:    "valid@example.com",
				Password: "a-genuinely-long-password-123",
			},
			wantErr: true,
		},
		{
			name: "empty email rejected",
			req: model.RegisterRequest{
				Password:    "a-genuinely-long-password-123",
				DisplayName: "Valid User",
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := binding.Validator.ValidateStruct(&tc.req)
			if tc.wantErr && err == nil {
				t.Fatalf("ValidateStruct(%+v) = nil error, want validation error", tc.req)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("ValidateStruct(%+v) unexpected error: %v", tc.req, err)
			}
		})
	}
}

func TestLoginRequest_RealValidatorRejectsInvalidInput(t *testing.T) {
	cases := []struct {
		name    string
		req     model.LoginRequest
		wantErr bool
	}{
		{
			name:    "valid request passes",
			req:     model.LoginRequest{Email: "valid@example.com", Password: "anything"},
			wantErr: false,
		},
		{
			name:    "malformed email rejected",
			req:     model.LoginRequest{Email: "nope", Password: "anything"},
			wantErr: true,
		},
		{
			name:    "empty password rejected",
			req:     model.LoginRequest{Email: "valid@example.com"},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := binding.Validator.ValidateStruct(&tc.req)
			if tc.wantErr && err == nil {
				t.Fatalf("ValidateStruct(%+v) = nil error, want validation error", tc.req)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("ValidateStruct(%+v) unexpected error: %v", tc.req, err)
			}
		})
	}
}

func TestValidateTokenRequest_RequiresToken(t *testing.T) {
	if err := binding.Validator.ValidateStruct(&model.ValidateTokenRequest{}); err == nil {
		t.Fatal("ValidateStruct(empty ValidateTokenRequest) = nil error, want validation error for missing required token")
	}
	if err := binding.Validator.ValidateStruct(&model.ValidateTokenRequest{Token: "abc"}); err != nil {
		t.Fatalf("ValidateStruct(populated ValidateTokenRequest) unexpected error: %v", err)
	}
}

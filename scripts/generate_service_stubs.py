#!/usr/bin/env python3
"""Generate Go microservice stubs for HelixTerminator."""

import os
import pathlib

SERVICES = [
    ("gateway-service", "github.com/helixdevelopment/gateway-service"),
    ("auth-service", "github.com/helixdevelopment/auth-service"),
    ("user-service", "github.com/helixdevelopment/user-service"),
    ("vault-service", "github.com/helixdevelopment/vault-service"),
    ("host-service", "github.com/helixdevelopment/host-service"),
    ("ssh-proxy-service", "github.com/helixdevelopment/ssh-proxy-service"),
    ("terminal-service", "github.com/helixdevelopment/terminal-service"),
    ("sftp-service", "github.com/helixdevelopment/sftp-service"),
    ("port-forward-service", "github.com/helixdevelopment/port-forward-service"),
    ("snippet-service", "github.com/helixdevelopment/snippet-service"),
    ("keychain-service", "github.com/helixdevelopment/keychain-service"),
    ("workspace-service", "github.com/helixdevelopment/workspace-service"),
    ("collaboration-service", "github.com/helixdevelopment/collaboration-service"),
    ("notification-service", "github.com/helixdevelopment/notification-service"),
    ("audit-service", "github.com/helixdevelopment/audit-service"),
    ("analytics-service", "github.com/helixdevelopment/analytics-service"),
    ("ai-service", "github.com/helixdevelopment/ai-service"),
    ("recording-service", "github.com/helixdevelopment/recording-service"),
    ("pki-service", "github.com/helixdevelopment/pki-service"),
    ("org-service", "github.com/helixdevelopment/org-service"),
    ("billing-service", "github.com/helixdevelopment/billing-service"),
    ("config-service", "github.com/helixdevelopment/config-service"),
    ("health-service", "github.com/helixdevelopment/health-service"),
    ("container-bridge-service", "github.com/helixdevelopment/container-bridge-service"),
    ("helixtrack-bridge-service", "github.com/helixdevelopment/helixtrack-bridge-service"),
]

BASE = pathlib.Path("/home/milos/Factory/projects/tools_and_research/helix_terminator/services")

MAIN_GO = """package main

import (
	"log"
	"os"

	"{module}/internal/server"
)

func main() {{
	port := os.Getenv("PORT")
	if port == "" {{
		port = "8080"
	}}

	// TODO: initialize config, logger, DB, tracer
	srv := server.New()

	log.Printf("{svc} starting on port %s", port)
	if err := srv.Run(":" + port); err != nil {{
		log.Fatalf("server failed: %v", err)
	}}
}}
"""

SERVER_GO = """package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"{module}/internal/handler"
)

// Server wraps the Gin engine.
type Server struct {{
	router *gin.Engine
}}

// New creates a new Server with routes wired.
func New() *Server {{
	// TODO: configure middleware (logging, recovery, auth, tracing)
	r := gin.New()
	h := handler.New()

	// Health endpoints
	r.GET("/health", h.HealthCheck)
	r.GET("/ready", h.ReadinessCheck)

	// TODO: wire service-specific routes

	return &Server{{router: r}}
}}

// Run starts the HTTP server.
func (s *Server) Run(addr string) error {{
	return s.router.Run(addr)
}}

// Router exposes the underlying engine for testing.
func (s *Server) Router() http.Handler {{
	return s.router
}}
"""

HANDLER_GO = """package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler holds service handlers.
type Handler struct{{}}

// New returns a new Handler.
func New() *Handler {{
	return &Handler{{}}
}}

// HealthCheck returns service health status.
func (h *Handler) HealthCheck(c *gin.Context) {{
	c.JSON(http.StatusOK, gin.H{{"status": "healthy"}})
}}

// ReadinessCheck returns readiness status.
func (h *Handler) ReadinessCheck(c *gin.Context) {{
	// TODO: check DB, cache, upstream dependencies
	c.JSON(http.StatusOK, gin.H{{"ready": true}})
}}

// TODO: add service-specific handlers
"""

REPO_GO = """package repository

import (
	"context"
	"errors"
)

// Repository defines the persistence interface.
// TODO: add methods for domain entities.
type Repository interface {{
	Ping(ctx context.Context) error
}}

// PostgresRepository implements Repository using PostgreSQL.
type PostgresRepository struct {{
	// TODO: inject *sql.DB or *pgxpool.Pool
}}

// NewPostgresRepository creates a new PostgresRepository.
func NewPostgresRepository() *PostgresRepository {{
	return &PostgresRepository{{}}
}}

// Ping verifies connectivity.
func (r *PostgresRepository) Ping(ctx context.Context) error {{
	// TODO: implement real ping
	return errors.New("not implemented")
}}
"""

MODEL_GO = """package model

import (
	"time"
)

// TODO: define domain models for {svc}

// BaseModel provides common fields.
type BaseModel struct {{
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}}
"""

PROTO_TMPL = """syntax = "proto3";

package {pkg};

option go_package = "{module}/api/proto;{pkg}";

// TODO: define service RPCs and messages

service {svc_pascal}Service {{
	rpc HealthCheck (HealthRequest) returns (HealthResponse);
}}

message HealthRequest {{}}

message HealthResponse {{
	bool healthy = 1;
}}
"""

DOCKERFILE = """# syntax=docker/dockerfile:1
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /bin/{svc} ./cmd/{svc}

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /bin/{svc} /bin/{svc}
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/bin/{svc}"]
"""

DOCKERFILE_DEV = """# syntax=docker/dockerfile:1
FROM golang:1.22-alpine
RUN go install github.com/cosmtrek/air@latest && \\
    go install github.com/go-delve/delve/cmd/dlv@latest
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
EXPOSE 8080 40000
CMD ["air", "-c", ".air.toml"]
"""

GO_MOD = """module {module}

go 1.22

require (
	github.com/gin-gonic/gin v1.9.1
	github.com/stretchr/testify v1.9.0
)

// TODO: pin additional dependencies (logger, tracer, config, DB driver)
"""

README_TMPL = """# {svc_pascal}

HelixTerminator microservice stub.

## TODO
- [ ] Implement domain logic
- [ ] Add gRPC server
- [ ] Wire persistence layer
- [ ] Add integration tests
- [ ] Add OpenTelemetry instrumentation
"""

TEST_GO = """package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"{module}/internal/handler"
)

func TestHealthCheck(t *testing.T) {{
	t.Skip("TODO: implement real health check test")
	gin.SetMode(gin.TestMode)
	h := handler.New()
	r := gin.New()
	r.GET("/health", h.HealthCheck)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/health", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}}
"""

MIGRATION = """-- 001_init.sql
-- TODO: create tables, indexes, and constraints for {svc}
"""


def pascal(s: str) -> str:
    return "".join(part.capitalize() for part in s.replace("-", "_").split("_"))


def generate():
    for svc, module in SERVICES:
        root = BASE / svc
        pkg = svc.replace("-", "_")
        svc_pascal = pascal(svc)

        files = {
            root / f"cmd/{svc}/main.go": MAIN_GO.format(svc=svc, module=module),
            root / "internal/server/server.go": SERVER_GO.format(module=module),
            root / "internal/handler/handler.go": HANDLER_GO,
            root / "internal/handler/handler_test.go": TEST_GO.format(module=module),
            root / "internal/repository/repository.go": REPO_GO,
            root / "internal/model/model.go": MODEL_GO.format(svc=svc),
            root / f"api/proto/{svc}.proto": PROTO_TMPL.format(pkg=pkg, module=module, svc_pascal=svc_pascal),
            root / "Dockerfile": DOCKERFILE.format(svc=svc),
            root / "Dockerfile.dev": DOCKERFILE_DEV,
            root / "go.mod": GO_MOD.format(module=module),
            root / "README.md": README_TMPL.format(svc_pascal=svc_pascal),
            root / "migrations/001_init.sql": MIGRATION.format(svc=svc),
        }

        for path, content in files.items():
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_text(content, encoding="utf-8")

    print(f"Generated {len(SERVICES)} service stubs.")


if __name__ == "__main__":
    generate()

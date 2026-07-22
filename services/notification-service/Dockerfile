# syntax=docker/dockerfile:1
#
# HEL-002 (2026-07-23): REPO-ROOT BUILD CONTEXT REQUIRED.
# Build from the repository root:
#
#   docker build -f services/notification-service/Dockerfile .
#
# Rationale: this module depends on Herald modules via local `replace`
# directives (`../../submodules/herald/...`), so the build context must
# contain BOTH the service and `submodules/herald` (populate it first:
# `git submodule update --init submodules/herald &&
#  git -C submodules/herald submodule update --init --recursive`).
# The previous service-local context (`COPY . .`) could not reach the
# replace targets at all, and its `golang:1.22` base predated even the
# module's own `go` directive. Built module-mode (GOWORK=off): go.work is
# git-ignored and never part of a build context; the manifest is
# module-mode-correct (HEL-002).
# NOTE: no `go mod download` pre-COPY cache layer — with local replaces
# the full source tree must be present before any module resolution.
FROM golang:1.26-alpine AS builder
WORKDIR /src/services/notification-service
COPY services/notification-service/ /src/services/notification-service/
COPY submodules/herald/ /src/submodules/herald/
RUN CGO_ENABLED=0 GOOS=linux GOWORK=off go build -ldflags="-w -s" -o /bin/notification-service ./cmd/notification-service

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /bin/notification-service /bin/notification-service
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/bin/notification-service"]

#!/bin/bash
set -euo pipefail

# dev/lint-all.sh - Run linters on all services

echo "Linting all services..."

for svc in services/*/; do
  if [ -f "$svc/go.mod" ]; then
    echo "Linting $svc..."
    (cd "$svc" && golangci-lint run ./...)
  fi
done

echo "Done."

#!/bin/bash
set -euo pipefail

# dev/build-all.sh - Build all services

echo "Building all services..."

for svc in services/*/; do
  if [ -f "$svc/go.mod" ]; then
    echo "Building $svc..."
    (cd "$svc" && go build -o bin/"$(basename "$svc")" ./cmd/"$(basename "$svc")")
  fi
done

echo "Done."

#!/bin/bash
set -euo pipefail

# dev/test-all.sh - Run tests on all services

echo "Testing all services..."

for svc in services/*/; do
  if [ -f "$svc/go.mod" ]; then
    echo "Testing $svc..."
    (cd "$svc" && go test ./...)
  fi
done

echo "Done."

#!/usr/bin/env bash
# pre_build_verification.sh
#
# Runs the constitution-inheritance gate before allowing a build/commit to
# proceed. Exits non-zero (blocking) if the gate script is missing or if it
# fails.
set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

GATE_SCRIPT="tests/verify_constitution_inheritance.sh"

if [ ! -f "${GATE_SCRIPT}" ]; then
    echo "============================================================"
    echo " PRE-BUILD BLOCKED"
    echo "============================================================"
    echo "Constitution-inheritance gate script not found: ${GATE_SCRIPT}"
    echo "Cannot verify constitution inheritance without this script."
    echo "Refusing to silently pass. Restore or create ${GATE_SCRIPT} and retry."
    echo "============================================================"
    exit 1
fi

if ! bash "${GATE_SCRIPT}"; then
    echo "============================================================"
    echo " PRE-BUILD BLOCKED"
    echo "============================================================"
    echo "Constitution-inheritance gate FAILED: ${GATE_SCRIPT}"
    echo "Fix the reported issues before building or committing."
    echo "============================================================"
    exit 1
fi

echo "Pre-build verification passed: constitution-inheritance gate OK."

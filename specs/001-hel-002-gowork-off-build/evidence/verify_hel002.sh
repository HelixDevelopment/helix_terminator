#!/usr/bin/env bash
# HEL-002 verification harness (T002) — specs/001-hel-002-gowork-off-build
# Purpose: prove notification-service builds in BOTH modes, drift resolved,
#          tidy-stable, container context reachable. Real exit codes only.
# Usage:   bash specs/001-hel-002-gowork-off-build/evidence/verify_hel002.sh <outdir>
# Exit:    0 iff every MANDATORY step passes. Container image build is
#          conditional: SKIP (container_runtime_absent) when no runtime.
set -u
ROOT="$(git rev-parse --show-toplevel)"; cd "$ROOT" || exit 99
SVC="services/notification-service"
OUT="${1:-specs/001-hel-002-gowork-off-build/evidence/run_$(date -u +%Y%m%dT%H%M%SZ)}"
mkdir -p "$OUT"
FAILS=0
step() { # step <name> <mandatory:1|0> <cmd...>
  local name="$1" mand="$2"; shift 2
  ( "$@" ) >"$OUT/$name.log" 2>&1
  local rc=$?
  if [ $rc -eq 0 ]; then echo "PASS $name (rc=0)";
  else echo "FAIL $name (rc=$rc)"; [ "$mand" = 1 ] && FAILS=$((FAILS+1)); fi
  return $rc
}
in_svc() { local d="$1"; shift; ( cd "$SVC" && env $d "$@" ); }

echo "== HEL-002 harness $(date -u +%Y-%m-%dT%H:%M:%SZ) OUT=$OUT =="

# 1) workspace matrix
step ws_build 1 bash -c "cd $SVC && go build ./..."
step ws_vet   1 bash -c "cd $SVC && go vet ./..."
step ws_test  1 bash -c "cd $SVC && go test ./..."
step ws_vet_integration 1 bash -c "cd $SVC && go vet -tags integration ./..."

# 2) module-mode (GOWORK=off) matrix
step off_build 1 bash -c "cd $SVC && GOWORK=off go build ./..."
step off_vet   1 bash -c "cd $SVC && GOWORK=off go vet ./..."
step off_test  1 bash -c "cd $SVC && GOWORK=off go test ./..."
step off_vet_integration 1 bash -c "cd $SVC && GOWORK=off go vet -tags integration ./..."

# 3) drift: recorded == built, both modes agree
step drift 1 bash -c '
  cd '"$SVC"' || exit 9
  rec=$(awk "/^\tgithub.com\/gin-gonic\/gin v/{print \$2}" go.mod)
  ws=$(go list -m -f "{{.Version}}" github.com/gin-gonic/gin) || exit 8
  off=$(GOWORK=off go list -m -f "{{.Version}}" github.com/gin-gonic/gin) || exit 7
  echo "recorded=$rec workspace=$ws gowork_off=$off"
  [ -n "$rec" ] && [ "$rec" = "$ws" ] && [ "$ws" = "$off" ]'

# 4) tidy-stability (non-mutating)
step tidy_stable 1 bash -c "cd $SVC && GOWORK=off go mod tidy -diff"

# 5) container context reachability (mechanical)
step ctx_reach 1 bash -c '
  svc='"$SVC"'
  ok=1
  # every ../../ replace target must exist (with go.mod) relative to repo root
  while read -r p; do
    rel="${p#../../}"
    if [ -f "$rel/go.mod" ]; then echo "OK  $rel"; else echo "MISS $rel"; ok=0; fi
  done < <(awk "/=> \.\.\/\.\.\//{print \$NF}" "$svc/go.mod")
  # Dockerfile must be repo-root-context invocable: its COPY lines must
  # reference the service path and submodules/herald from the root
  grep -q "COPY services/notification-service" "$svc/Dockerfile" || { echo "Dockerfile: no root-context COPY of service"; ok=0; }
  grep -q "COPY submodules/herald" "$svc/Dockerfile" || { echo "Dockerfile: no COPY of submodules/herald"; ok=0; }
  [ $ok -eq 1 ]'

# 6) real container image build — conditional (probe the ARTIFACT path, not prerequisites)
RUNTIME=""
command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1 && RUNTIME=docker
[ -z "$RUNTIME" ] && command -v podman >/dev/null 2>&1 && podman info >/dev/null 2>&1 && RUNTIME=podman
if [ -n "$RUNTIME" ]; then
  step docker_build 1 bash -c "$RUNTIME build -f $SVC/Dockerfile -t hel002-notification-service:proof ."
else
  echo "SKIP docker_build (container_runtime_absent — no working docker/podman on this host)" | tee "$OUT/docker_build.log"
fi

echo "== VERDICT: mandatory_failures=$FAILS =="
[ $FAILS -eq 0 ] && echo "OVERALL PASS" || echo "OVERALL FAIL"
exit $FAILS

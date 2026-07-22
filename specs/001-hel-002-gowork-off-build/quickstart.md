# Quickstart Validation: HEL-002 (notification-service builds outside the workspace)

**Prerequisites**: clean checkout; submodules populated:
`git submodule update --init submodules/herald && git -C submodules/herald submodule update --init --recursive`

All commands run from the repository root; exit codes captured directly
(never through a pipeline tail — Constitution Principle I).

## 1. Module-mode (GOWORK=off) matrix — must ALL exit 0

```bash
cd services/notification-service
GOWORK=off go build ./...
GOWORK=off go vet ./...
GOWORK=off go test ./...
GOWORK=off go vet -tags integration ./...
```

## 2. Workspace-mode matrix — must ALL exit 0

```bash
cd services/notification-service
go build ./... && go vet ./... && go test ./... && go vet -tags integration ./...
```

## 3. Drift resolved — recorded == built, both modes agree

```bash
cd services/notification-service
go list -m github.com/gin-gonic/gin            # expect v1.12.0
GOWORK=off go list -m github.com/gin-gonic/gin # expect v1.12.0 (same)
grep 'gin-gonic/gin v' go.mod                  # expect v1.12.0 recorded
```

## 4. Tidy-stable — repeat tidy is a no-op

```bash
cd services/notification-service
GOWORK=off go mod tidy -diff   # expect exit 0, empty diff
```

## 5. Container context reachability (mechanical)

```bash
# every local replace target must exist inside the repo-root build context
awk '/=> \.\.\/\.\.\//{print $NF}' services/notification-service/go.mod \
  | sed 's#^\.\./\.\./##' | while read -r p; do
    [ -e "$p/go.mod" ] && echo "OK $p" || { echo "MISSING $p"; exit 1; }
  done
# and the Dockerfile must be root-context invocable:
docker build -f services/notification-service/Dockerfile .   # iff a runtime exists
```

If no container runtime exists on the host, record an honest SKIP for the
real image build (the mechanical reachability check above still runs) — never
a faked pass.

## Expected outcome

Steps 1–4 green + step 5 reachability 100% ⇒ SC-001..SC-004 satisfied, with
the real `docker build` leg either green or an explicitly recorded SKIP.

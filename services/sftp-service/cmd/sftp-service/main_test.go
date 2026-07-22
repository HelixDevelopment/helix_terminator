package main

import "testing"

// TestMainStub is an honest §11.4.3 SKIP: main() opens a real HTTP
// listener and blocks on os/signal.Notify until SIGINT/SIGTERM, so it
// cannot be exercised as an in-process unit test without either leaking
// a background listener or requiring process-level signal injection.
// main()'s real behaviour (route wiring, graceful shutdown) is covered
// by internal/server's own tests plus integration/e2e tests that boot
// the actual binary - this stub deliberately does NOT assert a
// tautology. Mirrors the identical, already-reviewed §11.4.3 pattern
// used by org-service, workspace-service, pki-service, config-service,
// user-service, gateway-service, and keychain-service's own
// cmd/*/main_test.go.
func TestMainStub(t *testing.T) {
	t.Skip("§11.4.3: main() blocks on ListenAndServe + OS signal wait; covered by internal/server tests + integration/e2e boot tests, not an in-process unit test - tracked follow-up: promote to a real e2e boot test per §11.4.52 autonomous-validation")
}

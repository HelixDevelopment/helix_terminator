// Package testutil provides shared stress/chaos test helpers for
// port-forward-service. These are pure-logic utilities with no build-tag
// gating — they compile whenever test code imports them.
//
// Constitution §11.4.85 (stress + chaos test mandate) requires every fix
// or improvement to ship with full-automation stress AND chaos test suites.
// This package supplies the common plumbing: latency recording, percentile
// computation, concurrent goroutine orchestration, and an in-memory fake
// repository for tests that exercise handler logic without a real database.
package testutil

import (
	"context"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/helixdevelopment/port-forward-service/internal/model"
)

// LatencyRecorder is a thread-safe collector of per-operation durations.
// Sustained-load tests (§11.4.85) call Record once per iteration;
// Percentiles computes p50/p95/p99 from the collected samples.
type LatencyRecorder struct {
	mu   sync.Mutex
	durs []time.Duration
}

// NewLatencyRecorder returns a ready-to-use recorder.
func NewLatencyRecorder() *LatencyRecorder {
	return &LatencyRecorder{}
}

// Record appends d to the recorder's sample set. Thread-safe.
func (r *LatencyRecorder) Record(d time.Duration) {
	r.mu.Lock()
	r.durs = append(r.durs, d)
	r.mu.Unlock()
}

// Len returns the number of recorded samples.
func (r *LatencyRecorder) Len() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.durs)
}

// Percentiles returns the p50, p95, and p99 latencies from all recorded
// samples. Panics if zero samples have been recorded — callers MUST
// record at least one sample before calling.
func (r *LatencyRecorder) Percentiles() (p50, p95, p99 time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.durs) == 0 {
		panic("Percentiles called with zero samples")
	}

	sorted := make([]time.Duration, len(r.durs))
	copy(sorted, r.durs)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	p50 = sorted[percentileIndex(len(sorted), 50)]
	p95 = sorted[percentileIndex(len(sorted), 95)]
	p99 = sorted[percentileIndex(len(sorted), 99)]
	return
}

// percentileIndex returns the index into a sorted slice for the given
// percentile (0-100). Uses the nearest-rank method.
func percentileIndex(n, pct int) int {
	idx := (pct * (n - 1)) / 100
	if idx >= n {
		idx = n - 1
	}
	return idx
}

// RunConcurrent launches n goroutines, each calling fn(id) where id is
// 0..n-1. It blocks until all goroutines return. If any goroutine
// panics, the panic is captured and re-raised in the calling goroutine
// (so the test fails cleanly rather than silently losing the panic).
//
// Use for concurrent-contention tests (§11.4.85 sustained-load +
// concurrent contention invariant: N>=10 parallel invocations, no
// deadlock, no resource leak).
func RunConcurrent(t *testing.T, n int, fn func(id int)) {
	t.Helper()

	var wg sync.WaitGroup
	var errCount atomic.Int64

	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(id int) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					errCount.Add(1)
					t.Errorf("goroutine %d panicked: %v", id, r)
				}
			}()
			fn(id)
		}(i)
	}
	wg.Wait()

	if errCount.Load() > 0 {
		t.Fatalf("%d goroutine(s) panicked during concurrent execution", errCount.Load())
	}
}

// FakeRepo is a thread-safe in-memory implementation of
// handler.ForwardRepository for stress and chaos tests that exercise
// handler logic without requiring a real database. All operations are
// backed by a sync.Map so they are safe under concurrent access.
type FakeRepo struct {
	mu       sync.RWMutex
	forwards map[uuid.UUID]*model.PortForward
}

// NewFakeRepo returns a ready-to-use fake repository.
func NewFakeRepo() *FakeRepo {
	return &FakeRepo{forwards: make(map[uuid.UUID]*model.PortForward)}
}

// Ping always succeeds — the fake repo is always "ready".
func (r *FakeRepo) Ping(_ context.Context) error { return nil }

// CreateForward stores a deep copy of the forward.
func (r *FakeRepo) CreateForward(_ context.Context, fwd *model.PortForward) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *fwd
	r.forwards[fwd.ID] = &cp
	return nil
}

// GetForwardByID retrieves a forward by ID.
func (r *FakeRepo) GetForwardByID(_ context.Context, id uuid.UUID) (*model.PortForward, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	fwd, ok := r.forwards[id]
	if !ok {
		return nil, model.ErrForwardNotFound
	}
	cp := *fwd
	return &cp, nil
}

// ListForwards returns forwards filtered by hostID (zero-value = all),
// with limit/offset pagination.
func (r *FakeRepo) ListForwards(_ context.Context, hostID uuid.UUID, limit, offset int) ([]*model.PortForward, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var all []*model.PortForward
	for _, fwd := range r.forwards {
		if hostID != uuid.Nil && fwd.HostID != hostID {
			continue
		}
		cp := *fwd
		all = append(all, &cp)
	}

	total := len(all)
	if offset >= total {
		return nil, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return all[offset:end], total, nil
}

// UpdateForward applies a map of updates to the forward.
func (r *FakeRepo) UpdateForward(_ context.Context, id uuid.UUID, updates map[string]interface{}) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	fwd, ok := r.forwards[id]
	if !ok {
		return model.ErrForwardNotFound
	}
	if v, ok := updates["local_port"].(int); ok {
		fwd.LocalPort = v
	}
	if v, ok := updates["remote_port"].(int); ok {
		fwd.RemotePort = v
	}
	if v, ok := updates["remote_host"].(string); ok {
		fwd.RemoteHost = v
	}
	if v, ok := updates["protocol"].(string); ok {
		fwd.Protocol = v
	}
	if v, ok := updates["status"].(string); ok {
		fwd.Status = v
	}
	return nil
}

// UpdateStatus sets the forward's status field.
func (r *FakeRepo) UpdateStatus(_ context.Context, id uuid.UUID, status string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	fwd, ok := r.forwards[id]
	if !ok {
		return model.ErrForwardNotFound
	}
	fwd.Status = status
	return nil
}

// DeleteForward removes the forward from the in-memory store.
func (r *FakeRepo) DeleteForward(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.forwards[id]; !ok {
		return model.ErrForwardNotFound
	}
	delete(r.forwards, id)
	return nil
}

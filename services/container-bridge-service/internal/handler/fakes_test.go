package handler

import (
	"context"
	"fmt"
	"io"

	ctrruntime "digital.vasic.containers/pkg/runtime"
	"github.com/google/uuid"
	"github.com/helixdevelopment/container-bridge-service/internal/containerrt"
	"github.com/helixdevelopment/container-bridge-service/internal/model"
)

// fakeRepo is an in-memory BridgeStore for handler unit tests. It requires no
// live database, letting handler logic (including runtime reconciliation) be
// exercised deterministically per §11.4.50/§11.4.98.
type fakeRepo struct {
	createErr error
	created   []*model.ContainerBridge

	getResult *model.ContainerBridge
	getErr    error

	listResult []*model.ContainerBridge
	listTotal  int
	listErr    error

	updateCalls []fakeUpdateCall
	updateErr   error

	deleteErr   error
	deleteCalls []uuid.UUID

	pingErr error
}

type fakeUpdateCall struct {
	id      uuid.UUID
	updates map[string]interface{}
}

func (f *fakeRepo) CreateBridge(_ context.Context, bridge *model.ContainerBridge) error {
	if f.createErr != nil {
		return f.createErr
	}
	f.created = append(f.created, bridge)
	return nil
}

func (f *fakeRepo) GetBridgeByID(_ context.Context, _ uuid.UUID) (*model.ContainerBridge, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.getResult, nil
}

func (f *fakeRepo) ListBridges(
	_ context.Context, _ uuid.UUID, _, _ int,
) ([]*model.ContainerBridge, int, error) {
	return f.listResult, f.listTotal, f.listErr
}

func (f *fakeRepo) UpdateBridge(_ context.Context, id uuid.UUID, updates map[string]interface{}) error {
	f.updateCalls = append(f.updateCalls, fakeUpdateCall{id: id, updates: updates})
	return f.updateErr
}

func (f *fakeRepo) DeleteBridge(_ context.Context, id uuid.UUID) error {
	f.deleteCalls = append(f.deleteCalls, id)
	return f.deleteErr
}

func (f *fakeRepo) Ping(_ context.Context) error {
	return f.pingErr
}

// fakeBackend is an in-memory containerrt.Backend for handler unit tests. It
// implements the FULL runtime.ContainerRuntime interface plus RunFromImage so
// it satisfies containerrt.Backend, with every call recorded so tests can
// assert real lifecycle calls were made (not skipped/bluffed).
type fakeBackend struct {
	name      string
	available bool

	startErr   error
	startCalls []string

	stopErr   error
	stopCalls []string

	removeErr   error
	removeCalls []string

	statusFunc func(id string) (*ctrruntime.ContainerStatus, error)

	runFromImageFunc func(name, image string, ports []string) (string, error)
	runFromImageCalls []struct {
		name, image string
		ports       []string
	}
}

func (f *fakeBackend) Name() string { return f.name }

func (f *fakeBackend) Version(_ context.Context) (string, error) { return "fake-1.0", nil }

func (f *fakeBackend) IsAvailable(_ context.Context) bool { return f.available }

func (f *fakeBackend) Start(_ context.Context, id string, _ ...ctrruntime.StartOption) error {
	f.startCalls = append(f.startCalls, id)
	return f.startErr
}

func (f *fakeBackend) Stop(_ context.Context, id string, _ ...ctrruntime.StopOption) error {
	f.stopCalls = append(f.stopCalls, id)
	return f.stopErr
}

func (f *fakeBackend) Remove(_ context.Context, id string, _ ...ctrruntime.RemoveOption) error {
	f.removeCalls = append(f.removeCalls, id)
	return f.removeErr
}

func (f *fakeBackend) Status(_ context.Context, id string) (*ctrruntime.ContainerStatus, error) {
	if f.statusFunc == nil {
		return nil, fmt.Errorf("fakeBackend: no such container: %s", id)
	}
	return f.statusFunc(id)
}

func (f *fakeBackend) List(_ context.Context, _ ctrruntime.ListFilter) ([]ctrruntime.ContainerInfo, error) {
	return nil, nil
}

func (f *fakeBackend) Stats(_ context.Context, _ string) (*ctrruntime.ContainerStats, error) {
	return nil, fmt.Errorf("fakeBackend: Stats not implemented")
}

func (f *fakeBackend) Exec(_ context.Context, _ string, _ []string) (*ctrruntime.ExecResult, error) {
	return nil, fmt.Errorf("fakeBackend: Exec not implemented")
}

func (f *fakeBackend) Logs(_ context.Context, _ string, _ ...ctrruntime.LogOption) (io.ReadCloser, error) {
	return nil, fmt.Errorf("fakeBackend: Logs not implemented")
}

func (f *fakeBackend) RunFromImage(
	_ context.Context, name, image string, ports []string,
) (string, error) {
	f.runFromImageCalls = append(f.runFromImageCalls, struct {
		name, image string
		ports       []string
	}{name, image, ports})
	if f.runFromImageFunc == nil {
		return "", fmt.Errorf("fakeBackend: RunFromImage not configured")
	}
	return f.runFromImageFunc(name, image, ports)
}

var _ containerrt.Backend = (*fakeBackend)(nil)

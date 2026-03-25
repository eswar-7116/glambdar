package docker

import (
	"context"
	"testing"

	"github.com/moby/moby/client"
)

type MockDockerAPI struct {
	ContainerCreateFunc func(ctx context.Context, options client.ContainerCreateOptions) (client.ContainerCreateResult, error)
	ContainerStartFunc  func(ctx context.Context, containerID string, options client.ContainerStartOptions) (client.ContainerStartResult, error)
	ContainerKillFunc   func(ctx context.Context, containerID string, options client.ContainerKillOptions) (client.ContainerKillResult, error)
	CloseFunc           func() error
	ImagePullFunc       func(ctx context.Context, refStr string, options client.ImagePullOptions) (client.ImagePullResponse, error)
	ImageInspectFunc    func(ctx context.Context, imageID string, inspectOpts ...client.ImageInspectOption) (client.ImageInspectResult, error)
}

func (m *MockDockerAPI) ImagePull(ctx context.Context, refStr string, options client.ImagePullOptions) (client.ImagePullResponse, error) {
	if m.ImagePullFunc != nil {
		return m.ImagePullFunc(ctx, refStr, options)
	}
	return nil, nil
}

func (m *MockDockerAPI) ImageInspect(ctx context.Context, imageID string, inspectOpts ...client.ImageInspectOption) (client.ImageInspectResult, error) {
	if m.ImageInspectFunc != nil {
		return m.ImageInspectFunc(ctx, imageID, inspectOpts...)
	}
	return client.ImageInspectResult{}, nil
}

func (m *MockDockerAPI) ContainerCreate(ctx context.Context, options client.ContainerCreateOptions) (client.ContainerCreateResult, error) {
	return m.ContainerCreateFunc(ctx, options)
}

func (m *MockDockerAPI) ContainerStart(ctx context.Context, containerID string, options client.ContainerStartOptions) (client.ContainerStartResult, error) {
	return m.ContainerStartFunc(ctx, containerID, options)
}

func (m *MockDockerAPI) ContainerKill(ctx context.Context, containerID string, options client.ContainerKillOptions) (client.ContainerKillResult, error) {
	return m.ContainerKillFunc(ctx, containerID, options)
}

func (m *MockDockerAPI) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}

func TestDocker_ContainerCreate(t *testing.T) {
	mock := &MockDockerAPI{
		ContainerCreateFunc: func(ctx context.Context, options client.ContainerCreateOptions) (client.ContainerCreateResult, error) {
			if options.Name != "test-container" {
				t.Errorf("expected name test-container, got %s", options.Name)
			}
			return client.ContainerCreateResult{ID: "test-id"}, nil
		},
	}

	d := &Docker{
		client: mock,
	}

	id, err := d.ContainerCreate(context.Background(), "test-container", "/some/dir")
	if err != nil {
		t.Fatalf("ContainerCreate failed: %v", err)
	}

	if id != "test-id" {
		t.Errorf("expected id test-id, got %s", id)
	}
}

func TestDocker_ContainerStart(t *testing.T) {
	mock := &MockDockerAPI{
		ContainerStartFunc: func(ctx context.Context, containerID string, options client.ContainerStartOptions) (client.ContainerStartResult, error) {
			if containerID != "test-id" {
				t.Errorf("expected id test-id, got %s", containerID)
			}
			return client.ContainerStartResult{}, nil
		},
	}

	d := &Docker{
		client: mock,
	}

	err := d.ContainerStart(context.Background(), "test-id")
	if err != nil {
		t.Fatalf("ContainerStart failed: %v", err)
	}
}

func TestDocker_ContainerKill(t *testing.T) {
	mock := &MockDockerAPI{
		ContainerKillFunc: func(ctx context.Context, containerID string, options client.ContainerKillOptions) (client.ContainerKillResult, error) {
			if containerID != "test-id" {
				t.Errorf("expected id test-id, got %s", containerID)
			}
			if options.Signal != "SIGKILL" {
				t.Errorf("expected signal SIGKILL, got %s", options.Signal)
			}
			return client.ContainerKillResult{}, nil
		},
	}

	d := &Docker{
		client: mock,
	}

	err := d.ContainerKill(context.Background(), "test-id")
	if err != nil {
		t.Fatalf("ContainerKill failed: %v", err)
	}
}

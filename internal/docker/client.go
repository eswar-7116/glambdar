package docker

import (
	"context"
	"fmt"
	"sync"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/mount"
	"github.com/moby/moby/client"
)

type DockerAPI interface {
	ContainerCreate(ctx context.Context, options client.ContainerCreateOptions) (client.ContainerCreateResult, error)
	ContainerStart(ctx context.Context, containerID string, options client.ContainerStartOptions) (client.ContainerStartResult, error)
	ContainerKill(ctx context.Context, containerID string, options client.ContainerKillOptions) (client.ContainerKillResult, error)
	Close() error
}

type Docker struct {
	client     DockerAPI
	once       sync.Once
	UDSPath    string
	WorkerPath string
}

func (d *Docker) GetClient() (DockerAPI, error) {
	var err error

	d.once.Do(func() {
		if d.client == nil {
			d.client, err = client.New()
		}
	})

	if err != nil {
		return nil, fmt.Errorf("failed to initialize docker client: %w", err)
	}

	return d.client, nil
}

func (d *Docker) Close() error {
	if d.client != nil {
		return d.client.Close()
	}
	return nil
}

func (d *Docker) ContainerCreate(ctx context.Context, name, funcDir string) (string, error) {
	cli, err := d.GetClient()
	if err != nil {
		return "", err
	}

	container, err := cli.ContainerCreate(ctx, client.ContainerCreateOptions{
		Name:  name,
		Image: "node:25-slim",
		Config: &container.Config{
			Hostname: "glambdar",
			Cmd:      []string{"node", "/glambdar/worker.js", "/function"},
		},
		HostConfig: &container.HostConfig{
			AutoRemove: true, // TODO: Remove later
			Mounts: []mount.Mount{
				{
					Type:   mount.TypeBind,
					Source: funcDir,
					Target: "/function",
				},
				{
					Type:   mount.TypeBind,
					Source: d.UDSPath,
					Target: "/glambdar/glambdar.sock",
				},
				{
					Type:   mount.TypeBind,
					Source: d.WorkerPath,
					Target: "/glambdar/worker.js",
				},
			},
			Resources: container.Resources{
				Memory:   128 * 1024 * 1024, // 128m in bytes
				NanoCPUs: 500_000_000,       // 0.5 CPUs in nanocpus
			},
		},
	})
	if err != nil {
		return "", err
	}

	return container.ID, nil
}

func (d *Docker) ContainerStart(ctx context.Context, id string) error {
	cli, err := d.GetClient()
	if err != nil {
		return err
	}

	if _, err := cli.ContainerStart(ctx, id, client.ContainerStartOptions{}); err != nil {
		return err
	}

	return nil
}

func (d *Docker) ContainerKill(ctx context.Context, id string) error {
	cli, err := d.GetClient()
	if err != nil {
		return err
	}

	_, err = cli.ContainerKill(ctx, id, client.ContainerKillOptions{
		Signal: "SIGKILL",
	})
	return err
}

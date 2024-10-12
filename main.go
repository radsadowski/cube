package main

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/client"
	"github.com/radekska/cube/internal/task"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	d, r := createContainer(ctx)
	fmt.Printf("docker: %+v\n", d)
	fmt.Printf("result: %+v\n", r)

	time.Sleep(30 * time.Second)
	stopContainer(ctx, d, r.ContainerId)
}

func createContainer(ctx context.Context) (*task.Docker, *task.DockerResult) {
	c := task.Config{
		Name:  "test-container-1",
		Image: "postgres:13",
		Env: []string{
			"POSTGRES_USER=cube",
			"POSTGRES_PASSWORD=secret",
		},
	}
	dc, _ := client.NewClientWithOpts(client.FromEnv)
	d := task.Docker{
		Client: dc,
		Config: c,
	}
	result := d.Run(ctx)
	if result.Error != nil {
		fmt.Printf("%v\n", result.Error)
		return nil, nil
	}
	fmt.Printf("Container %s is running with config %v\n", result.ContainerId, c)
	return &d, &result
}

func stopContainer(ctx context.Context, d *task.Docker, id string) *task.DockerResult {
	t := d.Stop(ctx, id)
	if t.Error != nil {
		fmt.Print(t.Error)
		return nil
	}
	return &t
}

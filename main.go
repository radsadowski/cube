package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
	"github.com/radekska/cube/internal/task"
	"github.com/radekska/cube/internal/worker"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	db := make(map[uuid.UUID]*task.Task)
	w1 := worker.Worker{
		Queue: *queue.New(),
		Db:    db,
	}
	w2 := worker.Worker{
		Queue: *queue.New(),
		Db:    db,
	}
	w3 := worker.Worker{
		Queue: *queue.New(),
		Db:    db,
	}

	var wg sync.WaitGroup
	for _, w := range []worker.Worker{w1, w2, w3} {
		wg.Add(1)
		w := w
		go func() {
			defer wg.Done()

			t := task.Task{
				ID:    uuid.New(),
				Name:  "test-container-" + uuid.NewString(),
				State: task.Scheduled,
				Image: "strm/helloworld-http",
			}

			w.AddTask(t)

			fmt.Println("starting task")
			result := w.RunTask(ctx)
			if result.Error != nil {
				fmt.Printf("%+v\n", result)
				return
			}

			t.ContainerID = result.ContainerId
			fmt.Printf("task [%s] is running in container [%s]\n", t.ID, t.ContainerID)
			time.Sleep(30 * time.Second)

			fmt.Printf("stopping task [%s]\n", t.ID)
			t.State = task.Completed
			w.AddTask(t)
			result = w.RunTask(ctx)
			if result.Error != nil {
				fmt.Printf("%+v\n", result)
				return
			}
		}()
	}

	wg.Wait()
}

// func main() {
// 	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
// 	defer cancel()
// 	d, r := createContainer(ctx)
// 	fmt.Printf("docker: %+v\n", d)
// 	fmt.Printf("result: %+v\n", r)
//
// 	time.Sleep(30 * time.Second)
// 	stopContainer(ctx, d, r.ContainerId)
// }
//
// func createContainer(ctx context.Context) (*task.Docker, *task.DockerResult) {
// 	c := task.Config{
// 		Name:  "test-container-1",
// 		Image: "postgres:13",
// 		Env: []string{
// 			"POSTGRES_USER=cube",
// 			"POSTGRES_PASSWORD=secret",
// 		},
// 	}
// 	dc, _ := client.NewClientWithOpts(client.FromEnv)
// 	d := task.Docker{
// 		Client: dc,
// 		Config: c,
// 	}
// 	result := d.Run(ctx)
// 	if result.Error != nil {
// 		fmt.Printf("%v\n", result.Error)
// 		return nil, nil
// 	}
// 	fmt.Printf("Container %s is running with config %v\n", result.ContainerId, c)
// 	return &d, &result
// }
//
// func stopContainer(ctx context.Context, d *task.Docker, id string) *task.DockerResult {
// 	t := d.Stop(ctx, id)
// 	if t.Error != nil {
// 		fmt.Print(t.Error)
// 		return nil
// 	}
// 	return &t
// }

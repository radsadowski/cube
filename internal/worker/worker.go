package worker

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
	"github.com/radekska/cube/internal/task"
)

type Worker struct {
	Name      string
	Queue     queue.Queue
	Db        map[uuid.UUID]*task.Task
	TaskCount int
}

func (w *Worker) CollectStats() {
	fmt.Println("collecting stats")
}

func (w *Worker) RunTask() {
	fmt.Println("it will run a task")
}

func (w *Worker) AddTask(t task.Task) {
	w.Queue.Enqueue(t)
}

func (w *Worker) StartTask(ctx context.Context, t task.Task) task.DockerResult {
	t.StartTime = time.Now().UTC()

	config := task.NewConfig(&t)
	dc := task.NewDocker(config)

	result := dc.Run(ctx)
	if result.Error != nil {
		log.Printf("running task [%s]: %v\n", t.ID, result.Error)
		t.State = task.Failed
		w.Db[t.ID] = &t
		return result
	}

	t.ContainerID = result.ContainerId
	t.State = task.Running
	w.Db[t.ID] = &t
	return result
}

func (w *Worker) StopTask(ctx context.Context, t task.Task) task.DockerResult {
	config := task.NewConfig(&t)
	dc := task.NewDocker(config)

	result := dc.Stop(ctx, t.ContainerID)
	if result.Error != nil {
		log.Printf("stopping container [%s]: %v\n", result.ContainerId, result.Error)
	} else {
		log.Printf("stopped and removed container [%s] for task [%s]\n", t.ContainerID, t.ID)
	}
	t.State = task.Completed
	t.FinishTime = time.Now().UTC()
	w.Db[t.ID] = &t
	return result
}

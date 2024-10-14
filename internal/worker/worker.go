package worker

import (
	"context"
	"errors"
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

func (w *Worker) ListTasks() []task.Task {
	tasks := []task.Task{}
	for _, t := range w.Db {
		tasks = append(tasks, *t)
	}
	return tasks
}

func (w *Worker) RunTask(ctx context.Context) task.DockerResult {
	t := w.Queue.Dequeue()
	if t == nil {
		log.Println("no tasks in the queue")
		return task.DockerResult{Error: nil}
	}
	taskQueued := t.(task.Task)
	taskPersisted := w.Db[taskQueued.ID]

	if taskPersisted == nil {
		taskPersisted = &taskQueued
		w.Db[taskQueued.ID] = &taskQueued
	}

	var result task.DockerResult
	if task.ValidStateTransition(taskPersisted.State, taskQueued.State) {
		switch taskQueued.State {
		case task.Scheduled:
			result = w.StartTask(ctx, taskQueued)
		case task.Completed:
			result = w.StopTask(ctx, taskQueued)
		default:
			result = task.DockerResult{Error: errors.New("invalid task state")}
		}
	} else {
		err := fmt.Errorf("invalid transistion from [%v] to [%v]", taskPersisted.State, taskQueued.State)
		result = task.DockerResult{Error: err}
	}
	return result
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

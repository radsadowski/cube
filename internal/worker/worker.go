package worker

import (
	"fmt"

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

func (w *Worker) StartTask() {
	fmt.Println("it will start a task")
}

func (w *Worker) StopTask() {
	fmt.Println("it will stop a task")
}

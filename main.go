package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
	"github.com/radekska/cube/internal/manager"
	"github.com/radekska/cube/internal/task"
	"github.com/radekska/cube/internal/worker"
)

func main() {
	workerHost := os.Getenv("CUBE_WORKER_HOST")
	if workerHost == "" {
		workerHost = "0.0.0.0"
	}
	workerPort, _ := strconv.Atoi(os.Getenv("CUBE_WORKER_PORT"))
	if workerPort == 0 {
		workerPort = 5555
	}

	managerHost := os.Getenv("CUBE_MANAGER_HOST")
	if managerHost == "" {
		managerHost = "0.0.0.0"
	}
	managerPort, _ := strconv.Atoi(os.Getenv("CUBE_MANAGER_PORT"))
	if managerPort == 0 {
		managerPort = 5556
	}

	// Start worker
	fmt.Println("Starting Cube worker")
	w := worker.Worker{
		Name:  "worker-1",
		Queue: *queue.New(),
		Db:    make(map[uuid.UUID]*task.Task),
	}
	go w.CollectStats()

	workerAPI := worker.NewAPI(workerHost, workerPort, &w)
	go runTasks(&w)
	go workerAPI.Start()

	// Start manager pointing at the worker
	workerAddr := fmt.Sprintf("%s:%d", workerHost, workerPort)
	fmt.Println("Starting Cube manager")
	m := manager.New([]string{workerAddr})
	go m.DoWork()

	managerAPI := manager.NewAPI(managerHost, managerPort, m)
	managerAPI.Start()
}

func runTasks(w *worker.Worker) {
	ctx := context.Background()
	for {
		if w.Queue.Len() != 0 {
			result := w.RunTask(ctx)
			if result.Error != nil {
				log.Printf("running task: %v\n", result.Error)
			}
		} else {
			log.Println("no tasks in the workers queue")
		}
		log.Println("sleeping for 10s...")
		time.Sleep(10 * time.Second)
	}
}

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
	"github.com/radekska/cube/internal/task"
	"github.com/radekska/cube/internal/worker"
)

func main() {
	for {
		s, err := worker.GetStats()
		if err != nil {
			panic(err)
		}
		fmt.Println(s.MemTotalKb())
		fmt.Println(s.MemAvailableKb())
		fmt.Println(s.MemUsedPercent())

		fmt.Println(s.DiskTotal())
		fmt.Println(s.CpuUsage())

		time.Sleep(1 * time.Second)
	}

	host := os.Getenv("CUBE_HOST")
	port, _ := strconv.Atoi(os.Getenv("CUBE_PORT"))
	fmt.Println("Starting Cube worker")
	w := worker.Worker{
		Queue: *queue.New(),
		Db:    make(map[uuid.UUID]*task.Task),
	}

	api := worker.NewAPI(host, port, &w)
	go runTasks(&w)
	api.Start()
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

package manager

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
	"github.com/radekska/cube/internal/task"
)

type Manager struct {
	Pending        queue.Queue
	TaskDb         map[uuid.UUID]*task.Task
	EventDb        map[uuid.UUID]*task.TaskEvent
	Workers        []string
	WorkersTaskMap map[string][]uuid.UUID
	TaskWorkerMap  map[uuid.UUID]string
	LastWorker     int
}

func New(workers []string) *Manager {
	taskDb := make(map[uuid.UUID]*task.Task)
	eventDb := make(map[uuid.UUID]*task.TaskEvent)
	workersTaskMap := make(map[string][]uuid.UUID)
	taskWorkerMap := make(map[uuid.UUID]string)

	for _, w := range workers {
		workersTaskMap[w] = []uuid.UUID{}
	}

	return &Manager{
		Pending:        *queue.New(),
		TaskDb:         taskDb,
		EventDb:        eventDb,
		Workers:        workers,
		WorkersTaskMap: workersTaskMap,
		TaskWorkerMap:  taskWorkerMap,
	}
}

// SelectWorker picks the next worker using round-robin.
func (m *Manager) SelectWorker() string {
	if len(m.Workers) == 0 {
		return ""
	}
	w := m.Workers[m.LastWorker%len(m.Workers)]
	m.LastWorker++
	return w
}

// AddTask enqueues a TaskEvent for dispatch and records it in the event DB.
func (m *Manager) AddTask(te task.TaskEvent) {
	m.Pending.Enqueue(te)
	m.EventDb[te.ID] = &te
	m.TaskDb[te.Task.ID] = &te.Task
}

// SendWork dequeues one pending TaskEvent, selects a worker, and POSTs the event to it.
func (m *Manager) SendWork() {
	if m.Pending.Len() == 0 {
		log.Println("no pending tasks")
		return
	}

	w := m.SelectWorker()
	if w == "" {
		log.Println("no workers available")
		return
	}

	e := m.Pending.Dequeue()
	te, ok := e.(task.TaskEvent)
	if !ok {
		log.Println("unexpected type in pending queue")
		return
	}

	t := te.Task
	log.Printf("sending task [%s] to worker [%s]\n", t.ID, w)

	m.WorkersTaskMap[w] = append(m.WorkersTaskMap[w], t.ID)
	m.TaskWorkerMap[t.ID] = w

	t.State = task.Scheduled
	m.TaskDb[t.ID] = &t

	data, err := json.Marshal(te)
	if err != nil {
		log.Printf("serializing task event [%s]: %v\n", te.ID, err)
		return
	}

	url := fmt.Sprintf("http://%s/tasks", w)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		log.Printf("posting task [%s] to worker [%s]: %v\n", t.ID, w, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		log.Printf("unexpected status [%d] from worker [%s] for task [%s]\n", resp.StatusCode, w, t.ID)
		return
	}

	log.Printf("task [%s] accepted by worker [%s]\n", t.ID, w)
}

// UpdateTasks polls every known worker for its task list and syncs state into the manager DB.
func (m *Manager) UpdateTasks() {
	for _, w := range m.Workers {
		log.Printf("polling worker [%s] for task updates\n", w)

		url := fmt.Sprintf("http://%s/tasks", w)
		resp, err := http.Get(url)
		if err != nil {
			log.Printf("getting tasks from worker [%s]: %v\n", w, err)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			log.Printf("unexpected status [%d] from worker [%s]\n", resp.StatusCode, w)
			resp.Body.Close()
			continue
		}

		var tasks []task.Task
		if err := json.NewDecoder(resp.Body).Decode(&tasks); err != nil {
			log.Printf("decoding tasks from worker [%s]: %v\n", w, err)
			resp.Body.Close()
			continue
		}
		resp.Body.Close()

		for _, t := range tasks {
			local, exists := m.TaskDb[t.ID]
			if !exists {
				log.Printf("task [%s] from worker [%s] is unknown — skipping\n", t.ID, w)
				continue
			}
			if local.State != t.State {
				log.Printf("task [%s] state updated: [%v] -> [%v]\n", t.ID, local.State, t.State)
				local.State = t.State
				local.ContainerID = t.ContainerID
				local.StartTime = t.StartTime
				local.FinishTime = t.FinishTime
			}
		}
	}
}

// DoWork is the manager's main loop: it alternates between sending work and syncing state.
func (m *Manager) DoWork() {
	for {
		log.Println("processing pending tasks")
		for m.Pending.Len() > 0 {
			m.SendWork()
		}

		log.Println("syncing task state from workers")
		m.UpdateTasks()

		log.Println("sleeping for 15s...")
		time.Sleep(15 * time.Second)
	}
}

// ListTasks returns all tasks known to the manager.
func (m *Manager) ListTasks() []*task.Task {
	tasks := make([]*task.Task, 0, len(m.TaskDb))
	for _, t := range m.TaskDb {
		tasks = append(tasks, t)
	}
	return tasks
}

// StopTask enqueues a stop event for a task and forwards it to the owning worker.
func (m *Manager) StopTask(taskID uuid.UUID) error {
	t, ok := m.TaskDb[taskID]
	if !ok {
		return fmt.Errorf("task [%s] not found", taskID)
	}

	w, ok := m.TaskWorkerMap[taskID]
	if !ok {
		return fmt.Errorf("no worker found for task [%s]", taskID)
	}

	url := fmt.Sprintf("http://%s/tasks/%s", w, taskID)
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("creating stop request for task [%s]: %w", taskID, err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("sending stop request for task [%s]: %w", taskID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected status [%d] when stopping task [%s]", resp.StatusCode, taskID)
	}

	t.State = task.Completed
	log.Printf("task [%s] stop requested on worker [%s]\n", taskID, w)
	return nil
}

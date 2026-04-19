package manager

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/radekska/cube/internal/task"
)

type api struct {
	Address string
	Port    int
	Manager *Manager
	Router  *chi.Mux
}

func NewAPI(address string, port int, m *Manager) *api {
	return &api{
		Address: address,
		Port:    port,
		Manager: m,
	}
}

func (a *api) Start() {
	a.initRouter()
	addr := fmt.Sprintf("%s:%d", a.Address, a.Port)
	log.Printf("manager API listening on %s\n", addr)
	http.ListenAndServe(addr, a.Router)
}

func (a *api) initRouter() {
	a.Router = chi.NewRouter()
	a.Router.Route("/tasks", func(r chi.Router) {
		r.Post("/", a.StartTaskHandler)
		r.Get("/", a.ListTasksHandler)
		r.Delete("/{taskID}", a.StopTaskHandler)
	})
}

// StartTaskHandler accepts a TaskEvent, enqueues it in the manager's pending queue.
func (a *api) StartTaskHandler(w http.ResponseWriter, r *http.Request) {
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()

	te := task.TaskEvent{}
	if err := d.Decode(&te); err != nil {
		log.Printf("decoding task event: %v\n", err)
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if te.ID == uuid.Nil {
		te.ID = uuid.New()
	}
	if te.Timestamp.IsZero() {
		te.Timestamp = time.Now().UTC()
	}
	if te.Task.ID == uuid.Nil {
		te.Task.ID = uuid.New()
	}

	a.Manager.AddTask(te)
	log.Printf("task event [%s] enqueued (task [%s])\n", te.ID, te.Task.ID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(te.Task); err != nil {
		log.Printf("encoding task response: %v\n", err)
	}
}

// ListTasksHandler returns all tasks known to the manager.
func (a *api) ListTasksHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(a.Manager.ListTasks()); err != nil {
		log.Printf("encoding tasks: %v\n", err)
	}
}

// StopTaskHandler forwards a stop request for a task to its owning worker.
func (a *api) StopTaskHandler(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "taskID")
	if taskID == "" {
		http.Error(w, "missing taskID", http.StatusBadRequest)
		return
	}

	tID, err := uuid.Parse(taskID)
	if err != nil {
		http.Error(w, "invalid taskID", http.StatusBadRequest)
		return
	}

	if err := a.Manager.StopTask(tID); err != nil {
		log.Printf("stopping task [%s]: %v\n", tID, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

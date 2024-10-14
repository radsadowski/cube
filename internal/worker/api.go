package worker

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/radekska/cube/internal/task"
)

type api struct {
	Address string
	Port    int
	Worker  *Worker
	Router  *chi.Mux
}

func NewAPI(address string, port int, worker *Worker) api {
	return api{
		Address: address,
		Port:    port,
		Worker:  worker,
	}
}

func (a *api) Start() {
	a.initRouter()
	http.ListenAndServe(fmt.Sprintf("%s:%d", a.Address, a.Port), a.Router)
}

func (a *api) initRouter() {
	a.Router = chi.NewRouter()
	a.Router.Route("/tasks", func(r chi.Router) {
		r.Post("/", a.StartTaskHandler)
		r.Get("/", a.ListTasks)
		r.Route("/{taskID}", func(r chi.Router) {
			r.Delete("/", a.StopTask)
		})
	})
}

func (a *api) StartTaskHandler(w http.ResponseWriter, r *http.Request) {
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()

	te := task.TaskEvent{}

	if err := d.Decode(&te); err != nil {
		log.Printf("decoding task: %v\n", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	a.Worker.AddTask(te.Task)
	log.Printf("task [%s] added\n", te.Task.ID)
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(te.Task); err != nil {
		log.Printf("encoding task: %v\n", err)
	}
}

func (a *api) ListTasks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(a.Worker.ListTasks()); err != nil {
		log.Printf("encoding tasks: %v\n", err)
	}
}

func (a *api) StopTask(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "taskID")
	if taskID == "" {
		log.Printf("missing taskID")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	tID, err := uuid.Parse(taskID)
	if err != nil {
		log.Printf("parsing taskID: %v\n", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	taskToStop, ok := a.Worker.Db[tID]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	taskCopy := *taskToStop
	taskCopy.State = task.Completed
	a.Worker.AddTask(taskCopy)
	w.WriteHeader(http.StatusNoContent)
}

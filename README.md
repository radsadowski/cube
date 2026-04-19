# cube

A toy container orchestration system written in Go — a minimal Kubernetes/Nomad-style manager-worker architecture where tasks are Docker containers.

## Architecture

```
          Manager (control plane)
         /    |    \
      Worker Worker Worker   ← each exposes a REST API
        |
     Docker Engine           ← tasks run as Docker containers
```

### Packages

| Package | Description |
|---|---|
| `internal/task` | Core domain model (`Task`, `TaskEvent`) and Docker execution engine (`Docker.Run`, `Docker.Stop`) |
| `internal/worker` | Execution node — runs containers, tracks state, collects host metrics, exposes REST API |
| `internal/manager` | Control plane — dispatches tasks to workers, reconciles state (WIP) |
| `internal/scheduler` | Pluggable interface for worker selection (filter → score → pick) |
| `internal/node` | Data model for a cluster node (CPU, memory, disk, role) |

## Task State Machine

```
Pending → Scheduled → Running → Completed
                    ↘          ↘
                      Failed     Failed
```

## Worker HTTP API

| Method | Path | Description |
|---|---|---|
| `POST` | `/tasks` | Submit a task (enqueues a `TaskEvent`) |
| `GET` | `/tasks` | List all tasks |
| `DELETE` | `/tasks/{taskID}` | Stop a task |
| `GET` | `/metrics` | Host stats (CPU, memory, disk) |

## Configuration

The worker reads its bind address from environment variables:

```
CUBE_HOST=0.0.0.0
CUBE_PORT=5555
```

## Running

```bash
go run main.go
```

## Tech Stack

- **Go 1.22**
- **Docker SDK** (`docker/docker`) — container lifecycle management
- **chi** — HTTP router
- **gopsutil** — host metrics collection
- **golang-collections/queue** — FIFO task queue
- **google/uuid** — task/event identifiers

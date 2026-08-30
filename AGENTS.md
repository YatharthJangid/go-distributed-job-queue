# AGENTS.md — Developer & AI Agent Guide for GoRes

> Practical guide for AI agents and engineers working on the **GoRes** distributed job queue repository.

---

## 1. Project Overview & Mission

**GoRes** is a Redis-backed distributed background job queue written in Go. It targets high throughput, reliable queue handling, pooled job objects, and multi-node coordination.

### Core Metrics & Capabilities
* **Performance**: See the repository benchmarks for current throughput and latency results.
* **Job pooling**: `sync.Pool` recycles `Job` structs on the normal processing path.
* **Redis queue handling**: `BRPOPLPUSH` moves jobs into a processing list before execution; stranded entries remain inspectable after a worker crash.
* **Idempotency**: Lua enqueue deduplication and Redis execution-status keys coordinate duplicate submissions and completed deliveries.
* **Kubernetes**: Included manifests support local kind-based, multi-replica verification.

---

## 2. Repository Structure & Layout

```
gores/
├── main.go                       # CLI entry point (produce / consume / bench modes)
├── config.json                   # Local/default Redis connection configuration
├── Dockerfile                    # Multi-stage container build (Go 1.24 -> Alpine)
├── go.mod / go.sum               # Module definition: myproject/gores
├── AGENTS.md                     # Comprehensive agent & engineer operational guide
├── README.md                     # High-level documentation & architecture overview
├── DEMO.md                       # Interactive 7-part demonstration guide
├── DEPLOYMENT.md                 # 5 deployment architectures (Cloud, K8s, VPS)
│
├── k8s/                          # Kubernetes Manifests & Multi-Node Cluster Configuration
│   ├── kind-config.yaml          # Multi-node kind configuration (1 control-plane, 2 workers)
│   ├── configmap.yaml            # ConfigMap providing config.json with container hostnames
│   ├── redis.yaml                # Redis Deployment with health probes & ClusterIP Service
│   ├── consumer-deployment.yaml  # Scalable Consumer Deployment with graceful shutdown
│   ├── producer-job.yaml         # Batch Producer Job (300 jobs with idempotency)
│   ├── producer-duplicate-job.yaml # Idempotency verification Job
│   └── producer-1000.yaml        # 1,000-job workload testing scaled consumer pods
│
├── pkg/gores/                    # Core GoRes Engine Package (package gores)
│   ├── constants.go              # Key prefixes, queue suffixes, stats, and TTL defaults
│   ├── config.go                 # Config struct + JSON loader + REDIS_URL override
│   ├── job.go                    # Job struct, sync.Pool allocator, MessagePack codec
│   ├── gores.go                  # Gores client, connection pool, Lua enqueue, Info & status
│   ├── worker.go                 # Worker pool, BRPOPLPUSH loop, retry, DLQ, execution status
│   ├── live_bench.go             # Live 10,000-job Redis throughput benchmark runner
│   ├── config_test.go            # Config loading and environment override unit tests
│   ├── gores_test.go             # Enqueue, batch enqueue, and Info statistics unit tests
│   ├── job_test.go               # Job pooling, validation, and serialization unit tests
│   ├── worker_test.go            # Worker processing, retry backoff, and DLQ tests
│   ├── idempotency_test.go       # Enqueue deduplication and worker execution idempotency tests
│   └── benchmark_test.go         # Microbenchmarks (JobPool, MessagePack, Enqueue, Workers)
│
    └── graphify-out/                 # Generated Graphify output; normally do not edit
```

---

## 3. Core Architecture & Key Design Decisions

```
┌─────────────────────────────────────────────────────────────┐
│                     Producer (main.go)                      │
│   Enqueue() / EnqueueBatch()                                │
│   → MessagePack serialization (msgpack/v5)                  │
│   → Lua enqueue script; optional SET NX EX deduplication    │
└──────────────────────────────┬──────────────────────────────┘
                               │ LPUSH gores:<queue>:pending
                               ▼
┌─────────────────────────────────────────────────────────────┐
│                    Redis (Central Broker)                   │
│   gores:<queue>:pending      → Pending Job List             │
│   gores:<queue>:processing   → In-Flight Processing List    │
│   gores:<queue>_deadletter   → Dead-Letter Queue (DLQ)      │
│   gores:idempotency:<key>    → Enqueue deduplication key    │
│   gores:exec:<key>           → Execution state (processing/completed)
│   gores:stat:*               → Stat counters (enqueued, processed, duplicates)
└──────────────────────────────┬──────────────────────────────┘
                               │ BRPOPLPUSH (atomic, zero loss)
                               ▼
┌─────────────────────────────────────────────────────────────┐
│               Worker Pool (pkg/gores/worker.go)             │
│   N Goroutines (each pinned via runtime.LockOSThread)       │
│   → Check gores:exec:<key> execution status                 │
│   → Dispatch to registered task function                    │
│   → On success: LREM :processing, INCR stat:processed       │
│   → On failure: Exponential retry backoff (2^r seconds)     │
│   → On max retries: LPUSH gores:<queue>_deadletter          │
└─────────────────────────────────────────────────────────────┘
```

### Deep Dive into Core Components

#### 1. Job Serialization & Memory Management ([`pkg/gores/job.go`](file:///home/jangi/projects/gores/pkg/gores/job.go))
* **MessagePack (`msgpack/v5`)**: Chosen over JSON for binary compactness (~3× smaller) and faster serialization speed (~578 ns/op).
* **`sync.Pool` Zero GC Allocation**: `GetJob()` acquires a pre-allocated `*Job` from `jobPool`, while `PutJob(j)` clears fields, maps, and resets the struct before returning it to the pool (~11 ns/op).

#### 2. Redis Client & Atomicity ([`pkg/gores/gores.go`](file:///home/jangi/projects/gores/pkg/gores/gores.go))
* **Connection Pool**: Powered by Redigo with customizable `MaxIdle`, `MaxActive`, and `IdleTimeout`.
* **Atomic Lua Scripts**:
  * `luaEnqueue`: Enqueues a single job and increments `gores:stat:enqueued` in 1 network roundtrip.
  * `luaEnqueueIdempotent`: Atomically checks/sets `gores:idempotency:<key>` via `SET ... NX EX <ttl>`; enqueues only if novel, otherwise increments `gores:stat:duplicates`.
  * `luaEnqueueBatch`: Pipelined Lua script executing N batch enqueues with per-job queue targeting and atomic deduplication.

#### 3. Worker Pool & Fault Tolerance ([`pkg/gores/worker.go`](file:///home/jangi/projects/gores/pkg/gores/worker.go))
* **Thread-Pinning (`runtime.LockOSThread()`)**: Each worker goroutine is locked to an OS thread to prevent OS thread context switches on hot processing loops.
* **Dequeue (`BRPOPLPUSH`)**: Shifts job bytes from `:pending` to `:processing` before execution. If a worker crashes mid-task, the entry remains in `:processing` for inspection or recovery.
* **Exponential Backoff Retry**: Transient task failures retry up to 3 times with $2^r$ seconds backoff (1s, 2s, 4s).
* **Dead-Letter Queue (DLQ)**: Jobs failing all retries are pushed to `gores:<queue>_deadletter` for manual inspection.
* **Graceful Termination**: Listens for `SIGINT`/`SIGTERM` via `os/signal` and cancels context, allowing in-flight jobs to complete cleanly before shutdown.

---

## 4. Idempotency & Deduplication Engine

GoRes coordinates duplicate handling across two distinct layers:

### Layer 1: Enqueue Deduplication
* **Purpose**: Prevents duplicate job enqueues when producers retry network requests or multiple producers submit identical tasks.
* **Mechanism**: When a job includes `IdempotencyKey` (and optional `IdempotencyTTL`), Redis executes:
  ```lua
  local setRes = redis.call('SET', idempKey, 'enqueued', 'EX', ttl, 'NX')
  if not setRes then
      redis.call('INCR', dupStatKey)
      return 0
  end
  redis.call('LPUSH', queue, data)
  redis.call('INCR', statKey)
  ```
* **Behavior**: Subsequent submissions with the same key during the TTL window return immediately without inserting duplicate items into the pending queue.

### Layer 2: Execution Status
* **Purpose**: Skips completed duplicate deliveries when messages are re-delivered or distributed across multiple worker nodes.
* **Mechanism**:
  1. Worker checks `gores:exec:<key>` in Redis.
  2. If status is `"completed"`, the worker immediately acknowledges and skips execution.
  3. Otherwise, worker marks status as `"processing"` with TTL.
  4. Upon successful task completion, status is updated to `"completed"` and `gores:stat:processed` is incremented.

---

## 5. CLI Reference & Execution Modes

The CLI binary ([`main.go`](file:///home/jangi/projects/gores/main.go)) supports the following flags:

| Flag | Default | Description |
|---|---|---|
| `-o` | `produce` | Operation mode: `produce` or `consume` |
| `-w` | `3` | Number of concurrent worker goroutines (in consume mode) |
| `-n` | `100` | Number of jobs to generate (in produce mode) |
| `-idemp` | `false` | Enable idempotency keys on generated batch jobs |
| `-c` | `config.json`| Path to Redis configuration file |
| `-bench` | `false` | Run live end-to-end Redis throughput benchmark (10,000 jobs) |

### Common CLI Invocations
```bash
# Build binary
go build -o gores .

# Producer: Enqueue 100 regular jobs
./gores -o produce -n 100

# Producer: Enqueue 500 jobs with idempotency keys
./gores -o produce -n 500 -idemp

# Consumer: Start 5 workers
./gores -o consume -w 5

# Live Benchmark: Run 10,000 jobs through local Redis
./gores -bench
```

---

## 6. Kubernetes Multi-Node & Cloud Deployment

GoRes includes Kubernetes manifests in [`k8s/`](file:///home/jangi/projects/gores/k8s) for local kind-based verification and scalable consumers.

### Multi-Node Local Verification with `kind`
```bash
# 1. Create a 3-node cluster (1 control plane, 2 worker nodes)
kind create cluster --config k8s/kind-config.yaml --name gores-cluster

# 2. Build and load container image
docker build -t gores:latest .
kind load docker-image gores:latest --name gores-cluster

# 3. Deploy ConfigMap and Redis
kubectl apply -f k8s/configmap.yaml
kubectl apply -f k8s/redis.yaml
kubectl rollout status deployment/redis

# 4. Deploy 4 consumer replicas (distributed across worker nodes)
kubectl apply -f k8s/consumer-deployment.yaml
kubectl rollout status deployment/gores-consumer

# 5. Trigger batch producer Job
kubectl apply -f k8s/producer-job.yaml
kubectl wait --for=condition=complete job/gores-producer
kubectl logs job/gores-producer

# 6. Verify idempotency with duplicate producer job
kubectl apply -f k8s/producer-duplicate-job.yaml
kubectl wait --for=condition=complete job/gores-producer-duplicate
kubectl logs job/gores-producer-duplicate

# 7. Scale consumer deployment dynamically
kubectl scale deployment gores-consumer --replicas=8
```

---

## 7. Testing, Verification & Benchmarking

### Running the Test Suite
```bash
# 1. Run all unit and idempotency tests
go test -v ./pkg/gores/...

# 2. Run with Race Detector
go test -v -race ./pkg/gores/...

# 3. Generate HTML Coverage Report
go test -coverprofile=coverage.out ./pkg/gores/...
go tool cover -html=coverage.out

# 4. Run Microbenchmarks
go test -bench=. -benchmem ./pkg/gores/...

# 5. Run Static Analysis (Vet)
go vet ./...
```

---

## 8. Optional Knowledge Graph Maintenance (`/graphify`)

When Graphify is part of the workflow, it can maintain an up-to-date,
queryable knowledge graph of the codebase. It is not required for routine edits.

```bash
# Update code graph after making changes (deterministic AST extraction)
graphify update .
# or
graphify . --code-only

# Export browser visualization
graphify export html

# Query the codebase knowledge graph
graphify query "How does job idempotency work?"
```

---

## 9. Rules and Guidelines for Future AI Agents

When editing or extending this codebase, use the following principles:

1. **Preserve Package Structure**: Keep core queue logic in `pkg/gores` under `package gores`; avoid generic helper packages unless they earn their place.
2. **Use the Job Pool**: When modifying `Job` creation or parsing, acquire via `GetJob()` and release via `PutJob(j)`.
3. **Preserve Atomicity**: Keep queue operations that require atomicity inside Redis Lua scripts.
4. **Maintain Redis Compatibility**: Support explicit host/port config (`config.json`) and URL-based cloud config (`REDIS_URL`).
5. **Update Relevant Checks**: Add or update tests when behavior changes. Refresh Graphify output only when the repository workflow requires it; documentation-only edits do not need graph regeneration.

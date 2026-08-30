# GoRes — High-Performance Distributed Job Queue

> A production-grade, Redis-backed background job queue written in Go. Engineered for throughput, reliability, and zero-allocation hot paths.

---

## Performance Highlights

| Metric | Value |
|---|---|
| Peak throughput | **35,388 jobs/sec** |
| Avg. dispatch latency | **845 ns** |
| Test coverage | **77.3%** (core library) |

---

## Architecture

```
┌──────────────────────────────────────────────────────┐
│                    Producer (main.go)                │
│  EnqueueBatch() → Redis MULTI/EXEC pipeline          │
│  Lua script: atomic LPUSH + INCR stat counter        │
└──────────────────────┬───────────────────────────────┘
                       │ gores:<queue>:pending  (Redis List)
                       ▼
┌──────────────────────────────────────────────────────┐
│              Redis  (central broker)                 │
│  pending list → BRPOPLPUSH → processing list         │
│  Dead-Letter Queue on exhausted retries              │
└──────────────────────┬───────────────────────────────┘
                       │
           ┌───────────┴──────────┐
           ▼                      ▼
   ┌──────────────┐      ┌──────────────┐
   │  Worker 0    │ ...  │  Worker N    │   (goroutines, OS-thread-locked)
   │ processJob() │      │ processJob() │
   │ retry x3     │      │ retry x3     │
   └──────────────┘      └──────────────┘
```

### Key Design Decisions

| Component | Implementation | Why |
|---|---|---|
| **Job serialization** | MessagePack (`msgpack/v5`) | ~3× smaller than JSON, faster encode/decode |
| **Job pooling** | `sync.Pool` | Zero GC pressure on hot path |
| **Enqueue atomicity** | Lua script (`EVAL`) | Stat counter + LPUSH in one round-trip |
| **Batch enqueue** | Redis `MULTI/EXEC` pipeline | Single TCP round-trip for N jobs |
| **Reliable dequeue** | `BRPOPLPUSH` | Jobs survive worker crashes; never lost |
| **Dead-Letter Queue** | `gores:<queue>_deadletter` | Failed jobs after 3 retries are preserved |
| **Worker isolation** | `runtime.LockOSThread()` | Per-worker dedicated OS thread + Redis connection |
| **Graceful shutdown** | `SIGINT`/`SIGTERM` via `context.WithCancel` | In-flight jobs complete before exit |
| **Retry backoff** | Exponential (`2^r` seconds) | Avoids thundering-herd on transient failures |

---

## Project Structure

```
gores/
├── main.go                  # CLI entry-point (produce / consume / bench modes)
├── config.json              # Redis connection config
├── go.mod / go.sum
└── pkg/gores/
    ├── constants.go         # Redis key prefixes & suffixes
    ├── config.go            # Config struct + JSON loader
    ├── job.go               # Job struct, sync.Pool, msgpack encode/decode
    ├── gores.go             # Core: Enqueue, EnqueueBatch, Info (Lua + MULTI/EXEC)
    ├── worker.go            # Worker pool: BRPOPLPUSH loop, retry, DLQ
    ├── live_bench.go        # End-to-end throughput benchmark (--bench flag)
    └── *_test.go            # Unit + benchmark tests (77.3% coverage)
```

---

## Quick Start

### Prerequisites

- Go 1.21+
- Redis 6+ running locally (or see [Cloud Deployment](#cloud-deployment))

```bash
# 1. Clone & fetch dependencies
git clone https://github.com/YatharthJangid/go-distributed-job-queue
cd gores
go mod download

# 2. Start Redis locally (Docker is the easiest way)
docker run -d -p 6379:6379 redis:7-alpine

# 3. Build
go build -o gores .
```

### Running

```bash
# Enqueue 100 jobs (producer mode)
./gores -o produce

# Start 5 worker goroutines (consumer mode)
./gores -o consume -w 5

# Run end-to-end live benchmark (10,000 jobs through Redis)
./gores -bench
```

### CLI Flags

| Flag | Default | Description |
|---|---|---|
| `-o` | `produce` | Mode: `produce` or `consume` |
| `-w` | `3` | Number of worker goroutines |
| `-c` | `config.json` | Path to Redis config file |
| `--bench` | `false` | Run live throughput benchmark and exit |

### Config (`config.json`)

```json
{
  "redis": {
    "host": "127.0.0.1",
    "port": 6379,
    "db": 0,
    "pool_size": 10,
    "max_idle": 50,
    "max_active": 200,
    "idle_timeout": 300
  }
}
```

---

## Job Lifecycle

```
Enqueue (producer)
  → msgpack serialize
  → Lua: LPUSH gores:<queue>:pending + INCR gores:stat:enqueued

Dequeue (worker)
  → BRPOPLPUSH :pending → :processing   (atomic, no job loss)
  → processJob(): dispatch to task fn
  → on success: LREM :processing
  → on failure: retry up to 3× with exponential backoff
  → max retries exceeded: LPUSH gores:<queue>_deadletter
```

---

## Running Tests

```bash
# Unit tests
go test -v ./pkg/gores/...

# With coverage
go test -coverprofile=coverage.out ./pkg/gores/...
go tool cover -html=coverage.out

# Benchmark tests (Go microbenchmarks — no Redis needed)
go test -bench=. -benchmem ./pkg/gores/...
```

---

## Cloud Deployment

See **[DEPLOYMENT.md](DEPLOYMENT.md)** for a full guide covering:
- Docker container setup
- Railway / Render (free-tier, one-click deploy)
- AWS EC2 + ElastiCache (production)
- Environment-variable-based config for cloud Redis

---

## Demonstrating the System

See **[DEMO.md](DEMO.md)** for:
- Running producer + consumer in split terminals with real-time output
- Using `redis-cli MONITOR` to watch jobs flow live
- The `--bench` flag for measurable throughput numbers
- A `docker-compose` setup that spins up Redis + producer + consumer automatically

---

## Dependencies

| Package | Purpose |
|---|---|
| `github.com/garyburd/redigo` | Redis client with connection pooling |
| `github.com/vmihailenco/msgpack/v5` | Binary serialization for jobs |

---

## License

MIT
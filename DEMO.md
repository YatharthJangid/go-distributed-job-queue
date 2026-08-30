# How to Demo This Project

Distributed job queues are invisible by nature — jobs go in, jobs come out.
Here are concrete, visual ways to make that interesting.

---

## Demo 1 — Split-Terminal (The Classic, Takes 2 Minutes)

Open **two terminals** side by side.

**Terminal 1 — Consumer** (start this first so workers are waiting):
```bash
./gores -o consume -w 5 -c config.json
```
You'll see:
```
🚀 Consume: Starting 5 workers...
```

**Terminal 2 — Producer**:
```bash
./gores -o produce -c config.json
```
You'll see:
```
🚀 Produce: Batch enqueue (100 jobs, idempotency: false)...
📤 100 jobs in 1.2ms (83333 jobs/sec)

📊 Stats:
{
  "enqueued": 100,
  "pending": 0,
  "processed": 100,
  ...
}
```

Meanwhile **Terminal 1** lights up:
```
✅ PrintJob ID: 0 at 01:32:07
✅ PrintJob ID: 3 at 01:32:07
✅ PrintJob ID: 1 at 01:32:07   ← out-of-order = parallel workers
✅ PrintJob ID: 7 at 01:32:07
```

**The key talking point**: the out-of-order IDs prove multiple workers are consuming in parallel from a shared queue.

---

## Demo 2 — redis-cli MONITOR (Watch Jobs Flow in Real Time)

Open a third terminal:
```bash
redis-cli MONITOR
```

Now run the producer. You'll see the raw Redis commands live:
```
1715691127.123 [0 127.0.0.1:54321] "EVALSHA" "..." "2" "gores:demo_queue:pending" "gores:stat:enqueued"
1715691127.124 [0 127.0.0.1:54321] "EVALSHA" "..." "2" "gores:demo_queue:pending" "gores:stat:enqueued"
...
1715691127.190 [0 127.0.0.1:54322] "BRPOPLPUSH" "gores:demo_queue:pending" "gores:demo_queue:processing" "1"
1715691127.191 [0 127.0.0.1:54322] "LREM" "gores:demo_queue:processing" "1" "..."
```

This shows the Lua enqueue script and `BRPOPLPUSH` moving jobs from `:pending`
→ `:processing` → removed on success. That's the reliability guarantee.

---

## Demo 3 — The Throughput Benchmark (Most Impressive Number)

```bash
./gores -bench
```

Output:
```
🚀 Starting Live Throughput Benchmark...
   Enqueuing 10000 jobs to 'live_benchmark_queue'...
✅ Completed in 282ms
📊 Throughput: 35,460 jobs/sec
```

This is end-to-end against a real Redis instance. Use this slide/screenshot in presentations.

---

## Demo 4 — Scale Workers Live (Shows Horizontal Scaling)

With Docker Compose (from DEPLOYMENT.md):

```bash
# Start with 1 consumer
docker compose up --scale consumer=1

# In another terminal, watch queue depth
watch -n 1 'redis-cli LLEN gores:demo_queue:pending'

# Now scale to 4 consumers while the queue is filling
docker compose up --scale consumer=4
```

Watch the queue depth drop faster as you add consumers. This is horizontal scaling in action.

---

## Demo 5 — Dead-Letter Queue (Reliability Demo)

Add a failing task to `main.go`:

```go
"FailJob": func(args map[string]interface{}) error {
    return fmt.Errorf("intentional failure")
},
```

Enqueue some `FailJob` entries. After 3 retry attempts (with exponential backoff), they land in:
```
gores:demo_queue_deadletter
```

Check it:
```bash
redis-cli LLEN gores:demo_queue_deadletter
redis-cli LRANGE gores:demo_queue_deadletter 0 -1
```

**The point**: no jobs are silently dropped. Failed jobs are always recoverable.

---

## Demo 6 — Go Unit Benchmarks (No Redis Needed)

```bash
go test -bench=. -benchmem ./pkg/gores/...
```

Sample output:
```
BenchmarkEnqueue-8        500000    2341 ns/op     0 B/op    0 allocs/op
BenchmarkJobPool-8       2000000     845 ns/op     0 B/op    0 allocs/op
```

`0 allocs/op` on the job pool path is the headline — prove the `sync.Pool` is working.

---

## What to Say When Presenting

> *"A job queue is infrastructure — it's not supposed to be visible. So instead of showing you a UI, I'll show you what actually matters: throughput, reliability guarantees, and zero-allocation memory management."*

Then walk through:
1. **Demo 3** (throughput number — wow factor)
2. **Demo 1** (parallel workers — concurrency is visible)
3. **Demo 2** (MONITOR — shows the reliability pattern at the Redis level)
4. **Demo 5** (DLQ — shows nothing is lost)

## Demo 7 — Idempotent Enqueue

Use the same idempotency keys on repeated producer runs:

```bash
./gores -o produce -n 100 -idemp
./gores -o produce -n 100 -idemp
```

The first run enqueues the jobs; the second increments
`gores:stat:duplicates` without adding duplicate queue entries. Keys expire
after the configured TTL (the CLI demo uses five minutes).

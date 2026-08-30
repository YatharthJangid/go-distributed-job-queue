# Graph Report - gores  (2026-08-30)

## Corpus Check
- 19 files · ~10,645 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 131 nodes · 213 edges · 13 communities (12 shown, 1 thin omitted)
- Extraction: 75% EXTRACTED · 25% INFERRED · 0% AMBIGUOUS · INFERRED: 54 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `07682149`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- PutJob
- InitConfig
- NewGores
- newTestConfig
- Gores
- main
- GoRes — High-Performance Distributed Job Queue
- 3. Kubernetes Multi-Node & Multi-System Testing
- myproject/gores
- How to Demo This Project
- Cloud Deployment Guide

## God Nodes (most connected - your core abstractions)
1. `NewGores()` - 21 edges
2. `newTestConfig()` - 13 edges
3. `PutJob()` - 11 edges
4. `GoRes — High-Performance Distributed Job Queue` - 11 edges
5. `InitConfig()` - 9 edges
6. `GetJob()` - 9 edges
7. `FromBytes()` - 9 edges
8. `How to Demo This Project` - 9 edges
9. `Gores` - 8 edges
10. `Job` - 7 edges

## Surprising Connections (you probably didn't know these)
- `main()` --calls--> `InitConfig()`  [INFERRED]
  main.go → pkg/gores/config.go
- `main()` --calls--> `NewGores()`  [INFERRED]
  main.go → pkg/gores/gores.go
- `main()` --calls--> `RunLiveBenchmark()`  [INFERRED]
  main.go → pkg/gores/live_bench.go
- `BenchmarkJobFromBytes()` --calls--> `FromBytes()`  [INFERRED]
  pkg/gores/benchmark_test.go → pkg/gores/job.go
- `RunLiveBenchmark()` --calls--> `InitConfig()`  [INFERRED]
  pkg/gores/live_bench.go → pkg/gores/config.go

## Import Cycles
- None detected.

## Communities (13 total, 1 thin omitted)

### Community 0 - "PutJob"
Cohesion: 0.20
Nodes (13): Job, BenchmarkJobPool(), BenchmarkJobPoolNoWrite(), RunBenchmarks(), FromBytes(), GetJob(), PutJob(), T (+5 more)

### Community 1 - "InitConfig"
Cohesion: 0.40
Nodes (8): Config, InitConfig(), T, TestInitConfig_InvalidJSON(), TestInitConfig_MissingFile(), TestInitConfig_MissingRedisSection(), TestInitConfig_PartialDefaults(), TestInitConfig_Valid()

### Community 2 - "NewGores"
Cohesion: 0.27
Nodes (13): B, BenchmarkEnqueue(), BenchmarkEnqueueBatch10(), BenchmarkEnqueueBatch100(), BenchmarkInfo(), BenchmarkJobFromBytes(), BenchmarkJobToBytes(), BenchmarkProcessJob() (+5 more)

### Community 3 - "newTestConfig"
Cohesion: 0.32
Nodes (11): T, newTestConfig(), TestEnqueueAndInfoCounts(), TestEnqueueBatch100(), T, TestProcessJob_EmptyPayload(), TestProcessJob_NilArgs(), TestProcessJob_TaskNotFound() (+3 more)

### Community 4 - "Gores"
Cohesion: 0.39
Nodes (3): Gores, jobFromMap(), Pool

### Community 5 - "main"
Cohesion: 0.43
Nodes (5): Gores, main(), runConsumer(), runProducer(), RunLiveBenchmark()

### Community 6 - "GoRes — High-Performance Distributed Job Queue"
Cohesion: 0.12
Nodes (16): Architecture, CLI Flags, Cloud Deployment, Config (`config.json`), Demonstrating the System, Dependencies, GoRes — High-Performance Distributed Job Queue, Job Lifecycle (+8 more)

### Community 7 - "3. Kubernetes Multi-Node & Multi-System Testing"
Cohesion: 0.13
Nodes (14): 1. Graphify Knowledge Graph Generation, 2. Idempotency Implementation & Verification, 3. Kubernetes Multi-Node & Multi-System Testing, Architecture, Automated Idempotency Tests, Complete Goal Walkthrough: Refactoring, Idempotency & Kubernetes Multi-Node Verification, Distributed Pod Scheduling, Kubernetes Infrastructure (+6 more)

### Community 11 - "How to Demo This Project"
Cohesion: 0.17
Nodes (9): Demo 1 — Split-Terminal (The Classic, Takes 2 Minutes), Demo 2 — redis-cli MONITOR (Watch Jobs Flow in Real Time), Demo 3 — The Throughput Benchmark (Most Impressive Number), Demo 4 — Scale Workers Live (Shows Horizontal Scaling), Demo 5 — Dead-Letter Queue (Reliability Demo), Demo 6 — Go Unit Benchmarks (No Redis Needed), Demo 7 — Idempotent Enqueue, How to Demo This Project (+1 more)

### Community 12 - "Cloud Deployment Guide"
Cohesion: 0.17
Nodes (12): Cloud Deployment Guide, Deploying to Kubernetes, `docker-compose.yml`, `Dockerfile`, Manifests Overview, Option 1 — Railway (Easiest, Free Tier), Option 2 — Render (Free Tier, Similar to Railway), Option 3 — Docker Compose (Local or Any VPS) (+4 more)

## Knowledge Gaps
- **41 isolated node(s):** `myproject/gores`, `Demo 1 — Split-Terminal (The Classic, Takes 2 Minutes)`, `Demo 2 — redis-cli MONITOR (Watch Jobs Flow in Real Time)`, `Demo 3 — The Throughput Benchmark (Most Impressive Number)`, `Demo 4 — Scale Workers Live (Shows Horizontal Scaling)` (+36 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **1 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `NewGores()` connect `NewGores` to `InitConfig`, `newTestConfig`, `Gores`, `main`?**
  _High betweenness centrality (0.164) - this node is a cross-community bridge._
- **Why does `TestWorker_ExecutionIdempotency()` connect `NewGores` to `PutJob`, `newTestConfig`?**
  _High betweenness centrality (0.076) - this node is a cross-community bridge._
- **Why does `PutJob()` connect `PutJob` to `NewGores`, `Gores`?**
  _High betweenness centrality (0.064) - this node is a cross-community bridge._
- **Are the 18 inferred relationships involving `NewGores()` (e.g. with `main()` and `BenchmarkEnqueue()`) actually correct?**
  _`NewGores()` has 18 INFERRED edges - model-reasoned connections that need verification._
- **Are the 9 inferred relationships involving `newTestConfig()` (e.g. with `TestEnqueue_Idempotency()` and `TestEnqueueBatch_Idempotency()`) actually correct?**
  _`newTestConfig()` has 9 INFERRED edges - model-reasoned connections that need verification._
- **Are the 8 inferred relationships involving `PutJob()` (e.g. with `.Enqueue()` and `.EnqueueBatch()`) actually correct?**
  _`PutJob()` has 8 INFERRED edges - model-reasoned connections that need verification._
- **What connects `myproject/gores`, `Demo 1 — Split-Terminal (The Classic, Takes 2 Minutes)`, `Demo 2 — redis-cli MONITOR (Watch Jobs Flow in Real Time)` to the rest of the system?**
  _41 weakly-connected nodes found - possible documentation gaps or missing edges._
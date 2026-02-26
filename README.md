# High-Performance Distributed Job Queue (Go)

## Overview
A high-throughput, distributed job queue system written in Go, designed for low-latency task processing and efficient resource management. This project emphasizes system reliability and performance, featuring a custom worker-pool architecture and optimized memory allocation.

## Performance Metrics
- **Throughput:** Achieved peak processing of **35,388 jobs/second**.
- **Latency:** Average task dispatch latency of **845 ns**.
- **Test Coverage:** **77.3%** statement coverage across core library components.

## Key Architectural Features
- **Worker-Pool Pattern:** Efficiently manages goroutine lifecycles to prevent resource exhaustion.
- **Object Pooling:** Utilizes `sync.Pool` for zero-allocation task recycling, significantly reducing GC overhead.
- **CI/CD Integrated:** Automated testing pipeline via GitHub Actions ensuring code quality on every push.
- **Observability:** Built-in benchmarking and coverage reporting for performance bottlenecks.

## System Architecture
The following diagram illustrates the job lifecycle from Dispatcher to Worker execution:



## Quick Start
```bash
go mod download
cd lib_optimized
go test -v -cover
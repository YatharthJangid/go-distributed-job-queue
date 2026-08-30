# Cloud Deployment Guide

This guide covers five deployment options from simplest to most production-ready.

---

## Option 1 — Railway (Easiest, Free Tier)

Railway can run a Go binary + a managed Redis instance with almost no config.

### Steps

1. **Add a `Dockerfile`** to the project root (see below).
2. Push to GitHub.
3. Go to [railway.app](https://railway.app) → **New Project → Deploy from GitHub repo**.
4. Add a **Redis** plugin: Railway menu → **+ New → Database → Redis**.
5. Railway injects `REDIS_URL` automatically. GoRes reads it directly, so no
   code changes are required.

6. Set the **start command** in Railway settings:
   - Producer service: `./gores -o produce`
   - Consumer service: `./gores -o consume -w 5`

   You can deploy **two services from the same repo** (one per mode) by setting different start commands.

---

## Option 2 — Render (Free Tier, Similar to Railway)

1. Push to GitHub.
2. [render.com](https://render.com) → **New → Web Service** → connect repo.
3. Add a **Redis** instance: Render dashboard → **New → Redis**.
4. Set env var `REDIS_URL` on each service.
5. Create **two Web Services** from the same repo with start commands:
   - `go run main.go -o produce`
   - `go run main.go -o consume -w 5`

> Both Railway and Render have free tiers that sleep after inactivity — fine for demos, not for production.

---

## Option 3 — Docker Compose (Local or Any VPS)

This is the **recommended approach for a demo or a cheap VPS** (DigitalOcean, Hetzner, etc).

### `Dockerfile`

```dockerfile
# Build stage
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o gores .

# Run stage
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/gores .
COPY config.json .
ENTRYPOINT ["./gores"]
```

### `docker-compose.yml`

```yaml
version: "3.9"

services:
  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"

  producer:
    build: .
    command: ["-o", "produce", "-c", "/app/config.json"]
    depends_on:
      - redis
    environment:
      - REDIS_URL=redis://redis:6379
    restart: on-failure

  consumer:
    build: .
    command: ["-o", "consume", "-w", "8", "-c", "/app/config.json"]
    depends_on:
      - redis
    environment:
      - REDIS_URL=redis://redis:6379
    restart: unless-stopped
    deploy:
      replicas: 2   # run 2 consumer instances = horizontal scaling
```

### Running

```bash
docker compose up --build
# Scale consumers on the fly:
docker compose up --scale consumer=4
```

---

## Option 4 — AWS EC2 + ElastiCache (Production)

For real production traffic:

| Component | AWS Service |
|---|---|
| Redis broker | **ElastiCache for Redis** (managed, replicated) |
| Producer | EC2 / ECS task / Lambda |
| Consumer | **ECS Fargate** with auto-scaling on queue depth |
| Config | **AWS Secrets Manager** or Parameter Store |
| Monitoring | CloudWatch + custom metrics via `gores:stat:*` keys |

Key points:
- ElastiCache endpoint replaces `127.0.0.1:6379` in config.
- Use IAM roles instead of hardcoded credentials.
- Scale ECS consumer tasks based on the Redis list length (`LLEN gores:demo_queue:pending`).

---

## Option 5 — Kubernetes (Multi-Node / kind Verification)

The repository provides Kubernetes manifests in `k8s/` for running Redis and
scalable GoRes consumers across multiple nodes. The included kind config is for
local verification; production Redis should use a managed or persistent setup.

### Manifests Overview

| Manifest | Purpose |
|---|---|
| [`k8s/redis.yaml`](k8s/redis.yaml) | Redis Deployment with health probes & ClusterIP Service |
| [`k8s/configmap.yaml`](k8s/configmap.yaml) | ConfigMap providing `config.json` |
| [`k8s/consumer-deployment.yaml`](k8s/consumer-deployment.yaml) | Multi-replica Consumer Deployment with graceful termination |
| [`k8s/producer-job.yaml`](k8s/producer-job.yaml) | 300-job idempotency smoke test |
| [`k8s/producer-duplicate-job.yaml`](k8s/producer-duplicate-job.yaml) | Duplicate enqueue verification |
| [`k8s/producer-1000.yaml`](k8s/producer-1000.yaml) | 1,000-job scaling verification |
| [`k8s/kind-config.yaml`](k8s/kind-config.yaml) | Multi-node cluster config (1 control-plane, 2 worker nodes) |

### Deploying to Kubernetes

```bash
# 1. Build and load Docker image
docker build -t gores:latest .
# If using kind:
kind load docker-image gores:latest --name gores-cluster

# 2. Deploy ConfigMap & Redis
kubectl apply -f k8s/configmap.yaml
kubectl apply -f k8s/redis.yaml
kubectl rollout status deployment/redis

# 3. Deploy Consumer Pods (runs 4 replicas across nodes)
kubectl apply -f k8s/consumer-deployment.yaml
kubectl rollout status deployment/gores-consumer

# 4. Trigger Producer Job
kubectl apply -f k8s/producer-job.yaml
kubectl wait --for=condition=complete job/gores-producer
kubectl logs job/gores-producer

# Optional duplicate and scale checks
kubectl apply -f k8s/producer-duplicate-job.yaml
kubectl apply -f k8s/producer-1000.yaml

# 5. Scale consumers dynamically
kubectl scale deployment gores-consumer --replicas=8
```

The producer supports `-n` for batch size and `-idemp` for idempotency keys.
Duplicate enqueue attempts increment `gores:stat:duplicates`; successful work
increments `gores:stat:processed`.

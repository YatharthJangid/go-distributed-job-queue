# Cloud Deployment Guide

This guide covers three deployment options from simplest to most production-ready.

---

## Option 1 — Railway (Easiest, Free Tier)

Railway can run a Go binary + a managed Redis instance with almost no config.

### Steps

1. **Add a `Dockerfile`** to the project root (see below).
2. Push to GitHub.
3. Go to [railway.app](https://railway.app) → **New Project → Deploy from GitHub repo**.
4. Add a **Redis** plugin: Railway menu → **+ New → Database → Redis**.
5. Railway injects `REDIS_URL` automatically. Update `config.go` to read it:

```go
// In config.go, add a URL field to the Config struct:
type Config struct {
    Redis struct {
        // ...
        URL string `json:"url"`
    } `json:"redis"`
}

// In InitConfig(), read the env var:
if url := os.Getenv("REDIS_URL"); url != "" {
    cfg.Redis.URL = url
}

// In gores.go's NewGores(), use redis.DialURL:
Dial: func() (redis.Conn, error) {
    if config.Redis.URL != "" {
        return redis.DialURL(config.Redis.URL)
    }
    return redis.Dial("tcp", fmt.Sprintf("%s:%d", config.Redis.Host, config.Redis.Port))
},
```

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
      - REDIS_HOST=redis
    restart: on-failure

  consumer:
    build: .
    command: ["-o", "consume", "-w", "8", "-c", "/app/config.json"]
    depends_on:
      - redis
    environment:
      - REDIS_HOST=redis
    restart: unless-stopped
    deploy:
      replicas: 2   # run 2 consumer instances = horizontal scaling
```

> The `REDIS_HOST` env var override requires a small change in `config.go` (see below).

### Environment-variable config override (needed for containers)

Add this to `InitConfig()` in `lib_optimized/config.go`:

```go
import "os"

// After loading JSON, override with env vars if set
if h := os.Getenv("REDIS_HOST"); h != "" {
    cfg.Redis.Host = h
}
if p := os.Getenv("REDIS_PORT"); p != "" {
    port, _ := strconv.Atoi(p)
    cfg.Redis.Port = port
}
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

## Config for Cloud Redis (any provider)

Most managed Redis providers give you a URL like:
```
redis://:password@host:6379/0
```

Add a helper to `config.go`:

```go
func InitConfigFromURL(redisURL string) (*Config, error) {
    // Parse the URL and populate Config struct
    // github.com/gomodule/redigo has redis.DialURL() built in
}
```

Then in `NewGores()`:

```go
Dial: func() (redis.Conn, error) {
    return redis.DialURL(redisURL)
},
```

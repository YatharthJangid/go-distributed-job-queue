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

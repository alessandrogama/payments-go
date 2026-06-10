# Stage 1: Build binaries
FROM golang:1.26.4-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Copy dependency files and download
COPY go.mod go.sum ./
RUN go mod download

# Copy source tree
COPY . .

# Compile optimized static Go binaries
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/worker ./cmd/worker

# Stage 2: HTTP API Runtime Image
FROM alpine:latest AS api
WORKDIR /app
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/api .
COPY --from=builder /app/migrations ./migrations
EXPOSE 8080 2112
CMD ["./api"]

# Stage 3: Worker Daemon Runtime Image
FROM alpine:latest AS worker
WORKDIR /app
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/worker .
EXPOSE 2113
CMD ["./worker"]

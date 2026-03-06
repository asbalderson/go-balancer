# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /build

# Copy go.work and module files first for better layer caching
COPY go.work ./
COPY backend/go.mod backend/go.sum* ./backend/
COPY pkg/go.mod ./pkg/
COPY balancer/go.mod balancer/go.sum* ./balancer/
RUN cd backend && go mod download || true

# Copy source code
COPY backend/ ./backend/
COPY pkg/ ./pkg/

# Build the binary
RUN cd backend && CGO_ENABLED=0 GOOS=linux go build -o /build/out/backend cmd/backend/main.go

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /build/out/backend .

# Copy config file (will be overridden by ConfigMap in K8s)
COPY backend/config.json .

# Expose port (should match your config)
EXPOSE 8080

# Run the binary
CMD ["./backend"]

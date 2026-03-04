# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /build

# Copy go.work and module files first for better layer caching
COPY go.work ./
COPY balancer/go.mod balancer/go.sum* ./balancer/
COPY pkg/go.mod ./pkg/
COPY backend/go.mod backend/go.sum* ./backend/
RUN cd balancer && go mod download || true

# Copy source code
COPY balancer/ ./balancer/
COPY pkg/ ./pkg/

# Build the binary
RUN cd balancer && CGO_ENABLED=0 GOOS=linux go build -o /build/out/balancer cmd/balancer/main.go

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /build/out/balancer .

# Copy config file (will be overridden by ConfigMap in K8s)
COPY balancer/config.json .

# Expose port (should match your config)
EXPOSE 8080

# Run the binary
CMD ["./balancer"]

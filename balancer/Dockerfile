# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /build

# Copy go mod files first for better layer caching
COPY go.mod* go.sum* ./
RUN go mod download || true

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o balancer cmd/balancer/main.go

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /build/balancer .

# Copy config file (will be overridden by ConfigMap in K8s)
COPY config.json .

# Expose port (should match your config)
EXPOSE 8080

# Run the binary
CMD ["./balancer"]

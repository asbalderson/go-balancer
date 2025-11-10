.PHONY: help run test lint lint-fix build clean

# Set PATH to include Go binaries
export PATH := $(HOME)/go/bin:$(PATH)

# Default target - show help
help:
	@echo "Go Balancer - Available targets:"
	@echo "  make run        - Run the backend service"
	@echo "  make test       - Run all tests"
	@echo "  make lint       - Run golangci-lint checks"
	@echo "  make lint-fix   - Run golangci-lint and auto-fix issues"
	@echo "  make build      - Build backend binary"
	@echo "  make clean      - Clean build artifacts"

# Run the backend service
run:
	@echo "Running backend service..."
	cd backend && go run cmd/backend/main.go

# Run all tests
test:
	@echo "Running all tests..."
	cd backend && go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	cd backend && go test -cover ./...

# Run golangci-lint
lint:
	@echo "Running golangci-lint..."
	cd backend && golangci-lint run ./...

# Run golangci-lint with auto-fix
lint-fix:
	@echo "Running golangci-lint with auto-fix..."
	cd backend && golangci-lint run --fix ./...

# Build backend binary
build:
	@echo "Building backend..."
	cd backend && go build -o ../bin/backend cmd/backend/main.go
	@echo "Binary created at: bin/backend"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	cd backend && go clean

# Format code
fmt:
	@echo "Formatting Go code..."
	cd backend && go fmt ./...

# Run go mod tidy
tidy:
	@echo "Tidying Go modules..."
	cd backend && go mod tidy

# Quick check - format, lint, and test
check: fmt lint test
	@echo "All checks passed!"

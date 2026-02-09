.PHONY: help run test lint lint-fix imports build clean docker-build docker-build-backend docker-build-balancer kind kind-delete k8s-deploy k8s-deploy-backend k8s-deploy-balancer k8s-delete k8s-delete-backend k8s-delete-balancer k8s-load-image k8s-load-backend-image k8s-load-balancer-image k8s-status k8s-test k8s-test-backend-status k8s-test-backend-ping k8s-test-balancer-status k8s-update-backend k8s-update-balancer k8s-restart-backend k8s-restart-balancer

# Set PATH to include Go binaries
export PATH := $(HOME)/go/bin:$(PATH)

# Docker configuration
DOCKER_REGISTRY ?= localhost:5000
BACKEND_IMAGE ?= go-balancer-backend
BALANCER_IMAGE ?= go-balancer-balancer
DOCKER_TAG ?= latest

# Kind cluster configuration
KIND_CLUSTER_NAME ?= go-balancer

# Backend configuration
BACKEND_REPLICAS ?= 3

# Balancer configuration
BALANCER_REPLICAS ?= 1

# Default target - show help
help:
	@echo "Go Balancer - Available targets:"
	@echo ""
	@echo "Development:"
	@echo "  make run          - Run the backend service"
	@echo "  make test         - Run all tests"
	@echo "  make lint         - Run golangci-lint checks"
	@echo "  make lint-fix     - Run golangci-lint and auto-fix issues"
	@echo "  make imports      - Fix import order and formatting"
	@echo "  make build        - Build backend binary"
	@echo "  make clean        - Clean build artifacts"
	@echo ""
	@echo "Docker:"
	@echo "  make docker-build          - Build Docker images for both backend and balancer"
	@echo "  make docker-build-backend  - Build Docker image for backend only"
	@echo "  make docker-build-balancer - Build Docker image for balancer only"
	@echo ""
	@echo "Kubernetes:"
	@echo "  make kind                      - Create kind cluster (if not already running)"
	@echo "  make kind-delete               - Delete kind cluster"
	@echo "  make k8s-load-image            - Load both Docker images into kind cluster"
	@echo "  make k8s-load-backend-image    - Load backend Docker image only"
	@echo "  make k8s-load-balancer-image   - Load balancer Docker image only"
	@echo "  make k8s-deploy                - Deploy both backend and balancer"
	@echo "  make k8s-deploy-backend       - Deploy backend only"
	@echo "  make k8s-deploy-balancer      - Deploy balancer only"
	@echo "  make k8s-update-backend       - Rebuild, reload, and restart backend"
	@echo "  make k8s-update-balancer      - Rebuild, reload, and restart balancer"
	@echo "  make k8s-restart-backend      - Restart backend pods (no rebuild)"
	@echo "  make k8s-restart-balancer     - Restart balancer pods (no rebuild)"
	@echo "  make k8s-delete               - Delete all resources"
	@echo "  make k8s-delete-backend       - Delete backend only"
	@echo "  make k8s-delete-balancer      - Delete balancer only"
	@echo "  make k8s-status               - Show status of all pods and services"
	@echo "  make k8s-test                 - Test backend endpoints (status + ping)"
	@echo "  make k8s-test-backend-status  - Test backend /status endpoint"
	@echo "  make k8s-test-backend-ping    - Test backend /ping endpoint"
	@echo "  make k8s-test-balancer-status - Test balancer /status endpoint"
	@echo ""
	@echo "Variables (override with VARIABLE=value):"
	@echo "  DOCKER_REGISTRY      - Docker registry (default: localhost:5000)"
	@echo "  BACKEND_IMAGE        - Backend image name (default: go-balancer-backend)"
	@echo "  BALANCER_IMAGE       - Balancer image name (default: go-balancer-balancer)"
	@echo "  DOCKER_TAG           - Image tag (default: latest)"
	@echo "  KIND_CLUSTER_NAME    - Kind cluster name (default: go-balancer)"
	@echo "  BACKEND_REPLICAS     - Number of backend replicas (default: 3)"
	@echo "  BALANCER_REPLICAS    - Number of balancer replicas (default: 1)"

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

# Fix imports (order and format)
imports:
	@echo "Fixing imports..."
	cd backend && gci write -s standard -s default -s localmodule .

# Run go mod tidy
tidy:
	@echo "Tidying Go modules..."
	cd backend && go mod tidy

# Quick check - format, lint, and test
check: fmt lint test
	@echo "All checks passed!"

# Docker commands
docker-build-backend:
	@echo "Building backend Docker image..."
	docker build -t $(DOCKER_REGISTRY)/$(BACKEND_IMAGE):$(DOCKER_TAG) backend/
	@echo "Image built: $(DOCKER_REGISTRY)/$(BACKEND_IMAGE):$(DOCKER_TAG)"

docker-build-balancer:
	@echo "Building balancer Docker image..."
	docker build -t $(DOCKER_REGISTRY)/$(BALANCER_IMAGE):$(DOCKER_TAG) balancer/
	@echo "Image built: $(DOCKER_REGISTRY)/$(BALANCER_IMAGE):$(DOCKER_TAG)"

docker-build: docker-build-backend docker-build-balancer
	@echo "All Docker images built!"

# Kind cluster management
kind:
	@if kind get clusters 2>/dev/null | grep -q "^$(KIND_CLUSTER_NAME)$$"; then \
		echo "Kind cluster '$(KIND_CLUSTER_NAME)' already exists"; \
		kubectl config use-context kind-$(KIND_CLUSTER_NAME); \
		echo "Switched to context: kind-$(KIND_CLUSTER_NAME)"; \
	else \
		echo "Creating kind cluster '$(KIND_CLUSTER_NAME)' with NodePort mappings..."; \
		kind create cluster --name $(KIND_CLUSTER_NAME) --config kind-config.yaml; \
		echo "Cluster created and context set to: kind-$(KIND_CLUSTER_NAME)"; \
		echo "NodePort 30080 mapped to localhost:30080"; \
	fi

kind-delete:
	@echo "Deleting kind cluster '$(KIND_CLUSTER_NAME)'..."
	kind delete cluster --name $(KIND_CLUSTER_NAME)
	@echo "Cluster deleted"

# Kubernetes deployment commands
k8s-load-backend-image:
	@echo "Loading backend Docker image into kind cluster..."
	kind load docker-image $(DOCKER_REGISTRY)/$(BACKEND_IMAGE):$(DOCKER_TAG) --name $(KIND_CLUSTER_NAME)
	@echo "Backend image loaded into kind cluster"

k8s-load-balancer-image:
	@echo "Loading balancer Docker image into kind cluster..."
	kind load docker-image $(DOCKER_REGISTRY)/$(BALANCER_IMAGE):$(DOCKER_TAG) --name $(KIND_CLUSTER_NAME)
	@echo "Balancer image loaded into kind cluster"

k8s-load-image: k8s-load-backend-image k8s-load-balancer-image
	@echo "All Docker images loaded into kind cluster!"

k8s-deploy-backend:
	@echo "Deploying backend to Kubernetes (replicas: $(BACKEND_REPLICAS))..."
	kubectl apply -f k8s/namespace.yaml
	kubectl apply -f k8s/configmap.yaml
	sed 's/replicas: .*/replicas: $(BACKEND_REPLICAS)/' k8s/deployment.yaml | kubectl apply -f -
	kubectl apply -f k8s/service.yaml
	@echo "Backend deployed! Use 'make k8s-status' to check status"

k8s-restart-backend:
	@echo "Restarting backend deployment..."
	kubectl rollout restart deployment/backend -n go-balancer
	@echo "Waiting for rollout to complete..."
	kubectl rollout status deployment/backend -n go-balancer
	@echo "Backend restarted!"

k8s-update-backend: docker-build-backend k8s-load-backend-image k8s-restart-backend
	@echo "Backend updated successfully!"
	@echo ""
	@make k8s-status

k8s-deploy-balancer:
	@echo "Deploying balancer to Kubernetes (replicas: $(BALANCER_REPLICAS))..."
	kubectl apply -f k8s/namespace.yaml
	kubectl apply -f k8s/balancer-rbac.yaml
	kubectl apply -f k8s/balancer-configmap.yaml
	sed 's/replicas: .*/replicas: $(BALANCER_REPLICAS)/' k8s/balancer-deployment.yaml | kubectl apply -f -
	kubectl apply -f k8s/balancer-service.yaml
	@echo "Balancer deployed! Use 'make k8s-status' to check status"

k8s-restart-balancer:
	@echo "Restarting balancer deployment..."
	kubectl rollout restart deployment/balancer -n go-balancer
	@echo "Waiting for rollout to complete..."
	kubectl rollout status deployment/balancer -n go-balancer
	@echo "Balancer restarted!"

k8s-update-balancer: docker-build-balancer k8s-load-balancer-image k8s-restart-balancer
	@echo "Balancer updated successfully!"
	@echo ""
	@make k8s-status

k8s-deploy: k8s-deploy-backend k8s-deploy-balancer
	@echo "All components deployed!"

k8s-delete-backend:
	@echo "Deleting backend from Kubernetes..."
	kubectl delete -f k8s/service.yaml --ignore-not-found
	kubectl delete -f k8s/deployment.yaml --ignore-not-found
	kubectl delete -f k8s/configmap.yaml --ignore-not-found
	@echo "Backend deleted"

k8s-delete-balancer:
	@echo "Deleting balancer from Kubernetes..."
	kubectl delete -f k8s/balancer-service.yaml --ignore-not-found
	kubectl delete -f k8s/balancer-deployment.yaml --ignore-not-found
	kubectl delete -f k8s/balancer-configmap.yaml --ignore-not-found
	kubectl delete -f k8s/balancer-rbac.yaml --ignore-not-found
	@echo "Balancer deleted"

k8s-delete: k8s-delete-backend k8s-delete-balancer
	@echo "Deleting namespace..."
	kubectl delete -f k8s/namespace.yaml --ignore-not-found
	@echo "All components deleted"

k8s-status:
	@echo "=== Namespace ==="
	kubectl get namespace go-balancer 2>/dev/null || echo "Namespace not found"
	@echo ""
	@echo "=== Pods ==="
	kubectl get pods -n go-balancer -o wide 2>/dev/null || echo "No pods found"
	@echo ""
	@echo "=== Services ==="
	kubectl get svc -n go-balancer 2>/dev/null || echo "No services found"
	@echo ""
	@echo "=== Deployments ==="
	kubectl get deployment -n go-balancer 2>/dev/null || echo "No deployments found"

# Test backend endpoints via NodePort
k8s-test-backend-status:
	@echo "Testing backend /status endpoint via NodePort (localhost:30080)..."
	@echo ""
	@curl -s localhost:30080/status | jq . || curl -s localhost:30080/status
	@echo ""

k8s-test-backend-ping:
	@echo "Testing backend /ping endpoint - Load balancing across pods (10 requests)..."
	@echo "Watch for different podnames to see load balancing in action!"
	@echo ""
	@for i in 1 2 3 4 5 6 7 8 9 10; do \
		echo "Request $$i:"; \
		curl -s localhost:30080/ping | jq -r '"\(.podname) - count: \(.count)"' || curl -s localhost:30080/ping; \
	done
	@echo ""
	@echo "Notice: Different podnames = Kubernetes Service load balancing working!"

# Test balancer endpoints via NodePort
k8s-test-balancer-status:
	@echo "Testing balancer /status endpoint via NodePort (localhost:30081)..."
	@echo ""
	@curl -s localhost:30081/status | jq . || curl -s localhost:30081/status
	@echo ""

k8s-test: k8s-test-backend-status k8s-test-backend-ping k8s-test-balancer-status
	@echo "All tests completed!"

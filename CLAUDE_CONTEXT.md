# Claude Context: Go Load Balancer Project

## Project Overview
Alex is learning Go by building a load balancer in Kubernetes. This is a learning project focused on:
- Go fundamentals (coming from Python background)
- Kubernetes service discovery
- Container orchestration with kind

## Key Constraints
- **NEVER write Go code** - Alex wants to write all Go code themselves to learn
- Can review Go code and point out issues
- Can write Dockerfiles, Kubernetes manifests, and other non-Go files
- Can suggest Go libraries and approaches

## Architecture

### Backend Service
- Simple HTTP service that returns identifying information (pod name, IP)
- Listens on port 8080
- Returns JSON response with pod metadata
- Uses Kubernetes Downward API to get pod name

### Load Balancer
- Discovers backend pods using Kubernetes API (Endpoints)
- Implements round-robin load balancing with in-memory counter
- Uses `httputil.ReverseProxy` to forward requests
- Watches Endpoints via Informer pattern for dynamic discovery
- Requires RBAC (ServiceAccount, Role, RoleBinding) to read endpoints

### Deployment
- Using kind (Kubernetes in Docker) for local development
- Separate namespaces supported via ConfigMaps
- Both components use ConfigMaps mounted as JSON files at `/etc/config/config.json`

## Technology Decisions

### Configuration Management
- **ConfigMaps mounted as volumes** (JSON files)
- Location: `/etc/config/config.json` inside containers
- JSON format (not YAML) to avoid YAML-in-YAML complexity
- Apps should work with sensible defaults if config not provided
- Namespace injected via Kubernetes Downward API

### Logging
- Standard `log` package with structured prefixes
- All logs to stdout/stderr (12-factor app pattern)
- Log format: `[component] timestamp file:line message`
- Log levels via prefixes: `[INFO]`, `[ERROR]`, `[WARN]`, `[DEBUG]`
- View with `kubectl logs -f <pod>`

### Directory Structure
```
backend/
├── cmd/backend/main.go       # Entry point
├── internal/
│   ├── config/               # Config loading
│   └── handlers/             # HTTP handlers
├── config.json               # Local test config
├── go.mod
└── Dockerfile

loadbalancer/
├── cmd/loadbalancer/main.go  # Entry point
├── internal/
│   ├── config/               # Config loading
│   ├── discovery/            # K8s endpoint watching
│   ├── balancer/             # Round-robin logic
│   └── proxy/                # HTTP reverse proxy
├── config.json
├── go.mod
└── Dockerfile
```

## Go Libraries

### Backend (Standard Library Only)
- `net/http` - HTTP server
- `encoding/json` - JSON encoding/decoding
- `os` - Environment variables, file reading
- `log` - Logging

### Load Balancer
**Standard Library:**
- `net/http` - HTTP server
- `net/http/httputil` - `ReverseProxy`
- `encoding/json` - Config parsing
- `sync` - Mutex for concurrent backend list access
- `context` - Cancellation and timeouts
- `log` - Logging

**Kubernetes:**
- `k8s.io/client-go` - Kubernetes client
- `k8s.io/api/core/v1` - Core API types (Endpoints, Service, etc.)
- `k8s.io/apimachinery/pkg/apis/meta/v1` - Meta types
- `k8s.io/client-go/informers` - Informer pattern for watching
- `k8s.io/client-go/tools/cache` - Informer cache
- `k8s.io/client-go/rest` - REST config (InClusterConfig)

## Service Discovery Approach
- Use Kubernetes Endpoints API (not EndpointSlices - simpler for learning)
- Informer pattern to watch for endpoint changes
- Informer callbacks: OnAdd, OnUpdate, OnDelete
- Maintain in-memory list of backend pod IPs with mutex protection
- Auto-detect in-cluster vs out-of-cluster config

## Phase Plan

### Phase 1: Backend Service
1. Create Go module
2. Implement config loading (JSON from file)
3. Implement HTTP handler (returns pod name, IP)
4. Test locally
5. Write Dockerfile
6. Build and load into kind
7. Create ConfigMap, Deployment, Service manifests
8. Deploy and test in cluster

### Phase 2: Load Balancer
1. Create Go module
2. Implement config loading
3. Set up Kubernetes client (in-cluster + out-of-cluster)
4. Implement Endpoints informer
5. Implement round-robin balancer
6. Implement reverse proxy handler
7. Test locally (point at kind cluster)
8. Write Dockerfile
9. Create RBAC resources (ServiceAccount, Role, RoleBinding)
10. Create ConfigMap, Deployment, Service manifests
11. Deploy and test

### Phase 3: Enhancements (Later)
- Health checking
- Multiple load balancer instances
- Weighted round-robin based on load
- Artificial load generation endpoint on backend
- Persistent counter storage
- Graceful shutdown
- Metrics/observability

## Error Handling Philosophy
- Load balancer doesn't restart failed backends (Kubernetes does that)
- On backend failure, try next backend in rotation
- Log errors clearly
- Fail fast on startup issues (missing config, bad K8s connection)
- Let Kubernetes handle pod health (liveness/readiness probes)

## Testing Strategy
- Unit tests for each internal package
- Local testing with kind cluster
- Use `kubectl logs -f` to observe behavior
- Test pod scaling: `kubectl scale deployment backend --replicas=5`
- Test pod failures: `kubectl delete pod <backend-pod>`

## Configuration Examples

### Backend ConfigMap
```json
{
  "port": 8080,
  "serviceName": "backend"
}
```

### Load Balancer ConfigMap
```json
{
  "port": 8080,
  "backendServiceName": "backend-service",
  "backendPort": 8080
}
```

Namespace comes from Downward API env var.

## Common Go Patterns Alex Will Learn
- Structs with JSON tags
- Error handling (multiple return values)
- Pointers vs values
- Goroutines and channels (for endpoint watching)
- Mutex for concurrent access
- Interfaces (http.Handler, etc.)
- Package structure and exports (capitalization)
- Module system (go.mod, imports)

## Python → Go Translation Notes for Alex
- No exceptions: functions return `(result, error)`
- No classes: use structs + functions
- Explicit types: can't mix int/float/string without conversion
- Pointers are explicit: `&` to get address, `*` to dereference
- Multiple return values: `value, err := function()`
- Capitalization matters: uppercase = exported, lowercase = private
- No `self`: methods receive explicit receiver
- Slice ≈ Python list, Map ≈ Python dict
- `defer` for cleanup (like Python's `finally` or context managers)

## Communication Style with Alex
- We're coworkers, casual and collaborative
- Alex is smart but learning Go - don't condescend
- Be skeptical of assumptions, ask for evidence
- Push back when needed with citations
- Focus on teaching through doing, not lecturing
- Explain "why" not just "what"
- Relate Go concepts to Python where helpful

## Reminders
- Never write Go code - only review and suggest
- Can write: Dockerfiles, K8s YAML, bash scripts, documentation
- Encourage good practices: testing, logging, error handling
- Keep solutions simple and maintainable over clever
- This is a learning project - optimize for education, not performance

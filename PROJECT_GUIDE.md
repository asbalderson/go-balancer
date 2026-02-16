# Go Load Balancer Learning Project - Your Guide

## Project Goal
Build a working load balancer in Go deployed to Kubernetes (kind) to learn:
- Go fundamentals and idioms
- HTTP servers and reverse proxying
- Kubernetes service discovery and networking
- Container orchestration

## Project Status

### ✅ Phase 1: Backend Service - COMPLETE
- Backend HTTP service built and tested
- Configuration loaded from ConfigMap
- Pod metadata exposed via Downward API
- Deployed to Kubernetes with 3 replicas
- NodePort service on port 30080
- Responds to `/status` and `/ping` endpoints

**Tech used:**
- Go standard library (`net/http`, `encoding/json`, `os`, `log`)
- ConfigMaps for configuration
- Downward API for pod metadata (POD_NAME, POD_IP)
- Docker multi-stage builds
- Kubernetes Deployment, Service, ConfigMap

### 🚧 Phase 2: Load Balancer - IN PROGRESS
Next up: Build the load balancer component

### 📋 Phase 3: Enhancements - PLANNED
Future improvements and shared libraries

---

## What You're Building

### Component 1: Backend Service ✅
A simple HTTP API that responds with identifying information about itself.

**Configuration (JSON):**
```json
{
  "port": 8080,
  "serviceName": "backend"
}
```

**Endpoints:**
- `GET /status` - Returns pod metadata and config
- `GET /ping` - Returns pod name and request count

### Component 2: Load Balancer 🚧
Discovers backend pods via Kubernetes API and distributes requests using round-robin.

**Configuration (JSON):**
```json
{
  "port": 8080,
  "backendServiceName": "backend-service",
  "backendPort": 8080
}
```

**Responsibilities:**
- Load configuration from `/etc/config/config.json`
- Connect to Kubernetes API (both in-cluster and local kubeconfig)
- Watch Endpoints for the backend service
- Maintain list of healthy backend pod IPs
- Implement round-robin selection algorithm
- Proxy requests to selected backend using `httputil.ReverseProxy`
- Handle backend failures gracefully

**Endpoints:**
- `GET /health` - Returns health status and backend count
- `GET /info` - Returns backend list and request stats
- `GET /debug/next-backend` - Shows next backend selection (testing)
- Everything else → Proxied to backends

**Key learning:**
- Kubernetes client-go library
- Informer pattern for watching resources
- Concurrent programming (goroutines, mutexes)
- HTTP reverse proxy
- RBAC concepts

---

## Directory Structure

```
go-balancer/
├── backend/                    # ✅ COMPLETE
│   ├── cmd/backend/main.go
│   ├── internal/
│   │   ├── config/
│   │   └── handlers/
│   ├── config.json
│   ├── go.mod
│   └── Dockerfile
│
├── loadbalancer/               # 🚧 TODO
│   ├── cmd/loadbalancer/main.go
│   ├── internal/
│   │   ├── config/            # Config loading
│   │   ├── discovery/         # Kubernetes endpoint watching
│   │   ├── balancer/          # Round-robin selection
│   │   └── proxy/             # HTTP reverse proxy
│   ├── config.json
│   ├── go.mod
│   └── Dockerfile
│
├── k8s/
│   ├── namespace.yaml
│   ├── configmap.yaml         # Backend config
│   ├── deployment.yaml        # Backend deployment
│   ├── service.yaml           # Backend NodePort service
│   └── (loadbalancer manifests TODO)
│
├── Makefile                   # Build, test, deploy commands
├── kind-config.yaml           # Kind cluster with NodePort mapping
├── PROJECT_GUIDE.md           # This file
└── LOAD_BALANCER_CONCEPTS.md  # Load balancer theory and patterns
```

---

## Phase 2: Load Balancer Implementation

### Phase 2a: Get It Working (Local Development)

**Key Decisions:**
- Config: Service name, backend port, load balancing strategy (start with round-robin only)
- Config changes require manual pod restart (no hot-reload yet)
- Stream requests through without parsing/re-encoding
- Handle own routes (/health, /info, /debug) but proxy everything else
- Return backend errors directly to client (no retries yet)
- Build incrementally - test each piece before moving to next

---

### Incremental Implementation Steps

Each step should be working and testable before moving to the next. Days/weeks can pass between steps - each milestone is a clear checkpoint.

---

#### Step 1: Basic Server + Health Check
**Goal:** Prove the load balancer starts and responds

**What to build:**
- Initialize project: `go mod init loadbalancer`
- Create directory structure: `cmd/loadbalancer/`, `internal/`
- Write `cmd/loadbalancer/main.go` - starts HTTP server on port 8080
- Add handler for `GET /health` → returns `{"status": "healthy"}`

**Test it:**
```bash
go run cmd/loadbalancer/main.go
curl localhost:8080/health
# Expected: {"status": "healthy"}
```

**Success criteria:** ✅ Server starts, responds to health check
**Estimated time:** 30-60 minutes

---

#### Step 2: Add Config Package
**Goal:** Load configuration from file or environment

**What to build:**
- Create `internal/config/` package
- Define `Config` struct with: `BackendServiceName`, `BackendPort`, `Port`
- Implement `LoadFromFile(path string)` and `LoadFromEnv()` (reuse backend pattern)
- Update `main.go` to load config
- Write basic config tests

**Test it:**
```bash
# Create test config.json
echo '{"backendServiceName": "backend-service", "backendPort": 8080, "port": 8080}' > loadbalancer/config.json
go run cmd/loadbalancer/main.go
# Should start with config loaded
```

**Success criteria:** ✅ Config loads from file and environment
**Estimated time:** 1-2 hours

---

#### Step 3: Add Discovery (Read-Only)
**Goal:** Connect to Kubernetes and discover backend pods

**What to build:**
- Create `internal/discovery/` package
- Set up Kubernetes client (in-cluster config vs kubeconfig for local testing)
- Create Endpoints informer/watcher
- Maintain thread-safe list of backend IPs (use `sync.RWMutex`)
- Provide `GetBackends() []string` method
- Log when backends are added/removed
- Update `/health` to show backend count: `{"status": "healthy", "backends_available": 3}`

**Test it:**
```bash
# Point at your kind cluster
go run cmd/loadbalancer/main.go
curl localhost:8080/health
# Expected: {"status": "healthy", "backends_available": 3}

# Scale backends and watch logs
kubectl scale deployment/backend --replicas=5 -n go-balancer
# Should see discovery logs about new backends
```

**Success criteria:** ✅ Discovers 3 backend pods, count shows in /health, reacts to scaling
**Estimated time:** 2-3 hours (K8s client-go has learning curve)

---

#### Step 4: Add Info/Stats Endpoint
**Goal:** Observability before forwarding requests

**What to build:**
- Add request counters to main (use `sync.Map` or mutex-protected map)
- Create `GET /info` endpoint showing:
  - List of discovered backends
  - Requests forwarded per backend (all zeros initially)
  - Total requests
  - Uptime

**Example response:**
```json
{
  "uptime": "2m30s",
  "backends": [
    {"address": "10.244.0.5:8080", "requests_forwarded": 0},
    {"address": "10.244.0.6:8080", "requests_forwarded": 0},
    {"address": "10.244.0.7:8080", "requests_forwarded": 0}
  ],
  "total_requests": 0
}
```

**Test it:**
```bash
curl localhost:8080/info | jq .
# Should see all discovered backends with zero counts
```

**Success criteria:** ✅ Can see all backends and request counters
**Estimated time:** 1 hour

---

#### Step 5: Add Round-Robin Selection (No Forwarding)
**Goal:** Prove selection logic works before adding proxy complexity

**What to build:**
- Create `internal/balancer/` package
- Implement round-robin selection with counter (needs mutex)
- `SelectBackend(backends []string) (string, error)` - returns next backend, wraps around
- Add `GET /debug/next-backend` endpoint - shows which backend would be selected (doesn't forward)
- Write tests for balancer with mock backend lists

**How round-robin works with a changing backend list:**

The key insight: you don't loop over backends. Each request picks ONE backend:

```
1. Get current backends list (brief RLock, snapshot, RUnlock)
2. counter++ (atomic, no lock needed)
3. Pick: backends[counter % len(backends)]
4. Forward request to that backend
5. Done - lock released before forwarding
```

The counter lives separately and increments forever. Modulo handles changing list sizes:

```
Request 1: 3 backends, counter=1 → pick 1%3=1 → backend B
Request 2: 3 backends, counter=2 → pick 2%3=2 → backend C
(pod dies, now 2 backends)
Request 3: 2 backends, counter=3 → pick 3%2=1 → backend B
Request 4: 2 backends, counter=4 → pick 4%2=0 → backend A
```

Use `sync/atomic.AddUint64(&counter, 1)` for thread-safe counter increment without locks.

**Important:** The backend map can update between requests - that's fine! Each request gets a fresh snapshot. You're NOT holding a lock while forwarding.

**Test it:**
```bash
curl localhost:8080/debug/next-backend
# {"selected": "10.244.0.5:8080"}
curl localhost:8080/debug/next-backend
# {"selected": "10.244.0.6:8080"}
curl localhost:8080/debug/next-backend
# {"selected": "10.244.0.7:8080"}
curl localhost:8080/debug/next-backend
# {"selected": "10.244.0.5:8080"}  # wrapped around!
```

**Success criteria:** ✅ Selection rotates through all backends in order
**Estimated time:** 1-2 hours

---

#### Step 6: Add Proxy Forwarding
**Goal:** Actually forward requests to backends!

**What to build:**
- Create `internal/proxy/` package
- Use `httputil.ReverseProxy` to forward requests
- Create function that takes backend address and returns configured proxy
- Update main HTTP handler:
  - If path is `/health`, `/info`, or `/debug/*` → handle directly
  - Everything else → select backend and proxy
- Increment request counters when forwarding
- Log each forwarded request (which backend)

**Test it:**
```bash
# Forward a request through the LB to backend
curl localhost:8080/status
# Should see backend response with podname, etc.

# Check stats
curl localhost:8080/info
# Should see request counts incrementing

# Hit it multiple times, watch round-robin
for i in {1..9}; do curl -s localhost:8080/ping | jq -r .podname; done
# Should see all 3 backend pods in rotation
```

**Success criteria:**
- ✅ Requests forwarded to backends
- ✅ Backend responses returned to client
- ✅ Counters increment in `/info`
- ✅ Round-robin distribution working

**Estimated time:** 2-3 hours

---

#### Step 7: Test Locally End-to-End
**Goal:** Verify full functionality before K8s deployment

**What to test:**
- Start LB locally, pointing at kind cluster
- Scale backends up and down, verify LB adapts
- Send many requests, verify distribution
- Check `/info` for accurate stats
- Kill a backend pod, verify LB handles it (may fail requests - that's OK for now)

**Test commands:**
```bash
# Start LB
go run cmd/loadbalancer/main.go

# Send requests and watch distribution
for i in {1..30}; do curl -s localhost:8080/ping | jq -r .podname; done | sort | uniq -c
# Should see roughly even distribution (10, 10, 10)

# Check stats
curl localhost:8080/info | jq .

# Scale backends
kubectl scale deployment/backend --replicas=5 -n go-balancer
# Send more requests, should distribute across 5 backends
```

**Success criteria:**
- ✅ LB discovers backends automatically
- ✅ Requests distributed evenly
- ✅ Adapts to backend scaling
- ✅ All endpoints working

**Estimated time:** 1-2 hours of testing and fixing issues

---

**Summary of Step Boundaries:**

Each step has a clear "done" state you can demonstrate:
1. Health check responds ← **Start here**
2. Config loads successfully
3. Backends discovered and counted
4. Stats endpoint shows backends
5. Round-robin selection rotates
6. Requests actually proxied
7. Full local testing complete ← **End of Phase 2a**

**Total estimated time:** 10-15 hours (spread over days/weeks as needed)

**Next:** Phase 2b - Deploy to Kubernetes

---

### Phase 2b: Deploy to Kubernetes

**Focus:** Containerize and deploy load balancer to kind

**Steps:**
1. Create Dockerfile (multi-stage build like backend)
2. Build image: `docker build -t loadbalancer:latest loadbalancer/`
3. Load to kind: `kind load docker-image loadbalancer:latest --name go-balancer`
4. Create RBAC manifests (ServiceAccount, Role, RoleBinding)
   - Permissions needed: `get`, `list`, `watch` on `endpoints`
5. Create ConfigMap for load balancer config
6. Create Deployment manifest (use ServiceAccount, mount ConfigMap, inject NAMESPACE)
7. Create Service manifest (NodePort on different port, e.g., 30081)
8. Deploy: `kubectl apply -f k8s/loadbalancer-*.yaml`
9. Test via NodePort: `curl localhost:30081/`
10. Verify requests go through to backends
11. Scale backends and watch load balancer adapt
12. Check logs to see endpoint discovery working

---

### Phase 2c: Polish & Testing (Optional)

**Focus:** Add tests and improvements

**Steps:**
1. Add unit tests for all packages
2. Add integration tests (mock Kubernetes API)
3. Improve error messages and logging
4. Add metrics/observability (request counts, backend health)
5. Add multiple load balancing strategies (weighted, least-connections)
6. Add request retry logic (try different backend on failure)
7. Add circuit breaker (stop sending to failing backends)

---

## Phase 3: Enhancements & Shared Libraries

### Phase 3a: Shared Libraries & Refactoring (Planned)

**Focus:** Extract common code into shared packages

**Motivation:**
- Both backend and loadbalancer have similar config loading logic
- Only the config structure differs, the loading mechanism is the same
- Good opportunity to learn Go package design and interfaces
- Don't touch Phase 1 backend yet - wait until both components are working

**What to refactor:**

**Observed duplications (after building balancer):**
- `main.go` is 95% identical between backend and balancer
  - Config loading logic (lines 14-35)
  - Server setup (lines 42-52)
  - Only differs in which handler is created
- `getPodName()` - identical in both handlers
- `writeJSON()` - identical in both handlers
- `LoadFromFile()` / `LoadFromEnv()` - same pattern, just different field names

**Extraction candidates:**

1. **pkg/config/** (high priority)
   - Config file/env loading logic
   - Layered loading (file → env var overrides)
   - Common validation patterns

2. **pkg/httputil/** or **pkg/handlers/** (medium priority)
   - `writeJSON(w, status, data)` helper
   - `getPodName()` / `getPodIP()` helpers (from Downward API)
   - Error response helpers

3. **pkg/server/** (low priority - consider later)
   - Server creation with standard timeouts
   - Graceful shutdown logic (when added)
   - Signal handling

**Note:** Handlers themselves are unique enough - keep separate. The endpoints and business logic differ between backend and balancer, only the utilities are duplicated.

**New directory structure:**
```
go-balancer/
├── pkg/                        # Shared packages (can be imported by others)
│   └── config/
│       ├── loader.go          # Generic config loading logic
│       └── loader_test.go
├── backend/
│   ├── internal/
│   │   └── config/
│   │       ├── config.go      # Backend-specific Config struct
│   │       └── config_test.go # Now uses pkg/config/Loader
├── loadbalancer/
│   ├── internal/
│   │   └── config/
│   │       ├── config.go      # LoadBalancer-specific Config struct
│   │       └── config_test.go # Now uses pkg/config/Loader
```

**Implementation approach:**

1. **Define interface in `pkg/config/loader.go`:**
   - Interface for unmarshaling config (works with any struct)
   - Function: `Load(target interface{}, options ...Option) error`
   - Implements layered config loading:
     1. Start with zero values
     2. Load from first config file found (check multiple paths)
     3. Override with environment variables (if set)
   - All the common JSON parsing, file reading, error handling
   - Environment variable naming: `PREFIX_FIELD_NAME` (e.g., `BACKEND_PORT=8080`)

2. **Config loading order (12-factor app pattern):**
   ```
   Defaults → Config File → Environment Variables
   (lowest priority)              (highest priority)
   ```
   - **Example:** Backend config has `port: 8080` in file, but `BACKEND_PORT=9090` env var set → uses `9090`
   - **Why:** Allows base config in files, overrides for different environments (dev/staging/prod)
   - **Use case:** Same Docker image, different config per environment via env vars

3. **Update backend's `internal/config/config.go`:**
   - Keep `Config` struct (backend-specific)
   - Add struct tags for env var mapping: `json:"port" env:"BACKEND_PORT"`
   - Use `pkg/config.Load()` instead of custom implementation
   - Cleaner, less duplicated code

4. **Update loadbalancer's `internal/config/config.go`:**
   - Keep `Config` struct (loadbalancer-specific)
   - Add struct tags for env var mapping: `json:"port" env:"LOADBALANCER_PORT"`
   - Use same `pkg/config.Load()` with different prefix
   - Consistent behavior across components

5. **Write tests:**
   - Test generic loader with various struct types
   - Test file-only loading
   - Test env var overrides (file + env vars)
   - Test env-only loading (no file)
   - Test both backend and loadbalancer configs still work
   - Ensure no regressions

**Learning opportunities:**
- **Go package visibility:** `pkg/` vs `internal/` (public vs private packages)
- **Interface design:** Creating flexible, reusable interfaces
- **Generics:** Using `interface{}` or Go 1.18+ generics for type-safe config loading
- **Struct tags:** Using tags for metadata (`json:"port" env:"BACKEND_PORT"`)
- **Reflection:** Reading struct tags at runtime to map env vars to fields
- **12-factor app methodology:** Configuration via environment for cloud-native apps
- **Refactoring safely:** Making changes without breaking existing functionality
- **Dependency management:** How packages import each other
- **When to abstract:** Balance between DRY (Don't Repeat Yourself) and simplicity

**Key decisions to make:**
- Use `interface{}` (classic Go) or generics `[T any]` (Go 1.18+)?
- How much validation belongs in shared package vs component-specific?
- Add config validation interface? (e.g., `Validator` interface with `Validate() error`)
- How to handle type conversion from env vars (strings) to struct fields (int, bool, etc.)?
- Should config file paths be configurable? Search multiple locations? (`./config.json`, `/etc/config/config.json`)
- Naming convention for env vars: `PREFIX_FIELD_NAME` or `PREFIX__NESTED__FIELD`?

**Success criteria:**
- ✅ Both backend and loadbalancer use shared config loader
- ✅ No duplicated config loading logic
- ✅ Config loads from file by default
- ✅ Environment variables override file values when set
- ✅ Can run with env vars only (no config file)
- ✅ All existing tests still pass
- ✅ Code is cleaner and more maintainable
- ✅ You understand when to create shared packages vs keeping code separate

**Estimated time:** 3-5 hours (includes learning about package design patterns)

**Known issues to address during Phase 3a:**
- Balancer tests need updating to reflect current code
- Missing struct tag issue fixed (json struct tags had spaces)
- validate() error now properly checked and returned

**Note:** Don't start this until Phase 2b is complete and both components are deployed and working. Refactoring is easier when you have working code to test against!

---

### Other Enhancement Ideas

- **Health checking backends**
  - Active health checks from load balancer
  - Remove unhealthy backends from rotation
  - Configurable health check intervals

- **Multiple load balancer replicas**
  - Handle multiple LB instances
  - Consider shared state or accept independent round-robin counters

- **Weighted round-robin based on load**
  - Track backend response times or active connections
  - Send more traffic to less-loaded backends
  - Add `/load` endpoint to artificially increase backend load for testing

- **Persistent round-robin counter**
  - Use Redis or database to track state
  - Survive load balancer restarts
  - Coordinate across multiple LB replicas

- **Logging package/wrapper**
  - Control log levels via config (DEBUG, INFO, WARN, ERROR)
  - Structured logging with `slog` (stdlib since Go 1.21)
  - Optional JSON output for log aggregators
  - Could be a shared package in `pkg/logging/`

- **Metrics and observability (Prometheus)**
  - Request counts, latency histograms
  - Backend health status
  - Expose `/metrics` endpoint

- **Graceful shutdown handling**
  - Drain in-flight requests before shutdown
  - Handle SIGTERM properly

- **Request retry logic**
  - Retry failed requests to different backend
  - Configurable retry attempts and backoff

- **Circuit breaker pattern**
  - Temporarily stop sending to failing backends
  - Auto-recover when backend is healthy

---

## Go Packages & Libraries

### Load Balancer Dependencies

**Standard library:**
- `net/http` - HTTP server
- `net/http/httputil` - ReverseProxy for forwarding requests
- `encoding/json` - Config parsing
- `sync` - Mutex for protecting concurrent access to backend list
- `context` - Cancellation and timeouts
- `log` - Logging

**Kubernetes client (external):**
- `k8s.io/client-go` - Official Kubernetes Go client
- `k8s.io/api/core/v1` - Core Kubernetes types (Endpoints, Pods, Services)
- `k8s.io/apimachinery/pkg/apis/meta/v1` - Metadata types
- `k8s.io/client-go/informers` - Informer pattern for efficient watching
- `k8s.io/client-go/tools/cache` - Caching for informers
- `k8s.io/client-go/rest` - REST configuration (InClusterConfig)

**Installing dependencies:**
```bash
cd loadbalancer
go get k8s.io/client-go@latest
go get k8s.io/api@latest
go get k8s.io/apimachinery@latest
```

---

## Key Go Concepts You'll Use

### 1. Error Handling
Go doesn't have exceptions - functions return errors.

**Pattern:**
```go
result, err := someFunction()
if err != nil {
    // handle error
}
// use result
```

### 2. Concurrency
Goroutines and channels for concurrent operations.

**For this project:**
- Endpoint watcher runs in a goroutine
- Mutex protects backend list from concurrent access
- Multiple request handlers accessing shared state

**Concepts:**
- `go functionName()` to start goroutine
- `sync.Mutex` for protecting shared data
- Channels for communication (maybe used in discovery)

### 3. Interfaces
Go's approach to polymorphism.

**You'll encounter:**
- `http.Handler` interface
- Custom interfaces for testing (maybe)

---

## Kubernetes Concepts

### Service Discovery with Endpoints

**How it works:**
1. Your backend pods are managed by a Deployment
2. A Service selects those pods via labels
3. Kubernetes automatically creates/updates an Endpoints object
4. Endpoints object contains list of pod IPs and ports
5. Load balancer watches this Endpoints object
6. When pods are added/removed, load balancer is notified

**Informer Pattern:**
- More efficient than polling
- Maintains local cache of Endpoints
- Provides callbacks: OnAdd, OnUpdate, OnDelete
- Handles reconnection automatically

### RBAC (Role-Based Access Control)

Load balancer needs permission to read Endpoints.

**Resources needed:**
1. **ServiceAccount** - Identity for your pods
2. **Role** - Defines permissions (get, list, watch endpoints)
3. **RoleBinding** - Grants role to service account

**In your deployment:**
```yaml
spec:
  serviceAccountName: loadbalancer-sa
```

### Downward API

Inject pod metadata as environment variables.

**Example:**
```yaml
env:
- name: NAMESPACE
  valueFrom:
    fieldRef:
      fieldPath: metadata.namespace
```

Now your app can read `os.Getenv("NAMESPACE")`.

### ConfigMaps as Volumes

Mount configuration files into pods.

File appears at `/etc/config/config.json` inside container.

---

## Useful Commands

**Go:**
```bash
go mod init <name>          # Initialize module
go get <package>            # Add dependency
go mod tidy                 # Clean up dependencies
go run cmd/loadbalancer/main.go  # Run application
go test ./...               # Run all tests
go test -v ./internal/config # Run specific package tests
go build -o loadbalancer cmd/loadbalancer/main.go  # Build binary
```

**Makefile:**
```bash
make help                   # Show all available targets
make run                    # Run backend service
make test                   # Run tests
make lint                   # Run linter
make build                  # Build binary
make docker-build           # Build Docker image
make kind                   # Create/use kind cluster
make k8s-deploy-backend     # Deploy backend
make k8s-status             # Show K8s status
make k8s-test               # Test backend endpoints
```

**Kubernetes:**
```bash
kubectl get pods -n go-balancer              # List pods
kubectl get svc -n go-balancer               # List services
kubectl get endpoints backend-service -n go-balancer  # See discovered endpoints
kubectl logs -f <pod-name> -n go-balancer   # Follow logs
kubectl describe pod <pod-name> -n go-balancer  # Detailed pod info
kubectl scale deployment/backend --replicas=5 -n go-balancer  # Scale backends
```

---

## Testing Your Load Balancer

**Terminal setup:**
```bash
# Terminal 1: Watch load balancer logs
kubectl logs -f deployment/loadbalancer -n go-balancer

# Terminal 2: Watch backend logs
kubectl logs -f -l app=backend -n go-balancer

# Terminal 3: Send requests
curl localhost:30081/
```

**Test scenarios:**
```bash
# Scale backends up
kubectl scale deployment/backend --replicas=5 -n go-balancer
# Watch LB discover new pods, send requests, see distribution

# Delete a pod
kubectl delete pod <backend-pod-name> -n go-balancer
# Watch LB handle it, requests should still work

# Scale down
kubectl scale deployment/backend --replicas=2 -n go-balancer
# Verify LB removes old endpoints
```

---

## Common Go Gotchas (Coming from Python)

1. **No exceptions** - Check errors explicitly after every function call
2. **Types are strict** - Can't add int and float without conversion
3. **Pointers matter** - Passing by value copies, use pointers for modification
4. **Capitalization = visibility** - Uppercase = public, lowercase = private
5. **No default parameters** - Functions can't have optional args
6. **Nil is not None** - But similar concept for pointers, slices, maps
7. **Slices vs Arrays** - Arrays are fixed size, slices are dynamic (use slices)
8. **Range returns copies** - Modifying loop variable doesn't affect original
9. **Goroutines aren't free** - Don't spawn millions without thinking
10. **Zero values** - Variables have default values (0, "", false, nil)

---

## Resources

**Go:**
- Official Tour: https://go.dev/tour/
- Effective Go: https://go.dev/doc/effective_go
- Go by Example: https://gobyexample.com/

**Kubernetes Client-Go:**
- Examples: https://github.com/kubernetes/client-go/tree/master/examples
- Informer pattern: https://pkg.go.dev/k8s.io/client-go/informers

**Kubernetes:**
- Service Discovery: https://kubernetes.io/docs/concepts/services-networking/service/
- ConfigMaps: https://kubernetes.io/docs/concepts/configuration/configmap/
- RBAC: https://kubernetes.io/docs/reference/access-authn-authz/rbac/

---

## Success Criteria

**Phase 1 Complete:** ✅
- Backend pods running in kind
- Can curl backend and get pod name in response
- Different replicas return different pod names
- Configuration loaded from ConfigMap

**Phase 2 Complete:** 🚧
- Load balancer discovers backend endpoints automatically
- Requests distributed evenly across backends (check logs)
- Scaling backends up/down updates load balancer dynamically
- Deleting backend pods doesn't break load balancer
- All components log clearly to stdout

---

Have fun! Remember: this is about learning, so take time to understand each piece. Break problems down, test incrementally, and don't hesitate to ask questions.

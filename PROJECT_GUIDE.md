# Go Load Balancer Learning Project - Your Guide

## Project Goal
Build a working load balancer in Go deployed to Kubernetes (kind) to learn:
- Go fundamentals and idioms
- HTTP servers and reverse proxying
- Kubernetes service discovery and networking
- Container orchestration

## What You're Building

### Component 1: Backend Service
A simple HTTP API that responds with identifying information about itself.

**Responsibilities:**
- Start HTTP server on port 8080
- Respond to requests with JSON containing pod name and IP address
- Load configuration from `/etc/config/config.json` (mounted ConfigMap)
- Log requests to stdout

**Configuration (JSON):**
```json
{
  "port": 8080,
  "serviceName": "backend"
}
```

**Should learn:**
- HTTP server basics in Go
- JSON parsing and encoding
- File I/O
- Environment variables (for Downward API)
- Logging to stdout

### Component 2: Load Balancer
Discovers backend pods via Kubernetes API and distributes requests using round-robin.

**Responsibilities:**
- Load configuration from `/etc/config/config.json`
- Connect to Kubernetes API (both in-cluster and local kubeconfig)
- Watch Endpoints for the backend service
- Maintain list of healthy backend pod IPs
- Implement round-robin selection algorithm
- Proxy requests to selected backend using reverse proxy
- Handle backend failures gracefully

**Configuration (JSON):**
```json
{
  "port": 8080,
  "backendServiceName": "backend-service",
  "backendPort": 8080
}
```

Namespace auto-injected via Kubernetes Downward API.

**Should learn:**
- Kubernetes client-go library
- Informer pattern for watching resources
- Concurrent programming (goroutines, mutexes)
- HTTP reverse proxy
- RBAC concepts

## Directory Structure

```
go-balancer/
├── backend/
│   ├── cmd/
│   │   └── backend/
│   │       └── main.go          # Entry point - wires everything together
│   ├── internal/
│   │   ├── config/
│   │   │   ├── config.go        # Config struct and loading logic
│   │   │   └── config_test.go   # Tests
│   │   └── handlers/
│   │       ├── handlers.go      # HTTP request handlers
│   │       └── handlers_test.go # Tests
│   ├── config.json               # Local test config
│   ├── go.mod                    # Go module definition
│   ├── go.sum                    # Dependency checksums
│   └── Dockerfile
│
├── loadbalancer/
│   ├── cmd/
│   │   └── loadbalancer/
│   │       └── main.go          # Entry point
│   ├── internal/
│   │   ├── config/
│   │   │   ├── config.go        # Config loading
│   │   │   └── config_test.go
│   │   ├── discovery/
│   │   │   ├── discovery.go     # Kubernetes endpoint watching
│   │   │   └── discovery_test.go
│   │   ├── balancer/
│   │   │   ├── balancer.go      # Round-robin selection logic
│   │   │   └── balancer_test.go
│   │   └── proxy/
│   │       ├── proxy.go         # HTTP reverse proxy handler
│   │       └── proxy_test.go
│   ├── config.json
│   ├── go.mod
│   ├── go.sum
│   └── Dockerfile
│
└── k8s/
    ├── backend-configmap.yaml
    ├── backend-deployment.yaml
    ├── backend-service.yaml
    ├── loadbalancer-configmap.yaml
    ├── loadbalancer-rbac.yaml      # ServiceAccount, Role, RoleBinding
    ├── loadbalancer-deployment.yaml
    └── loadbalancer-service.yaml
```

**Why this structure?**
- `cmd/` contains entry points (main packages)
- `internal/` contains packages that can't be imported by other projects
- Each subdirectory is a separate package
- Separation of concerns: config, handlers, business logic
- Easy to test each component independently
- This is standard Go project layout for real applications

## Component Responsibilities & Contracts

This section defines what each package should do and what it should expose to other parts of the system.

### Backend Service Components

#### `internal/config` Package

**Job:**
- Load configuration from `/etc/config/config.json`
- Provide config data to other parts of the application

**Should expose:**
- A `Config` struct with fields: `Port` (int), `ServiceName` (string)
- A `LoadConfig(path string)` function that returns `(*Config, error)`

**Responsibilities:**
- Read file from disk
- Parse JSON into struct
- Return errors if file missing or JSON invalid
- Provide sensible defaults if needed

---

#### `internal/handlers` Package

**Job:**
- Handle incoming HTTP requests
- Return JSON response with pod identity information

**Should expose:**
- An HTTP handler function (signature depends on your design)
- Could be `func Handler(w http.ResponseWriter, r *http.Request)`
- Or `func NewHandler(podName, podIP, serviceName string)` that returns a handler

**Responsibilities:**
- Get pod metadata (from env vars or passed in)
- Format response as JSON
- Write response to HTTP response writer
- Log the request

**Response should contain:**
```json
{
  "podName": "backend-pod-xyz",
  "podIP": "10.244.1.5",
  "serviceName": "backend",
  "timestamp": "2025-11-02T10:30:00Z"
}
```

---

#### `cmd/backend/main.go`

**Job:**
- Wire everything together
- Entry point for the application

**Responsibilities:**
1. Load config using the config package
2. Get pod metadata from environment variables (POD_NAME, POD_IP via Downward API)
3. Set up HTTP handler with the handlers package
4. Start HTTP server on configured port
5. Set up logging
6. Handle startup errors

**Flow:**
```
main() →
  Load config →
  Get pod metadata from env →
  Create handler →
  Register handler with http server →
  Start listening →
  Log errors/info
```

---

### Load Balancer Components

#### `internal/config` Package

**Job:**
- Load configuration from `/etc/config/config.json`

**Should expose:**
- A `Config` struct with: `BackendServiceName` (string), `BackendPort` (int), `Port` (int)
- A `LoadConfig(path string)` function returning `(*Config, error)`

**Responsibilities:**
- Same as backend config package
- Read, parse, validate, return errors

---

#### `internal/discovery` Package

**Job:**
- Connect to Kubernetes API
- Watch Endpoints for the backend service
- Maintain current list of backend pod IPs
- Notify when backends are added/removed

**Should expose:**
- A function to create the K8s client (handles in-cluster vs out-of-cluster)
- A function/struct to start watching endpoints
- A way to get the current list of backend addresses
- Callback mechanism for when backends change (or a channel)

**Responsibilities:**
- Set up Kubernetes client-go
- Create Informer for Endpoints resource
- Set up callbacks: OnAdd, OnUpdate, OnDelete
- Convert Endpoints object to list of "host:port" strings
- Thread-safe access to backend list (uses mutex)

**Should provide to other packages:**
- Current list of backends as `[]string` (e.g., `["10.244.1.5:8080", "10.244.1.6:8080"]`)
- Way to register for updates (or just poll current list)

---

#### `internal/balancer` Package

**Job:**
- Implement round-robin selection algorithm
- Pick which backend to send the next request to

**Should expose:**
- A struct or function that maintains round-robin state
- A `SelectBackend(backends []string)` function that returns the next backend address
- Thread-safe selection (uses mutex for counter)

**Responsibilities:**
- Maintain counter for round-robin position
- Increment counter on each call
- Wrap around when reaching end of list
- Handle empty backend list gracefully
- Thread-safe (multiple requests happening concurrently)

**Returns:**
- A backend address string (e.g., `"10.244.1.5:8080"`)
- Or error if no backends available

---

#### `internal/proxy` Package

**Job:**
- Forward HTTP requests to selected backend
- Handle the actual proxying

**Should expose:**
- A handler function that takes backend address and forwards the request
- Possibly `NewProxy()` that returns an http.Handler
- Or `ProxyRequest(w, r, backendAddr)` function

**Responsibilities:**
- Use `httputil.ReverseProxy` from standard library
- Set target backend URL
- Forward request headers, body, method
- Return response to client
- Handle errors (backend down, timeout, etc.)
- Log requests and errors

**Needs:**
- Backend address (from balancer)
- Original request (from HTTP handler)
- Response writer (to send response back)

---

#### `cmd/loadbalancer/main.go`

**Job:**
- Wire all components together
- Entry point

**Responsibilities:**
1. Load config
2. Get namespace from environment (Downward API)
3. Initialize Kubernetes discovery (pass namespace and service name)
4. Start watching endpoints (runs in goroutine)
5. Create HTTP handler that:
   - Gets current backend list from discovery
   - Selects backend using balancer
   - Proxies request using proxy package
6. Start HTTP server on configured port
7. Handle graceful shutdown
8. Log everything

**Flow:**
```
main() →
  Load config →
  Get namespace from env →
  Initialize K8s discovery →
  Start endpoint watcher (goroutine) →
  Create HTTP handler:
    - Get backends from discovery
    - Select one with balancer
    - Proxy with proxy package
  →
  Start HTTP server →
  Handle errors
```

---

### Component Interactions

**Backend request flow:**
```
HTTP Request →
  main (HTTP server) →
    handlers.Handler() →
      Returns JSON with pod info
```

**Load Balancer request flow:**
```
HTTP Request →
  main (HTTP server) →
    Get backends from discovery.GetBackends() →
    Select backend with balancer.SelectBackend(backends) →
    Proxy request with proxy.ProxyRequest(w, r, backend) →
      ReverseProxy forwards to backend →
        Backend responds →
          Response sent to client
```

**Load Balancer background process:**
```
main() starts goroutine →
  discovery.Watch() →
    K8s Informer callbacks (OnAdd/OnUpdate/OnDelete) →
      Updates internal backend list →
        (main HTTP handler reads from this list)
```

---

### Key Data Flow Summary

**Configuration:**
- ConfigMap mounted as `/etc/config/config.json`
- Config package reads and parses it
- Main gets Config struct with all settings

**Pod Metadata (Backend):**
- Kubernetes Downward API injects POD_NAME, POD_IP as env vars
- Main reads from `os.Getenv()`
- Passes to handlers for response

**Service Discovery (Load Balancer):**
- Discovery package watches Kubernetes Endpoints
- Maintains `[]string` of backend addresses
- Balancer package selects one from the list
- Proxy package forwards request to selected backend

**Thread Safety:**
- Discovery updates backend list (one goroutine)
- HTTP handlers read backend list (many goroutines)
- Both protected by `sync.Mutex`

## Go Packages & Libraries

### Backend Service

**Standard library only:**
- `net/http` - HTTP server and routing
- `encoding/json` - JSON encoding/decoding
- `os` - File reading, environment variables, hostname
- `log` - Logging

### Load Balancer

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

## Key Go Concepts You'll Use

### 1. Structs and JSON Tags
Define data structures and map them to JSON fields.

**Concepts:**
- Struct fields with JSON tags for marshaling/unmarshaling
- Exported (capitalized) vs unexported (lowercase) fields
- Pointers vs values

### 2. Error Handling
Go doesn't have exceptions - functions return errors.

**Pattern:**
```
result, err := someFunction()
if err != nil {
    // handle error
}
// use result
```

**Concepts:**
- Multiple return values
- Explicit error checking
- `log.Fatal()` for unrecoverable errors
- Wrapping errors with context

### 3. HTTP Server
Build web servers with the standard library.

**Concepts:**
- `http.HandleFunc()` to register routes
- `http.ResponseWriter` and `http.Request`
- `http.ListenAndServe()` to start server
- JSON encoding to response writer

### 4. Concurrency
Goroutines and channels for concurrent operations.

**For this project:**
- Endpoint watcher runs in a goroutine
- Mutex protects backend list from concurrent access
- Multiple request handlers accessing shared state

**Concepts:**
- `go functionName()` to start goroutine
- `sync.Mutex` for protecting shared data
- Channels for communication (maybe used in discovery)

### 5. Packages and Imports
Organize code into reusable packages.

**Concepts:**
- `package` declaration at top of each file
- Import paths based on directory structure
- Exported names start with capital letter
- Import your own packages: `backend/internal/config`

### 6. Interfaces
Go's approach to polymorphism.

**You'll encounter:**
- `http.Handler` interface
- Custom interfaces for testing (maybe)

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
- name: POD_NAME
  valueFrom:
    fieldRef:
      fieldPath: metadata.name
```

Now your app can read `os.Getenv("NAMESPACE")`.

### ConfigMaps as Volumes

Mount configuration files into pods.

**ConfigMap:**
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: backend-config
data:
  config.json: |
    {
      "port": 8080
    }
```

**Deployment:**
```yaml
volumeMounts:
- name: config
  mountPath: /etc/config
  readOnly: true
volumes:
- name: config
  configMap:
    name: backend-config
```

File appears at `/etc/config/config.json` inside container.

## Logging Strategy

**Best practice for containers:**
- Write all logs to stdout/stderr
- Let Kubernetes capture and aggregate them
- Use `kubectl logs` to view

**In Go:**
- Use standard `log` package
- Set helpful prefix: `log.SetPrefix("[backend] ")`
- Add flags for timestamp and file:line: `log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)`
- Use prefixes for levels: `log.Println("[INFO] message")`

**Viewing logs:**
```bash
kubectl logs -f deployment/backend
kubectl logs -f deployment/loadbalancer
kubectl logs <pod-name> --tail=50
kubectl logs <pod-name> --previous  # Previous crash
```

## Implementation Phases

### Phase 1: Backend Service

**Steps:**
1. Initialize Go module: `go mod init backend`
2. Create directory structure (cmd/, internal/)
3. Implement config package (load JSON from file)
4. Implement handlers package (HTTP response with pod info)
5. Write main.go to wire everything together
6. Test locally: `go run cmd/backend/main.go`
7. Write unit tests: `go test ./...`
8. Create Dockerfile (multi-stage build)
9. Build image and load to kind
10. Create Kubernetes manifests (ConfigMap, Deployment, Service)
11. Deploy: `kubectl apply -f k8s/backend-*.yaml`
12. Test: `kubectl port-forward svc/backend-service 8080:8080`

### Phase 2: Load Balancer

**Steps:**
1. Initialize Go module: `go mod init loadbalancer`
2. Create directory structure
3. Implement config package
4. Implement discovery package (K8s client setup, Endpoints informer)
5. Implement balancer package (round-robin algorithm)
6. Implement proxy package (reverse proxy handler)
7. Write main.go
8. Test locally (point at kind cluster with kubeconfig)
9. Write unit tests
10. Create Dockerfile
11. Build and load to kind
12. Create RBAC manifests
13. Create ConfigMap, Deployment, Service manifests
14. Deploy: `kubectl apply -f k8s/loadbalancer-*.yaml`
15. Expose with NodePort and test

### Phase 3: Future Enhancements

**Ideas:**
- Health checking backends
- Multiple load balancer replicas
- Weighted round-robin based on load
- Backend endpoint for artificial load generation
- Persistent round-robin counter (database or Redis)
- Metrics and observability (Prometheus)
- Graceful shutdown handling
- Request retry logic
- Circuit breaker pattern
- Logging package/wrapper for consistent logging
  - Control log levels via config (DEBUG, INFO, WARN, ERROR)
  - Structured logging with `slog` (stdlib since Go 1.21)
  - Optional JSON output for log aggregators
  - Good learning experience for package design patterns

## Testing Your Load Balancer

**Terminal setup:**
```bash
# Terminal 1: Watch load balancer logs
kubectl logs -f deployment/loadbalancer

# Terminal 2: Watch backend logs
kubectl logs -f -l app=backend

# Terminal 3: Send requests
kubectl port-forward svc/loadbalancer-service 8080:8080
# Then: curl http://localhost:8080/
```

**Test scenarios:**
```bash
# Scale backends
kubectl scale deployment/backend --replicas=5

# Watch load balancer discover new pods
# Send requests and see distribution

# Delete a pod
kubectl delete pod <backend-pod-name>

# Watch load balancer handle it
# Send requests - should still work

# Scale down
kubectl scale deployment/backend --replicas=2

# Verify load balancer removes old endpoints
```

## kind Cluster Setup

```bash
# Install kind (if not already)
# https://kind.sigs.k8s.io/docs/user/quick-start/

# Create cluster
kind create cluster --name loadbalancer-learning

# Verify
kubectl cluster-info --context kind-loadbalancer-learning

# Load images to kind (after building)
kind load docker-image backend:latest --name loadbalancer-learning
kind load docker-image loadbalancer:latest --name loadbalancer-learning
```

## Useful Commands

**Go:**
```bash
go mod init <name>          # Initialize module
go get <package>            # Add dependency
go mod tidy                 # Clean up dependencies
go run cmd/backend/main.go  # Run application
go test ./...               # Run all tests
go test -v ./internal/config # Run specific package tests
go build -o backend cmd/backend/main.go  # Build binary
```

**Docker:**
```bash
docker build -t backend:latest backend/
docker build -t loadbalancer:latest loadbalancer/
kind load docker-image backend:latest --name <cluster-name>
```

**Kubernetes:**
```bash
kubectl apply -f k8s/                    # Apply all manifests
kubectl get pods                         # List pods
kubectl get svc                          # List services
kubectl get endpoints backend-service    # See discovered endpoints
kubectl logs -f <pod-name>              # Follow logs
kubectl describe pod <pod-name>         # Detailed pod info
kubectl port-forward svc/<name> 8080:8080  # Access service locally
kubectl delete -f k8s/                  # Clean up
```

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

## Questions to Consider While Building

1. What happens when all backends are down?
2. Should the round-robin counter be per-request or maintained across requests?
3. How do you handle backends being added while requests are in-flight?
4. What happens if the Kubernetes API connection drops?
5. Should you remove backends immediately on delete, or wait for failures?
6. How do you test the load balancer behavior without deploying to Kubernetes?
7. What information should be in logs for debugging?
8. Should configuration changes require pod restart?

## Success Criteria

**Phase 1 Complete:**
- ✅ Backend pods running in kind
- ✅ Can curl backend and get pod name in response
- ✅ Different replicas return different pod names
- ✅ Configuration loaded from ConfigMap

**Phase 2 Complete:**
- ✅ Load balancer discovers backend endpoints automatically
- ✅ Requests distributed evenly across backends (check logs)
- ✅ Scaling backends up/down updates load balancer dynamically
- ✅ Deleting backend pods doesn't break load balancer
- ✅ All components log clearly to stdout

---

Have fun! Remember: this is about learning, so take time to understand each piece. Break problems down, test incrementally, and don't hesitate to ask questions.

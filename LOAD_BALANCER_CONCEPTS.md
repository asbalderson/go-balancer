# Load Balancer Concepts - Connection Management & Architecture

This document summarizes key concepts about how load balancers work, manage connections, handle failures, and interact with Kubernetes.

## Table of Contents
- [Long-Lived Connections](#long-lived-connections)
- [Load Balancing the Load Balancers](#load-balancing-the-load-balancers)
- [Multiple LB Instances - State Coordination](#multiple-lb-instances---state-coordination)
- [Rolling Restarts - The Race Condition](#rolling-restarts---the-race-condition)
- [Connection Tracking in Go](#connection-tracking-in-go)
- [Graceful Shutdown](#graceful-shutdown)
- [The Long Connection Problem](#the-long-connection-problem)

---

## Long-Lived Connections

### How Load Balancers Handle Streaming/Large Files

**Key insight:** Load balancers don't "keep track" of streaming data - they just keep the connection open.

**The process:**
1. Client connects to LB
2. LB picks a backend (round-robin, least connections, etc.)
3. LB opens connection to that backend
4. Data **streams through** the LB without buffering
5. Connection stays pinned to that backend until closed

**The LB is just a pipe:**
- Doesn't buffer the video/file in memory
- Just forwards TCP packets back and forth
- OS handles the connection state
- Connection can stay open for hours if needed

**What LBs track:**
- Which client connections map to which backend connections
- How many active connections each backend has (for "least connections" algorithm)
- Backend health status

**Common Load Balancing Strategies:**
- **Round-robin**: Pick next backend for each NEW connection
- **Least connections**: Send to backend with fewest active connections (better for long-lived connections)
- **IP hash**: Same client IP always goes to same backend (sticky sessions)
- **Weighted**: More capable backends get more traffic

---

## Load Balancing the Load Balancers

**Yes, you typically do need another load balancer in front!**

### Common Patterns

#### 1. DNS Round-Robin
```
clients → DNS (returns multiple IPs) → LB1, LB2, LB3 → backends
```
- Simplest approach
- DNS returns different LB IP for each query
- ❌ Doesn't know if an LB is down
- ❌ Can't do health checks

#### 2. Layer 4 LB in front of Layer 7 LBs
```
clients → L4 LB (fast, simple TCP) → L7 LB1, L7 LB2 → backends
```
- Common pattern in production
- L4 LB just forwards TCP (very fast, high throughput)
- L7 LBs do smart routing (URL-based, headers, SSL termination)
- Cloud providers often use this pattern

#### 3. Anycast Routing
```
clients → network routes to closest LB → many LBs with same IP
```
- Multiple LBs advertise same IP address
- Network routing (BGP) picks closest one
- Used by CDNs (Cloudflare, Akamai, etc.)
- Requires network infrastructure support

#### 4. Cloud Provider Load Balancers
```
clients → cloud LB endpoint → [hidden LB instances] → your apps
```
- AWS ELB/ALB, GCP Load Balancer, Azure LB
- You see one DNS name/IP
- Cloud provider runs many LB instances behind the scenes
- Automatically scales and distributes traffic

#### 5. Kubernetes Service (Our Project!)
```
External traffic → NodePort Service (kube-proxy load balances)
  ↓
Your LB pods (3 replicas, running your Go code)
  ↓ (your code does round-robin)
Backend pods (3 replicas)
```

**In our architecture:**
- Load balancer runs as a Deployment (multiple replicas)
- Kubernetes Service sits in front of LB pods
- kube-proxy on each node load balances to LB pods using iptables/IPVS
- **Kubernetes is load-balancing your load balancers!**
- Your LB pods discover and load-balance to backend pods

**Complete flow:**
```
curl localhost:30081 (NodePort)
  ↓
kube-proxy (on node, load balances to LB pods)
  ↓
LB Pod 1, 2, or 3 (our Go code selects backend)
  ↓
Backend Pod A, B, or C
```

---

## Multiple LB Instances - State Coordination

### Do LB Instances Need to Coordinate?

**Short answer: No!** Each LB instance maintains its own state.

### Round-Robin Without Coordination

Each LB instance has its own counter in memory. They don't talk to each other.

**Example:**
```
Request 1 → LB1 (counter=0) → Backend A
Request 2 → LB2 (counter=0) → Backend A
Request 3 → LB1 (counter=1) → Backend B
Request 4 → LB2 (counter=1) → Backend B
Request 5 → LB1 (counter=2) → Backend C
Request 6 → LB2 (counter=2) → Backend C
```

Even though both LBs start at 0, distribution evens out because:
- Requests are distributed across LB instances by Kubernetes Service
- Each LB does round-robin independently
- Over many requests (hundreds/thousands), it balances out
- Exact distribution per request doesn't matter - average distribution does

### Least Connections Without Coordination

Each LB tracks connections **it** established:
- LB1 knows: "I have 5 connections to Backend A, 3 to Backend B"
- LB2 knows: "I have 4 connections to Backend A, 2 to Backend B"
- Neither knows about the other's connections

**This still works reasonably well:**
- Each LB makes locally optimal decisions
- Coarse-grained balancing is achieved
- Perfect precision isn't necessary

### When Would You Need Coordination?

**Using shared state (Redis, etcd, database):**
- **Pros:** Precise load balancing across all LB instances
- **Cons:**
  - Adds latency to every request (network call to Redis)
  - Single point of failure (what if Redis is down?)
  - Much more complex

**Real-world practice:**
- Most load balancers (HAProxy, Nginx, Envoy) don't coordinate
- The added complexity and latency isn't worth it
- Independent operation is more reliable

**Our approach:**
- Start with 1 LB instance
- Scale to 3 instances
- No shared state needed
- Let Kubernetes distribute requests to LB instances

---

## Rolling Restarts - The Race Condition

### The Problem

When Kubernetes does a rolling restart of your backend pods, there's a race condition.

**Kubernetes rolling restart flow:**
```
1. K8s starts new backend pod
2. New pod becomes ready (readiness probe passes)
3. K8s adds new pod to Endpoints object
4. K8s sends SIGTERM to old pod
5. Old pod has 30 seconds (terminationGracePeriodSeconds) to finish
6. Old pod removed from Endpoints
7. K8s force-kills pod if still running after grace period
```

**The race condition:**
```
Your LB discovers new backend at step 3 ✅
Your LB might try to send request to old backend
  → Right as K8s is terminating it (steps 4-6) ❌
  → Connection refused or reset
  → Client sees error!
```

### How Kubernetes Helps

#### 1. Readiness Probes
```yaml
readinessProbe:
  httpGet:
    path: /status
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 5
```
- Pod not added to Endpoints until probe succeeds
- Prevents sending traffic to pods that aren't ready yet

#### 2. Termination Grace Period
```yaml
terminationGracePeriodSeconds: 30
```
- Pod gets SIGTERM signal
- Has 30 seconds to finish existing requests
- Should stop accepting new requests
- After 30s, K8s sends SIGKILL (force kill)

#### 3. PreStop Hook (Optional but Helpful)
```yaml
lifecycle:
  preStop:
    exec:
      command: ["/bin/sh", "-c", "sleep 5"]
```
- Delays shutdown by 5 seconds
- Gives time for Endpoints update to propagate to all watchers
- Prevents new connections during drain

**Our backend already has readiness probes configured!**

### How Your LB Should Handle It

**Phase 2a (simple approach):**
- If connection to backend fails → return 503 to client
- User sees error, retries manually
- Not ideal, but simple to implement and understand

**Phase 2c (better approach - retry logic):**
```go
func (lb *LoadBalancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    var lastErr error
    maxRetries := 3

    for attempt := 0; attempt < maxRetries; attempt++ {
        backend := lb.selectBackend()
        if backend == "" {
            http.Error(w, "No backends available", 503)
            return
        }

        // Try to proxy the request
        err := lb.proxyToBackend(w, r, backend)
        if err == nil {
            return // Success!
        }

        lastErr = err
        log.Printf("Backend %s failed (attempt %d): %v", backend, attempt+1, err)
        // Try next backend...
    }

    http.Error(w, fmt.Sprintf("All backends failed: %v", lastErr), 503)
}
```

**Important:** Only retry safe methods (GET, HEAD). Don't retry POST/PUT to avoid duplicate operations.

### Advanced Solutions (Phase 3)

**1. Active Health Checks**
- LB actively probes backends (`/health` endpoint)
- Removes unhealthy backends before trying to send traffic
- Catches failures proactively

**2. Circuit Breaker**
- After N failures to a backend, stop trying for X seconds
- "Open circuit" = backend is considered dead
- Auto-retry later to see if recovered
- Prevents wasting time on dead backends

**3. Connection Draining**
- LB tracks which backends are in "draining" state
- Stops sending NEW requests to draining backends
- Waits for existing requests to complete
- Requires watching Endpoints for deletion events

**4. Graceful Shutdown in Backend**
- Backend handles SIGTERM signal
- Stops accepting new connections
- Finishes in-flight requests
- Exits cleanly within grace period

---

## Connection Tracking in Go

### Using `defer` for Cleanup

Go's `defer` is like Python's `finally` - it always runs when a function exits.

**Tracking active connections:**
```go
type LoadBalancer struct {
    activeConnections map[string]int  // backend IP -> count
    mu                sync.Mutex       // protects the map
}

func (lb *LoadBalancer) trackConnection(backend string) func() {
    // Increment counter
    lb.mu.Lock()
    lb.activeConnections[backend]++
    count := lb.activeConnections[backend]
    lb.mu.Unlock()

    log.Printf("Connection opened to %s (active: %d)", backend, count)

    // Return cleanup function
    return func() {
        lb.mu.Lock()
        lb.activeConnections[backend]--
        count := lb.activeConnections[backend]
        lb.mu.Unlock()
        log.Printf("Connection closed to %s (active: %d)", backend, count)
    }
}

func (lb *LoadBalancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    backend := lb.selectBackend()

    cleanup := lb.trackConnection(backend)
    defer cleanup()  // Always runs when function exits

    // Proxy the request...
    lb.proxyToBackend(w, r, backend)

    // cleanup() runs here automatically, even if proxy panics
}
```

**How `defer` works:**
- Schedules function to run when surrounding function returns
- Runs even if panic occurs
- Multiple defers run in LIFO order (last deferred runs first)
- Perfect for cleanup operations

### Data Structures for Connection Management

**For connection counts (least connections algorithm):**
```go
map[string]int  // backend IP -> active connection count
```

Simple and sufficient. Protected by mutex.

**For tracking individual connections (if needed):**
```go
type Connection struct {
    ID        string
    Backend   string
    StartTime time.Time
}

activeConns := make(map[string]*Connection)  // connection ID -> Connection
```

Generate unique connection ID:
```go
connID := fmt.Sprintf("%s-%d", r.RemoteAddr, time.Now().UnixNano())
```

**For most use cases, just tracking counts is enough.**

---

## Graceful Shutdown

### The Problem

When Kubernetes wants to terminate your load balancer pod:
```
Client → (long-lived connection) → LB Pod → Backend Pod
```

What happens to the active connection?

### Graceful Shutdown Implementation

**Handle signals and shutdown cleanly:**
```go
package main

import (
    "context"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"
)

func main() {
    handler := // your LB handler
    server := &http.Server{
        Addr:    ":8080",
        Handler: handler,
    }

    // Start server in background goroutine
    go func() {
        log.Println("Starting server on :8080")
        if err := server.ListenAndServe(); err != http.ErrServerClosed {
            log.Fatalf("Server error: %v", err)
        }
    }()

    // Set up signal handling
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

    // Block until signal received
    sig := <-quit
    log.Printf("Received signal: %v. Shutting down gracefully...", sig)

    // Give existing requests time to finish
    // Use 25s timeout (less than K8s terminationGracePeriodSeconds of 30s)
    ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
    defer cancel()

    // Shutdown stops accepting new connections and waits for existing ones
    if err := server.Shutdown(ctx); err != nil {
        log.Printf("Server forced to shutdown: %v", err)
    }

    log.Println("Server stopped cleanly")
}
```

**What `server.Shutdown(ctx)` does:**
1. Stops listening for new connections
2. Closes all idle connections (keep-alive connections with no active request)
3. Waits for active connections to complete
4. Returns when all connections finished OR ctx deadline exceeded

**Kubernetes termination flow with graceful shutdown:**
```
1. K8s removes pod from Service endpoints (no new traffic)
2. K8s sends SIGTERM to pod
3. Your code catches SIGTERM
4. server.Shutdown() is called
5. New connections rejected
6. Existing connections allowed to finish (up to 25s)
7. After 30s total, K8s sends SIGKILL (force kill)
```

### Configuration

**In your deployment:**
```yaml
spec:
  terminationGracePeriodSeconds: 30  # Default, can increase
  containers:
  - name: loadbalancer
    # Your container config
```

**Recommendation:**
- Set shutdown timeout to 5s less than terminationGracePeriodSeconds
- Ensures your code finishes before K8s force-kills

---

## The Long Connection Problem

### Fundamental Issue

**Kubernetes doesn't know when it's safe to kill a pod.**

**K8s approach:**
```
1. Send SIGTERM
2. Wait terminationGracePeriodSeconds (30s default)
3. If still running, send SIGKILL (force kill)
4. Active connections get dropped
```

**For long-lived connections (streaming video, large file downloads, WebSockets):**
- If connection takes > 30 seconds, it WILL be killed mid-stream
- Client sees broken connection (connection reset, EOF, etc.)
- Client must detect and retry

### Real-World Example: Video Streaming

**What happens during a rolling restart:**
```
Client streaming video → LB Pod → Backend Pod
K8s decides to do rolling update of LB pods
```

1. K8s removes LB pod from Service endpoints (new requests go elsewhere)
2. K8s sends SIGTERM to LB pod
3. LB pod keeps existing streaming connection alive
4. After 30 seconds, K8s force-kills LB pod
5. TCP connection drops
6. Client video player detects error/buffering
7. Client automatically reconnects
8. Gets routed to different LB pod
9. Video resumes (might see brief buffering or quality drop)

**This is normal!** Even major services (Netflix, YouTube, Twitch) have this behavior.

### Solutions and Trade-offs

#### 1. Increase Termination Grace Period
```yaml
terminationGracePeriodSeconds: 300  # 5 minutes
```
- **Pros:** More time for long requests to finish
- **Cons:**
  - Delays pod termination
  - Slows down deployments
  - Pods consume resources while draining

#### 2. Sticky Sessions / Session Affinity
```yaml
service:
  sessionAffinity: ClientIP
  sessionAffinityConfig:
    clientIP:
      timeoutSeconds: 10800  # 3 hours
```
- **Pros:** Same client goes to same backend
- **Cons:**
  - When that backend dies, connection breaks anyway
  - Uneven load distribution
  - Doesn't solve the fundamental problem

#### 3. Application-Level Retry (Recommended)
- Client detects connection loss
- Automatically reconnects
- Picks up where it left off (chunked transfer, resumable uploads)
- All modern video/file clients do this

#### 4. Accept Failure for Short Requests
- Most HTTP requests complete in < 1 second
- 30 second grace period is plenty
- Long-lived connections (hours) are the exception
- Design for them to break and clients to handle it

#### 5. Blue/Green or Canary Deployments
- Deploy new version alongside old version
- Gradually shift traffic to new version
- Keep old version running until all connections finished
- More complex but handles long connections better
- Used by large-scale services

### What We're Building

**Phase 2a:**
- No graceful shutdown (K8s just kills after 30s)
- Fine for learning and short requests

**Phase 2b/2c:**
- Add `server.Shutdown()` with signal handling
- Set timeout to 25s (less than termination grace period)
- Log active connections on shutdown
- Add retry logic for failed requests

**Reality Check:**
- 99% of HTTP requests finish in < 1 second
- 30 second grace period handles most cases fine
- Long-lived connections (hours) must be designed to handle reconnection
- This is an accepted trade-off in distributed systems

---

## Summary

**Key Takeaways:**

1. **Streaming connections:** LBs are just pipes - data streams through without buffering
2. **Multiple LBs:** Independent state is fine - don't need coordination for most use cases
3. **Rolling restarts:** Race conditions happen - use retries and health checks
4. **Connection tracking:** Use `defer` in Go for guaranteed cleanup
5. **Graceful shutdown:** Handle SIGTERM, use `server.Shutdown()`, set appropriate timeouts
6. **Long connections:** Will break during restarts - design clients to handle it

**Architecture Recap:**
```
Internet → Kubernetes NodePort → kube-proxy
  ↓
Your LB Pods (3 replicas, independent state)
  ↓ (round-robin selection in your Go code)
Your Backend Pods (3 replicas)
```

**Implementation Priority:**
1. Get it working (Phase 2a)
2. Add resilience (retries, health checks)
3. Add graceful shutdown
4. Add observability (metrics, logging)
5. Add advanced features (circuit breaker, multiple strategies)

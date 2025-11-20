package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync/atomic"
	"time"
)

type StatusResponse struct {
	PodName     string `json:"podname"`
	PodIP       string `json:"podip"`
	ServiceName string `json:"servicename"`
	StartTime   string `json:"starttime"`
}

type PingResponse struct {
	ServiceName string `json:"servicename"`
	Timestamp   string `json:"timestamp"`
	Count       int64  `json:"count"`
}

type ServiceHandler struct {
	ServiceName string
	StartTime   string
	Count       int64
}

func NewServiceHandler(serviceName string, startTime string) *ServiceHandler {
	return &ServiceHandler{
		ServiceName: serviceName,
		StartTime:   startTime,
	}
}

func (s ServiceHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/status", s.status)
	mux.HandleFunc("/ping", s.ping)
}

func (s *ServiceHandler) status(w http.ResponseWriter, r *http.Request) {
	podname, podok := os.LookupEnv("POD_NAME")
	if !podok {
		hostname, err := os.Hostname()
		if err != nil {
			hostname = "unknown"
		}
		podname = hostname
	}

	podip, ipok := os.LookupEnv("POD_IP")
	if !ipok {
		podip = "127.0.0.1"
	}

	response := StatusResponse{
		PodName:     podname,
		PodIP:       podip,
		ServiceName: s.ServiceName,
		StartTime:   s.StartTime,
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *ServiceHandler) ping(w http.ResponseWriter, r *http.Request) {
	count := atomic.AddInt64(&s.Count, 1)
	response := PingResponse{
		ServiceName: s.ServiceName,
		Timestamp:   time.Now().Format(time.RFC3339),
		Count:       count,
	}
	writeJSON(w, http.StatusOK, response)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	// Endcoding errors are uncommon so we just log the error. I bet they never happen
	// with this code
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("[ERROR] Failed to encode respoonse json: %v", err)
	}
}

/*
DESIGN APPROACH: Handler Type with Dependency Injection (Approach B)

OVERVIEW:
- Create a Handler struct that holds configuration/state
- Use a New() constructor to initialize the handler with dependencies
- Handler methods receive (h *Handler) to access config and state
- Counter uses atomic operations for thread-safety (no mutex needed for simple int64)
- Generate timestamp fresh on each request
- Get pod info from environment variables (POD_NAME, POD_IP) with fallbacks

COMPONENTS TO IMPLEMENT:

1. Handler Struct
   - serviceName string (injected via New())
   - count int64 (for atomic operations)

2. New() Constructor
   - Takes serviceName as parameter
   - Returns *Handler
   - Initializes struct with provided config

3. StatusResponse Struct
   - PodName string (from env or hostname fallback)
   - PodIP string (from env or "127.0.0.1" fallback)
   - ServiceName string (from Handler config)
   - Timestamp string (time.Now().Format(time.RFC3339))
   - Count int64 (atomic counter)
   - All fields need `json:"fieldName"` tags

4. Status Method (h *Handler)
   - Signature: func (h *Handler) Status(w http.ResponseWriter, r *http.Request)
   - Increment counter using atomic.AddInt64(&h.count, 1)
   - Read POD_NAME from os.Getenv(), fallback to os.Hostname()
   - Read POD_IP from os.Getenv(), fallback to "127.0.0.1"
   - Create StatusResponse with current timestamp
   - Call writeJSON() helper to send response

5. writeJSON Helper Function
   - Signature: func writeJSON(w http.ResponseWriter, status int, data interface{})
   - Set Content-Type header to "application/json"
   - Write HTTP status code with w.WriteHeader(status)
   - Encode data to JSON using json.NewEncoder(w).Encode(data)
   - Should handle encoding errors (log them)

PACKAGES NEEDED:
- encoding/json (for JSON encoding)
- net/http (for HTTP handler types)
- os (for environment variables and hostname)
- sync/atomic (for thread-safe counter - atomic.AddInt64)
- time (for timestamps - time.Now(), time.RFC3339)

USAGE IN MAIN:
- handler := handlers.New(serviceName)
- http.HandleFunc("/", handler.Status)
- http.ListenAndServe(":8080", nil)

WHY THIS APPROACH:
- Clean dependency injection (serviceName passed in, not hardcoded)
- Easy to test (can create handler with test config)
- No package-level state (everything in Handler struct)
- Thread-safe counter with atomic operations
- Flexible - can add more methods to Handler later
*/

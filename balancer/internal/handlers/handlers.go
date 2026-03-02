package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"sync"
	"time"

	"balancer/internal/discovery"
	"balancer/internal/strategy"
)

type StatusResponse struct {
	PodName            string `json:"podname"`
	PodIP              string `json:"podip"`
	BackendName        string `json:"servicename"`
	BackendPort        int    `json:"backendport"`
	LoadbalancerPort   int    `json:"loadbalancerport"`
	LoadbalancerMethod string `json:"loadbalancermethod"`
	ConnectedHosts     int    `json:"connectedhosts"`
	StartTime          string `json:"starttime"`
}

type NextResponse struct {
	NextHost string `json:"nexthost"`
}

type BalanceHandler struct {
	BackendName        string
	BackendPort        int
	LoadbalancerPort   int
	LoadbalancerMethod string
	StartTime          string
	Backends           *discovery.BackendList
	Requests           int
	Strategy           strategy.Strategy
	Proxy              *httputil.ReverseProxy
	mu                 sync.RWMutex
}

func NewBalanceHandler(
	backendName string,
	backendPort int,
	loadbalancerPort int,
	loadbalancerMethod string,
	backends *discovery.BackendList,
) *BalanceHandler {
	bh := &BalanceHandler{
		BackendName:        backendName,
		BackendPort:        backendPort,
		LoadbalancerPort:   loadbalancerPort,
		LoadbalancerMethod: loadbalancerMethod,
		StartTime:          time.Now().Format(time.RFC3339),
		Backends:           backends,
		Strategy:           strategy.NewStrategy(loadbalancerMethod),
	}
	bh.createProxy()
	return bh
}

func (bh *BalanceHandler) selectBackend(requests int) discovery.Backend {
	return bh.Strategy.Next(bh.Backends.GetAll(), requests)
}

func (bh BalanceHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/status", bh.status)
	mux.HandleFunc("/next-backend", bh.nextBackend)
	mux.Handle("/", bh.Proxy)
}

func getPodName() string {
	podname, ok := os.LookupEnv("POD_NAME")
	if !ok {
		hostname, err := os.Hostname()
		if err != nil {
			hostname = "unknown"
		}
		podname = hostname
	}
	return podname
}

func (bh *BalanceHandler) nextBackend(w http.ResponseWriter, r *http.Request) {
	bh.mu.RLock()
	requests := bh.Requests
	bh.mu.RUnlock()

	backend := bh.selectBackend(requests)

	next := NextResponse{
		NextHost: fmt.Sprintf("%s:%d", backend.Address, bh.BackendPort),
	}
	writeJSON(w, http.StatusOK, next)
}

func (bh *BalanceHandler) createProxy() {
	bh.Proxy = &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			bh.mu.Lock()
			bh.Requests++
			requests := bh.Requests
			bh.mu.Unlock()
			backend := bh.selectBackend(requests)
			host := fmt.Sprintf("%s:%d", backend.Address, bh.BackendPort)
			url, err := url.Parse(fmt.Sprintf("http://%s", host))
			if err != nil {
				log.Printf("[ERROR] Failed to parse the url from %v", host)
			}
			pr.SetURL(url)
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			log.Printf("Proxy error: %v", err)
			w.WriteHeader(http.StatusBadGateway)
		},
	}
}

func (bh *BalanceHandler) status(w http.ResponseWriter, r *http.Request) {
	podname := getPodName()

	podip, ok := os.LookupEnv("POD_IP")
	if !ok {
		podip = "127.0.0.1"
	}

	response := StatusResponse{
		PodName:            podname,
		PodIP:              podip,
		BackendName:        bh.BackendName,
		BackendPort:        bh.BackendPort,
		LoadbalancerPort:   bh.LoadbalancerPort,
		LoadbalancerMethod: bh.LoadbalancerMethod,
		StartTime:          bh.StartTime,
		ConnectedHosts:     len(bh.Backends.GetAll()),
	}
	writeJSON(w, http.StatusOK, response)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	// Encoding errors are uncommon so we just log the error. I bet they never happen
	// with this code
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("[ERROR] Failed to encode response json: %v", err)
	}
}

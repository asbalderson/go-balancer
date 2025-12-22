package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"
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

type BalanceHandler struct {
	BackendName        string
	BackendPort        int
	LoadbalancerPort   int
	LoadbalancerMethod string
	StartTime          string
}

func NewBalanceHandler(backendName string, backendPort int, loadbalancerPort int, loadbalancerMethod string) *BalanceHandler {
	return &BalanceHandler{
		BackendName:        backendName,
		BackendPort:        backendPort,
		LoadbalancerPort:   loadbalancerPort,
		LoadbalancerMethod: loadbalancerMethod,
		StartTime:          time.Now().Format(time.RFC3339),
	}
}

func (s BalanceHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/status", s.status)
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

func (s *BalanceHandler) status(w http.ResponseWriter, r *http.Request) {
	podname := getPodName()

	podip, ok := os.LookupEnv("POD_IP")
	if !ok {
		podip = "127.0.0.1"
	}

	response := StatusResponse{
		PodName:            podname,
		PodIP:              podip,
		BackendName:        s.BackendName,
		BackendPort:        s.BackendPort,
		LoadbalancerPort:   s.LoadbalancerPort,
		LoadbalancerMethod: s.LoadbalancerMethod,
		StartTime:          s.StartTime,
		ConnectedHosts:     0,
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

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

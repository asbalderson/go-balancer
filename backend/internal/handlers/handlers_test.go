package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStatus_NoEnv(t *testing.T) {
	handler := NewServiceHandler("test")

	req := httptest.NewRequest("GET", "/status", nil)
	rr := httptest.NewRecorder()

	handler.status(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected ok status, got %d", rr.Code)
	}
	var response StatusResponse
	json.Unmarshal(rr.Body.Bytes(), &response)
	if response.PodIP != "127.0.0.1" {
		t.Errorf("podIP should be default, was %s", response.PodIP)
	}
	if response.ServiceName != "test" {
		t.Errorf("expected service name 'test' but got %s", response.ServiceName)
	}
	if response.StartTime == "" {
		t.Errorf("got empty timestamp from response")
	}
}

func TestStatus_Env(t *testing.T) {
	os.Setenv("POD_NAME", "env_name")
	os.Setenv("POD_IP", "1.2.3.4")
	t.Cleanup(func() {
		os.Unsetenv("POD_NAME")
		os.Unsetenv("POD_IP")
	})
	handler := NewServiceHandler("test")

	req := httptest.NewRequest("GET", "/status", nil)
	rr := httptest.NewRecorder()

	handler.status(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected ok status, got %d", rr.Code)
	}
	var response StatusResponse
	json.Unmarshal(rr.Body.Bytes(), &response)
	if response.PodIP != "1.2.3.4" {
		t.Errorf("podIP should be default, was %s", response.PodIP)
	}
	if response.PodName != "env_name" {
		t.Errorf("podName should be default, was %s", response.PodName)
	}
}

func TestStatus_counter(t *testing.T) {
	handler := NewServiceHandler("test")

	req := httptest.NewRequest("GET", "/status", nil)
	rr := httptest.NewRecorder()

	handler.status(rr, req)

	assert.Equal(t, int64(0), handler.Count)
}

func TestPing(t *testing.T) {
	handler := NewServiceHandler("test")

	req := httptest.NewRequest("GET", "/ping", nil)
	rr := httptest.NewRecorder()

	for i := int64(1); i < 10; i++ {
		handler.ping(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected ok status, got %d", rr.Code)
		}
		assert.Equal(t, i, handler.Count)
	}
}

func TestWriteJSON(t *testing.T) {
	rr := httptest.NewRecorder()
	data := struct {
		Foo string `json:foo`
	}{
		Foo: "bar",
	}
	writeJSON(rr, 169, data)

	assert.Equal(t, 169, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
}

package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"
)

// TestHealthCheckHandler tests the health check handler
func TestHealthCheckHandler(t *testing.T) {
	// Set lastPingSuccess to simulate a recent successful ping
	atomic.StoreInt64(&lastPingSuccess, time.Now().Unix())

	// Create a request to pass to our handler
	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create a ResponseRecorder to record the response
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(healthCheckHandler)

	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check the response body is what we expect
	expected := "Ping-Pong Server is healthy\n"
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
	}
}

// TestHealthCheckHandlerNoPing tests the health check handler when no ping has occurred
func TestHealthCheckHandlerNoPing(t *testing.T) {
	// Reset lastPingSuccess to simulate no successful ping
	atomic.StoreInt64(&lastPingSuccess, 0)

	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(healthCheckHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusServiceUnavailable {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusServiceUnavailable)
	}
}

// TestPingServer tests the pingServer function
func TestPingServer(t *testing.T) {
	// Create a test server that checks for a custom header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if the custom header is present
		if r.Header.Get("X-Custom-Header") != "test-value" {
			t.Errorf("Expected custom header 'X-Custom-Header' with value 'test-value', got: %v", r.Header.Get("X-Custom-Header"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := Config{
		ServerURL:    server.URL,
		PingInterval: 1 * time.Second,
		OwnURL:       "http://localhost:8080/health",
		Headers: map[string]string{
			"X-Custom-Header": "test-value",
		},
	}

	// Call pingServer
	pingServer(config)

	// Verify lastPingSuccess was updated
	lastPing := atomic.LoadInt64(&lastPingSuccess)
	if lastPing == 0 {
		t.Error("lastPingSuccess was not updated after successful ping")
	}
}

// TestIntegration tests a basic integration scenario
func TestIntegration(t *testing.T) {
	// Create a test server for the target server
	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer targetServer.Close()

	// Create a test server for the own health endpoint
	ownHealthServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ownHealthServer.Close()

	// Set environment variables for the test
	os.Setenv("SERVER_URL", targetServer.URL)
	os.Setenv("OWN_URL", ownHealthServer.URL)
	os.Setenv("PING_INTERVAL", "1000")

	// Call pingServer
	config := Config{
		ServerURL:    targetServer.URL,
		PingInterval: 1 * time.Second,
		OwnURL:       ownHealthServer.URL,
	}
	pingServer(config)

	// Verify lastPingSuccess was updated
	lastPing := atomic.LoadInt64(&lastPingSuccess)
	if lastPing == 0 {
		t.Error("lastPingSuccess was not updated after successful ping")
	}
}

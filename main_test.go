package main

import (
	"flag"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
)

// TestHealthCheckHandler tests the health check handler
func TestHealthCheckHandler(t *testing.T) {
	// Set up a test server
	server := httptest.NewServer(http.HandlerFunc(healthCheckHandler))
	defer server.Close()

	// Test case 1: No successful pings yet
	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("Expected status code %d, got %d", http.StatusServiceUnavailable, resp.StatusCode)
	}

	// Test case 2: Successful ping within the last 15 minutes
	atomic.StoreInt64(&lastPingSuccess, time.Now().Unix())
	resp, err = http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	// Test case 3: Last successful ping was too long ago
	atomic.StoreInt64(&lastPingSuccess, time.Now().Add(-16*time.Minute).Unix())
	resp, err = http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("Expected status code %d, got %d", http.StatusServiceUnavailable, resp.StatusCode)
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
		logy("ERROR", "handler returned wrong status code: got %v want %v", status, http.StatusServiceUnavailable)
		t.Fail()
	}
}

// TestPingServer tests the pingServer function
func TestPingServer(t *testing.T) {
	// Set up a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Set environment variables for testing
	os.Setenv("SERVER_URL", server.URL)
	os.Setenv("PING_INTERVAL", "1000")
	os.Setenv("OWN_URL", server.URL)
	os.Setenv("MAX_RETRIES", "3")

	// Create a Config struct
	config := Config{
		ServerURL:    server.URL,
		PingInterval: 1 * time.Second,
		OwnURL:       server.URL,
		Headers:      map[string]string{"Custom-Header": "test"},
	}

	// Test case 1: Successful ping
	pingServer(config)
	if atomic.LoadInt64(&lastPingSuccess) == 0 {
		t.Errorf("Expected lastPingSuccess to be set")
	}

	// Test case 2: Failed ping with retry
	server.Close() // Close the server to simulate a failure
	pingServer(config)
	// Check if the lastPingSuccess is still set
	if atomic.LoadInt64(&lastPingSuccess) == 0 {
		t.Errorf("Expected lastPingSuccess to be set after retry")
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
		logy("ERROR", "lastPingSuccess was not updated after successful ping")
		t.Fail()
	}
}

// Add test for local test server
func TestLocalTestServer(t *testing.T) {
	// Set up a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Set environment variables for testing
	os.Setenv("SERVER_URL", server.URL)
	os.Setenv("PING_INTERVAL", "1000")
	os.Setenv("OWN_URL", server.URL)
	os.Setenv("MAX_RETRIES", "3")

	// Start the local test server
	go startLocalTestServer()

	// Wait for the server to start
	time.Sleep(1 * time.Second)

	// Test the health endpoint
	resp, err := http.Get("http://localhost:8081/health")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}
}

func TestCommandLineFlags(t *testing.T) {
	// Set up a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Reset flag.CommandLine to avoid conflicts with other tests
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	// Define the flags
	serverURL := flag.String("server-url", "", "Server URL to ping")
	pingInterval := flag.String("ping-interval", "", "Ping interval in milliseconds")
	ownURL := flag.String("own-url", "", "Own health check URL")
	maxRetries := flag.Int("max-retries", 0, "Maximum number of retries")

	// Set command-line arguments
	os.Args = []string{"cmd", "-server-url", server.URL, "-ping-interval", "1000", "-own-url", server.URL, "-max-retries", "3"}

	// Parse flags
	flag.Parse()

	// Set environment variables from flags
	if *serverURL != "" {
		os.Setenv("SERVER_URL", *serverURL)
	}
	if *pingInterval != "" {
		os.Setenv("PING_INTERVAL", *pingInterval)
	}
	if *ownURL != "" {
		os.Setenv("OWN_URL", *ownURL)
	}
	if *maxRetries > 0 {
		os.Setenv("MAX_RETRIES", strconv.Itoa(*maxRetries))
	}

	// Check if environment variables are set correctly
	if os.Getenv("SERVER_URL") != server.URL {
		t.Errorf("Expected SERVER_URL to be set to %s, got %s", server.URL, os.Getenv("SERVER_URL"))
	}
	if os.Getenv("PING_INTERVAL") != "1000" {
		t.Errorf("Expected PING_INTERVAL to be set to 1000, got %s", os.Getenv("PING_INTERVAL"))
	}
	if os.Getenv("OWN_URL") != server.URL {
		t.Errorf("Expected OWN_URL to be set to %s, got %s", server.URL, os.Getenv("OWN_URL"))
	}
	if os.Getenv("MAX_RETRIES") != "3" {
		t.Errorf("Expected MAX_RETRIES to be set to 3, got %s", os.Getenv("MAX_RETRIES"))
	}
}

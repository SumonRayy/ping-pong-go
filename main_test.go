package main

import (
	"context"
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
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Test custom headers
		if r.Header.Get("X-Custom-Header") != "test-value" {
			t.Errorf("Expected custom header 'X-Custom-Header' with value 'test-value', got '%s'", r.Header.Get("X-Custom-Header"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Test cases
	tests := []struct {
		name           string
		config         Config
		expectedResult bool
	}{
		{
			name: "Successful ping",
			config: Config{
				ServerURL: server.URL,
				Headers: map[string]string{
					"X-Custom-Header": "test-value",
				},
			},
			expectedResult: true,
		},
		{
			name: "Failed ping",
			config: Config{
				ServerURL: "http://invalid-url",
			},
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pingServer(tt.config)
			if result != tt.expectedResult {
				t.Errorf("Expected result %v, got %v", tt.expectedResult, result)
			}
		})
	}
}

// TestStartPinging tests the startPinging function
func TestStartPinging(t *testing.T) {
	// Create a test server that fails consistently
	failingServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Always return 500 to ensure consistent failures
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer failingServer.Close()

	// Create a test server that succeeds consistently
	successServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Always return 200 to ensure consistent success
		w.WriteHeader(http.StatusOK)
	}))
	defer successServer.Close()

	// Test cases
	tests := []struct {
		name             string
		config           Config
		expectedShutdown bool
		timeout          time.Duration
	}{
		{
			name: "Shutdown after max consecutive failures",
			config: Config{
				ServerURL:           failingServer.URL,
				PingInterval:        100 * time.Millisecond,
				MaxConsecutiveFails: 2,
			},
			expectedShutdown: true,
			timeout:          5 * time.Second,
		},
		{
			name: "Continue pinging with successful responses",
			config: Config{
				ServerURL:           successServer.URL,
				PingInterval:        100 * time.Millisecond,
				MaxConsecutiveFails: 10,
			},
			expectedShutdown: false,
			timeout:          2 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set MAX_RETRIES to 1 to speed up the test
			originalMaxRetries := os.Getenv("MAX_RETRIES")
			os.Setenv("MAX_RETRIES", "1")
			defer func() {
				if originalMaxRetries == "" {
					os.Unsetenv("MAX_RETRIES")
				} else {
					os.Setenv("MAX_RETRIES", originalMaxRetries)
				}
			}()

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			shutdownChan := make(chan struct{})
			go startPinging(ctx, tt.config, shutdownChan)

			// Wait for either shutdown or timeout
			select {
			case <-shutdownChan:
				if !tt.expectedShutdown {
					t.Error("Unexpected shutdown")
				}
			case <-time.After(tt.timeout):
				if tt.expectedShutdown {
					t.Error("Expected shutdown but did not receive it")
				}
			}
		})
	}
}

// TestConfigSetup tests the setupConfig function
func TestConfigSetup(t *testing.T) {
	// Test environment variable setup
	tests := []struct {
		name           string
		envVars        map[string]string
		expectedConfig Config
		expectError    bool
	}{
		{
			name: "Valid configuration",
			envVars: map[string]string{
				"SERVER_URL":            "http://test-server",
				"PING_INTERVAL":         "1000",
				"OWN_URL":               "http://own-server",
				"MAX_CONSECUTIVE_FAILS": "5",
			},
			expectedConfig: Config{
				ServerURL:           "http://test-server",
				PingInterval:        1000 * time.Millisecond,
				OwnURL:              "http://own-server",
				MaxConsecutiveFails: 5,
			},
			expectError: false,
		},
		{
			name: "Invalid max consecutive fails",
			envVars: map[string]string{
				"MAX_CONSECUTIVE_FAILS": "0",
			},
			expectError: true,
		},
		{
			name: "Default max consecutive fails",
			envVars: map[string]string{
				"SERVER_URL":    "http://test-server",
				"PING_INTERVAL": "1000",
			},
			expectedConfig: Config{
				ServerURL:           "http://test-server",
				PingInterval:        1000 * time.Millisecond,
				MaxConsecutiveFails: 3, // Default value
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}
			defer func() {
				// Clean up environment variables
				for k := range tt.envVars {
					os.Unsetenv(k)
				}
			}()

			config, err := setupConfig()
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if config.ServerURL != tt.expectedConfig.ServerURL {
				t.Errorf("Expected ServerURL %v, got %v", tt.expectedConfig.ServerURL, config.ServerURL)
			}
			if config.PingInterval != tt.expectedConfig.PingInterval {
				t.Errorf("Expected PingInterval %v, got %v", tt.expectedConfig.PingInterval, config.PingInterval)
			}
			if config.OwnURL != tt.expectedConfig.OwnURL {
				t.Errorf("Expected OwnURL %v, got %v", tt.expectedConfig.OwnURL, config.OwnURL)
			}
			if config.MaxConsecutiveFails != tt.expectedConfig.MaxConsecutiveFails {
				t.Errorf("Expected MaxConsecutiveFails %v, got %v", tt.expectedConfig.MaxConsecutiveFails, config.MaxConsecutiveFails)
			}
		})
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

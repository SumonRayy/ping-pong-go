package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/fatih/color"
	"github.com/joho/godotenv"
)

type Config struct {
	ServerURL    string
	OwnURL       string
	PingInterval time.Duration
	Headers      map[string]string // Custom headers for ping requests
}

var (
	lastPingSuccess int64
)

// Add custom logger with timestamps and colors
func logy(level string, format string, args ...interface{}) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	message := fmt.Sprintf(format, args...)
	switch level {
	case "INFO":
		color.Green("[%s] INFO: %s", timestamp, message)
	case "ERROR":
		color.Red("[%s] ERROR: %s", timestamp, message)
	case "WARN":
		color.Yellow("[%s] WARN: %s", timestamp, message)
	default:
		color.White("[%s] %s: %s", timestamp, level, message)
	}
}

// Add environment variable validations
func validateEnv() error {
	serverURL := os.Getenv("SERVER_URL")
	ownURL := os.Getenv("OWN_URL")
	pingIntervalStr := os.Getenv("PING_INTERVAL")

	if serverURL == "" {
		return fmt.Errorf("missing required environment variable: SERVER_URL")
	}
	if ownURL == "" {
		return fmt.Errorf("missing required environment variable: OWN_URL")
	}
	if pingIntervalStr == "" {
		return fmt.Errorf("missing required environment variable: PING_INTERVAL")
	}

	pingInterval, err := strconv.Atoi(pingIntervalStr)
	if err != nil {
		return fmt.Errorf("invalid PING_INTERVAL: %v", err)
	}
	if pingInterval <= 0 {
		return fmt.Errorf("PING_INTERVAL must be greater than 0")
	}

	return nil
}

func main() {
	// Print a colorful banner
	color.Cyan("=========================================")
	color.Cyan("      Welcome to Ping-Pong Service       ")
	color.Cyan("=========================================")

	logy("INFO", "Starting Ping-Pong Server...")

	// Check if .env file exists
	if _, err := os.Stat(".env"); err == nil {
		// Read configuration from .env file
		err := godotenv.Load()
		if err != nil {
			logy("ERROR", "Error reading .env file: %v", err)
			os.Exit(1)
		}
	}

	// Validate environment variables
	if err := validateEnv(); err != nil {
		logy("ERROR", "Environment validation error: %v", err)
		os.Exit(1)
	}

	// Read configuration from environment variables
	serverURL := os.Getenv("SERVER_URL")
	ownURL := os.Getenv("OWN_URL")
	pingIntervalStr := os.Getenv("PING_INTERVAL")

	// Convert PING_INTERVAL to an integer
	pingInterval, err := strconv.Atoi(pingIntervalStr)
	if err != nil {
		logy("ERROR", "Error reading PING_INTERVAL environment variable: %v", err)
		os.Exit(1)
	}

	// Convert PING_INTERVAL to a time.Duration
	pingIntervalDuration := time.Duration(pingInterval) * time.Millisecond

	// Create a Config struct
	config := Config{
		ServerURL:    serverURL,
		PingInterval: pingIntervalDuration,
		OwnURL:       ownURL,
	}

	// Start the ping routine
	go startPinging(config)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Start the HTTP server for health checks
	http.HandleFunc("/health", healthCheckHandler)
	logy("INFO", "Health check endpoint available at /health")
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func startPinging(config Config) {
	ticker := time.NewTicker(config.PingInterval)
	defer ticker.Stop()

	for range ticker.C {
		pingServer(config)
	}
}

func pingServer(config Config) {
	logy("INFO", "Pinging server: %s", config.ServerURL)

	req, err := http.NewRequest("GET", config.ServerURL, nil)
	if err != nil {
		logy("ERROR", "Error creating request: %v", err)
		return
	}

	// Add custom headers if provided
	for key, value := range config.Headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logy("ERROR", "Error pinging server: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		atomic.StoreInt64(&lastPingSuccess, time.Now().Unix())
		logy("INFO", "Ping successful! to server : %s", config.ServerURL)

		// Call the server's own health check API
		callOwnHealthCheck(config.OwnURL)
	} else {
		logy("ERROR", "Ping failed with status code: %d", resp.StatusCode)
	}
}

func callOwnHealthCheck(ownURL string) {
	if ownURL == "" {
		logy("WARN", "OwnURL not provided, skipping health check call")
		return
	}

	logy("INFO", "Calling own health check endpoint: %s", ownURL)

	resp, err := http.Get(ownURL)
	if err != nil {
		logy("ERROR", "Error calling own health check: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		logy("INFO", "Own health check successful!")
	} else {
		logy("ERROR", "Own health check failed with status code: %d", resp.StatusCode)
	}
}

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	lastPing := atomic.LoadInt64(&lastPingSuccess)
	if lastPing == 0 {
		http.Error(w, "No successful pings yet", http.StatusServiceUnavailable)
		return
	}

	// Check if the last ping was within the last 15 minutes
	if time.Since(time.Unix(lastPing, 0)) > 15*time.Minute {
		http.Error(w, "Last successful ping was too long ago", http.StatusServiceUnavailable)
		return
	}

	fmt.Fprintln(w, "Ping-Pong Server is healthy")
}

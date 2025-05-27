package main

import (
	"flag"
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

	// Set default values if not provided
	if serverURL == "" {
		serverURL = "http://localhost:8081/health" // Default to local test server
	}
	if ownURL == "" {
		ownURL = "http://localhost:8080/health" // Default to own health endpoint
	}
	if pingIntervalStr == "" {
		pingIntervalStr = "2000" // Default to 2 seconds
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

func printBanner() {
	color.Cyan("=========================================")
	color.Cyan("      Welcome to Ping-Pong Service       ")
	color.Cyan("=========================================")
}

// Start local test server
func startLocalTestServer() {
	// Print a colorful banner
	printBanner()

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "Local test server is healthy")
	})

	logy("INFO", "Starting local test server on :8081")
	logy("INFO", "Local test server is ready to accept requests")
	log.Fatal(http.ListenAndServe(":8081", nil))
}

func main() {
	// Parse command line arguments
	useLocalServer := flag.Bool("local", false, "Use local test server")
	serverURL := flag.String("server-url", "", "Server URL to ping")
	pingInterval := flag.String("ping-interval", "", "Ping interval in milliseconds")
	ownURL := flag.String("own-url", "", "Own health check URL")
	maxRetries := flag.Int("max-retries", 0, "Maximum number of retries")
	flag.Parse()

	if *useLocalServer {
		startLocalTestServer()
		return
	}

	// Set environment variables from flags if provided
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

	// Print a colorful banner
	printBanner()

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
	serverURLEnv := os.Getenv("SERVER_URL")
	ownURLEnv := os.Getenv("OWN_URL")
	pingIntervalStr := os.Getenv("PING_INTERVAL")

	// Convert PING_INTERVAL to an integer
	pingIntervalInt, err := strconv.Atoi(pingIntervalStr)
	if err != nil {
		logy("ERROR", "Error reading PING_INTERVAL environment variable: %v", err)
		os.Exit(1)
	}

	// Convert PING_INTERVAL to a time.Duration
	pingIntervalDuration := time.Duration(pingIntervalInt) * time.Millisecond

	// Create a Config struct
	config := Config{
		ServerURL:    serverURLEnv,
		PingInterval: pingIntervalDuration,
		OwnURL:       ownURLEnv,
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

// Add retry mechanism for pingServer
func pingServer(config Config) {
	logy("INFO", "Pinging server: %s", config.ServerURL)

	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		req, err := http.NewRequest("GET", config.ServerURL, nil)
		if err != nil {
			logy("ERROR", "Error creating request: %v", err)
			continue
		}

		// Add custom headers if provided
		for key, value := range config.Headers {
			req.Header.Set(key, value)
		}

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			logy("ERROR", "Error pinging server: %v", err)
			if i < maxRetries-1 {
				logy("INFO", "Retrying...")
				time.Sleep(1 * time.Second)
				continue
			}
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			atomic.StoreInt64(&lastPingSuccess, time.Now().Unix())
			logy("INFO", "Ping successful! to server : %s", config.ServerURL)

			// Call the server's own health check API
			callOwnHealthCheck(config.OwnURL)
			return
		} else {
			logy("ERROR", "Ping failed with status code: %d", resp.StatusCode)
			if i < maxRetries-1 {
				logy("INFO", "Retrying...")
				time.Sleep(1 * time.Second)
				continue
			}
		}
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

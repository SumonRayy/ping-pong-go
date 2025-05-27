package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"
)

// Config struct to hold configuration settings
type Config struct {
	ServerURLs   []string
	OwnURL       string
	PingInterval time.Duration
}

// Global variable to store server statuses
var (
	serverStatuses sync.Map
)

// Main function
func main() {
	log.Println("Starting Ping-Pong Server...")

	// Check if .env file exists
	if _, err := os.Stat(".env"); err == nil {
		// Read configuration from .env file
		err := godotenv.Load()
		if err != nil {
			log.Fatalf("Error reading .env file: %v", err)
		}
	}

	// Read configuration from environment variables
	serverURLsStr := os.Getenv("SERVER_URLS")
	ownURL := os.Getenv("OWN_URL")
	pingIntervalStr := os.Getenv("PING_INTERVAL")

	if serverURLsStr == "" || pingIntervalStr == "" || ownURL == "" {
		log.Fatalf("Missing required environment variables: SERVER_URLS, PING_INTERVAL, OWN_URL")
	}

	// Parse server URLs
	serverURLs := parseServerURLs(serverURLsStr)

	// Convert PING_INTERVAL to a time.Duration
	pingInterval, err := strconv.Atoi(pingIntervalStr)
	if err != nil {
		log.Fatalf("Error reading PING_INTERVAL environment variable: %v", err)
	}

	config := Config{
		ServerURLs:   serverURLs,
		PingInterval: time.Duration(pingInterval) * time.Millisecond,
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
	log.Println("Health check endpoint available at /health")
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// Function to parse server URLs
func parseServerURLs(urlsStr string) []string {
	var urls []string
	for _, url := range strings.Split(urlsStr, ",") {
		trimmed := strings.TrimSpace(url)
		if trimmed != "" {
			urls = append(urls, trimmed)
		}
	}
	return urls
}

// Function to start the ping routine
func startPinging(config Config) {
	for _, serverURL := range config.ServerURLs {
		go pingServerPeriodically(serverURL, config.PingInterval, config.OwnURL)
	}
}

func pingServerPeriodically(serverURL string, interval time.Duration, ownURL string) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		pingServer(serverURL, ownURL)
	}
}

// Function to ping a server
func pingServer(serverURL, ownURL string) {
	log.Printf("Pinging server: %s\n", serverURL)

	resp, err := http.Get(serverURL)
	if err != nil {
		log.Printf("Error pinging server %s: %v\n", serverURL, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		serverStatuses.Store(serverURL, time.Now().Unix()) // Store the timestamp
		log.Printf("Ping successful for server: %s\n", serverURL)

		// Call the server's own health check API
		callOwnHealthCheck(ownURL)
	} else {
		log.Printf("Ping failed for server %s with status code: %d\n", serverURL, resp.StatusCode)
	}
}

// Function to call the server's own health check API
func callOwnHealthCheck(ownURL string) {
	if ownURL == "" {
		log.Println("OwnURL not provided, skipping health check call")
		return
	}

	log.Printf("Calling own health check endpoint: %s\n", ownURL)

	resp, err := http.Get(ownURL)
	if err != nil {
		log.Printf("Error calling own health check: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		log.Println("Own health check successful!")
	} else {
		log.Printf("Own health check failed with status code: %d\n", resp.StatusCode)
	}
}

// Function to handle health check requests
func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	statusReport := make(map[string]string)

	// Iterate over server statuses
	serverStatuses.Range(func(key, value interface{}) bool {
		serverURL := key.(string)
		lastPing := value.(int64)

		if time.Since(time.Unix(lastPing, 0)) > 15*time.Minute {
			statusReport[serverURL] = "unhealthy"
		} else {
			statusReport[serverURL] = "healthy"
		}

		return true
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(statusReport)
}

package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerURL    string
	OwnURL       string
	PingInterval time.Duration
}

var (
	lastPingSuccess int64
)

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
	serverURL := os.Getenv("SERVER_URL")
	ownURL := os.Getenv("OWN_URL")
	pingIntervalStr := os.Getenv("PING_INTERVAL")
	if serverURL == "" || pingIntervalStr == "" || ownURL == "" {
		log.Fatalf("Missing required environment variables: SERVER_URL, PING_INTERVAL, OWN_URL")
	}

	// Convert PING_INTERVAL to an integer
	pingInterval, err := strconv.Atoi(pingIntervalStr)
	if err != nil {
		log.Fatalf("Error reading PING_INTERVAL environment variable: %v", err)
	}

	// Convert PING_INTERVAL to a time.Duration
	pingIntervalDuration := time.Duration(pingInterval) * time.Second

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
	log.Println("Health check endpoint available at /health")
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
	log.Printf("Pinging server: %s\n", config.ServerURL)

	resp, err := http.Get(config.ServerURL)
	if err != nil {
		log.Printf("Error pinging server: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		atomic.StoreInt64(&lastPingSuccess, time.Now().Unix())
		log.Println("Ping successful! to server :", config.ServerURL)

		// Call the server's own health check API
		callOwnHealthCheck(config.OwnURL)
	} else {
		log.Printf("Ping failed with status code: %d\n", resp.StatusCode)
	}
}

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

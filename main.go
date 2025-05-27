// Written by [SumonRayy](https://sumonrayy.xyz)
// Follow me on Github: https://github.com/SumonRayy

// Package pingpong provides a health check service that implements a ping-pong mechanism
// between services. It can be used to monitor the health of distributed systems by
// having services ping each other at configurable intervals.
//
// The package provides functionality for:
// - Configurable ping intervals
// - Custom headers for ping requests
// - Health check endpoints
// - Consecutive failure tracking
// - Graceful shutdown on repeated failures
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/joho/godotenv"
)

type Config struct {
	ServerURL           string
	OwnURL              string
	PingInterval        time.Duration
	Headers             map[string]string // Custom headers for ping requests
	MaxConsecutiveFails int               // Maximum number of consecutive failures before shutdown
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
	color.Cyan("================================================")
	color.Cyan("      Welcome to Ping-Pong-Go Service           ")
	color.Cyan("================================================")
}

// Start local test server
func startLocalTestServer() {

	// Start local test server
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "Local test server is healthy")
	})

	logy("INFO", "Starting local test server on :8081")
	logy("INFO", "Local test server is ready to accept requests")
	log.Fatal(http.ListenAndServe(":8081", nil))
}

// setupConfig reads and validates configuration from environment variables and flags
func setupConfig() (Config, error) {
	// Read configuration from environment variables
	serverURLEnv := os.Getenv("SERVER_URL")
	ownURLEnv := os.Getenv("OWN_URL")
	pingIntervalStr := os.Getenv("PING_INTERVAL")
	maxConsecutiveFailsStr := os.Getenv("MAX_CONSECUTIVE_FAILS")

	// Set default for max consecutive failures if not provided
	if maxConsecutiveFailsStr == "" {
		maxConsecutiveFailsStr = "3" // Default to 3 consecutive failures
	}

	// Convert MAX_CONSECUTIVE_FAILS to an integer
	maxConsecutiveFails, err := strconv.Atoi(maxConsecutiveFailsStr)
	if err != nil {
		return Config{}, fmt.Errorf("error reading MAX_CONSECUTIVE_FAILS environment variable: %v", err)
	}

	// Validate max consecutive failures
	if maxConsecutiveFails <= 0 {
		return Config{}, fmt.Errorf("MAX_CONSECUTIVE_FAILS must be greater than 0")
	}

	// Convert PING_INTERVAL to an integer
	pingIntervalInt, err := strconv.Atoi(pingIntervalStr)
	if err != nil {
		return Config{}, fmt.Errorf("error reading PING_INTERVAL environment variable: %v", err)
	}

	// Convert PING_INTERVAL to a time.Duration
	pingIntervalDuration := time.Duration(pingIntervalInt) * time.Millisecond

	// Create a Config struct
	return Config{
		ServerURL:           serverURLEnv,
		PingInterval:        pingIntervalDuration,
		OwnURL:              ownURLEnv,
		MaxConsecutiveFails: maxConsecutiveFails,
	}, nil
}

// setupEnvironment loads environment variables from .env file if it exists
func setupEnvironment() error {
	if _, err := os.Stat(".env"); err == nil {
		// Read configuration from .env file
		err := godotenv.Load()
		if err != nil {
			return fmt.Errorf("error reading .env file: %v", err)
		}
	}
	return nil
}

// Flags represents all command line flags
type Flags struct {
	UseLocalServer      bool
	ServerURL           string
	PingInterval        string
	OwnURL              string
	MaxRetries          int
	MaxConsecutiveFails int
}

// parseFlags parses and returns command line flags
func parseFlags() Flags {
	flags := Flags{}

	flag.BoolVar(&flags.UseLocalServer, "local", false, "Use local test server")
	flag.StringVar(&flags.ServerURL, "server-url", "", "Server URL to ping")
	flag.StringVar(&flags.PingInterval, "ping-interval", "", "Ping interval in milliseconds")
	flag.StringVar(&flags.OwnURL, "own-url", "", "Own health check URL")
	flag.IntVar(&flags.MaxRetries, "max-retries", 0, "Maximum number of retries")
	flag.IntVar(&flags.MaxConsecutiveFails, "max-consecutive-fails", 0, "Maximum number of consecutive failures before shutdown")

	flag.Parse()
	return flags
}

// applyFlags sets environment variables from flags if provided
func applyFlags(flags Flags) {
	if flags.ServerURL != "" {
		os.Setenv("SERVER_URL", flags.ServerURL)
	}
	if flags.PingInterval != "" {
		os.Setenv("PING_INTERVAL", flags.PingInterval)
	}
	if flags.OwnURL != "" {
		os.Setenv("OWN_URL", flags.OwnURL)
	}
	if flags.MaxRetries > 0 {
		os.Setenv("MAX_RETRIES", strconv.Itoa(flags.MaxRetries))
	}
	if flags.MaxConsecutiveFails > 0 {
		os.Setenv("MAX_CONSECUTIVE_FAILS", strconv.Itoa(flags.MaxConsecutiveFails))
	}
}

// setupFlags parses and sets command line flags
func setupFlags() {
	flags := parseFlags()

	if flags.UseLocalServer {
		startLocalTestServer()
		os.Exit(0)
	}

	applyFlags(flags)
}

// startServer starts the HTTP server for health checks with graceful shutdown
func startServer(ctx context.Context, port string) {
	server := &http.Server{
		Addr:    ":" + port,
		Handler: http.DefaultServeMux,
	}

	http.HandleFunc("/health", healthCheckHandler)
	logy("INFO", "Health check endpoint available at /health")

	// Start server in a goroutine
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logy("ERROR", "Server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	<-ctx.Done()
	logy("INFO", "Shutting down server...")

	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logy("ERROR", "Server shutdown error: %v", err)
	}
	logy("INFO", "Server stopped")
}

func startPinging(ctx context.Context, config Config, shutdownChan chan<- struct{}) {
	ticker := time.NewTicker(config.PingInterval)
	defer ticker.Stop()

	consecutiveFailures := 0
	maxConsecutiveFailures := config.MaxConsecutiveFails

	// Create a channel to signal when to stop pinging
	stopChan := make(chan struct{})

	// Start a goroutine to monitor the health check
	go func() {
		for {
			select {
			case <-ctx.Done():
				logy("INFO", "Received shutdown signal, stopping ping routine...")
				close(stopChan)
				return
			case <-stopChan:
				return
			case <-ticker.C:
				success := pingServer(config)
				if success {
					consecutiveFailures = 0
				} else {
					consecutiveFailures++
					if consecutiveFailures >= maxConsecutiveFailures {
						logy("ERROR", "Stopping ping routine after %d consecutive complete failures", maxConsecutiveFailures)
						close(stopChan)
						// Signal main to initiate shutdown
						shutdownChan <- struct{}{}
						return
					}
				}
			}
		}
	}()

	// Wait for the stop signal
	<-stopChan
	logy("INFO", "Ping routine stopped")
}

// Add retry mechanism for pingServer
func pingServer(config Config) bool {
	logy("INFO", "Pinging server: %s", config.ServerURL)

	maxRetries := os.Getenv("MAX_RETRIES")
	if maxRetries == "" {
		maxRetries = "3" // Default to 3 retries if not set
	}
	maxRetriesInt, err := strconv.Atoi(maxRetries)
	if err != nil {
		logy("ERROR", "Error reading MAX_RETRIES environment variable: %v", err)
		return false
	}

	logy("INFO", "Maximum retries set to: %d", maxRetriesInt)

	for i := 0; i < maxRetriesInt; i++ {
		logy("INFO", "Attempt %d of %d", i+1, maxRetriesInt)

		req, err := http.NewRequest("GET", config.ServerURL, nil)
		if err != nil {
			logy("ERROR", "Error creating request: %v", err)
			if i == maxRetriesInt-1 {
				logy("ERROR", "Max retries reached, giving up")
				return false
			}
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
			if i < maxRetriesInt-1 {
				logy("INFO", "Connection failed, retrying... (Attempt %d of %d)", i+1, maxRetriesInt)
				time.Sleep(1 * time.Second)
				continue
			}
			logy("ERROR", "Max retries reached, giving up")
			return false
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			atomic.StoreInt64(&lastPingSuccess, time.Now().Unix())
			logy("INFO", "Ping successful! to server : %s", config.ServerURL)

			// Call the server's own health check API
			callOwnHealthCheck(config.OwnURL)
			return true
		} else {
			logy("ERROR", "Ping failed with status code: %d", resp.StatusCode)
			if i < maxRetriesInt-1 {
				logy("INFO", "Ping failed with status code: %d, retrying... (Attempt %d of %d)", resp.StatusCode, i+1, maxRetriesInt)
				time.Sleep(1 * time.Second)
				continue
			}
			logy("ERROR", "Max retries reached, giving up")
			return false
		}
	}
	return false
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

	fmt.Fprintln(w, "Ping-Pong-Go Server is healthy")
}

func main() {
	// Print welcome banner
	printBanner()

	// Setup command line flags
	setupFlags()

	logy("INFO", "Starting Ping-Pong-Go Server...")

	// Setup environment
	if err := setupEnvironment(); err != nil {
		logy("ERROR", "%v", err)
		os.Exit(1)
	}

	// Validate environment variables
	if err := validateEnv(); err != nil {
		logy("ERROR", "Environment validation error: %v", err)
		os.Exit(1)
	}

	// Setup configuration
	config, err := setupConfig()
	if err != nil {
		logy("ERROR", "%v", err)
		os.Exit(1)
	}

	// Create context that listens for the interrupt signal
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Create channel for automatic shutdown
	shutdownChan := make(chan struct{})

	// Start the ping routine
	go startPinging(ctx, config, shutdownChan)

	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Start the HTTP server in a goroutine
	go startServer(ctx, port)

	// Wait for either manual interrupt or automatic shutdown
	select {
	case <-ctx.Done():
		logy("INFO", "Received manual shutdown signal")
	case <-shutdownChan:
		logy("INFO", "Initiating automatic shutdown due to ping failures")
		stop() // Cancel the context to trigger graceful shutdown
	}

	// Wait a moment for cleanup
	time.Sleep(2 * time.Second)
	logy("INFO", "Application shutdown complete")
}

package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/SumonRayy/ping-pong-go/pkg/pingpong"
	"github.com/fatih/color"
	"github.com/joho/godotenv"
)

// ColorLogger implements the pingpong.Logger interface with colored output
type ColorLogger struct{}

func (l *ColorLogger) Info(format string, args ...interface{}) {
	color.Green(format, args...)
}

func (l *ColorLogger) Error(format string, args ...interface{}) {
	color.Red(format, args...)
}

func (l *ColorLogger) Warn(format string, args ...interface{}) {
	color.Yellow(format, args...)
}

func main() {
	// Load environment variables from .env file if it exists
	if _, err := os.Stat(".env"); err == nil {
		if err := godotenv.Load(); err != nil {
			log.Fatalf("Error loading .env file: %v", err)
		}
	}

	// Parse command line flags
	serverURL := flag.String("server-url", "", "Server URL to ping")
	pingInterval := flag.String("ping-interval", "", "Ping interval in milliseconds")
	ownURL := flag.String("own-url", "", "Own health check URL")
	maxRetries := flag.Int("max-retries", 0, "Maximum number of retries")
	maxConsecutiveFails := flag.Int("max-consecutive-fails", 0, "Maximum number of consecutive failures before shutdown")
	flag.Parse()

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
	if *maxConsecutiveFails > 0 {
		os.Setenv("MAX_CONSECUTIVE_FAILS", strconv.Itoa(*maxConsecutiveFails))
	}

	// Get configuration from environment variables
	config := pingpong.Config{
		ServerURL:           getEnvOrDefault("SERVER_URL", "http://localhost:8081/health"),
		OwnURL:              getEnvOrDefault("OWN_URL", "http://localhost:8080/health"),
		PingInterval:        time.Duration(getEnvIntOrDefault("PING_INTERVAL", 2000)) * time.Millisecond,
		MaxConsecutiveFails: getEnvIntOrDefault("MAX_CONSECUTIVE_FAILS", 3),
		MaxRetries:          getEnvIntOrDefault("MAX_RETRIES", 3),
		Logger:              &ColorLogger{},
	}

	// Create and start the service
	service := pingpong.NewService(config)

	// Create context that listens for the interrupt signal
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Start the service
	if err := service.Start(ctx); err != nil {
		log.Fatalf("Failed to start service: %v", err)
	}

	// Wait for interrupt signal
	<-ctx.Done()

	// Gracefully shutdown the service
	if err := service.Stop(); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}
}

// Helper functions
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvIntOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

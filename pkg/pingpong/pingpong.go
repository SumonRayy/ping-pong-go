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
package pingpong

import (
	"context"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"
)

// Config represents the configuration for the ping-pong service
type Config struct {
	ServerURL           string
	OwnURL              string
	PingInterval        time.Duration
	Headers             map[string]string // Custom headers for ping requests
	MaxConsecutiveFails int               // Maximum number of consecutive failures before shutdown
	MaxRetries          int               // Maximum number of retries for each ping
	Logger              Logger            // Custom logger interface
}

// Logger interface for custom logging
type Logger interface {
	Info(format string, args ...interface{})
	Error(format string, args ...interface{})
	Warn(format string, args ...interface{})
}

// DefaultLogger implements the Logger interface with basic logging
type DefaultLogger struct{}

func (l *DefaultLogger) Info(format string, args ...interface{}) {
	fmt.Printf("[INFO] "+format+"\n", args...)
}
func (l *DefaultLogger) Error(format string, args ...interface{}) {
	fmt.Printf("[ERROR] "+format+"\n", args...)
}
func (l *DefaultLogger) Warn(format string, args ...interface{}) {
	fmt.Printf("[WARN] "+format+"\n", args...)
}

// Service represents a ping-pong service instance
type Service struct {
	config          Config
	lastPingSuccess int64
	logger          Logger
	server          *http.Server
}

// NewService creates a new ping-pong service with the given configuration
func NewService(config Config) *Service {
	if config.Logger == nil {
		config.Logger = &DefaultLogger{}
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	return &Service{
		config: config,
		logger: config.Logger,
	}
}

// Start starts the ping-pong service
func (s *Service) Start(ctx context.Context) error {
	// Start the HTTP server
	if err := s.startServer(); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	// Start the ping routine
	go s.startPinging(ctx)

	return nil
}

// Stop gracefully stops the service
func (s *Service) Stop() error {
	if s.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.server.Shutdown(ctx)
	}
	return nil
}

// startServer starts the HTTP server for health checks
func (s *Service) startServer() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.healthCheckHandler)

	s.server = &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("Server error: %v", err)
		}
	}()

	return nil
}

// startPinging starts the ping routine
func (s *Service) startPinging(ctx context.Context) {
	ticker := time.NewTicker(s.config.PingInterval)
	defer ticker.Stop()

	consecutiveFailures := 0

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			success := s.pingServer()
			if success {
				consecutiveFailures = 0
			} else {
				consecutiveFailures++
				if consecutiveFailures >= s.config.MaxConsecutiveFails {
					s.logger.Error("Stopping ping routine after %d consecutive failures", s.config.MaxConsecutiveFails)
					return
				}
			}
		}
	}
}

// pingServer attempts to ping the configured server
func (s *Service) pingServer() bool {
	s.logger.Info("Pinging server: %s", s.config.ServerURL)

	for i := 0; i < s.config.MaxRetries; i++ {
		s.logger.Info("Attempt %d of %d", i+1, s.config.MaxRetries)

		req, err := http.NewRequest("GET", s.config.ServerURL, nil)
		if err != nil {
			s.logger.Error("Error creating request: %v", err)
			continue
		}

		// Add custom headers
		for key, value := range s.config.Headers {
			req.Header.Set(key, value)
		}

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			s.logger.Error("Error pinging server: %v", err)
			if i < s.config.MaxRetries-1 {
				time.Sleep(1 * time.Second)
				continue
			}
			return false
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			atomic.StoreInt64(&s.lastPingSuccess, time.Now().Unix())
			s.logger.Info("Ping successful!")
			s.callOwnHealthCheck()
			return true
		}

		s.logger.Error("Ping failed with status code: %d", resp.StatusCode)
		if i < s.config.MaxRetries-1 {
			time.Sleep(1 * time.Second)
			continue
		}
	}
	return false
}

// callOwnHealthCheck calls the service's own health check endpoint
func (s *Service) callOwnHealthCheck() {
	if s.config.OwnURL == "" {
		return
	}

	resp, err := http.Get(s.config.OwnURL)
	if err != nil {
		s.logger.Error("Error calling own health check: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		s.logger.Info("Own health check successful!")
	} else {
		s.logger.Error("Own health check failed with status code: %d", resp.StatusCode)
	}
}

// healthCheckHandler handles health check requests
func (s *Service) healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	lastPing := atomic.LoadInt64(&s.lastPingSuccess)
	if lastPing == 0 {
		http.Error(w, "No successful pings yet", http.StatusServiceUnavailable)
		return
	}

	if time.Since(time.Unix(lastPing, 0)) > 15*time.Minute {
		http.Error(w, "Last successful ping was too long ago", http.StatusServiceUnavailable)
		return
	}

	fmt.Fprintln(w, "Ping-Pong-Go Server is healthy")
}

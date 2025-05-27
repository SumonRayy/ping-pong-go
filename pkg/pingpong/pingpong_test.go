package pingpong

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// TestLogger implements the Logger interface for testing
type TestLogger struct {
	InfoLogs  []string
	ErrorLogs []string
	WarnLogs  []string
}

func (l *TestLogger) Info(format string, args ...interface{}) {
	l.InfoLogs = append(l.InfoLogs, format)
}
func (l *TestLogger) Error(format string, args ...interface{}) {
	l.ErrorLogs = append(l.ErrorLogs, format)
}
func (l *TestLogger) Warn(format string, args ...interface{}) {
	l.WarnLogs = append(l.WarnLogs, format)
}

func TestNewService(t *testing.T) {
	logger := &TestLogger{}
	config := Config{
		ServerURL:           "http://example.com/health",
		OwnURL:              "http://localhost:8080/health",
		PingInterval:        time.Second,
		MaxConsecutiveFails: 3,
		MaxRetries:          3,
		Logger:              logger,
	}

	service := NewService(config)
	if service == nil || service.config.ServerURL != config.ServerURL {
		t.Errorf("NewService failed: service is nil or ServerURL mismatch (expected %s)", config.ServerURL)
	}
}

func TestService_Start(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	logger := &TestLogger{}
	config := Config{
		ServerURL:           server.URL,
		OwnURL:              server.URL,
		PingInterval:        100 * time.Millisecond,
		MaxConsecutiveFails: 3,
		MaxRetries:          3,
		Logger:              logger,
	}

	service := NewService(config)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	err := service.Start(ctx)
	if err != nil {
		t.Errorf("Start returned error: %v", err)
	}

	// Wait for context to be done
	<-ctx.Done()

	// Stop the service
	if err := service.Stop(); err != nil {
		t.Errorf("Stop returned error: %v", err)
	}
}

func TestService_HealthCheck(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	logger := &TestLogger{}
	config := Config{
		ServerURL:           server.URL,
		OwnURL:              server.URL,
		PingInterval:        100 * time.Millisecond,
		MaxConsecutiveFails: 3,
		MaxRetries:          3,
		Logger:              logger,
	}

	service := NewService(config)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	err := service.Start(ctx)
	if err != nil {
		t.Errorf("Start returned error: %v", err)
	}

	// Simulate a successful ping
	atomic.StoreInt64(&service.lastPingSuccess, time.Now().Unix())

	// Create a test request
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	// Call the health check handler
	service.healthCheckHandler(w, req)

	// Check the response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	// Stop the service
	if err := service.Stop(); err != nil {
		t.Errorf("Stop returned error: %v", err)
	}
}

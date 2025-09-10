package http_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/wasim-nihal/k8s-watcher/pkg/config"
	client "github.com/wasim-nihal/k8s-watcher/pkg/http"
	"github.com/wasim-nihal/k8s-watcher/pkg/logger"
)

type testPayload struct {
	Message string `json:"message"`
}

func TestNewClient(t *testing.T) {
	tests := []struct {
		name   string
		config config.RequestConfig
	}{
		{
			name: "basic client",
			config: config.RequestConfig{
				Timeout:       10,
				SkipTLSVerify: false,
			},
		},
		{
			name: "client with TLS skip verify",
			config: config.RequestConfig{
				Timeout:       5,
				SkipTLSVerify: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := client.NewClient(tt.config)
			if c == nil {
				t.Error("NewClient() returned nil")
			}
		})
	}
}

func TestSendNotification(t *testing.T) {
	// Initialize logger
	err := logger.Initialize(config.LoggingConfig{
		Level:  "INFO",
		Format: "JSON",
	})
	if err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}
	tests := []struct {
		name         string
		config       config.RequestConfig
		payload      interface{}
		serverFunc   func(http.ResponseWriter, *http.Request)
		wantErr      bool
		expectedBody string
	}{
		{
			name: "successful GET request",
			config: config.RequestConfig{
				Method:  "GET",
				Timeout: 5,
			},
			serverFunc: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					t.Errorf("Expected GET request, got %s", r.Method)
				}
				w.WriteHeader(http.StatusOK)
			},
			wantErr: false,
		},
		{
			name: "successful POST request with payload",
			config: config.RequestConfig{
				Method:  "POST",
				Timeout: 5,
			},
			payload: testPayload{Message: "test message"},
			serverFunc: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("Expected POST request, got %s", r.Method)
				}
				var p testPayload
				if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
					t.Errorf("Failed to decode request body: %v", err)
				}
				if p.Message != "test message" {
					t.Errorf("Expected message 'test message', got %s", p.Message)
				}
				w.WriteHeader(http.StatusOK)
			},
			wantErr: false,
		},
		{
			name: "server error with retry",
			config: config.RequestConfig{
				Method:  "GET",
				Timeout: 5,
				Retry: config.RetryConfig{
					Total:         2,
					BackoffFactor: 0.1,
				},
			},
			serverFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverFunc))
			defer server.Close()

			tt.config.URL = server.URL
			c := client.NewClient(tt.config)

			err := c.SendNotification(tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("SendNotification() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRetryLogic(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts <= 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := config.RequestConfig{
		URL:     server.URL,
		Method:  "GET",
		Timeout: 5,
		Retry: config.RetryConfig{
			Total:         3,
			BackoffFactor: 0.1,
		},
	}

	c := client.NewClient(cfg)
	err := c.SendNotification(nil)
	if err != nil {
		t.Errorf("Expected success after retries, got error: %v", err)
	}
	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := config.RequestConfig{
		URL:     server.URL,
		Method:  "GET",
		Timeout: 1, // 1 second timeout
	}

	c := client.NewClient(cfg)
	err := c.SendNotification(nil)
	if err == nil {
		t.Error("Expected timeout error, got nil")
	}
}

func TestAuthentication(t *testing.T) {
	const (
		username = "testuser"
		password = "testpass"
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != username || pass != password {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := config.RequestConfig{
		URL:     server.URL,
		Method:  "GET",
		Timeout: 5,
		Auth: config.AuthConfig{
			Basic: config.BasicAuth{
				Username: username,
				Password: password,
			},
		},
	}

	c := client.NewClient(cfg)
	err := c.SendNotification(nil)
	if err != nil {
		t.Errorf("Expected successful authenticated request, got error: %v", err)
	}
}

func BenchmarkSendNotification(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := config.RequestConfig{
		URL:     server.URL,
		Method:  "POST",
		Timeout: 5,
	}

	c := client.NewClient(cfg)
	payload := testPayload{Message: "benchmark test"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := c.SendNotification(payload)
		if err != nil {
			b.Fatalf("SendNotification failed: %v", err)
		}
	}
}

package http

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/wasim-nihal/k8s-watcher/pkg/config"
	"github.com/wasim-nihal/k8s-watcher/pkg/logger"
)

// Client handles HTTP requests with retry logic
type Client struct {
	client *http.Client
	config config.RequestConfig
}

// NewClient creates a new HTTP client with the given configuration
func NewClient(config config.RequestConfig) *Client {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: config.SkipTLSVerify,
		},
	}

	client := &http.Client{
		Timeout:   time.Duration(config.Timeout) * time.Second,
		Transport: transport,
	}

	return &Client{
		client: client,
		config: config,
	}
}

// SendNotification sends an HTTP request with retry logic
func (c *Client) SendNotification(payload interface{}) error {
	var body []byte
	var err error

	if payload != nil {
		body, err = json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("marshaling payload: %w", err)
		}
	}

	method := c.config.Method
	if method == "" {
		method = "GET"
	}

	return c.doWithRetry(method, c.config.URL, body)
}

// doWithRetry performs the HTTP request with retry logic
func (c *Client) doWithRetry(method, url string, body []byte) error {
	var lastErr error
	retryConfig := c.config.Retry

	for attempt := 0; attempt <= retryConfig.Total; attempt++ {
		if attempt > 0 {
			backoffDuration := time.Duration(float64(attempt) * retryConfig.BackoffFactor * float64(time.Second))
			time.Sleep(backoffDuration)
		}

		err := c.doRequest(method, url, body)
		if err == nil {
			return nil
		}

		lastErr = err
		logger.Warn("Request failed, retrying",
			"attempt", attempt+1,
			"maxAttempts", retryConfig.Total+1,
			"error", err,
		)
	}

	return fmt.Errorf("all retry attempts failed: %w", lastErr)
}

// doRequest performs a single HTTP request
func (c *Client) doRequest(method, url string, body []byte) error {
	var req *http.Request
	var err error

	if len(body) > 0 {
		req, err = http.NewRequest(method, url, bytes.NewReader(body))
	} else {
		req, err = http.NewRequest(method, url, nil)
	}

	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}

	// Add basic auth if configured
	if c.config.Auth.Basic.Username != "" {
		req.SetBasicAuth(c.config.Auth.Basic.Username, c.config.Auth.Basic.Password)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body for error reporting
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	logger.Info("Request completed successfully",
		"method", method,
		"url", url,
		"status", resp.StatusCode,
	)

	return nil
}

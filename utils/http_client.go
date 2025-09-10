package utils

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// BaseHTTPClient provides common HTTP functionality for AI clients
type BaseHTTPClient struct {
	HttpClient *http.Client
	baseURL    string
	ApiKey     string
	logger     *Logger
}

// NewBaseHTTPClient creates a new base HTTP client with timeout and retry logic
func NewBaseHTTPClient(baseURL, apiKey string, timeout time.Duration) *BaseHTTPClient {
	return &BaseHTTPClient{
		HttpClient: &http.Client{
			Timeout: timeout,
		},
		baseURL: strings.TrimSuffix(baseURL, "/"),
		ApiKey:  apiKey,
		logger:  NewLogger("HTTPClient"),
	}
}

// HTTPRequest represents an HTTP request configuration
type HTTPRequest struct {
	Method  string
	Path    string
	Headers map[string]string
	Body    io.Reader
}

// HTTPResponse represents an HTTP response
type HTTPResponse struct {
	StatusCode int
	Body       []byte
	Headers    map[string][]string
}

// DoRequest executes an HTTP request with retry logic and network status awareness
func (c *BaseHTTPClient) DoRequest(ctx context.Context, req HTTPRequest) (*HTTPResponse, error) {
	url := c.baseURL + req.Path

	// Check network status before making request
	if globalNetworkMonitor != nil && globalNetworkMonitor.IsOffline() {
		c.logger.Warn("Network is offline, request will likely fail: %s %s", req.Method, url)
		return nil, fmt.Errorf("network is offline")
	}

	httpReq, err := http.NewRequestWithContext(ctx, req.Method, url, req.Body)
	if err != nil {
		c.logger.Error("Failed to create HTTP request: %v", err)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set default headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", "VSCode-Assist/1.0")

	// Set custom headers
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	// Execute request with retry logic and network-aware backoff
	var resp *http.Response
	maxRetries := 3
	baseDelay := time.Millisecond * 500

	// Adjust retry strategy based on network status
	if globalNetworkMonitor != nil {
		status := globalNetworkMonitor.GetStatus()
		if status == NetworkStatusLimited {
			maxRetries = 5          // More retries for limited connectivity
			baseDelay = time.Second // Longer delays
		}
	}

	for attempt := 0; attempt <= maxRetries; attempt++ {
		resp, err = c.HttpClient.Do(httpReq)
		if err != nil {
			// Check if this is a network-related error
			isNetworkError := c.isNetworkError(err)

			if attempt == maxRetries {
				c.logger.Error("HTTP request failed after %d attempts: %v", maxRetries+1, err)

				// Update network status if this appears to be a connectivity issue
				if isNetworkError && globalNetworkMonitor != nil {
					go func() {
						ctx := context.Background()
						globalNetworkMonitor.CheckConnectivity(ctx)
					}()
				}

				return nil, fmt.Errorf("request failed after %d attempts: %w", maxRetries+1, err)
			}

			// Exponential backoff with network-aware delays
			delay := baseDelay * time.Duration(1<<attempt)
			if isNetworkError {
				delay *= 2 // Longer delays for network errors
			}

			c.logger.Warn("HTTP request attempt %d failed, retrying in %v: %v", attempt+1, delay, err)

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
				continue
			}
		}
		break
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logger.Error("Failed to read response body: %v", err)
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	response := &HTTPResponse{
		StatusCode: resp.StatusCode,
		Body:       body,
		Headers:    resp.Header,
	}

	c.logger.Info("HTTP request completed: %s %s -> %d", req.Method, url, resp.StatusCode)

	return response, nil
}

// isNetworkError determines if an error is network-related
func (c *BaseHTTPClient) isNetworkError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	networkErrorPatterns := []string{
		"connection refused",
		"connection reset",
		"connection timeout",
		"network is unreachable",
		"no such host",
		"timeout",
		"context deadline exceeded",
		"i/o timeout",
	}

	for _, pattern := range networkErrorPatterns {
		if strings.Contains(strings.ToLower(errStr), pattern) {
			return true
		}
	}

	return false
}

// IsRetryableError determines if an HTTP error should be retried
func (c *BaseHTTPClient) IsRetryableError(statusCode int) bool {
	switch statusCode {
	case http.StatusTooManyRequests, // 429
		http.StatusInternalServerError, // 500
		http.StatusBadGateway,          // 502
		http.StatusServiceUnavailable,  // 503
		http.StatusGatewayTimeout:      // 504
		return true
	default:
		return false
	}
}

// ValidateResponse checks if the HTTP response indicates success
func (c *BaseHTTPClient) ValidateResponse(resp *HTTPResponse) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	if c.IsRetryableError(resp.StatusCode) {
		return fmt.Errorf("retryable error: HTTP %d", resp.StatusCode)
	}

	return fmt.Errorf("HTTP error: %d - %s", resp.StatusCode, string(resp.Body))
}

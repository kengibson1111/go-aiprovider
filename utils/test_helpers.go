package utils

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// TestHelpers provides common mock scenarios and utilities for testing
type TestHelpers struct{}

// NewTestHelpers creates a new test helpers instance
func NewTestHelpers() *TestHelpers {
	return &TestHelpers{}
}

// CreateSuccessResponse creates a standard success HTTP response
func (th *TestHelpers) CreateSuccessResponse(body string) *HTTPResponse {
	return &HTTPResponse{
		StatusCode: http.StatusOK,
		Body:       []byte(body),
		Headers: map[string][]string{
			"Content-Type": {"application/json"},
		},
	}
}

// CreateErrorResponse creates a standard error HTTP response
func (th *TestHelpers) CreateErrorResponse(statusCode int, message string) *HTTPResponse {
	body := fmt.Sprintf(`{"error": {"message": "%s", "code": %d}}`, message, statusCode)
	return &HTTPResponse{
		StatusCode: statusCode,
		Body:       []byte(body),
		Headers: map[string][]string{
			"Content-Type": {"application/json"},
		},
	}
}

// CreateTimeoutError creates a context timeout error for testing
func (th *TestHelpers) CreateTimeoutError() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancel()
	<-ctx.Done()
	return ctx.Err()
}

// CreateNetworkError creates a network-related error for testing
func (th *TestHelpers) CreateNetworkError(message string) error {
	return fmt.Errorf("network error: %s", message)
}

// SetupMockHTTPClientForSuccess configures a mock HTTP client for successful responses
func (th *TestHelpers) SetupMockHTTPClientForSuccess(mock *MockHTTPClient, responses map[string]string) {
	for pattern, body := range responses {
		mock.SetResponse("POST", pattern, http.StatusOK, body)
		mock.SetResponse("GET", pattern, http.StatusOK, body)
	}
}

// SetupMockHTTPClientForErrors configures a mock HTTP client for error responses
func (th *TestHelpers) SetupMockHTTPClientForErrors(mock *MockHTTPClient, errors map[string]int) {
	for pattern, statusCode := range errors {
		errorBody := fmt.Sprintf(`{"error": {"message": "Mock error response", "code": %d}}`, statusCode)
		mock.SetResponse("POST", pattern, statusCode, errorBody)
		mock.SetResponse("GET", pattern, statusCode, errorBody)
	}
}

// SetupMockHTTPClientForNetworkErrors configures a mock HTTP client for network errors
func (th *TestHelpers) SetupMockHTTPClientForNetworkErrors(mock *MockHTTPClient, patterns []string) {
	for _, pattern := range patterns {
		mock.SetError("POST", pattern, th.CreateNetworkError("connection refused"))
		mock.SetError("GET", pattern, th.CreateNetworkError("connection refused"))
	}
}

// SetupMockHTTPClientForRetryScenario configures a mock HTTP client for retry testing
func (th *TestHelpers) SetupMockHTTPClientForRetryScenario(mock *MockHTTPClient, pattern string, failCount int, finalStatusCode int, finalBody string) {
	// This is a simplified approach - in a real scenario, you might need more sophisticated
	// retry simulation that tracks call counts and changes behavior accordingly
	if failCount > 0 {
		mock.SetError("POST", pattern, th.CreateNetworkError("temporary failure"))
	}
	if finalStatusCode > 0 {
		mock.SetResponse("POST", pattern, finalStatusCode, finalBody)
	}
}

// SetupMockNetworkMonitorOnline configures a mock network monitor for online status
func (th *TestHelpers) SetupMockNetworkMonitorOnline(mock *MockNetworkMonitor) {
	mock.SetStatus(NetworkStatusOnline)
	mock.SetEndpointResult("https://api.anthropic.com/v1/messages", true)
	mock.SetEndpointResult("https://api.openai.com/v1/chat/completions", true)
	mock.SetEndpointResult("https://www.google.com", true)
}

// SetupMockNetworkMonitorOffline configures a mock network monitor for offline status
func (th *TestHelpers) SetupMockNetworkMonitorOffline(mock *MockNetworkMonitor) {
	mock.SetStatus(NetworkStatusOffline)
	mock.SetEndpointResult("https://api.anthropic.com/v1/messages", false)
	mock.SetEndpointResult("https://api.openai.com/v1/chat/completions", false)
	mock.SetEndpointResult("https://www.google.com", false)
}

// SetupMockNetworkMonitorLimited configures a mock network monitor for limited connectivity
func (th *TestHelpers) SetupMockNetworkMonitorLimited(mock *MockNetworkMonitor) {
	mock.SetStatus(NetworkStatusLimited)
	mock.SetEndpointResult("https://api.anthropic.com/v1/messages", false)
	mock.SetEndpointResult("https://api.openai.com/v1/chat/completions", true)
	mock.SetEndpointResult("https://www.google.com", true)
}

// CreateMockClaudeResponse creates a mock response for Claude API
func (th *TestHelpers) CreateMockClaudeResponse(content string) *HTTPResponse {
	body := fmt.Sprintf(`{
		"id": "msg_test123",
		"type": "message",
		"role": "assistant",
		"content": [
			{
				"type": "text",
				"text": "%s"
			}
		],
		"model": "claude-3-sonnet-20240229",
		"stop_reason": "end_turn",
		"stop_sequence": null,
		"usage": {
			"input_tokens": 10,
			"output_tokens": 20
		}
	}`, content)

	return th.CreateSuccessResponse(body)
}

// CreateMockOpenAIResponse creates a mock response for OpenAI API
func (th *TestHelpers) CreateMockOpenAIResponse(content string) *HTTPResponse {
	body := fmt.Sprintf(`{
		"id": "chatcmpl-test123",
		"object": "chat.completion",
		"created": 1677652288,
		"model": "gpt-3.5-turbo",
		"choices": [
			{
				"index": 0,
				"message": {
					"role": "assistant",
					"content": "%s"
				},
				"finish_reason": "stop"
			}
		],
		"usage": {
			"prompt_tokens": 10,
			"completion_tokens": 20,
			"total_tokens": 30
		}
	}`, content)

	return th.CreateSuccessResponse(body)
}

// CreateMockRateLimitResponse creates a mock rate limit error response
func (th *TestHelpers) CreateMockRateLimitResponse() *HTTPResponse {
	body := `{
		"error": {
			"message": "Rate limit exceeded",
			"type": "rate_limit_error",
			"code": "rate_limit_exceeded"
		}
	}`

	return &HTTPResponse{
		StatusCode: http.StatusTooManyRequests,
		Body:       []byte(body),
		Headers: map[string][]string{
			"Content-Type":          {"application/json"},
			"Retry-After":           {"60"},
			"X-RateLimit-Remaining": {"0"},
		},
	}
}

// CreateMockAuthErrorResponse creates a mock authentication error response
func (th *TestHelpers) CreateMockAuthErrorResponse() *HTTPResponse {
	body := `{
		"error": {
			"message": "Invalid API key",
			"type": "authentication_error",
			"code": "invalid_api_key"
		}
	}`

	return &HTTPResponse{
		StatusCode: http.StatusUnauthorized,
		Body:       []byte(body),
		Headers: map[string][]string{
			"Content-Type": {"application/json"},
		},
	}
}

// CreateMockServerErrorResponse creates a mock server error response
func (th *TestHelpers) CreateMockServerErrorResponse() *HTTPResponse {
	body := `{
		"error": {
			"message": "Internal server error",
			"type": "server_error",
			"code": "internal_error"
		}
	}`

	return &HTTPResponse{
		StatusCode: http.StatusInternalServerError,
		Body:       []byte(body),
		Headers: map[string][]string{
			"Content-Type": {"application/json"},
		},
	}
}

// AssertRequestMade verifies that a specific request was made to the mock HTTP client
func (th *TestHelpers) AssertRequestMade(mock *MockHTTPClient, method, urlPattern string) error {
	count := mock.GetRequestCount(method, urlPattern)
	if count == 0 {
		return fmt.Errorf("expected request %s %s was not made", method, urlPattern)
	}
	return nil
}

// AssertRequestCount verifies the number of requests made to the mock HTTP client
func (th *TestHelpers) AssertRequestCount(mock *MockHTTPClient, method, urlPattern string, expectedCount int) error {
	actualCount := mock.GetRequestCount(method, urlPattern)
	if actualCount != expectedCount {
		return fmt.Errorf("expected %d requests to %s %s, but got %d", expectedCount, method, urlPattern, actualCount)
	}
	return nil
}

// AssertNetworkStatus verifies the network monitor status
func (th *TestHelpers) AssertNetworkStatus(mock *MockNetworkMonitor, expectedStatus NetworkStatus) error {
	actualStatus := mock.GetStatus()
	if actualStatus != expectedStatus {
		return fmt.Errorf("expected network status %s, but got %s", expectedStatus.String(), actualStatus.String())
	}
	return nil
}

// AssertMonitoringState verifies the monitoring state
func (th *TestHelpers) AssertMonitoringState(mock *MockNetworkMonitor, expectedState bool) error {
	actualState := mock.IsMonitoring()
	if actualState != expectedState {
		return fmt.Errorf("expected monitoring state %t, but got %t", expectedState, actualState)
	}
	return nil
}

// WaitForStatusChange waits for a network status change with timeout
func (th *TestHelpers) WaitForStatusChange(mock *MockNetworkMonitor, expectedStatus NetworkStatus, timeout time.Duration) error {
	statusChanged := make(chan bool, 1)

	callback := func(status NetworkStatus) {
		if status == expectedStatus {
			select {
			case statusChanged <- true:
			default:
			}
		}
	}

	mock.AddStatusCallback(callback)

	select {
	case <-statusChanged:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("timeout waiting for network status change to %s", expectedStatus.String())
	}
}

// SimulateNetworkFluctuation simulates network status changes over time
func (th *TestHelpers) SimulateNetworkFluctuation(mock *MockNetworkMonitor, statuses []NetworkStatus, interval time.Duration) {
	go func() {
		for _, status := range statuses {
			mock.SetStatus(status)
			time.Sleep(interval)
		}
	}()
}

// CreateTestContext creates a context with timeout for testing
func (th *TestHelpers) CreateTestContext(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}

// CreateCancelledContext creates a pre-cancelled context for testing
func (th *TestHelpers) CreateCancelledContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

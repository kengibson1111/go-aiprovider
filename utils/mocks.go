package utils

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// MockHTTPClient implements a mock HTTP client for testing
type MockHTTPClient struct {
	responses    map[string]*http.Response
	errors       map[string]error
	requestCount map[string]int
	lastRequest  *http.Request
	doFunc       func(*http.Request) (*http.Response, error)
	// Enhanced fields for better testing
	requestHistory  []MockHTTPRequest
	delayMap        map[string]time.Duration
	defaultResponse *http.Response
	defaultError    error
	mu              sync.RWMutex
}

// MockHTTPRequest represents a captured HTTP request for testing
type MockHTTPRequest struct {
	Method  string
	URL     string
	Headers map[string]string
	Body    string
	Context context.Context
}

// NewMockHTTPClient creates a new mock HTTP client
func NewMockHTTPClient() *MockHTTPClient {
	return &MockHTTPClient{
		responses:      make(map[string]*http.Response),
		errors:         make(map[string]error),
		requestCount:   make(map[string]int),
		requestHistory: make([]MockHTTPRequest, 0),
		delayMap:       make(map[string]time.Duration),
	}
}

// Do implements the http.Client.Do method for mocking
func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	m.mu.Lock()

	m.lastRequest = req
	key := req.Method + " " + req.URL.String()
	m.requestCount[key]++

	// Record the request in history
	body := ""
	if req.Body != nil {
		if bodyBytes, err := io.ReadAll(req.Body); err == nil {
			body = string(bodyBytes)
			// Reset the body for potential re-reading
			req.Body = io.NopCloser(strings.NewReader(body))
		}
	}

	headers := make(map[string]string)
	for k, v := range req.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}

	mockReq := MockHTTPRequest{
		Method:  req.Method,
		URL:     req.URL.String(),
		Headers: headers,
		Body:    body,
		Context: req.Context(),
	}
	m.requestHistory = append(m.requestHistory, mockReq)

	// Check for artificial delay
	if delay, exists := m.delayMap[key]; exists {
		m.mu.Unlock() // Unlock before delay
		select {
		case <-req.Context().Done():
			return nil, req.Context().Err()
		case <-time.After(delay):
		}
		m.mu.Lock() // Re-acquire lock after delay
	}

	// Use custom doFunc if set
	if m.doFunc != nil {
		result, err := m.doFunc(req)
		m.mu.Unlock()
		return result, err
	}

	if err, exists := m.errors[key]; exists {
		m.mu.Unlock()
		return nil, err
	}

	if resp, exists := m.responses[key]; exists {
		m.mu.Unlock()
		return resp, nil
	}

	// Use defaults
	if m.defaultError != nil {
		m.mu.Unlock()
		return nil, m.defaultError
	}

	if m.defaultResponse != nil {
		m.mu.Unlock()
		return m.defaultResponse, nil
	}

	// Default response
	m.mu.Unlock()
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(`{"success": true}`)),
		Header:     make(http.Header),
	}, nil
}

// SetResponse sets a mock response for a specific request
func (m *MockHTTPClient) SetResponse(method, url string, statusCode int, body string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := method + " " + url
	// Clear any existing error for this key when setting a response
	delete(m.errors, key)
	m.responses[key] = &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

// SetHTTPResponse sets a mock HTTPResponse for a specific request pattern
func (m *MockHTTPClient) SetHTTPResponse(method, urlPattern string, response *HTTPResponse) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := method + " " + urlPattern

	headers := make(http.Header)
	for k, v := range response.Headers {
		headers[k] = v
	}

	m.responses[key] = &http.Response{
		StatusCode: response.StatusCode,
		Body:       io.NopCloser(strings.NewReader(string(response.Body))),
		Header:     headers,
	}
}

// SetError sets a mock error for a specific request
func (m *MockHTTPClient) SetError(method, url string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := method + " " + url
	m.errors[key] = err
}

// SetDoFunc sets a custom function for the Do method
func (m *MockHTTPClient) SetDoFunc(fn func(*http.Request) (*http.Response, error)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.doFunc = fn
}

// SetDelay configures an artificial delay for a specific request pattern
func (m *MockHTTPClient) SetDelay(method, urlPattern string, delay time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := method + " " + urlPattern
	m.delayMap[key] = delay
}

// SetDefaultResponse sets the default response for unmatched requests
func (m *MockHTTPClient) SetDefaultResponse(statusCode int, body string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.defaultResponse = &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

// SetDefaultError sets the default error for unmatched requests
func (m *MockHTTPClient) SetDefaultError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.defaultError = err
}

// GetRequestCount returns the number of times a request was made
func (m *MockHTTPClient) GetRequestCount(method, url string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	key := method + " " + url
	return m.requestCount[key]
}

// GetLastRequest returns the last request made
func (m *MockHTTPClient) GetLastRequest() *http.Request {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastRequest
}

// GetLastMockRequest returns the last request as MockHTTPRequest with more details
func (m *MockHTTPClient) GetLastMockRequest() *MockHTTPRequest {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if len(m.requestHistory) == 0 {
		return nil
	}
	return &m.requestHistory[len(m.requestHistory)-1]
}

// GetAllRequests returns all requests made to the mock
func (m *MockHTTPClient) GetAllRequests() []MockHTTPRequest {
	m.mu.RLock()
	defer m.mu.RUnlock()
	requests := make([]MockHTTPRequest, len(m.requestHistory))
	copy(requests, m.requestHistory)
	return requests
}

// Reset clears all mock configuration and history
func (m *MockHTTPClient) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.responses = make(map[string]*http.Response)
	m.errors = make(map[string]error)
	m.requestCount = make(map[string]int)
	m.requestHistory = make([]MockHTTPRequest, 0)
	m.delayMap = make(map[string]time.Duration)
	m.lastRequest = nil
	m.doFunc = nil
	m.defaultResponse = nil
	m.defaultError = nil
}

// MockNetworkMonitor implements a mock network monitor for testing
type MockNetworkMonitor struct {
	status          NetworkStatus
	lastCheck       time.Time
	timeout         time.Duration
	maxTimeout      time.Duration
	testEndpoints   []string
	statusCallbacks []func(NetworkStatus)
	isMonitoring    bool
	checkHistory    []NetworkStatus
	endpointResults map[string]bool
	mu              sync.RWMutex
}

// NewMockNetworkMonitor creates a new mock network monitor
func NewMockNetworkMonitor() *MockNetworkMonitor {
	return &MockNetworkMonitor{
		status:          NetworkStatusOnline,
		timeout:         5 * time.Second,
		maxTimeout:      10 * time.Second,
		testEndpoints:   []string{"https://api.example.com"},
		statusCallbacks: make([]func(NetworkStatus), 0),
		checkHistory:    make([]NetworkStatus, 0),
		endpointResults: make(map[string]bool),
	}
}

// GetStatus returns the mock network status
func (m *MockNetworkMonitor) GetStatus() NetworkStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.status
}

// GetLastCheck returns the time of the last connectivity check
func (m *MockNetworkMonitor) GetLastCheck() time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastCheck
}

// IsOnline returns the mock online status
func (m *MockNetworkMonitor) IsOnline() bool {
	return m.GetStatus() == NetworkStatusOnline
}

// IsOffline returns the mock offline status
func (m *MockNetworkMonitor) IsOffline() bool {
	status := m.GetStatus()
	return status == NetworkStatusOffline || status == NetworkStatusUnknown
}

// SetStatus sets the mock network status and triggers callbacks
func (m *MockNetworkMonitor) SetStatus(status NetworkStatus) {
	m.mu.Lock()
	oldStatus := m.status
	m.status = status
	m.lastCheck = time.Now()
	callbacks := make([]func(NetworkStatus), len(m.statusCallbacks))
	copy(callbacks, m.statusCallbacks)
	m.mu.Unlock()

	// Notify callbacks if status changed
	if oldStatus != status {
		for _, callback := range callbacks {
			go callback(status)
		}
	}
}

// SetTimeout updates the network check timeout
func (m *MockNetworkMonitor) SetTimeout(timeout time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if timeout > m.maxTimeout {
		timeout = m.maxTimeout
	}
	if timeout < time.Second {
		timeout = time.Second
	}
	m.timeout = timeout
}

// GetTimeout returns the current network timeout
func (m *MockNetworkMonitor) GetTimeout() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.timeout
}

// AddStatusCallback registers a callback for status changes
func (m *MockNetworkMonitor) AddStatusCallback(callback func(NetworkStatus)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.statusCallbacks = append(m.statusCallbacks, callback)
}

// CheckConnectivity simulates a connectivity check
func (m *MockNetworkMonitor) CheckConnectivity(ctx context.Context) NetworkStatus {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Record the check
	m.checkHistory = append(m.checkHistory, m.status)
	m.lastCheck = time.Now()

	return m.status
}

// StartMonitoring simulates starting network monitoring
func (m *MockNetworkMonitor) StartMonitoring(ctx context.Context) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.isMonitoring = true
}

// StopMonitoring simulates stopping network monitoring
func (m *MockNetworkMonitor) StopMonitoring() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.isMonitoring = false
}

// GetNetworkInfo returns mock network information
func (m *MockNetworkMonitor) GetNetworkInfo() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return map[string]interface{}{
		"status":       m.status.String(),
		"lastCheck":    m.lastCheck.Format(time.RFC3339),
		"timeout":      m.timeout.String(),
		"maxTimeout":   m.maxTimeout.String(),
		"isMonitoring": m.isMonitoring,
		"endpoints":    m.testEndpoints,
	}
}

// TestEndpointConnectivity simulates testing endpoint connectivity
func (m *MockNetworkMonitor) TestEndpointConnectivity(ctx context.Context, endpoint string, timeout time.Duration) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if result, exists := m.endpointResults[endpoint]; exists && !result {
		return fmt.Errorf("mock endpoint %s unreachable", endpoint)
	}

	return nil
}

// SetEndpointResult configures the result for a specific endpoint check
func (m *MockNetworkMonitor) SetEndpointResult(endpoint string, reachable bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.endpointResults[endpoint] = reachable
}

// GetCheckHistory returns the history of connectivity checks
func (m *MockNetworkMonitor) GetCheckHistory() []NetworkStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	history := make([]NetworkStatus, len(m.checkHistory))
	copy(history, m.checkHistory)
	return history
}

// IsMonitoring returns whether monitoring is active
func (m *MockNetworkMonitor) IsMonitoring() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.isMonitoring
}

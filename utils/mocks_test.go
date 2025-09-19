package utils

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestMockHTTPClient_BasicFunctionality(t *testing.T) {
	mock := NewMockHTTPClient()

	// Test setting and getting responses
	expectedBody := `{"message": "success"}`
	mock.SetResponse("POST", "https://example.com/api/test", http.StatusOK, expectedBody)

	// Create a mock request
	req := &http.Request{
		Method: "POST",
		URL:    &url.URL{Scheme: "https", Host: "example.com", Path: "/api/test"},
		Header: make(http.Header),
	}
	req.Header.Set("Authorization", "Bearer test-token")

	resp, err := mock.Do(req)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) != expectedBody {
		t.Errorf("Expected body %s, got %s", expectedBody, string(body))
	}

	// Verify request was recorded
	if mock.GetRequestCount("POST", "https://example.com/api/test") != 1 {
		t.Errorf("Expected 1 request, got %d", mock.GetRequestCount("POST", "https://example.com/api/test"))
	}

	lastReq := mock.GetLastMockRequest()
	if lastReq == nil {
		t.Fatal("Expected last request to be recorded")
	}

	if lastReq.Method != "POST" || !strings.Contains(lastReq.URL, "/api/test") {
		t.Errorf("Expected POST /api/test, got %s %s", lastReq.Method, lastReq.URL)
	}
}

func TestMockHTTPClient_ErrorHandling(t *testing.T) {
	mock := NewMockHTTPClient()
	helpers := NewTestHelpers()

	// Test error responses
	expectedError := helpers.CreateNetworkError("connection refused")
	mock.SetError("POST", "https://example.com/api/error", expectedError)

	req := &http.Request{
		Method: "POST",
		URL:    &url.URL{Scheme: "https", Host: "example.com", Path: "/api/error"},
		Header: make(http.Header),
	}

	_, err := mock.Do(req)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if !strings.Contains(err.Error(), "connection refused") {
		t.Errorf("Expected error to contain 'connection refused', got: %v", err)
	}
}

func TestMockHTTPClient_DelaySimulation(t *testing.T) {
	mock := NewMockHTTPClient()

	// Set up delay
	delay := 50 * time.Millisecond
	mock.SetDelay("GET", "https://example.com/api/slow", delay)
	mock.SetResponse("GET", "https://example.com/api/slow", http.StatusOK, `{"slow": "response"}`)

	req := &http.Request{
		Method: "GET",
		URL:    &url.URL{Scheme: "https", Host: "example.com", Path: "/api/slow"},
		Header: make(http.Header),
	}
	req = req.WithContext(context.Background())

	start := time.Now()
	_, err := mock.Do(req)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if elapsed < delay {
		t.Errorf("Expected delay of at least %v, got %v", delay, elapsed)
	}
}

func TestMockHTTPClient_ContextCancellation(t *testing.T) {
	mock := NewMockHTTPClient()

	// Set up a long delay
	mock.SetDelay("GET", "https://example.com/api/timeout", 5*time.Second)

	req := &http.Request{
		Method: "GET",
		URL:    &url.URL{Scheme: "https", Host: "example.com", Path: "/api/timeout"},
		Header: make(http.Header),
	}

	// Create context that cancels quickly
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	req = req.WithContext(ctx)

	_, err := mock.Do(req)
	if err == nil {
		t.Fatal("Expected context cancellation error, got nil")
	}

	if err != context.DeadlineExceeded {
		t.Errorf("Expected context.DeadlineExceeded, got: %v", err)
	}
}

func TestMockHTTPClient_DefaultResponses(t *testing.T) {
	mock := NewMockHTTPClient()

	// Set default response
	mock.SetDefaultResponse(http.StatusOK, `{"default": "response"}`)

	req := &http.Request{
		Method: "GET",
		URL:    &url.URL{Scheme: "https", Host: "example.com", Path: "/api/unknown"},
		Header: make(http.Header),
	}

	resp, err := mock.Do(req)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) != `{"default": "response"}` {
		t.Errorf("Expected default response body, got: %s", string(body))
	}
}

func TestMockHTTPClient_Reset(t *testing.T) {
	mock := NewMockHTTPClient()

	// Set up some state
	mock.SetResponse("GET", "https://example.com/api/test", http.StatusOK, "test")
	req := &http.Request{
		Method: "GET",
		URL:    &url.URL{Scheme: "https", Host: "example.com", Path: "/api/test"},
		Header: make(http.Header),
	}
	mock.Do(req)

	// Verify state exists
	if mock.GetRequestCount("GET", "https://example.com/api/test") != 1 {
		t.Fatal("Expected request count to be 1 before reset")
	}

	// Reset and verify state is cleared
	mock.Reset()

	if mock.GetRequestCount("GET", "https://example.com/api/test") != 0 {
		t.Error("Expected request count to be 0 after reset")
	}

	if len(mock.GetAllRequests()) != 0 {
		t.Error("Expected request history to be empty after reset")
	}
}

func TestMockNetworkMonitor_BasicFunctionality(t *testing.T) {
	mock := NewMockNetworkMonitor()

	// Test initial state
	if mock.GetStatus() != NetworkStatusOnline {
		t.Errorf("Expected initial status to be online, got %s", mock.GetStatus().String())
	}

	if mock.IsOffline() {
		t.Error("Expected IsOffline to be false initially")
	}

	if !mock.IsOnline() {
		t.Error("Expected IsOnline to be true initially")
	}

	// Test status change
	mock.SetStatus(NetworkStatusOffline)

	if mock.GetStatus() != NetworkStatusOffline {
		t.Errorf("Expected status to be offline, got %s", mock.GetStatus().String())
	}

	if !mock.IsOffline() {
		t.Error("Expected IsOffline to be true after setting offline")
	}

	if mock.IsOnline() {
		t.Error("Expected IsOnline to be false after setting offline")
	}
}

func TestMockNetworkMonitor_StatusCallbacks(t *testing.T) {
	mock := NewMockNetworkMonitor()

	// Set up callback
	callbackCalled := false
	var receivedStatus NetworkStatus

	mock.AddStatusCallback(func(status NetworkStatus) {
		callbackCalled = true
		receivedStatus = status
	})

	// Change status
	mock.SetStatus(NetworkStatusLimited)

	// Give callback time to execute
	time.Sleep(10 * time.Millisecond)

	if !callbackCalled {
		t.Error("Expected callback to be called")
	}

	if receivedStatus != NetworkStatusLimited {
		t.Errorf("Expected callback to receive NetworkStatusLimited, got %s", receivedStatus.String())
	}
}

func TestMockNetworkMonitor_EndpointTesting(t *testing.T) {
	mock := NewMockNetworkMonitor()

	// Set endpoint results
	mock.SetEndpointResult("https://api.example.com", false)

	err := mock.TestEndpointConnectivity(context.Background(), "https://api.example.com", 5*time.Second)
	if err == nil {
		t.Error("Expected error for unreachable endpoint")
	}

	if !strings.Contains(err.Error(), "unreachable") {
		t.Errorf("Expected error to contain 'unreachable', got: %v", err)
	}

	// Test reachable endpoint
	mock.SetEndpointResult("https://api.example.com", true)

	err = mock.TestEndpointConnectivity(context.Background(), "https://api.example.com", 5*time.Second)
	if err != nil {
		t.Errorf("Expected no error for reachable endpoint, got: %v", err)
	}
}

func TestTestHelpers_ResponseCreation(t *testing.T) {
	helpers := NewTestHelpers()

	// Test success response
	successResp := helpers.CreateSuccessResponse(`{"test": "data"}`)
	if successResp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, successResp.StatusCode)
	}

	if string(successResp.Body) != `{"test": "data"}` {
		t.Errorf("Expected body to match, got: %s", string(successResp.Body))
	}

	// Test error response
	errorResp := helpers.CreateErrorResponse(http.StatusBadRequest, "Bad request")
	if errorResp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, errorResp.StatusCode)
	}

	if !strings.Contains(string(errorResp.Body), "Bad request") {
		t.Errorf("Expected body to contain error message, got: %s", string(errorResp.Body))
	}
}

func TestTestHelpers_MockSetup(t *testing.T) {
	helpers := NewTestHelpers()
	mock := NewMockHTTPClient()

	// Test success setup
	responses := map[string]string{
		"https://example.com/api/test1": `{"result": "success1"}`,
		"https://example.com/api/test2": `{"result": "success2"}`,
	}

	helpers.SetupMockHTTPClientForSuccess(mock, responses)

	// Verify responses are set
	req1 := &http.Request{
		Method: "POST",
		URL:    &url.URL{Scheme: "https", Host: "example.com", Path: "/api/test1"},
		Header: make(http.Header),
	}
	resp1, err := mock.Do(req1)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	body1, _ := io.ReadAll(resp1.Body)
	if string(body1) != `{"result": "success1"}` {
		t.Errorf("Expected success1 response, got: %s", string(body1))
	}

	// Test error setup
	errors := map[string]int{
		"https://example.com/api/error1": http.StatusInternalServerError,
		"https://example.com/api/error2": http.StatusBadRequest,
	}

	helpers.SetupMockHTTPClientForErrors(mock, errors)

	req2 := &http.Request{
		Method: "POST",
		URL:    &url.URL{Scheme: "https", Host: "example.com", Path: "/api/error1"},
		Header: make(http.Header),
	}
	resp2, err := mock.Do(req2)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if resp2.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected status code %d, got %d", http.StatusInternalServerError, resp2.StatusCode)
	}
}

func TestTestHelpers_NetworkMonitorSetup(t *testing.T) {
	helpers := NewTestHelpers()
	mock := NewMockNetworkMonitor()

	// Test online setup
	helpers.SetupMockNetworkMonitorOnline(mock)
	if mock.GetStatus() != NetworkStatusOnline {
		t.Errorf("Expected online status, got %s", mock.GetStatus().String())
	}

	// Test offline setup
	helpers.SetupMockNetworkMonitorOffline(mock)
	if mock.GetStatus() != NetworkStatusOffline {
		t.Errorf("Expected offline status, got %s", mock.GetStatus().String())
	}

	// Test limited setup
	helpers.SetupMockNetworkMonitorLimited(mock)
	if mock.GetStatus() != NetworkStatusLimited {
		t.Errorf("Expected limited status, got %s", mock.GetStatus().String())
	}
}

func TestTestHelpers_Assertions(t *testing.T) {
	helpers := NewTestHelpers()
	mock := NewMockHTTPClient()

	// Make a request
	req := &http.Request{
		Method: "GET",
		URL:    &url.URL{Scheme: "https", Host: "example.com", Path: "/api/test"},
		Header: make(http.Header),
	}
	mock.Do(req)

	// Test assertion success
	err := helpers.AssertRequestMade(mock, "GET", "https://example.com/api/test")
	if err != nil {
		t.Errorf("Expected assertion to pass, got error: %v", err)
	}

	// Test assertion failure
	err = helpers.AssertRequestMade(mock, "POST", "https://example.com/api/nonexistent")
	if err == nil {
		t.Error("Expected assertion to fail for non-existent request")
	}

	// Test count assertion
	err = helpers.AssertRequestCount(mock, "GET", "https://example.com/api/test", 1)
	if err != nil {
		t.Errorf("Expected count assertion to pass, got error: %v", err)
	}

	err = helpers.AssertRequestCount(mock, "GET", "https://example.com/api/test", 2)
	if err == nil {
		t.Error("Expected count assertion to fail for wrong count")
	}
}

func TestTestHelpers_SpecialResponses(t *testing.T) {
	helpers := NewTestHelpers()

	// Test Claude response
	claudeResp := helpers.CreateMockClaudeResponse("Hello from Claude")
	if !strings.Contains(string(claudeResp.Body), "Hello from Claude") {
		t.Error("Expected Claude response to contain the message")
	}

	if !strings.Contains(string(claudeResp.Body), "claude-3-sonnet") {
		t.Error("Expected Claude response to contain model name")
	}

	// Test OpenAI response
	openaiResp := helpers.CreateMockOpenAIResponse("Hello from OpenAI")
	if !strings.Contains(string(openaiResp.Body), "Hello from OpenAI") {
		t.Error("Expected OpenAI response to contain the message")
	}

	if !strings.Contains(string(openaiResp.Body), "gpt-4o-mini") {
		t.Error("Expected OpenAI response to contain model name")
	}

	// Test rate limit response
	rateLimitResp := helpers.CreateMockRateLimitResponse()
	if rateLimitResp.StatusCode != http.StatusTooManyRequests {
		t.Errorf("Expected status code %d, got %d", http.StatusTooManyRequests, rateLimitResp.StatusCode)
	}

	// Test auth error response
	authErrorResp := helpers.CreateMockAuthErrorResponse()
	if authErrorResp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status code %d, got %d", http.StatusUnauthorized, authErrorResp.StatusCode)
	}

	// Test server error response
	serverErrorResp := helpers.CreateMockServerErrorResponse()
	if serverErrorResp.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected status code %d, got %d", http.StatusInternalServerError, serverErrorResp.StatusCode)
	}
}

// Example integration test showing how to use mocks together
func TestMockIntegration_HTTPClientWithNetworkMonitor(t *testing.T) {
	helpers := NewTestHelpers()
	httpMock := NewMockHTTPClient()
	networkMock := NewMockNetworkMonitor()

	// Set up network as offline
	helpers.SetupMockNetworkMonitorOffline(networkMock)

	// Set up HTTP client to simulate network dependency
	// In a real scenario, the HTTP client would check the network monitor
	// For this test, we'll simulate the behavior
	if networkMock.IsOffline() {
		httpMock.SetError("POST", "https://example.com/api/test", fmt.Errorf("network is offline"))
	}

	req := &http.Request{
		Method: "POST",
		URL:    &url.URL{Scheme: "https", Host: "example.com", Path: "/api/test"},
		Header: make(http.Header),
	}

	_, err := httpMock.Do(req)
	if err == nil {
		t.Error("Expected error when network is offline")
	}

	if !strings.Contains(err.Error(), "offline") {
		t.Errorf("Expected error to mention offline status, got: %v", err)
	}

	// Now bring network online
	helpers.SetupMockNetworkMonitorOnline(networkMock)
	httpMock.SetResponse("POST", "https://example.com/api/test", http.StatusOK, `{"status": "online"}`)

	resp, err := httpMock.Do(req)
	if err != nil {
		t.Fatalf("Expected no error when network is online, got: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected success response when network is online, got status: %d", resp.StatusCode)
	}
}

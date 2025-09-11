package utils

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// DemoMockInfrastructure demonstrates the mock infrastructure functionality
// This function can be called from other parts of the codebase to verify mocks work
func DemoMockInfrastructure() error {
	// Create mock HTTP client
	mockHTTP := NewMockHTTPClient()
	helpers := NewTestHelpers()

	// Set up a mock response
	mockHTTP.SetResponse("GET", "https://api.example.com/test", http.StatusOK, `{"demo": "success"}`)

	// Create a test request
	req := &http.Request{
		Method: "GET",
		URL:    &url.URL{Scheme: "https", Host: "api.example.com", Path: "/test"},
		Header: make(http.Header),
	}

	// Make the request
	resp, err := mockHTTP.Do(req)
	if err != nil {
		return fmt.Errorf("mock HTTP request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	// Verify request was recorded
	if mockHTTP.GetRequestCount("GET", "https://api.example.com/test") != 1 {
		return fmt.Errorf("request count not recorded correctly")
	}

	// Create mock network monitor
	mockNetwork := NewMockNetworkMonitor()

	// Test network status changes
	if !mockNetwork.IsOnline() {
		return fmt.Errorf("expected network to be online initially")
	}

	mockNetwork.SetStatus(NetworkStatusOffline)
	if !mockNetwork.IsOffline() {
		return fmt.Errorf("expected network to be offline after setting status")
	}

	// Test helpers
	successResp := helpers.CreateSuccessResponse(`{"helper": "works"}`)
	if successResp.StatusCode != http.StatusOK {
		return fmt.Errorf("test helper failed to create success response")
	}

	errorResp := helpers.CreateErrorResponse(http.StatusBadRequest, "test error")
	if errorResp.StatusCode != http.StatusBadRequest {
		return fmt.Errorf("test helper failed to create error response")
	}

	// Test network monitor setup helpers
	helpers.SetupMockNetworkMonitorOnline(mockNetwork)
	if !mockNetwork.IsOnline() {
		return fmt.Errorf("helper failed to set network online")
	}

	helpers.SetupMockNetworkMonitorOffline(mockNetwork)
	if !mockNetwork.IsOffline() {
		return fmt.Errorf("helper failed to set network offline")
	}

	// Test HTTP client setup helpers
	responses := map[string]string{
		"https://api.example.com/success": `{"status": "ok"}`,
	}
	helpers.SetupMockHTTPClientForSuccess(mockHTTP, responses)

	// Verify the setup worked
	req2 := &http.Request{
		Method: "POST",
		URL:    &url.URL{Scheme: "https", Host: "api.example.com", Path: "/success"},
		Header: make(http.Header),
	}
	resp2, err := mockHTTP.Do(req2)
	if err != nil {
		return fmt.Errorf("helper setup failed: %w", err)
	}

	if resp2.StatusCode != http.StatusOK {
		return fmt.Errorf("helper setup response incorrect: got %d", resp2.StatusCode)
	}

	// Test assertions
	err = helpers.AssertRequestMade(mockHTTP, "POST", "https://api.example.com/success")
	if err != nil {
		return fmt.Errorf("assertion failed: %w", err)
	}

	err = helpers.AssertRequestCount(mockHTTP, "POST", "https://api.example.com/success", 1)
	if err != nil {
		return fmt.Errorf("count assertion failed: %w", err)
	}

	return nil
}

// DemoConfigurableErrorResponses demonstrates configurable error responses in mocks
func DemoConfigurableErrorResponses() error {
	mockHTTP := NewMockHTTPClient()
	helpers := NewTestHelpers()

	// Test different error scenarios
	testCases := []struct {
		name       string
		statusCode int
		pattern    string
	}{
		{"Rate Limit", http.StatusTooManyRequests, "https://api.example.com/ratelimit"},
		{"Auth Error", http.StatusUnauthorized, "https://api.example.com/auth"},
		{"Server Error", http.StatusInternalServerError, "https://api.example.com/server"},
		{"Bad Request", http.StatusBadRequest, "https://api.example.com/bad"},
	}

	for _, tc := range testCases {
		// Set up error response
		mockHTTP.SetResponse("POST", tc.pattern, tc.statusCode, fmt.Sprintf(`{"error": "%s"}`, tc.name))

		// Make request
		req := &http.Request{
			Method: "POST",
			URL:    mustParseURL(tc.pattern),
			Header: make(http.Header),
		}

		resp, err := mockHTTP.Do(req)
		if err != nil {
			return fmt.Errorf("error response test failed for %s: %w", tc.name, err)
		}

		if resp.StatusCode != tc.statusCode {
			return fmt.Errorf("expected status %d for %s, got %d", tc.statusCode, tc.name, resp.StatusCode)
		}
	}

	// Test network errors
	networkPatterns := []string{
		"https://api.example.com/network1",
		"https://api.example.com/network2",
	}
	helpers.SetupMockHTTPClientForNetworkErrors(mockHTTP, networkPatterns)

	for _, pattern := range networkPatterns {
		req := &http.Request{
			Method: "GET",
			URL:    mustParseURL(pattern),
			Header: make(http.Header),
		}

		_, err := mockHTTP.Do(req)
		if err == nil {
			return fmt.Errorf("expected network error for %s", pattern)
		}
	}

	// Test delay simulation
	mockHTTP.SetDelay("GET", "https://api.example.com/slow", 10) // Very short delay for demo
	mockHTTP.SetResponse("GET", "https://api.example.com/slow", http.StatusOK, `{"slow": "response"}`)

	req := &http.Request{
		Method: "GET",
		URL:    mustParseURL("https://api.example.com/slow"),
		Header: make(http.Header),
	}
	req = req.WithContext(context.Background())

	_, err := mockHTTP.Do(req)
	if err != nil {
		return fmt.Errorf("delay simulation failed: %w", err)
	}

	return nil
}

// mustParseURL is a helper function for the demo
func mustParseURL(rawURL string) *url.URL {
	u, err := url.Parse(rawURL)
	if err != nil {
		panic(err)
	}
	return u
}

// DemoSpecialAPIResponses demonstrates mock responses for specific AI APIs
func DemoSpecialAPIResponses() error {
	helpers := NewTestHelpers()

	// Test Claude API response
	claudeResp := helpers.CreateMockClaudeResponse("Hello from Claude mock")
	if claudeResp.StatusCode != http.StatusOK {
		return fmt.Errorf("the Claude response has wrong status: %d", claudeResp.StatusCode)
	}

	// Test OpenAI API response
	openaiResp := helpers.CreateMockOpenAIResponse("Hello from OpenAI mock")
	if openaiResp.StatusCode != http.StatusOK {
		return fmt.Errorf("the OpenAI response has wrong status: %d", openaiResp.StatusCode)
	}

	// Test rate limit response
	rateLimitResp := helpers.CreateMockRateLimitResponse()
	if rateLimitResp.StatusCode != http.StatusTooManyRequests {
		return fmt.Errorf("rate limit response has wrong status: %d", rateLimitResp.StatusCode)
	}

	// Test auth error response
	authErrorResp := helpers.CreateMockAuthErrorResponse()
	if authErrorResp.StatusCode != http.StatusUnauthorized {
		return fmt.Errorf("auth error response has wrong status: %d", authErrorResp.StatusCode)
	}

	// Test server error response
	serverErrorResp := helpers.CreateMockServerErrorResponse()
	if serverErrorResp.StatusCode != http.StatusInternalServerError {
		return fmt.Errorf("server error response has wrong status: %d", serverErrorResp.StatusCode)
	}

	return nil
}

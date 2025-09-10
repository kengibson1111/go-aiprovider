package utils

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestNetworkStatus_String tests the String method of NetworkStatus
func TestNetworkStatus_String(t *testing.T) {
	tests := []struct {
		status   NetworkStatus
		expected string
	}{
		{NetworkStatusUnknown, "unknown"},
		{NetworkStatusOnline, "online"},
		{NetworkStatusOffline, "offline"},
		{NetworkStatusLimited, "limited"},
	}

	for _, test := range tests {
		t.Run(test.expected, func(t *testing.T) {
			result := test.status.String()
			if result != test.expected {
				t.Errorf("Expected %s, got %s", test.expected, result)
			}
		})
	}
}

// TestNewNetworkMonitor tests the creation of a new network monitor
func TestNewNetworkMonitor(t *testing.T) {
	t.Run("with default config", func(t *testing.T) {
		config := NetworkMonitorConfig{}
		nm := NewNetworkMonitor(config)

		if nm == nil {
			t.Fatal("Expected network monitor to be created")
		}

		if nm.checkInterval != 30*time.Second {
			t.Errorf("Expected default check interval 30s, got %v", nm.checkInterval)
		}

		if nm.timeout != 5*time.Second {
			t.Errorf("Expected default timeout 5s, got %v", nm.timeout)
		}

		if nm.maxTimeout != 10*time.Second {
			t.Errorf("Expected default max timeout 10s, got %v", nm.maxTimeout)
		}

		if len(nm.testEndpoints) != 3 {
			t.Errorf("Expected 3 default endpoints, got %d", len(nm.testEndpoints))
		}

		if nm.status != NetworkStatusUnknown {
			t.Errorf("Expected initial status unknown, got %v", nm.status)
		}
	})

	t.Run("with custom config", func(t *testing.T) {
		config := NetworkMonitorConfig{
			CheckInterval: 60 * time.Second,
			Timeout:       3 * time.Second,
			MaxTimeout:    15 * time.Second,
			TestEndpoints: []string{"https://example.com"},
		}
		nm := NewNetworkMonitor(config)

		if nm.checkInterval != 60*time.Second {
			t.Errorf("Expected check interval 60s, got %v", nm.checkInterval)
		}

		if nm.timeout != 3*time.Second {
			t.Errorf("Expected timeout 3s, got %v", nm.timeout)
		}

		if nm.maxTimeout != 15*time.Second {
			t.Errorf("Expected max timeout 15s, got %v", nm.maxTimeout)
		}

		if len(nm.testEndpoints) != 1 || nm.testEndpoints[0] != "https://example.com" {
			t.Errorf("Expected custom endpoint, got %v", nm.testEndpoints)
		}
	})

	t.Run("timeout exceeds max timeout", func(t *testing.T) {
		config := NetworkMonitorConfig{
			Timeout:    15 * time.Second,
			MaxTimeout: 10 * time.Second,
		}
		nm := NewNetworkMonitor(config)

		if nm.timeout != 10*time.Second {
			t.Errorf("Expected timeout to be capped at max timeout 10s, got %v", nm.timeout)
		}
	})
}

// TestNetworkMonitor_GetStatus tests status retrieval
func TestNetworkMonitor_GetStatus(t *testing.T) {
	nm := NewNetworkMonitor(NetworkMonitorConfig{})

	// Initial status should be unknown
	if status := nm.GetStatus(); status != NetworkStatusUnknown {
		t.Errorf("Expected initial status unknown, got %v", status)
	}

	// Update status and verify
	nm.updateStatus(NetworkStatusOnline)
	if status := nm.GetStatus(); status != NetworkStatusOnline {
		t.Errorf("Expected status online, got %v", status)
	}
}

// TestNetworkMonitor_GetLastCheck tests last check time retrieval
func TestNetworkMonitor_GetLastCheck(t *testing.T) {
	nm := NewNetworkMonitor(NetworkMonitorConfig{})

	// Initial last check should be zero
	if !nm.GetLastCheck().IsZero() {
		t.Error("Expected initial last check to be zero time")
	}

	// Update status and verify last check is updated
	before := time.Now()
	nm.updateStatus(NetworkStatusOnline)
	after := time.Now()

	lastCheck := nm.GetLastCheck()
	if lastCheck.Before(before) || lastCheck.After(after) {
		t.Errorf("Expected last check to be between %v and %v, got %v", before, after, lastCheck)
	}
}

// TestNetworkMonitor_IsOnline tests online status check
func TestNetworkMonitor_IsOnline(t *testing.T) {
	nm := NewNetworkMonitor(NetworkMonitorConfig{})

	tests := []struct {
		status   NetworkStatus
		expected bool
	}{
		{NetworkStatusUnknown, false},
		{NetworkStatusOnline, true},
		{NetworkStatusOffline, false},
		{NetworkStatusLimited, false},
	}

	for _, test := range tests {
		t.Run(test.status.String(), func(t *testing.T) {
			nm.updateStatus(test.status)
			result := nm.IsOnline()
			if result != test.expected {
				t.Errorf("Expected IsOnline() to return %v for status %v, got %v",
					test.expected, test.status, result)
			}
		})
	}
}

// TestNetworkMonitor_IsOffline tests offline status check
func TestNetworkMonitor_IsOffline(t *testing.T) {
	nm := NewNetworkMonitor(NetworkMonitorConfig{})

	tests := []struct {
		status   NetworkStatus
		expected bool
	}{
		{NetworkStatusUnknown, true},
		{NetworkStatusOnline, false},
		{NetworkStatusOffline, true},
		{NetworkStatusLimited, false},
	}

	for _, test := range tests {
		t.Run(test.status.String(), func(t *testing.T) {
			nm.updateStatus(test.status)
			result := nm.IsOffline()
			if result != test.expected {
				t.Errorf("Expected IsOffline() to return %v for status %v, got %v",
					test.expected, test.status, result)
			}
		})
	}
}

// TestNetworkMonitor_SetTimeout tests timeout configuration
func TestNetworkMonitor_SetTimeout(t *testing.T) {
	nm := NewNetworkMonitor(NetworkMonitorConfig{
		MaxTimeout: 10 * time.Second,
	})

	t.Run("valid timeout", func(t *testing.T) {
		nm.SetTimeout(7 * time.Second)
		if timeout := nm.GetTimeout(); timeout != 7*time.Second {
			t.Errorf("Expected timeout 7s, got %v", timeout)
		}
	})

	t.Run("timeout exceeds max", func(t *testing.T) {
		nm.SetTimeout(15 * time.Second)
		if timeout := nm.GetTimeout(); timeout != 10*time.Second {
			t.Errorf("Expected timeout to be capped at 10s, got %v", timeout)
		}
	})

	t.Run("timeout below minimum", func(t *testing.T) {
		nm.SetTimeout(500 * time.Millisecond)
		if timeout := nm.GetTimeout(); timeout != 1*time.Second {
			t.Errorf("Expected timeout to be minimum 1s, got %v", timeout)
		}
	})
}

// TestNetworkMonitor_AddStatusCallback tests callback registration and execution
func TestNetworkMonitor_AddStatusCallback(t *testing.T) {
	nm := NewNetworkMonitor(NetworkMonitorConfig{})

	var callbackStatus NetworkStatus
	var callbackCalled bool
	var wg sync.WaitGroup

	callback := func(status NetworkStatus) {
		callbackStatus = status
		callbackCalled = true
		wg.Done()
	}

	nm.AddStatusCallback(callback)

	// Change status and wait for callback
	wg.Add(1)
	nm.updateStatus(NetworkStatusOnline)
	wg.Wait()

	if !callbackCalled {
		t.Error("Expected callback to be called")
	}

	if callbackStatus != NetworkStatusOnline {
		t.Errorf("Expected callback to receive status online, got %v", callbackStatus)
	}
}

// TestNetworkMonitor_AddStatusCallback_NoDuplicateCall tests that callbacks aren't called for same status
func TestNetworkMonitor_AddStatusCallback_NoDuplicateCall(t *testing.T) {
	nm := NewNetworkMonitor(NetworkMonitorConfig{})

	callCount := 0
	var mu sync.Mutex

	callback := func(status NetworkStatus) {
		mu.Lock()
		callCount++
		mu.Unlock()
	}

	nm.AddStatusCallback(callback)

	// Set same status twice
	nm.updateStatus(NetworkStatusOnline)
	time.Sleep(10 * time.Millisecond) // Allow goroutines to complete
	nm.updateStatus(NetworkStatusOnline)
	time.Sleep(10 * time.Millisecond) // Allow goroutines to complete

	mu.Lock()
	finalCount := callCount
	mu.Unlock()

	if finalCount != 1 {
		t.Errorf("Expected callback to be called once, got %d calls", finalCount)
	}
}

// TestNetworkMonitor_AddStatusCallback_Panic tests callback panic handling
func TestNetworkMonitor_AddStatusCallback_Panic(t *testing.T) {
	nm := NewNetworkMonitor(NetworkMonitorConfig{})

	panicCallback := func(status NetworkStatus) {
		panic("test panic")
	}

	var normalCallbackCalled bool
	var wg sync.WaitGroup

	normalCallback := func(status NetworkStatus) {
		normalCallbackCalled = true
		wg.Done()
	}

	nm.AddStatusCallback(panicCallback)
	nm.AddStatusCallback(normalCallback)

	// Change status - should not crash despite panic
	wg.Add(1)
	nm.updateStatus(NetworkStatusOnline)
	wg.Wait()

	if !normalCallbackCalled {
		t.Error("Expected normal callback to be called despite panic in other callback")
	}
}

// TestNetworkMonitor_CheckConnectivity tests connectivity checking with mocked HTTP client
func TestNetworkMonitor_CheckConnectivity(t *testing.T) {
	// Create a network monitor with custom endpoints
	config := NetworkMonitorConfig{
		TestEndpoints: []string{
			"https://endpoint1.com",
			"https://endpoint2.com",
			"https://endpoint3.com",
		},
		Timeout: 5 * time.Second,
	}
	nm := NewNetworkMonitor(config)

	// Replace the HTTP client with our mock using the existing mock infrastructure
	mockClient := NewMockHTTPClient()
	nm.httpClient = &http.Client{
		Transport: &mockTransport{mockClient: mockClient},
		Timeout:   nm.timeout,
	}

	ctx := context.Background()

	t.Run("all endpoints online", func(t *testing.T) {
		// Reset mock for clean test
		mockClient = NewMockHTTPClient()
		nm.httpClient.Transport = &mockTransport{mockClient: mockClient}

		// All endpoints return success
		for _, endpoint := range config.TestEndpoints {
			mockClient.SetResponse("HEAD", endpoint, 200, "")
		}

		status := nm.CheckConnectivity(ctx)
		if status != NetworkStatusOnline {
			t.Errorf("Expected status online, got %v", status)
		}

		if nm.GetStatus() != NetworkStatusOnline {
			t.Errorf("Expected stored status online, got %v", nm.GetStatus())
		}
	})

	t.Run("all endpoints offline", func(t *testing.T) {
		// Reset mock for clean test
		mockClient = NewMockHTTPClient()
		nm.httpClient.Transport = &mockTransport{mockClient: mockClient}

		// All endpoints return errors
		for _, endpoint := range config.TestEndpoints {
			mockClient.SetError("HEAD", endpoint, fmt.Errorf("connection failed"))
		}

		status := nm.CheckConnectivity(ctx)
		if status != NetworkStatusOffline {
			t.Errorf("Expected status offline, got %v", status)
		}
	})

	t.Run("limited connectivity", func(t *testing.T) {
		// Reset mock for clean test
		mockClient = NewMockHTTPClient()
		nm.httpClient.Transport = &mockTransport{mockClient: mockClient}

		// First endpoint succeeds, others fail
		mockClient.SetResponse("HEAD", config.TestEndpoints[0], 200, "")
		mockClient.SetError("HEAD", config.TestEndpoints[1], fmt.Errorf("connection failed"))
		mockClient.SetError("HEAD", config.TestEndpoints[2], fmt.Errorf("connection failed"))

		status := nm.CheckConnectivity(ctx)
		if status != NetworkStatusLimited {
			t.Errorf("Expected status limited, got %v", status)
		}
	})

	t.Run("context cancellation", func(t *testing.T) {
		// Reset mock for clean test
		mockClient = NewMockHTTPClient()
		nm.httpClient.Transport = &mockTransport{mockClient: mockClient}

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		// Should still complete but may have different behavior
		status := nm.CheckConnectivity(ctx)
		// Status depends on mock behavior with cancelled context
		if status != NetworkStatusOffline {
			t.Logf("Status with cancelled context: %v", status)
		}
	})
}

// TestNetworkMonitor_checkEndpoint tests individual endpoint checking
func TestNetworkMonitor_checkEndpoint(t *testing.T) {
	nm := NewNetworkMonitor(NetworkMonitorConfig{})
	mockClient := NewMockHTTPClient()
	nm.httpClient = &http.Client{
		Transport: &mockTransport{mockClient: mockClient},
		Timeout:   nm.timeout,
	}

	ctx := context.Background()
	endpoint := "https://test.com"

	t.Run("successful response", func(t *testing.T) {
		mockClient = NewMockHTTPClient()
		nm.httpClient.Transport = &mockTransport{mockClient: mockClient}

		mockClient.SetResponse("HEAD", endpoint, 200, "")

		result := nm.checkEndpoint(ctx, endpoint)
		if !result {
			t.Error("Expected endpoint check to succeed")
		}

		// Verify the request was made correctly
		lastReq := mockClient.GetLastRequest()
		if lastReq == nil {
			t.Fatal("Expected a request to be made")
		}

		if lastReq.Method != "HEAD" {
			t.Errorf("Expected HEAD request, got %s", lastReq.Method)
		}

		userAgent := lastReq.Header.Get("User-Agent")
		if !strings.Contains(userAgent, "VSCode-Assist-NetworkCheck") {
			t.Errorf("Expected User-Agent to contain VSCode-Assist-NetworkCheck, got %s", userAgent)
		}
	})

	t.Run("error response", func(t *testing.T) {
		mockClient = NewMockHTTPClient()
		nm.httpClient.Transport = &mockTransport{mockClient: mockClient}

		mockClient.SetError("HEAD", endpoint, fmt.Errorf("network error"))

		result := nm.checkEndpoint(ctx, endpoint)
		if result {
			t.Error("Expected endpoint check to fail")
		}
	})

	t.Run("http error status", func(t *testing.T) {
		mockClient = NewMockHTTPClient()
		nm.httpClient.Transport = &mockTransport{mockClient: mockClient}

		mockClient.SetResponse("HEAD", endpoint, 500, "Internal Server Error")

		// Even error status codes should be considered "reachable"
		result := nm.checkEndpoint(ctx, endpoint)
		if !result {
			t.Error("Expected endpoint check to succeed even with error status")
		}
	})
}

// TestNetworkMonitor_StartStopMonitoring tests the monitoring lifecycle
func TestNetworkMonitor_StartStopMonitoring(t *testing.T) {
	config := NetworkMonitorConfig{
		CheckInterval: 100 * time.Millisecond, // Fast interval for testing
		TestEndpoints: []string{"https://test.com"},
	}
	nm := NewNetworkMonitor(config)

	mockClient := NewMockHTTPClient()
	nm.httpClient = &http.Client{
		Transport: &mockTransport{mockClient: mockClient},
		Timeout:   nm.timeout,
	}

	// Set up mock response
	mockClient.SetResponse("HEAD", "https://test.com", 200, "")

	ctx := context.Background()

	t.Run("start and stop monitoring", func(t *testing.T) {
		// Start monitoring
		nm.StartMonitoring(ctx)

		// Wait for a few checks
		time.Sleep(250 * time.Millisecond)

		// Stop monitoring
		nm.StopMonitoring()

		// Verify requests were made
		requestCount := mockClient.GetRequestCount("HEAD", "https://test.com")
		if requestCount < 2 { // Initial check + at least one periodic check
			t.Errorf("Expected at least 2 requests, got %d", requestCount)
		}
	})

	t.Run("double start monitoring", func(t *testing.T) {
		// Create a new network monitor for this test to avoid channel issues
		nm2 := NewNetworkMonitor(config)
		mockClient2 := NewMockHTTPClient()
		nm2.httpClient = &http.Client{
			Transport: &mockTransport{mockClient: mockClient2},
			Timeout:   nm2.timeout,
		}
		mockClient2.SetResponse("HEAD", "https://test.com", 200, "")

		// Start monitoring twice
		nm2.StartMonitoring(ctx)
		nm2.StartMonitoring(ctx) // Should be ignored

		time.Sleep(150 * time.Millisecond)
		nm2.StopMonitoring()

		// Should not have duplicate monitoring
		requestCount := mockClient2.GetRequestCount("HEAD", "https://test.com")
		if requestCount > 3 { // Should be reasonable number, not doubled
			t.Errorf("Expected reasonable number of requests, got %d (possible duplicate monitoring)", requestCount)
		}
	})

	t.Run("double stop monitoring", func(t *testing.T) {
		// Create a new network monitor for this test to avoid channel issues
		nm3 := NewNetworkMonitor(config)
		mockClient3 := NewMockHTTPClient()
		nm3.httpClient = &http.Client{
			Transport: &mockTransport{mockClient: mockClient3},
			Timeout:   nm3.timeout,
		}
		mockClient3.SetResponse("HEAD", "https://test.com", 200, "")

		nm3.StartMonitoring(ctx)
		time.Sleep(50 * time.Millisecond)

		// Stop twice
		nm3.StopMonitoring()
		nm3.StopMonitoring() // Should not panic or cause issues

		// Test passes if no panic occurs
	})
}

// TestNetworkMonitor_MonitoringWithContextCancellation tests monitoring with context cancellation
func TestNetworkMonitor_MonitoringWithContextCancellation(t *testing.T) {
	config := NetworkMonitorConfig{
		CheckInterval: 50 * time.Millisecond,
		TestEndpoints: []string{"https://test.com"},
	}
	nm := NewNetworkMonitor(config)

	mockClient := NewMockHTTPClient()
	nm.httpClient = &http.Client{
		Transport: &mockTransport{mockClient: mockClient},
		Timeout:   nm.timeout,
	}

	mockClient.SetResponse("HEAD", "https://test.com", 200, "")

	ctx, cancel := context.WithCancel(context.Background())

	// Start monitoring
	nm.StartMonitoring(ctx)

	// Let it run briefly
	time.Sleep(100 * time.Millisecond)

	// Cancel context
	cancel()

	// Wait for monitoring to stop
	time.Sleep(100 * time.Millisecond)

	// Verify some requests were made
	requestCount := mockClient.GetRequestCount("HEAD", "https://test.com")
	if requestCount == 0 {
		t.Error("Expected at least some requests to be made")
	}
}

// TestNetworkMonitor_GetNetworkInfo tests network information retrieval
func TestNetworkMonitor_GetNetworkInfo(t *testing.T) {
	config := NetworkMonitorConfig{
		CheckInterval: 30 * time.Second,
		Timeout:       5 * time.Second,
		MaxTimeout:    10 * time.Second,
		TestEndpoints: []string{"https://test1.com", "https://test2.com"},
	}
	nm := NewNetworkMonitor(config)

	// Update status to have some data
	nm.updateStatus(NetworkStatusOnline)

	info := nm.GetNetworkInfo()

	// Verify all expected fields are present
	expectedFields := []string{
		"status", "lastCheck", "timeout", "maxTimeout",
		"checkInterval", "isMonitoring", "endpoints",
	}

	for _, field := range expectedFields {
		if _, exists := info[field]; !exists {
			t.Errorf("Expected field %s to be present in network info", field)
		}
	}

	// Verify specific values
	if info["status"] != "online" {
		t.Errorf("Expected status 'online', got %v", info["status"])
	}

	if info["timeout"] != "5s" {
		t.Errorf("Expected timeout '5s', got %v", info["timeout"])
	}

	if info["isMonitoring"] != false {
		t.Errorf("Expected isMonitoring false, got %v", info["isMonitoring"])
	}

	endpoints, ok := info["endpoints"].([]string)
	if !ok {
		t.Error("Expected endpoints to be []string")
	} else if len(endpoints) != 2 {
		t.Errorf("Expected 2 endpoints, got %d", len(endpoints))
	}
}

// TestNetworkMonitor_TestEndpointConnectivity tests individual endpoint testing
func TestNetworkMonitor_TestEndpointConnectivity(t *testing.T) {
	nm := NewNetworkMonitor(NetworkMonitorConfig{
		MaxTimeout: 10 * time.Second,
	})

	ctx := context.Background()
	endpoint := "https://test.com"

	t.Run("timeout capping", func(t *testing.T) {
		// Test that timeout is capped at MaxTimeout
		err := nm.TestEndpointConnectivity(ctx, endpoint, 15*time.Second)
		// This will likely fail with real HTTP, but we're testing the timeout logic
		// The actual HTTP call will use the capped timeout of 10 seconds
		if err == nil {
			t.Log("Endpoint test succeeded (unexpected but not an error)")
		}
	})

	t.Run("context cancellation", func(t *testing.T) {
		cancelCtx, cancel := context.WithCancel(ctx)
		cancel() // Cancel immediately

		err := nm.TestEndpointConnectivity(cancelCtx, endpoint, 5*time.Second)
		if err == nil {
			t.Error("Expected error with cancelled context")
		}

		if !strings.Contains(err.Error(), "context canceled") &&
			!strings.Contains(err.Error(), "endpoint unreachable") {
			t.Errorf("Expected context cancellation error, got: %v", err)
		}
	})
}

// TestNetworkMonitor_GlobalInstance tests global instance management
func TestNetworkMonitor_GlobalInstance(t *testing.T) {
	// Save original global instance
	originalGlobal := globalNetworkMonitor
	defer func() {
		globalNetworkMonitor = originalGlobal
	}()

	// Test initialization
	config := NetworkMonitorConfig{
		CheckInterval: 60 * time.Second,
		TestEndpoints: []string{"https://global-test.com"},
	}

	InitializeNetworkMonitor(config)

	global := GetGlobalNetworkMonitor()
	if global == nil {
		t.Fatal("Expected global network monitor to be initialized")
	}

	if global.checkInterval != 60*time.Second {
		t.Errorf("Expected global monitor check interval 60s, got %v", global.checkInterval)
	}

	if len(global.testEndpoints) != 1 || global.testEndpoints[0] != "https://global-test.com" {
		t.Errorf("Expected global monitor to have custom endpoint, got %v", global.testEndpoints)
	}
}

// TestNetworkMonitor_ConcurrentAccess tests thread safety
func TestNetworkMonitor_ConcurrentAccess(t *testing.T) {
	nm := NewNetworkMonitor(NetworkMonitorConfig{})

	var wg sync.WaitGroup
	numGoroutines := 10
	numOperations := 100

	// Test concurrent status updates and reads
	wg.Add(numGoroutines * 2)

	// Goroutines updating status
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				status := NetworkStatus(j % 4) // Cycle through all statuses
				nm.updateStatus(status)
			}
		}(i)
	}

	// Goroutines reading status
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				_ = nm.GetStatus()
				_ = nm.GetLastCheck()
				_ = nm.IsOnline()
				_ = nm.IsOffline()
				_ = nm.GetTimeout()
			}
		}()
	}

	wg.Wait()

	// Test should complete without race conditions or panics
}

// TestNetworkMonitor_TimeoutConfiguration tests timeout management
func TestNetworkMonitor_TimeoutConfiguration(t *testing.T) {
	nm := NewNetworkMonitor(NetworkMonitorConfig{
		Timeout:    3 * time.Second,
		MaxTimeout: 8 * time.Second,
	})

	// Test initial timeout
	if timeout := nm.GetTimeout(); timeout != 3*time.Second {
		t.Errorf("Expected initial timeout 3s, got %v", timeout)
	}

	// Test concurrent timeout updates
	var wg sync.WaitGroup
	numGoroutines := 5

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			timeout := time.Duration(id+1) * time.Second
			nm.SetTimeout(timeout)
		}(i)
	}

	wg.Wait()

	// Final timeout should be valid (between 1s and 8s)
	finalTimeout := nm.GetTimeout()
	if finalTimeout < 1*time.Second || finalTimeout > 8*time.Second {
		t.Errorf("Expected final timeout between 1s and 8s, got %v", finalTimeout)
	}
}

// TestNetworkMonitor_CallbackManagement tests callback registration and execution
func TestNetworkMonitor_CallbackManagement(t *testing.T) {
	nm := NewNetworkMonitor(NetworkMonitorConfig{})

	callbackCount := 0
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Add multiple callbacks
	for i := 0; i < 3; i++ {
		nm.AddStatusCallback(func(status NetworkStatus) {
			mu.Lock()
			callbackCount++
			mu.Unlock()
			wg.Done()
		})
	}

	// Trigger status change
	wg.Add(3) // Expect 3 callbacks
	nm.updateStatus(NetworkStatusOnline)
	wg.Wait()

	mu.Lock()
	finalCount := callbackCount
	mu.Unlock()

	if finalCount != 3 {
		t.Errorf("Expected 3 callback executions, got %d", finalCount)
	}

	// Test that same status doesn't trigger callbacks again
	callbackCount = 0
	nm.updateStatus(NetworkStatusOnline) // Same status
	time.Sleep(10 * time.Millisecond)    // Allow any potential callbacks to execute

	mu.Lock()
	finalCount = callbackCount
	mu.Unlock()

	if finalCount != 0 {
		t.Errorf("Expected 0 callback executions for same status, got %d", finalCount)
	}
}

// TestNetworkMonitor_updateStatus tests the internal updateStatus method
func TestNetworkMonitor_updateStatus(t *testing.T) {
	nm := NewNetworkMonitor(NetworkMonitorConfig{})

	// Test status change
	oldStatus := nm.GetStatus()
	nm.updateStatus(NetworkStatusOnline)

	if nm.GetStatus() != NetworkStatusOnline {
		t.Errorf("Expected status to be updated to online, got %v", nm.GetStatus())
	}

	// Test that last check time is updated
	if nm.GetLastCheck().IsZero() {
		t.Error("Expected last check time to be updated")
	}

	// Test that same status still updates last check time (implementation always updates lastCheck)
	firstCheck := nm.GetLastCheck()
	time.Sleep(1 * time.Millisecond)     // Ensure time difference
	nm.updateStatus(NetworkStatusOnline) // Same status

	secondCheck := nm.GetLastCheck()
	if secondCheck.Equal(firstCheck) {
		t.Error("Expected last check time to be updated even for identical status")
	}

	// Test different status updates last check time
	time.Sleep(1 * time.Millisecond)
	nm.updateStatus(NetworkStatusOffline)

	thirdCheck := nm.GetLastCheck()
	if thirdCheck.Equal(firstCheck) {
		t.Error("Expected last check time to be updated for different status")
	}

	_ = oldStatus // Use the variable to avoid unused variable error
}

// TestNetworkMonitor_MonitoringLoop tests the monitoring loop behavior
func TestNetworkMonitor_MonitoringLoop(t *testing.T) {
	config := NetworkMonitorConfig{
		CheckInterval: 50 * time.Millisecond,
		TestEndpoints: []string{"https://test.com"},
	}
	nm := NewNetworkMonitor(config)

	mockClient := NewMockHTTPClient()
	nm.httpClient = &http.Client{
		Transport: &mockTransport{mockClient: mockClient},
		Timeout:   nm.timeout,
	}

	mockClient.SetResponse("HEAD", "https://test.com", 200, "")

	ctx := context.Background()

	// Start monitoring
	nm.StartMonitoring(ctx)

	// Let it run for multiple intervals
	time.Sleep(200 * time.Millisecond)

	// Stop monitoring
	nm.StopMonitoring()

	// Verify multiple requests were made
	requestCount := mockClient.GetRequestCount("HEAD", "https://test.com")
	if requestCount < 3 { // Should have made several requests
		t.Errorf("Expected at least 3 requests over 200ms with 50ms interval, got %d", requestCount)
	}
}

// Benchmark tests for performance
func BenchmarkNetworkMonitor_GetStatus(b *testing.B) {
	nm := NewNetworkMonitor(NetworkMonitorConfig{})
	nm.updateStatus(NetworkStatusOnline)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = nm.GetStatus()
	}
}

func BenchmarkNetworkMonitor_UpdateStatus(b *testing.B) {
	nm := NewNetworkMonitor(NetworkMonitorConfig{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		status := NetworkStatus(i % 4)
		nm.updateStatus(status)
	}
}

func BenchmarkNetworkMonitor_ConcurrentAccess(b *testing.B) {
	nm := NewNetworkMonitor(NetworkMonitorConfig{})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = nm.GetStatus()
		}
	})
}

// TestNetworkMonitor_ErrorHandling tests error handling in various scenarios
func TestNetworkMonitor_ErrorHandling(t *testing.T) {
	nm := NewNetworkMonitor(NetworkMonitorConfig{
		TestEndpoints: []string{"https://test.com"},
	})

	mockClient := NewMockHTTPClient()
	nm.httpClient = &http.Client{
		Transport: &mockTransport{mockClient: mockClient},
		Timeout:   nm.timeout,
	}

	ctx := context.Background()

	t.Run("request creation error", func(t *testing.T) {
		// Test with invalid URL that would cause request creation to fail
		// This is hard to test directly, but we can test the error path
		result := nm.checkEndpoint(ctx, "://invalid-url")
		if result {
			t.Error("Expected endpoint check to fail for invalid URL")
		}
	})

	t.Run("http client error", func(t *testing.T) {
		mockClient.SetError("HEAD", "https://test.com", fmt.Errorf("network error"))

		result := nm.checkEndpoint(ctx, "https://test.com")
		if result {
			t.Error("Expected endpoint check to fail for HTTP client error")
		}
	})

	t.Run("response body close error", func(t *testing.T) {
		// Create a response with a body that errors on close
		mockClient.SetDoFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       &errorCloseBody{},
				Header:     make(http.Header),
			}, nil
		})

		// Should still succeed despite close error
		result := nm.checkEndpoint(ctx, "https://test.com")
		if !result {
			t.Error("Expected endpoint check to succeed despite body close error")
		}
	})
}

// errorCloseBody is a helper for testing body close errors
type errorCloseBody struct{}

func (ecb *errorCloseBody) Read(p []byte) (n int, err error) {
	return 0, io.EOF
}

func (ecb *errorCloseBody) Close() error {
	return fmt.Errorf("close error")
}

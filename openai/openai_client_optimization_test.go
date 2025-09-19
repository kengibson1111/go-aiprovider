package openai

import (
	"net/http"
	"testing"
	"time"

	"github.com/kengibson1111/go-aiprovider/types"
)

// TestCreateOptimizedHTTPClient tests that the optimized HTTP client is configured correctly
func TestCreateOptimizedHTTPClient(t *testing.T) {
	client := createOptimizedHTTPClient()

	// Verify client is not nil
	if client == nil {
		t.Fatal("Expected HTTP client to be created, got nil")
	}

	// Verify timeout is set correctly
	expectedTimeout := 30 * time.Second
	if client.Timeout != expectedTimeout {
		t.Errorf("Expected timeout %v, got %v", expectedTimeout, client.Timeout)
	}

	// Verify transport is configured
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatal("Expected *http.Transport, got different type")
	}

	// Verify connection pooling settings
	if transport.MaxIdleConns != 100 {
		t.Errorf("Expected MaxIdleConns 100, got %d", transport.MaxIdleConns)
	}

	if transport.MaxIdleConnsPerHost != 10 {
		t.Errorf("Expected MaxIdleConnsPerHost 10, got %d", transport.MaxIdleConnsPerHost)
	}

	expectedIdleTimeout := 90 * time.Second
	if transport.IdleConnTimeout != expectedIdleTimeout {
		t.Errorf("Expected IdleConnTimeout %v, got %v", expectedIdleTimeout, transport.IdleConnTimeout)
	}

	// Verify keep-alive is enabled
	if transport.DisableKeepAlives {
		t.Error("Expected keep-alive to be enabled, but it's disabled")
	}

	// Verify response header timeout
	expectedResponseTimeout := 15 * time.Second
	if transport.ResponseHeaderTimeout != expectedResponseTimeout {
		t.Errorf("Expected ResponseHeaderTimeout %v, got %v", expectedResponseTimeout, transport.ResponseHeaderTimeout)
	}

	// Verify TLS handshake timeout
	expectedTLSTimeout := 10 * time.Second
	if transport.TLSHandshakeTimeout != expectedTLSTimeout {
		t.Errorf("Expected TLSHandshakeTimeout %v, got %v", expectedTLSTimeout, transport.TLSHandshakeTimeout)
	}
}

// TestNewOpenAIClientOptimizations tests that the client is created with performance optimizations
func TestNewOpenAIClientOptimizations(t *testing.T) {
	config := &types.AIConfig{
		APIKey: "test-key",
		Model:  "gpt-4o-mini",
	}

	client, err := NewOpenAIClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Verify client is created
	if client == nil {
		t.Fatal("Expected client to be created, got nil")
	}

	// Verify HTTP client is stored for resource management
	if client.httpClient == nil {
		t.Error("Expected httpClient to be stored for resource management")
	}

	// Verify HTTP client has correct timeout
	expectedTimeout := 30 * time.Second
	if client.httpClient.Timeout != expectedTimeout {
		t.Errorf("Expected HTTP client timeout %v, got %v", expectedTimeout, client.httpClient.Timeout)
	}

	// Verify transport configuration
	transport, ok := client.httpClient.Transport.(*http.Transport)
	if !ok {
		t.Fatal("Expected *http.Transport in HTTP client")
	}

	// Verify connection pooling is configured
	if transport.MaxIdleConns != 100 {
		t.Errorf("Expected MaxIdleConns 100, got %d", transport.MaxIdleConns)
	}

	if transport.MaxIdleConnsPerHost != 10 {
		t.Errorf("Expected MaxIdleConnsPerHost 10, got %d", transport.MaxIdleConnsPerHost)
	}
}

// TestCloseIdleConnections tests the resource cleanup functionality
func TestCloseIdleConnections(t *testing.T) {
	config := &types.AIConfig{
		APIKey: "test-key",
		Model:  "gpt-4o-mini",
	}

	client, err := NewOpenAIClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Test that CloseIdleConnections doesn't panic and can be called multiple times
	client.CloseIdleConnections()
	client.CloseIdleConnections() // Should be safe to call multiple times

	// Verify the method exists and is callable
	if client.httpClient == nil {
		t.Error("Expected httpClient to be available for resource management")
	}
}

// TestClientWithNilHTTPClient tests that CloseIdleConnections handles nil httpClient gracefully
func TestClientWithNilHTTPClient(t *testing.T) {
	client := &OpenAIClient{
		httpClient: nil, // Simulate a client without HTTP client reference
	}

	// Should not panic when httpClient is nil
	client.CloseIdleConnections()
}

// TestPerformanceOptimizedParameters tests that API calls use performance-optimized parameters
func TestPerformanceOptimizedParameters(t *testing.T) {
	// This test verifies that the parameters include performance optimizations
	// We can't easily test the actual API call without mocking, but we can verify
	// the client is configured correctly for performance

	config := &types.AIConfig{
		APIKey:      "test-key",
		Model:       "gpt-4o-mini",
		MaxTokens:   1000,
		Temperature: 0.7,
	}

	client, err := NewOpenAIClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Verify client configuration includes performance settings
	if client.model != "gpt-4o-mini" {
		t.Errorf("Expected model gpt-4o-mini, got %s", client.model)
	}

	if client.maxTokens != 1000 {
		t.Errorf("Expected maxTokens 1000, got %d", client.maxTokens)
	}

	if client.temperature != 0.7 {
		t.Errorf("Expected temperature 0.7, got %f", client.temperature)
	}

	// Verify HTTP client is optimized
	if client.httpClient == nil {
		t.Error("Expected optimized HTTP client to be configured")
	}
}

// BenchmarkCreateOptimizedHTTPClient benchmarks the HTTP client creation
func BenchmarkCreateOptimizedHTTPClient(b *testing.B) {
	for i := 0; i < b.N; i++ {
		client := createOptimizedHTTPClient()
		if client == nil {
			b.Fatal("Failed to create HTTP client")
		}
	}
}

// BenchmarkNewOpenAIClientCreation benchmarks the optimized client creation
func BenchmarkNewOpenAIClientCreation(b *testing.B) {
	config := &types.AIConfig{
		APIKey: "test-key",
		Model:  "gpt-4o-mini",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client, err := NewOpenAIClient(config)
		if err != nil {
			b.Fatalf("Failed to create client: %v", err)
		}
		if client == nil {
			b.Fatal("Client is nil")
		}
	}
}

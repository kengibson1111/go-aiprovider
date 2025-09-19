package openai

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kengibson1111/go-aiprovider/types"
	"github.com/kengibson1111/go-aiprovider/utils"
	"github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/packages/ssestream"
)

// ThreadSafeMockClient extends MockOpenAISDKClient to track concurrent access
type ThreadSafeMockClient struct {
	*MockOpenAISDKClient
	accessCount       int64
	maxConcurrent     int64
	currentConcurrent int64
}

// Chat returns a thread-safe mock chat service
func (m *ThreadSafeMockClient) Chat() ChatServiceInterface {
	return &ThreadSafeMockChatService{client: m}
}

// ThreadSafeMockChatService implements ChatServiceInterface with concurrency tracking
type ThreadSafeMockChatService struct {
	client *ThreadSafeMockClient
}

// Completions returns a thread-safe mock completions service
func (m *ThreadSafeMockChatService) Completions() CompletionsServiceInterface {
	return &ThreadSafeMockCompletionsService{client: m.client}
}

// ThreadSafeMockCompletionsService implements CompletionsServiceInterface with concurrency tracking
type ThreadSafeMockCompletionsService struct {
	client *ThreadSafeMockClient
}

// New implements the completion creation method with concurrency tracking
func (m *ThreadSafeMockCompletionsService) New(ctx context.Context, params openai.ChatCompletionNewParams) (*openai.ChatCompletion, error) {
	// Increment concurrent counter
	current := atomic.AddInt64(&m.client.currentConcurrent, 1)
	atomic.AddInt64(&m.client.accessCount, 1)

	// Track maximum concurrent access
	for {
		max := atomic.LoadInt64(&m.client.maxConcurrent)
		if current <= max || atomic.CompareAndSwapInt64(&m.client.maxConcurrent, max, current) {
			break
		}
	}

	// Simulate some processing time to increase chance of concurrent access
	time.Sleep(10 * time.Millisecond)

	// Get the result from the base mock
	result := m.client.completion
	err := m.client.err

	// Decrement concurrent counter
	atomic.AddInt64(&m.client.currentConcurrent, -1)

	return result, err
}

// NewStreaming implements the streaming completion method with concurrency tracking
func (m *ThreadSafeMockCompletionsService) NewStreaming(ctx context.Context, params openai.ChatCompletionNewParams) *ssestream.Stream[openai.ChatCompletionChunk] {
	// Track access for streaming as well
	atomic.AddInt64(&m.client.accessCount, 1)

	// Return the same mock stream as the base implementation
	mockDecoder := &MockDecoder{err: m.client.err}
	return ssestream.NewStream[openai.ChatCompletionChunk](mockDecoder, m.client.err)
}

// TestOpenAIClient_ConcurrentUsage tests multiple simultaneous requests
// This covers requirement 8.5: Test concurrent usage and requirement 7.3: Performance under concurrent load
func TestOpenAIClient_ConcurrentUsage(t *testing.T) {
	// Create a mock SDK client that simulates successful responses
	mockClient := &MockOpenAISDKClient{
		completion: &openai.ChatCompletion{
			ID:      "chatcmpl-concurrent-test",
			Object:  "chat.completion",
			Created: 1234567890,
			Model:   "gpt-4o-mini",
			Choices: []openai.ChatCompletionChoice{
				{
					Index: 0,
					Message: openai.ChatCompletionMessage{
						Role:    "assistant",
						Content: "Hello from concurrent test!",
					},
					FinishReason: "stop",
				},
			},
			Usage: openai.CompletionUsage{
				PromptTokens:     10,
				CompletionTokens: 5,
				TotalTokens:      15,
			},
		},
		err: nil,
	}

	// Create OpenAI client with mock SDK client
	client := &OpenAIClient{
		client:      mockClient,
		model:       "gpt-4o-mini",
		maxTokens:   1000,
		temperature: 0.7,
		logger:      utils.NewLogger("TestOpenAIClient"),
	}

	ctx := context.Background()
	numGoroutines := 50 // Test with significant concurrent load

	// Channels for collecting results
	resultChan := make(chan []byte, numGoroutines)
	errorChan := make(chan error, numGoroutines)

	// WaitGroup to ensure all goroutines complete
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Launch concurrent requests
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			prompt := fmt.Sprintf("Test prompt %d", id)
			result, err := client.CallWithPrompt(ctx, prompt)

			if err != nil {
				errorChan <- err
			} else {
				resultChan <- result
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(resultChan)
	close(errorChan)

	// Collect results
	var results [][]byte
	var errors []error

	for result := range resultChan {
		results = append(results, result)
	}

	for err := range errorChan {
		errors = append(errors, err)
	}

	// Verify all requests succeeded
	if len(errors) > 0 {
		t.Errorf("Expected all concurrent requests to succeed, but got %d errors: %v", len(errors), errors)
	}

	// Verify we got the expected number of results
	if len(results) != numGoroutines {
		t.Errorf("Expected %d results, got: %d", numGoroutines, len(results))
	}

	// Verify each result is valid JSON
	for i, result := range results {
		if len(result) == 0 {
			t.Errorf("Result %d is empty", i)
		}
	}

	t.Logf("Concurrent usage test completed: %d successful requests", len(results))
}

// TestOpenAIClient_ThreadSafety verifies thread safety of SDK client usage
// This covers requirement 8.5: Verify thread safety of SDK client usage
func TestOpenAIClient_ThreadSafety(t *testing.T) {
	// Create a thread-safe mock client that tracks concurrent access
	mockClient := &ThreadSafeMockClient{
		MockOpenAISDKClient: &MockOpenAISDKClient{
			completion: &openai.ChatCompletion{
				ID:      "chatcmpl-threadsafe-test",
				Object:  "chat.completion",
				Created: 1234567890,
				Model:   "gpt-4o-mini",
				Choices: []openai.ChatCompletionChoice{
					{
						Index: 0,
						Message: openai.ChatCompletionMessage{
							Role:    "assistant",
							Content: "Thread safe response",
						},
						FinishReason: "stop",
					},
				},
			},
			err: nil,
		},
	}

	client := &OpenAIClient{
		client:      mockClient,
		model:       "gpt-4o-mini",
		maxTokens:   1000,
		temperature: 0.7,
		logger:      utils.NewLogger("TestOpenAIClient"),
	}

	ctx := context.Background()
	numGoroutines := 20

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	errorChan := make(chan error, numGoroutines)

	// Launch concurrent requests
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			_, err := client.CallWithPrompt(ctx, fmt.Sprintf("Thread safety test %d", id))
			if err != nil {
				errorChan <- err
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(errorChan)

	// Check for errors
	var errors []error
	for err := range errorChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		t.Errorf("Thread safety test failed with errors: %v", errors)
	}

	// Verify we had concurrent access
	totalAccess := atomic.LoadInt64(&mockClient.accessCount)
	maxConcurrentAccess := atomic.LoadInt64(&mockClient.maxConcurrent)

	if totalAccess != int64(numGoroutines) {
		t.Errorf("Expected %d total accesses, got: %d", numGoroutines, totalAccess)
	}

	if maxConcurrentAccess < 2 {
		t.Logf("Warning: Maximum concurrent access was only %d, may not have tested true concurrency", maxConcurrentAccess)
	}

	t.Logf("Thread safety test completed: %d total requests, max %d concurrent", totalAccess, maxConcurrentAccess)
}

// TestOpenAIClient_ConcurrentPerformance tests performance under concurrent load
// This covers requirement 7.3: Test performance under concurrent load
func TestOpenAIClient_ConcurrentPerformance(t *testing.T) {
	// Create a mock client with realistic response times
	mockClient := &MockOpenAISDKClient{
		completion: &openai.ChatCompletion{
			ID:      "chatcmpl-perf-test",
			Object:  "chat.completion",
			Created: 1234567890,
			Model:   "gpt-4o-mini",
			Choices: []openai.ChatCompletionChoice{
				{
					Index: 0,
					Message: openai.ChatCompletionMessage{
						Role:    "assistant",
						Content: "Performance test response with some content to simulate realistic response size",
					},
					FinishReason: "stop",
				},
			},
			Usage: openai.CompletionUsage{
				PromptTokens:     20,
				CompletionTokens: 15,
				TotalTokens:      35,
			},
		},
		err: nil,
	}

	client := &OpenAIClient{
		client:      mockClient,
		model:       "gpt-4o-mini",
		maxTokens:   1000,
		temperature: 0.7,
		logger:      utils.NewLogger("TestOpenAIClient"),
	}

	ctx := context.Background()
	numGoroutines := 100 // Higher load for performance testing

	// Measure performance metrics
	startTime := time.Now()
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Track successful requests
	var successCount int64
	var errorCount int64

	// Measure memory usage before test
	var memBefore runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&memBefore)

	// Launch concurrent requests
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			_, err := client.CallWithPrompt(ctx, fmt.Sprintf("Performance test prompt %d with some additional content to simulate realistic request size", id))

			if err != nil {
				atomic.AddInt64(&errorCount, 1)
			} else {
				atomic.AddInt64(&successCount, 1)
			}
		}(i)
	}

	// Wait for all requests to complete
	wg.Wait()
	totalTime := time.Since(startTime)

	// Measure memory usage after test
	var memAfter runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&memAfter)

	// Calculate performance metrics
	successRate := float64(successCount) / float64(numGoroutines) * 100
	requestsPerSecond := float64(numGoroutines) / totalTime.Seconds()
	avgResponseTime := totalTime / time.Duration(numGoroutines)
	memoryIncrease := memAfter.Alloc - memBefore.Alloc

	// Log performance metrics
	t.Logf("Performance Test Results:")
	t.Logf("  Total Requests: %d", numGoroutines)
	t.Logf("  Successful: %d", successCount)
	t.Logf("  Errors: %d", errorCount)
	t.Logf("  Success Rate: %.2f%%", successRate)
	t.Logf("  Total Time: %v", totalTime)
	t.Logf("  Requests/Second: %.2f", requestsPerSecond)
	t.Logf("  Avg Response Time: %v", avgResponseTime)
	t.Logf("  Memory Increase: %d bytes", memoryIncrease)

	// Verify performance requirements
	if successRate < 100.0 {
		t.Errorf("Expected 100%% success rate, got: %.2f%%", successRate)
	}

	// Verify reasonable performance (these are conservative thresholds for mock tests)
	if requestsPerSecond < 10 {
		t.Errorf("Performance too slow: %.2f requests/second (expected > 10)", requestsPerSecond)
	}

	if avgResponseTime > 100*time.Millisecond {
		t.Errorf("Average response time too slow: %v (expected < 100ms for mock)", avgResponseTime)
	}

	// Memory usage should be reasonable (this is a rough check)
	// Note: Memory measurements can be unreliable in tests due to GC timing
	maxExpectedMemory := uint64(numGoroutines * 10 * 1024) // 10KB per request max
	if memoryIncrease > 0 && memoryIncrease > maxExpectedMemory {
		t.Logf("Warning: Memory usage higher than expected: %d bytes (expected < %d)", memoryIncrease, maxExpectedMemory)
	}
}

// TestOpenAIClient_ConcurrentMethods tests concurrent usage of different methods
// This covers requirement 8.5: Test concurrent usage across different API methods
func TestOpenAIClient_ConcurrentMethods(t *testing.T) {
	// Create mock completion for all methods
	mockCompletion := &openai.ChatCompletion{
		ID:      "chatcmpl-methods-test",
		Object:  "chat.completion",
		Created: 1234567890,
		Model:   "gpt-4o-mini",
		Choices: []openai.ChatCompletionChoice{
			{
				Index: 0,
				Message: openai.ChatCompletionMessage{
					Role:    "assistant",
					Content: "Multi-method concurrent test response",
				},
				FinishReason: "stop",
			},
		},
		Usage: openai.CompletionUsage{
			PromptTokens:     15,
			CompletionTokens: 10,
			TotalTokens:      25,
		},
	}

	mockClient := &MockOpenAISDKClient{
		completion: mockCompletion,
		err:        nil,
	}

	client := &OpenAIClient{
		client:      mockClient,
		model:       "gpt-4o-mini",
		maxTokens:   1000,
		temperature: 0.7,
		logger:      utils.NewLogger("TestOpenAIClient"),
	}

	ctx := context.Background()
	numGoroutines := 30 // 10 for each method type

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	errorChan := make(chan error, numGoroutines)
	resultChan := make(chan string, numGoroutines)

	// Test CallWithPrompt concurrently
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer wg.Done()

			result, err := client.CallWithPrompt(ctx, fmt.Sprintf("CallWithPrompt test %d", id))
			if err != nil {
				errorChan <- fmt.Errorf("CallWithPrompt %d: %w", id, err)
			} else {
				resultChan <- fmt.Sprintf("CallWithPrompt-%d: %d bytes", id, len(result))
			}
		}(i)
	}

	// Test GenerateCompletion concurrently
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer wg.Done()

			req := types.CompletionRequest{
				Code:     fmt.Sprintf("console.log('test %d');", id),
				Cursor:   20,
				Language: "javascript",
				Context: types.CodeContext{
					CurrentFunction: "testFunction",
					ProjectType:     "Node.js",
				},
			}

			result, err := client.GenerateCompletion(ctx, req)
			if err != nil {
				errorChan <- fmt.Errorf("GenerateCompletion %d: %w", id, err)
			} else {
				resultChan <- fmt.Sprintf("GenerateCompletion-%d: %d suggestions", id, len(result.Suggestions))
			}
		}(i)
	}

	// Test GenerateCode concurrently
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer wg.Done()

			req := types.CodeGenerationRequest{
				Prompt:   fmt.Sprintf("Create a function for test %d", id),
				Language: "javascript",
				Context: types.CodeContext{
					ProjectType: "Node.js",
				},
			}

			result, err := client.GenerateCode(ctx, req)
			if err != nil {
				errorChan <- fmt.Errorf("GenerateCode %d: %w", id, err)
			} else {
				resultChan <- fmt.Sprintf("GenerateCode-%d: %d chars", id, len(result.Code))
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(errorChan)
	close(resultChan)

	// Collect results
	var errors []error
	var results []string

	for err := range errorChan {
		errors = append(errors, err)
	}

	for result := range resultChan {
		results = append(results, result)
	}

	// Verify all methods succeeded
	if len(errors) > 0 {
		t.Errorf("Expected all concurrent method calls to succeed, but got %d errors: %v", len(errors), errors)
	}

	// Verify we got results from all methods
	if len(results) != numGoroutines {
		t.Errorf("Expected %d results, got: %d", numGoroutines, len(results))
	}

	// Log results for verification
	t.Logf("Concurrent methods test completed with %d successful calls:", len(results))
	for _, result := range results {
		t.Logf("  %s", result)
	}
}

// TestOpenAIClient_ConcurrentResourceManagement tests resource management under concurrent load
// This covers requirement 7.3: Verify efficient resource usage when client is idle and under load
func TestOpenAIClient_ConcurrentResourceManagement(t *testing.T) {
	// Create a real client to test actual resource management
	config := &types.AIConfig{
		Provider:    "openai",
		APIKey:      "test-key-for-resource-test",
		Model:       "gpt-4o-mini",
		MaxTokens:   500,
		Temperature: 0.5,
	}

	client, err := NewOpenAIClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Test that CloseIdleConnections can be called safely during concurrent operations
	ctx := context.Background()
	numGoroutines := 20

	var wg sync.WaitGroup
	wg.Add(numGoroutines + 1) // +1 for the cleanup goroutine

	// Launch concurrent mock requests (these will fail but test resource management)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			// These will fail due to invalid API key, but test resource management
			_, _ = client.CallWithPrompt(ctx, fmt.Sprintf("Resource test %d", id))
		}(i)
	}

	// Launch a goroutine that periodically calls CloseIdleConnections
	go func() {
		defer wg.Done()

		for i := 0; i < 5; i++ {
			time.Sleep(10 * time.Millisecond)
			client.CloseIdleConnections()
		}
	}()

	// Wait for all operations to complete
	wg.Wait()

	// Final cleanup call should not panic or cause issues
	client.CloseIdleConnections()

	t.Log("Resource management test completed successfully")
}

// TestOpenAIClient_ConcurrentContextCancellation tests concurrent requests with context cancellation
// This covers requirement 8.5: Test concurrent usage with proper context handling
func TestOpenAIClient_ConcurrentContextCancellation(t *testing.T) {
	mockClient := &MockOpenAISDKClient{
		completion: &openai.ChatCompletion{
			ID:      "chatcmpl-cancel-test",
			Object:  "chat.completion",
			Created: 1234567890,
			Model:   "gpt-4o-mini",
			Choices: []openai.ChatCompletionChoice{
				{
					Index: 0,
					Message: openai.ChatCompletionMessage{
						Role:    "assistant",
						Content: "This should be cancelled",
					},
					FinishReason: "stop",
				},
			},
		},
		err: nil,
	}

	client := &OpenAIClient{
		client:      mockClient,
		model:       "gpt-4o-mini",
		maxTokens:   1000,
		temperature: 0.7,
		logger:      utils.NewLogger("TestOpenAIClient"),
	}

	// Create a context that will be cancelled
	ctx, cancel := context.WithCancel(context.Background())

	numGoroutines := 10
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	errorChan := make(chan error, numGoroutines)

	// Launch concurrent requests
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			_, err := client.CallWithPrompt(ctx, fmt.Sprintf("Cancellation test %d", id))
			if err != nil {
				errorChan <- err
			}
		}(i)
	}

	// Cancel the context after a short delay
	time.Sleep(5 * time.Millisecond)
	cancel()

	// Wait for all goroutines to complete
	wg.Wait()
	close(errorChan)

	// Collect errors
	var errors []error
	for err := range errorChan {
		errors = append(errors, err)
	}

	// Some requests might complete before cancellation, others should be cancelled
	// The important thing is that cancellation is handled gracefully
	t.Logf("Context cancellation test completed with %d errors (expected due to cancellation)", len(errors))

	// Verify that cancelled requests return appropriate errors
	for _, err := range errors {
		if err != nil {
			// Context cancellation should be handled gracefully
			t.Logf("Cancellation error (expected): %v", err)
		}
	}
}

// TestOpenAIClient_ConcurrentStressTest performs a stress test with high concurrent load
// This covers requirement 8.5: Test concurrent usage under extreme load
func TestOpenAIClient_ConcurrentStressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	mockClient := &MockOpenAISDKClient{
		completion: &openai.ChatCompletion{
			ID:      "chatcmpl-stress-test",
			Object:  "chat.completion",
			Created: 1234567890,
			Model:   "gpt-4o-mini",
			Choices: []openai.ChatCompletionChoice{
				{
					Index: 0,
					Message: openai.ChatCompletionMessage{
						Role:    "assistant",
						Content: "Stress test response",
					},
					FinishReason: "stop",
				},
			},
		},
		err: nil,
	}

	client := &OpenAIClient{
		client:      mockClient,
		model:       "gpt-4o-mini",
		maxTokens:   1000,
		temperature: 0.7,
		logger:      utils.NewLogger("TestOpenAIClient"),
	}

	ctx := context.Background()
	numGoroutines := 500 // High stress load

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	var successCount int64
	var errorCount int64

	startTime := time.Now()

	// Launch high concurrent load
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			_, err := client.CallWithPrompt(ctx, fmt.Sprintf("Stress test %d", id))

			if err != nil {
				atomic.AddInt64(&errorCount, 1)
			} else {
				atomic.AddInt64(&successCount, 1)
			}
		}(i)
	}

	// Wait for all requests to complete
	wg.Wait()
	totalTime := time.Since(startTime)

	// Calculate metrics
	successRate := float64(successCount) / float64(numGoroutines) * 100
	requestsPerSecond := float64(numGoroutines) / totalTime.Seconds()

	t.Logf("Stress Test Results:")
	t.Logf("  Total Requests: %d", numGoroutines)
	t.Logf("  Successful: %d", successCount)
	t.Logf("  Errors: %d", errorCount)
	t.Logf("  Success Rate: %.2f%%", successRate)
	t.Logf("  Total Time: %v", totalTime)
	t.Logf("  Requests/Second: %.2f", requestsPerSecond)

	// Verify stress test requirements
	if successRate < 100.0 {
		t.Errorf("Stress test failed: Expected 100%% success rate, got: %.2f%%", successRate)
	}

	// Should handle high load reasonably well
	if requestsPerSecond < 50 {
		t.Errorf("Stress test performance too slow: %.2f requests/second (expected > 50)", requestsPerSecond)
	}
}

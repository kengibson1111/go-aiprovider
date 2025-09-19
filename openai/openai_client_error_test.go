package openai

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/kengibson1111/go-aiprovider/types"
	"github.com/kengibson1111/go-aiprovider/utils"
	"github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/packages/ssestream"
)

// TestOpenAIClient_ErrorScenarios tests comprehensive error scenarios
// This test covers requirement 8.4: Test error scenarios and requirement 6.3: SDK retry logic
func TestOpenAIClient_ErrorScenarios(t *testing.T) {
	tests := []struct {
		name          string
		setupMock     func() *MockOpenAISDKClient
		setupServer   func() *httptest.Server
		expectError   bool
		errorContains string
		testMethod    string
		validateError func(t *testing.T, err error)
	}{
		// Network failure scenarios
		{
			name: "connection refused error",
			setupMock: func() *MockOpenAISDKClient {
				return &MockOpenAISDKClient{
					err: &net.OpError{
						Op:  "dial",
						Net: "tcp",
						Err: syscall.ECONNREFUSED,
					},
				}
			},
			expectError:   true,
			errorContains: "network error",
			testMethod:    "CallWithPrompt",
			validateError: func(t *testing.T, err error) {
				if !strings.Contains(err.Error(), "unable to connect") {
					t.Errorf("Expected connection guidance in error message")
				}
			},
		},
		{
			name: "DNS resolution failure",
			setupMock: func() *MockOpenAISDKClient {
				return &MockOpenAISDKClient{
					err: &net.DNSError{
						Err:        "no such host",
						Name:       "api.openai.com",
						IsNotFound: true,
					},
				}
			},
			expectError:   true,
			errorContains: "network error",
			testMethod:    "CallWithPrompt",
		},
		{
			name: "request timeout",
			setupMock: func() *MockOpenAISDKClient {
				return &MockOpenAISDKClient{
					err: context.DeadlineExceeded,
				}
			},
			expectError:   true,
			errorContains: "request timeout",
			testMethod:    "CallWithPrompt",
			validateError: func(t *testing.T, err error) {
				if !strings.Contains(err.Error(), "took too long") {
					t.Errorf("Expected timeout guidance in error message")
				}
			},
		},
		{
			name: "context cancelled",
			setupMock: func() *MockOpenAISDKClient {
				return &MockOpenAISDKClient{
					err: context.Canceled,
				}
			},
			expectError:   true,
			errorContains: "request failed",
			testMethod:    "CallWithPrompt",
		},

		// API error scenarios - Authentication
		{
			name: "invalid API key",
			setupMock: func() *MockOpenAISDKClient {
				return &MockOpenAISDKClient{
					err: &openai.Error{
						Code:    "invalid_api_key",
						Message: "Invalid API key provided",
						Type:    "invalid_request_error",
					},
				}
			},
			expectError:   true,
			errorContains: "invalid API key",
			testMethod:    "ValidateCredentials",
			validateError: func(t *testing.T, err error) {
				if !strings.Contains(err.Error(), "check your OpenAI API key") {
					t.Errorf("Expected API key guidance in error message")
				}
			},
		},
		{
			name: "insufficient quota",
			setupMock: func() *MockOpenAISDKClient {
				return &MockOpenAISDKClient{
					err: &openai.Error{
						Code:    "insufficient_quota",
						Message: "You exceeded your current quota",
						Type:    "insufficient_quota",
					},
				}
			},
			expectError:   true,
			errorContains: "quota exceeded",
			testMethod:    "CallWithPrompt",
			validateError: func(t *testing.T, err error) {
				if !strings.Contains(err.Error(), "check your billing") {
					t.Errorf("Expected billing guidance in error message")
				}
			},
		},

		// API error scenarios - Rate limiting
		{
			name: "rate limit exceeded",
			setupMock: func() *MockOpenAISDKClient {
				return &MockOpenAISDKClient{
					err: &openai.Error{
						Code:    "rate_limit_exceeded",
						Message: "Rate limit reached for requests",
						Type:    "rate_limit_error",
					},
				}
			},
			expectError:   true,
			errorContains: "rate limit exceeded",
			testMethod:    "CallWithPrompt",
			validateError: func(t *testing.T, err error) {
				if !strings.Contains(err.Error(), "wait before retrying") {
					t.Errorf("Expected retry guidance in error message")
				}
			},
		},

		// API error scenarios - Invalid requests
		{
			name: "model not found",
			setupMock: func() *MockOpenAISDKClient {
				return &MockOpenAISDKClient{
					err: &openai.Error{
						Code:    "model_not_found",
						Message: "The model 'invalid-model' does not exist",
						Type:    "invalid_request_error",
					},
				}
			},
			expectError:   true,
			errorContains: "model not found",
			testMethod:    "CallWithPrompt",
		},
		{
			name: "context length exceeded",
			setupMock: func() *MockOpenAISDKClient {
				return &MockOpenAISDKClient{
					err: &openai.Error{
						Code:    "context_length_exceeded",
						Message: "This model's maximum context length is 4096 tokens",
						Type:    "invalid_request_error",
					},
				}
			},
			expectError:   true,
			errorContains: "context length exceeded",
			testMethod:    "CallWithPrompt",
			validateError: func(t *testing.T, err error) {
				if !strings.Contains(err.Error(), "context window") {
					t.Errorf("Expected context window guidance in error message")
				}
			},
		},

		// Server error scenarios
		{
			name: "internal server error",
			setupMock: func() *MockOpenAISDKClient {
				return &MockOpenAISDKClient{
					err: &openai.Error{
						Message: "Internal server error",
						Type:    "server_error",
					},
				}
			},
			expectError:   true,
			errorContains: "server error",
			testMethod:    "CallWithPrompt",
			validateError: func(t *testing.T, err error) {
				if !strings.Contains(err.Error(), "try again later") {
					t.Errorf("Expected retry guidance in error message")
				}
			},
		},
		{
			name: "service unavailable",
			setupMock: func() *MockOpenAISDKClient {
				return &MockOpenAISDKClient{
					err: &openai.Error{
						Message: "Service temporarily unavailable",
						Type:    "service_unavailable",
					},
				}
			},
			expectError:   true,
			errorContains: "service unavailable",
			testMethod:    "CallWithPrompt",
		},

		// HTTP status code errors (fallback handling)
		{
			name: "HTTP 401 Unauthorized",
			setupMock: func() *MockOpenAISDKClient {
				return &MockOpenAISDKClient{
					err: errors.New("HTTP 401 Unauthorized"),
				}
			},
			expectError:   true,
			errorContains: "invalid API key",
			testMethod:    "CallWithPrompt",
		},
		{
			name: "HTTP 403 Forbidden",
			setupMock: func() *MockOpenAISDKClient {
				return &MockOpenAISDKClient{
					err: errors.New("HTTP 403 Forbidden"),
				}
			},
			expectError:   true,
			errorContains: "insufficient permissions",
			testMethod:    "CallWithPrompt",
		},
		{
			name: "HTTP 429 Too Many Requests",
			setupMock: func() *MockOpenAISDKClient {
				return &MockOpenAISDKClient{
					err: errors.New("HTTP 429 Too Many Requests"),
				}
			},
			expectError:   true,
			errorContains: "rate limit exceeded",
			testMethod:    "CallWithPrompt",
		},
		{
			name: "HTTP 500 Internal Server Error",
			setupMock: func() *MockOpenAISDKClient {
				return &MockOpenAISDKClient{
					err: errors.New("HTTP 500 Internal Server Error"),
				}
			},
			expectError:   true,
			errorContains: "server error",
			testMethod:    "CallWithPrompt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock client
			mockClient := tt.setupMock()

			// Create OpenAI client with mock
			client := &OpenAIClient{
				client:      mockClient,
				model:       "gpt-4o-mini",
				maxTokens:   1000,
				temperature: 0.7,
				logger:      utils.NewLogger("TestOpenAIClient"),
			}

			ctx := context.Background()
			var err error

			// Test different methods based on testMethod
			switch tt.testMethod {
			case "CallWithPrompt":
				_, err = client.CallWithPrompt(ctx, "test prompt")
			case "ValidateCredentials":
				err = client.ValidateCredentials(ctx)
			case "CallWithMessages":
				messages := []openai.ChatCompletionMessageParamUnion{
					openai.UserMessage("test"),
				}
				_, err = client.CallWithMessages(ctx, messages)
			case "CallWithTools":
				tools := []openai.ChatCompletionToolUnionParam{}
				_, err = client.CallWithTools(ctx, "test", tools)
			default:
				_, err = client.CallWithPrompt(ctx, "test prompt")
			}

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.errorContains)) {
					t.Errorf("Expected error to contain '%s', got: %s", tt.errorContains, err.Error())
				}

				// Run custom validation if provided
				if tt.validateError != nil {
					tt.validateError(t, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

// TestOpenAIClient_InvalidConfigurationScenarios tests invalid configuration scenarios
// This covers requirement 8.4: Test invalid configuration scenarios
func TestOpenAIClient_InvalidConfigurationScenarios(t *testing.T) {
	tests := []struct {
		name        string
		config      *types.AIConfig
		expectError bool
		errorMsg    string
	}{
		{
			name:        "nil configuration",
			config:      nil,
			expectError: true,
			errorMsg:    "configuration is required",
		},
		{
			name: "empty API key",
			config: &types.AIConfig{
				Provider: "openai",
				APIKey:   "",
			},
			expectError: true,
			errorMsg:    "API key is required",
		},
		{
			name: "whitespace-only API key",
			config: &types.AIConfig{
				Provider: "openai",
				APIKey:   "   ",
			},
			expectError: true,
			errorMsg:    "API key is required",
		},
		{
			name: "invalid base URL format",
			config: &types.AIConfig{
				Provider: "openai",
				APIKey:   "sk-test123",
				BaseURL:  "not-a-valid-url",
			},
			expectError: false, // SDK handles URL validation
		},
		{
			name: "negative max tokens",
			config: &types.AIConfig{
				Provider:  "openai",
				APIKey:    "sk-test123",
				MaxTokens: -100,
			},
			expectError: false, // Should use default
		},
		{
			name: "negative temperature",
			config: &types.AIConfig{
				Provider:    "openai",
				APIKey:      "sk-test123",
				Temperature: -1.0,
			},
			expectError: false, // Should use default
		},
		{
			name: "temperature too high",
			config: &types.AIConfig{
				Provider:    "openai",
				APIKey:      "sk-test123",
				Temperature: 3.0,
			},
			expectError: false, // SDK will validate
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewOpenAIClient(tt.config)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error message to contain '%s', got: %s", tt.errorMsg, err.Error())
				}
				if client != nil {
					t.Errorf("Expected nil client on error, got: %v", client)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
					return
				}
				if client == nil {
					t.Errorf("Expected non-nil client")
				}
			}
		})
	}
}

// TestOpenAIClient_StreamingErrorScenarios tests streaming-specific error scenarios
// This covers requirement 8.4: Test streaming error scenarios
func TestOpenAIClient_StreamingErrorScenarios(t *testing.T) {
	tests := []struct {
		name          string
		mockError     error
		expectError   bool
		errorContains string
		validateError func(t *testing.T, err error)
	}{
		{
			name: "streaming connection error",
			mockError: &openai.Error{
				Message: "Connection lost during streaming",
				Type:    "connection_error",
			},
			expectError:   true,
			errorContains: "streaming error",
			validateError: func(t *testing.T, err error) {
				if !strings.Contains(err.Error(), "Connection lost during streaming") {
					t.Errorf("Expected connection error message in streaming error")
				}
			},
		},
		{
			name:          "streaming timeout",
			mockError:     context.DeadlineExceeded,
			expectError:   true,
			errorContains: "streaming timeout",
			validateError: func(t *testing.T, err error) {
				if !strings.Contains(err.Error(), "increasing timeout") {
					t.Errorf("Expected timeout configuration guidance in error message")
				}
			},
		},
		{
			name: "stream initialization failure",
			mockError: &openai.Error{
				Code:    "invalid_request_error",
				Message: "Failed to initialize stream",
				Type:    "invalid_request_error",
			},
			expectError:   true,
			errorContains: "streaming error",
		},
		{
			name: "streaming rate limit",
			mockError: &openai.Error{
				Code:    "rate_limit_exceeded",
				Message: "Rate limit exceeded for streaming requests",
				Type:    "rate_limit_error",
			},
			expectError:   true,
			errorContains: "rate limit exceeded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client with streaming error
			mockClient := &MockOpenAISDKClient{
				err: tt.mockError,
			}

			client := &OpenAIClient{
				client:      mockClient,
				model:       "gpt-4o-mini",
				maxTokens:   1000,
				temperature: 0.7,
				logger:      utils.NewLogger("TestOpenAIClient"),
			}

			ctx := context.Background()
			stream, err := client.CallWithPromptStream(ctx, "test streaming prompt")

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.errorContains)) {
					t.Errorf("Expected error to contain '%s', got: %s", tt.errorContains, err.Error())
				}
				if stream != nil {
					t.Errorf("Expected nil stream on error, got: %v", stream)
				}

				// Run custom validation if provided
				if tt.validateError != nil {
					tt.validateError(t, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if stream == nil {
					t.Errorf("Expected non-nil stream")
				}
			}
		})
	}
}

// TestOpenAIClient_SDKRetryLogic tests that SDK retry logic and backoff work correctly
// This covers requirement 6.3: Verify SDK retry logic and backoff work correctly
func TestOpenAIClient_SDKRetryLogic(t *testing.T) {
	// Test with a real HTTP server to verify retry behavior
	retryCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		retryCount++

		// Fail the first 2 requests, succeed on the 3rd
		if retryCount <= 2 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error": {"message": "Rate limit exceeded", "type": "rate_limit_error", "code": "rate_limit_exceeded"}}`))
			return
		}

		// Success response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": "chatcmpl-test",
			"object": "chat.completion",
			"created": 1234567890,
			"model": "gpt-4o-mini",
			"choices": [
				{
					"index": 0,
					"message": {
						"role": "assistant",
						"content": "Success after retries!"
					},
					"finish_reason": "stop"
				}
			],
			"usage": {
				"prompt_tokens": 10,
				"completion_tokens": 5,
				"total_tokens": 15
			}
		}`))
	}))
	defer server.Close()

	config := &types.AIConfig{
		Provider: "openai",
		APIKey:   "test-key",
		BaseURL:  server.URL,
		Model:    "gpt-4o-mini",
	}

	client, err := NewOpenAIClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	// Test that the SDK retries and eventually succeeds
	completion, err := client.callWithPrompt(ctx, "test prompt")

	if err != nil {
		t.Errorf("Expected success after retries, got error: %v", err)
		return
	}

	if completion == nil {
		t.Errorf("Expected non-nil completion")
		return
	}

	// Verify that retries occurred (should be at least 3 requests)
	if retryCount < 3 {
		t.Errorf("Expected at least 3 requests (with retries), got: %d", retryCount)
	}

	// Verify the final response
	if len(completion.Choices) == 0 {
		t.Errorf("Expected at least one choice in completion")
		return
	}

	if completion.Choices[0].Message.Content != "Success after retries!" {
		t.Errorf("Expected specific success message, got: %s", completion.Choices[0].Message.Content)
	}
}

// TestOpenAIClient_ConcurrentErrorHandling tests error handling under concurrent load
// This covers requirement 8.5: Test concurrent usage and requirement 7.3: Performance under concurrent load
func TestOpenAIClient_ConcurrentErrorHandling(t *testing.T) {
	// Create a mock client that returns different errors for different requests
	mockClient := &MockOpenAISDKClient{
		err: &openai.Error{
			Code:    "rate_limit_exceeded",
			Message: "Rate limit exceeded",
			Type:    "rate_limit_error",
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
	numGoroutines := 10
	errorChan := make(chan error, numGoroutines)

	// Launch concurrent requests
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			_, err := client.CallWithPrompt(ctx, fmt.Sprintf("test prompt %d", id))
			errorChan <- err
		}(i)
	}

	// Collect all errors
	var errors []error
	for i := 0; i < numGoroutines; i++ {
		err := <-errorChan
		if err != nil {
			errors = append(errors, err)
		}
	}

	// Verify all requests failed with the expected error
	if len(errors) != numGoroutines {
		t.Errorf("Expected %d errors, got: %d", numGoroutines, len(errors))
	}

	// Verify all errors are properly formatted
	for i, err := range errors {
		if !strings.Contains(err.Error(), "rate limit exceeded") {
			t.Errorf("Error %d does not contain expected message: %v", i, err)
		}
	}
}

// MockRecoveryClient implements recovery behavior for testing
type MockRecoveryClient struct {
	callCount int
}

func (m *MockRecoveryClient) Chat() ChatServiceInterface {
	return &MockRecoveryChatService{client: m}
}

type MockRecoveryChatService struct {
	client *MockRecoveryClient
}

func (m *MockRecoveryChatService) Completions() CompletionsServiceInterface {
	return &MockRecoveryCompletionsService{client: m.client}
}

type MockRecoveryCompletionsService struct {
	client *MockRecoveryClient
}

func (m *MockRecoveryCompletionsService) New(ctx context.Context, params openai.ChatCompletionNewParams) (*openai.ChatCompletion, error) {
	m.client.callCount++

	if m.client.callCount == 1 {
		// First call fails
		return nil, &openai.Error{
			Code:    "rate_limit_exceeded",
			Message: "Rate limit exceeded",
			Type:    "rate_limit_error",
		}
	}

	// Second call succeeds
	return &openai.ChatCompletion{
		ID:      "chatcmpl-recovery",
		Object:  "chat.completion",
		Created: 1234567890,
		Model:   "gpt-4o-mini",
		Choices: []openai.ChatCompletionChoice{
			{
				Index: 0,
				Message: openai.ChatCompletionMessage{
					Role:    "assistant",
					Content: "Recovery successful!",
				},
				FinishReason: "stop",
			},
		},
	}, nil
}

func (m *MockRecoveryCompletionsService) NewStreaming(ctx context.Context, params openai.ChatCompletionNewParams) *ssestream.Stream[openai.ChatCompletionChunk] {
	// Not used in this test
	return nil
}

// TestOpenAIClient_ErrorRecovery tests error recovery scenarios
func TestOpenAIClient_ErrorRecovery(t *testing.T) {
	// Test that client can recover from errors and make successful requests
	mockClient := &MockRecoveryClient{}

	client := &OpenAIClient{
		client:      mockClient,
		model:       "gpt-4o-mini",
		maxTokens:   1000,
		temperature: 0.7,
		logger:      utils.NewLogger("TestOpenAIClient"),
	}

	ctx := context.Background()

	// First call should fail
	_, err := client.callWithPrompt(ctx, "test prompt")
	if err == nil {
		t.Errorf("Expected first call to fail")
		return
	}
	if !strings.Contains(err.Error(), "rate limit exceeded") {
		t.Errorf("Expected rate limit error, got: %v", err)
	}

	// Second call should succeed
	completion, err := client.callWithPrompt(ctx, "test prompt")
	if err != nil {
		t.Errorf("Expected second call to succeed, got error: %v", err)
		return
	}

	if completion == nil {
		t.Errorf("Expected non-nil completion")
		return
	}

	if completion.Choices[0].Message.Content != "Recovery successful!" {
		t.Errorf("Expected recovery message, got: %s", completion.Choices[0].Message.Content)
	}
}

// TestOpenAIClient_TimeoutConfiguration tests timeout handling
func TestOpenAIClient_TimeoutConfiguration(t *testing.T) {
	// Test with very short timeout to ensure timeout handling works
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	mockClient := &MockOpenAISDKClient{
		err: context.DeadlineExceeded,
	}

	client := &OpenAIClient{
		client:      mockClient,
		model:       "gpt-4o-mini",
		maxTokens:   1000,
		temperature: 0.7,
		logger:      utils.NewLogger("TestOpenAIClient"),
	}

	_, err := client.CallWithPrompt(ctx, "test prompt")

	if err == nil {
		t.Errorf("Expected timeout error")
		return
	}

	if !strings.Contains(err.Error(), "timeout") {
		t.Errorf("Expected timeout error message, got: %v", err)
	}
}

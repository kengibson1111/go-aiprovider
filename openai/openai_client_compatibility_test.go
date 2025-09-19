package openai

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/kengibson1111/go-aiprovider/types"
	"github.com/kengibson1111/go-aiprovider/utils"
	"github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/packages/ssestream"
)

// CompatibilityMockClient implements OpenAIClientInterface for testing
type CompatibilityMockClient struct {
	chatCompletion *openai.ChatCompletion
	chatError      error
}

func (m *CompatibilityMockClient) Chat() ChatServiceInterface {
	return &CompatibilityMockChatService{
		completion: m.chatCompletion,
		error:      m.chatError,
	}
}

// CompatibilityMockChatService implements ChatServiceInterface for testing
type CompatibilityMockChatService struct {
	completion *openai.ChatCompletion
	error      error
}

func (m *CompatibilityMockChatService) Completions() CompletionsServiceInterface {
	return &CompatibilityMockCompletionsService{
		completion: m.completion,
		error:      m.error,
	}
}

// CompatibilityMockCompletionsService implements CompletionsServiceInterface for testing
type CompatibilityMockCompletionsService struct {
	completion *openai.ChatCompletion
	error      error
}

func (m *CompatibilityMockCompletionsService) New(ctx context.Context, params openai.ChatCompletionNewParams) (*openai.ChatCompletion, error) {
	if m.error != nil {
		return nil, m.error
	}
	return m.completion, nil
}

func (m *CompatibilityMockCompletionsService) NewStreaming(ctx context.Context, params openai.ChatCompletionNewParams) *ssestream.Stream[openai.ChatCompletionChunk] {
	// For compatibility tests, we don't need streaming functionality
	return nil
}

// createTestClient creates a test OpenAI client with the given mock
func createTestClient(mockClient OpenAIClientInterface) *OpenAIClient {
	return &OpenAIClient{
		client:      mockClient,
		model:       "gpt-4o-mini",
		maxTokens:   1000,
		temperature: 0.7,
		logger:      utils.NewLogger("TestOpenAIClient"),
	}
}

// TestHighLevelMethodsCompatibility tests that GenerateCompletion and GenerateCode
// maintain their expected interfaces and response structures after SDK migration
func TestHighLevelMethodsCompatibility(t *testing.T) {
	t.Run("GenerateCompletion maintains response structure", func(t *testing.T) {
		testGenerateCompletionResponseStructure(t)
	})

	t.Run("GenerateCode maintains response structure", func(t *testing.T) {
		testGenerateCodeResponseStructure(t)
	})

	t.Run("existing application code patterns work", func(t *testing.T) {
		testExistingApplicationCodePatterns(t)
	})
}

// testGenerateCompletionResponseStructure verifies that GenerateCompletion returns
// the same types.CompletionResponse structure with all expected fields
func testGenerateCompletionResponseStructure(t *testing.T) {
	tests := []struct {
		name                string
		mockCompletion      *openai.ChatCompletion
		mockError           error
		expectedSuggestions int
	}{
		{
			name: "successful completion response",
			mockCompletion: &openai.ChatCompletion{
				ID:      "test-completion-id",
				Object:  "chat.completion",
				Created: 1234567890,
				Model:   "gpt-4",
				Choices: []openai.ChatCompletionChoice{
					{
						Index: 0,
						Message: openai.ChatCompletionMessage{
							Role:    "assistant",
							Content: "Test completion suggestion",
						},
						FinishReason: "stop",
					},
				},
				Usage: openai.CompletionUsage{
					PromptTokens:     10,
					CompletionTokens: 20,
					TotalTokens:      30,
				},
			},
			mockError:           nil,
			expectedSuggestions: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client
			mockClient := &CompatibilityMockClient{
				chatCompletion: tt.mockCompletion,
				chatError:      tt.mockError,
			}

			client := createTestClient(mockClient)

			// Test GenerateCompletion
			response, err := client.GenerateCompletion(context.Background(), types.CompletionRequest{
				Code:     "func main() {",
				Cursor:   13,
				Language: "go",
				Context: types.CodeContext{
					CurrentFunction: "main",
					ProjectType:     "cli",
				},
			})

			if tt.mockError != nil {
				if err == nil {
					t.Errorf("Expected error %v, got nil", tt.mockError)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Verify response structure
			if response == nil {
				t.Fatal("Response is nil")
			}

			// Check expected fields exist and have correct types
			if len(response.Suggestions) != tt.expectedSuggestions {
				t.Errorf("Expected %d suggestions, got %d", tt.expectedSuggestions, len(response.Suggestions))
			}

			if response.Confidence < 0 || response.Confidence > 1 {
				t.Errorf("Confidence should be between 0 and 1, got %f", response.Confidence)
			}

			if response.Error != "" {
				t.Errorf("Expected no error in response, got %s", response.Error)
			}
		})
	}
}

// testGenerateCodeResponseStructure verifies that GenerateCode returns
// the same types.CodeGenerationResponse structure with all expected fields
func testGenerateCodeResponseStructure(t *testing.T) {
	tests := []struct {
		name           string
		mockCompletion *openai.ChatCompletion
		mockError      error
		expectedCode   string
	}{
		{
			name: "successful code generation response",
			mockCompletion: &openai.ChatCompletion{
				ID:      "test-code-completion-id",
				Object:  "chat.completion",
				Created: 1234567890,
				Model:   "gpt-4",
				Choices: []openai.ChatCompletionChoice{
					{
						Index: 0,
						Message: openai.ChatCompletionMessage{
							Role:    "assistant",
							Content: "func main() {\n\tfmt.Println(\"Hello, World!\")\n}",
						},
						FinishReason: "stop",
					},
				},
				Usage: openai.CompletionUsage{
					PromptTokens:     15,
					CompletionTokens: 25,
					TotalTokens:      40,
				},
			},
			mockError:    nil,
			expectedCode: "func main() {\n\tfmt.Println(\"Hello, World!\")\n}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client
			mockClient := &CompatibilityMockClient{
				chatCompletion: tt.mockCompletion,
				chatError:      tt.mockError,
			}

			client := createTestClient(mockClient)

			// Test GenerateCode
			response, err := client.GenerateCode(context.Background(), types.CodeGenerationRequest{
				Prompt:   "Write a Hello World function in Go",
				Language: "go",
				Context: types.CodeContext{
					ProjectType: "cli",
				},
			})

			if tt.mockError != nil {
				if err == nil {
					t.Errorf("Expected error %v, got nil", tt.mockError)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Verify response structure
			if response == nil {
				t.Fatal("Response is nil")
			}

			// Check that the response contains expected code
			if !strings.Contains(response.Code, "func main()") {
				t.Errorf("Expected code to contain 'func main()', got %v", response.Code)
			}

			if response.Error != "" {
				t.Errorf("Expected no error in response, got %s", response.Error)
			}
		})
	}
}

// testExistingApplicationCodePatterns verifies that common usage patterns
// from existing applications continue to work after SDK migration
func testExistingApplicationCodePatterns(t *testing.T) {
	t.Run("chaining multiple requests", func(t *testing.T) {
		mockClient := &CompatibilityMockClient{
			chatCompletion: &openai.ChatCompletion{
				ID:      "test-chain-id",
				Object:  "chat.completion",
				Created: 1234567890,
				Model:   "gpt-4",
				Choices: []openai.ChatCompletionChoice{
					{
						Index: 0,
						Message: openai.ChatCompletionMessage{
							Role:    "assistant",
							Content: "Chained response",
						},
						FinishReason: "stop",
					},
				},
				Usage: openai.CompletionUsage{
					PromptTokens:     5,
					CompletionTokens: 10,
					TotalTokens:      15,
				},
			},
		}

		client := createTestClient(mockClient)

		ctx := context.Background()

		// First request
		response1, err := client.GenerateCompletion(ctx, types.CompletionRequest{
			Code:     "func test() {",
			Cursor:   13,
			Language: "go",
			Context: types.CodeContext{
				CurrentFunction: "test",
			},
		})

		if err != nil {
			t.Errorf("First request failed: %v", err)
			return
		}

		// Second request
		response2, err := client.GenerateCode(ctx, types.CodeGenerationRequest{
			Prompt:   "Generate a test function",
			Language: "go",
			Context: types.CodeContext{
				ProjectType: "test",
			},
		})

		if err != nil {
			t.Errorf("Second request failed: %v", err)
			return
		}

		// Verify both responses have expected structure
		if len(response1.Suggestions) == 0 || response2.Code == "" {
			t.Error("Chained requests should return content")
		}
	})

	t.Run("error handling patterns", func(t *testing.T) {
		mockClient := &CompatibilityMockClient{
			chatError: errors.New("rate limit exceeded"),
		}

		client := createTestClient(mockClient)

		response, err := client.GenerateCompletion(context.Background(), types.CompletionRequest{
			Code:     "func test() {",
			Cursor:   13,
			Language: "go",
			Context: types.CodeContext{
				CurrentFunction: "test",
			},
		})

		// Verify error handling works as expected
		if err != nil {
			t.Errorf("GenerateCompletion should not return error, got %v", err)
			return
		}

		// Check that error is captured in response
		if response.Error == "" {
			t.Error("Expected error in response but got empty error")
		}

		if !strings.Contains(response.Error, "ERROR:") {
			t.Errorf("Expected error to contain 'ERROR:', got %s", response.Error)
		}
	})

	t.Run("context cancellation", func(t *testing.T) {
		mockClient := &CompatibilityMockClient{
			chatCompletion: &openai.ChatCompletion{
				ID:      "test-context-id",
				Object:  "chat.completion",
				Created: 1234567890,
				Model:   "gpt-4",
				Choices: []openai.ChatCompletionChoice{
					{
						Index: 0,
						Message: openai.ChatCompletionMessage{
							Role:    "assistant",
							Content: "Context test response",
						},
						FinishReason: "stop",
					},
				},
			},
		}

		client := createTestClient(mockClient)

		// Create a context that can be cancelled
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		response, err := client.GenerateCompletion(ctx, types.CompletionRequest{
			Code:     "func test() {",
			Cursor:   13,
			Language: "go",
			Context: types.CodeContext{
				CurrentFunction: "test",
			},
		})

		// Should handle context cancellation appropriately
		// Note: Our mock doesn't actually check context, so this tests the interface
		if err != nil {
			t.Errorf("Unexpected error with cancelled context: %v", err)
		}

		if response == nil {
			t.Error("Response should not be nil even with cancelled context")
		}
	})
}

// TestConfigurationCompatibility tests that client configuration
// works the same way after SDK migration
func TestConfigurationCompatibility(t *testing.T) {
	t.Run("client initialization with different configs", func(t *testing.T) {
		configs := []*types.AIConfig{
			{
				APIKey:      "test-key-1",
				Model:       "gpt-4",
				MaxTokens:   1000,
				Temperature: 0.7,
			},
			{
				APIKey:      "test-key-2",
				Model:       "gpt-3.5-turbo",
				MaxTokens:   500,
				Temperature: 0.5,
			},
		}

		for i, config := range configs {
			t.Run(strings.Join([]string{"config", string(rune(i + 49))}, "_"), func(t *testing.T) {
				client, err := NewOpenAIClient(config)
				if err != nil {
					t.Errorf("Failed to create client with config %d: %v", i+1, err)
					return
				}

				if client == nil {
					t.Errorf("Client is nil for config %d", i+1)
				}

				// Verify client has expected configuration
				if client.model != config.Model {
					t.Errorf("Expected model %s, got %s", config.Model, client.model)
				}

				if client.maxTokens != config.MaxTokens {
					t.Errorf("Expected maxTokens %d, got %d", config.MaxTokens, client.maxTokens)
				}

				if client.temperature != config.Temperature {
					t.Errorf("Expected temperature %f, got %f", config.Temperature, client.temperature)
				}
			})
		}
	})

	t.Run("invalid configuration handling", func(t *testing.T) {
		invalidConfigs := []*types.AIConfig{
			{APIKey: ""}, // Empty API key
			nil,          // Nil config
		}

		for i, config := range invalidConfigs {
			t.Run(strings.Join([]string{"invalid_config", string(rune(i + 49))}, "_"), func(t *testing.T) {
				_, err := NewOpenAIClient(config)
				if err == nil {
					t.Errorf("Expected error for invalid config %d, got nil", i+1)
				}
			})
		}
	})
}

// TestBackwardCompatibilityEdgeCases tests edge cases that might break
// backward compatibility
func TestBackwardCompatibilityEdgeCases(t *testing.T) {
	t.Run("empty response handling", func(t *testing.T) {
		mockClient := &CompatibilityMockClient{
			chatCompletion: &openai.ChatCompletion{
				ID:      "empty-response-test",
				Object:  "chat.completion",
				Created: 1234567890,
				Model:   "gpt-4",
				Choices: []openai.ChatCompletionChoice{
					{
						Index: 0,
						Message: openai.ChatCompletionMessage{
							Role:    "assistant",
							Content: "",
						},
						FinishReason: "stop",
					},
				},
			},
		}

		client := createTestClient(mockClient)

		response, err := client.GenerateCompletion(context.Background(), types.CompletionRequest{
			Code:     "func test() {",
			Cursor:   13,
			Language: "go",
			Context: types.CodeContext{
				CurrentFunction: "test",
			},
		})

		if err != nil {
			t.Errorf("Unexpected error with empty response: %v", err)
			return
		}

		// Should handle empty content gracefully
		if len(response.Suggestions) != 0 {
			t.Errorf("Expected empty suggestions, got %v", response.Suggestions)
		}
	})

	t.Run("missing usage information", func(t *testing.T) {
		mockClient := &CompatibilityMockClient{
			chatCompletion: &openai.ChatCompletion{
				ID:      "no-usage-test",
				Object:  "chat.completion",
				Created: 1234567890,
				Model:   "gpt-4",
				Choices: []openai.ChatCompletionChoice{
					{
						Index: 0,
						Message: openai.ChatCompletionMessage{
							Role:    "assistant",
							Content: "Response without usage",
						},
						FinishReason: "stop",
					},
				},
				// Usage field omitted to test missing usage handling
			},
		}

		client := createTestClient(mockClient)

		response, err := client.GenerateCompletion(context.Background(), types.CompletionRequest{
			Code:     "func test() {",
			Cursor:   13,
			Language: "go",
			Context: types.CodeContext{
				CurrentFunction: "test",
			},
		})

		if err != nil {
			t.Errorf("Unexpected error with missing usage: %v", err)
			return
		}

		// Should handle missing usage gracefully
		if len(response.Suggestions) == 0 {
			t.Error("Expected suggestions even without usage info")
		}
	})
}

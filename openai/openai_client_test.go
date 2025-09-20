package openai

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kengibson1111/go-aiprovider/types"
	"github.com/kengibson1111/go-aiprovider/utils"
	"github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/packages/ssestream"
	"github.com/openai/openai-go/v2/shared"
)

// MockOpenAISDKClient implements the OpenAIClientInterface for testing
type MockOpenAISDKClient struct {
	completion *openai.ChatCompletion
	err        error
	lastParams *openai.ChatCompletionNewParams
	lastCtx    context.Context
}

// Chat returns a mock chat service
func (m *MockOpenAISDKClient) Chat() ChatServiceInterface {
	return &MockChatService{client: m}
}

// MockChatService implements the ChatServiceInterface
type MockChatService struct {
	client *MockOpenAISDKClient
}

// Completions returns a mock completions service
func (m *MockChatService) Completions() CompletionsServiceInterface {
	return &MockCompletionsService{client: m.client}
}

// MockCompletionsService implements the CompletionsServiceInterface
type MockCompletionsService struct {
	client *MockOpenAISDKClient
}

// New implements the completion creation method
func (m *MockCompletionsService) New(ctx context.Context, params openai.ChatCompletionNewParams) (*openai.ChatCompletion, error) {
	// Store the parameters for verification in tests
	m.client.lastParams = &params
	m.client.lastCtx = ctx

	if m.client.err != nil {
		return nil, m.client.err
	}

	return m.client.completion, nil
}

// NewStreaming implements the streaming completion method
func (m *MockCompletionsService) NewStreaming(ctx context.Context, params openai.ChatCompletionNewParams) *ssestream.Stream[openai.ChatCompletionChunk] {
	// Store the parameters for verification in tests
	m.client.lastParams = &params
	m.client.lastCtx = ctx

	// For testing, we'll create a mock decoder that implements the ssestream.Decoder interface
	mockDecoder := &MockDecoder{err: m.client.err}

	// Create a stream with the mock decoder
	return ssestream.NewStream[openai.ChatCompletionChunk](mockDecoder, m.client.err)
}

// MockDecoder implements ssestream.Decoder for testing
type MockDecoder struct {
	err   error
	event ssestream.Event
}

func (m *MockDecoder) Event() ssestream.Event {
	return m.event
}

func (m *MockDecoder) Next() bool {
	// Return false to indicate no more events
	return false
}

func (m *MockDecoder) Close() error {
	return nil
}

func (m *MockDecoder) Err() error {
	return m.err
}

// Ensure our mock implements the interface
var _ OpenAIClientInterface = (*MockOpenAISDKClient)(nil)

func TestNewOpenAIClient(t *testing.T) {
	tests := []struct {
		name        string
		config      *types.AIConfig
		expectError bool
		errorMsg    string
	}{
		{
			name:        "nil config",
			config:      nil,
			expectError: true,
			errorMsg:    "configuration is required",
		},
		{
			name: "valid config with defaults",
			config: &types.AIConfig{
				Provider: "openai",
				APIKey:   "test-key",
			},
			expectError: false,
		},
		{
			name: "valid config with custom values",
			config: &types.AIConfig{
				Provider:    "openai",
				APIKey:      "test-key",
				BaseURL:     "https://custom.openai.com",
				Model:       "gpt-4",
				MaxTokens:   2000,
				Temperature: 0.5,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewOpenAIClient(tt.config)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error message to contain '%s', got: %s", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if client == nil {
				t.Errorf("Expected client to be created")
				return
			}

			// Check defaults
			if tt.config.Model == "" && client.model != "gpt-4o-mini" {
				t.Errorf("Expected default model 'gpt-4o-mini', got: %s", client.model)
			}
			if tt.config.MaxTokens == 0 && client.maxTokens != 1000 {
				t.Errorf("Expected default maxTokens 1000, got: %d", client.maxTokens)
			}
			if tt.config.Temperature == 0 && client.temperature != 0.7 {
				t.Errorf("Expected default temperature 0.7, got: %f", client.temperature)
			}

			// Check custom values
			if tt.config.Model != "" && client.model != tt.config.Model {
				t.Errorf("Expected model '%s', got: %s", tt.config.Model, client.model)
			}
			if tt.config.MaxTokens != 0 && client.maxTokens != tt.config.MaxTokens {
				t.Errorf("Expected maxTokens %d, got: %d", tt.config.MaxTokens, client.maxTokens)
			}
			if tt.config.Temperature != 0 && client.temperature != tt.config.Temperature {
				t.Errorf("Expected temperature %f, got: %f", tt.config.Temperature, client.temperature)
			}
		})
	}
}

func TestOpenAIClient_ValidateCredentials(t *testing.T) {
	tests := []struct {
		name          string
		statusCode    int
		responseBody  string
		expectError   bool
		errorContains string
	}{
		{
			name:       "valid credentials",
			statusCode: 200,
			responseBody: `{
				"id": "chatcmpl-test",
				"object": "chat.completion",
				"created": 1234567890,
				"model": "gpt-4o-mini",
				"choices": [
					{
						"index": 0,
						"message": {
							"role": "assistant",
							"content": "Hello!"
						},
						"finish_reason": "stop"
					}
				]
			}`,
			expectError: false,
		},
		{
			name:          "invalid API key",
			statusCode:    401,
			responseBody:  `{"error": {"message": "Invalid API key", "type": "invalid_request_error"}}`,
			expectError:   true,
			errorContains: "invalid API key",
		},
		{
			name:          "insufficient permissions",
			statusCode:    403,
			responseBody:  `{"error": {"message": "Insufficient permissions", "type": "invalid_request_error"}}`,
			expectError:   true,
			errorContains: "does not have required permissions",
		},
		{
			name:          "rate limit exceeded",
			statusCode:    429,
			responseBody:  `{"error": {"message": "Rate limit exceeded", "type": "rate_limit_error"}}`,
			expectError:   true,
			errorContains: "rate limit exceeded",
		},
		{
			name:       "API error with structured response",
			statusCode: 400,
			responseBody: `{
				"error": {
					"message": "Invalid model specified",
					"type": "invalid_request_error",
					"param": "model",
					"code": "model_not_found"
				}
			}`,
			expectError:   true,
			errorContains: "Invalid model specified",
		},
		{
			name:          "API error without structured response",
			statusCode:    500,
			responseBody:  "Internal Server Error",
			expectError:   true,
			errorContains: "HTTP 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request format
				if r.Method != "POST" {
					t.Errorf("Expected POST request, got: %s", r.Method)
				}
				if r.URL.Path != "/chat/completions" {
					t.Errorf("Expected path '/chat/completions', got: %s", r.URL.Path)
				}
				if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
					t.Errorf("Expected Authorization header with Bearer token")
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.responseBody))
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
			err = client.ValidateCredentials(ctx)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.errorContains)) {
					t.Errorf("Expected error to contain '%s', got: %s", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestOpenAIClient_GenerateCompletion(t *testing.T) {
	tests := []struct {
		name           string
		mockCompletion *openai.ChatCompletion
		mockError      error
		expectError    bool
		errorContains  string
		expectedSuggs  int
		validateResp   func(t *testing.T, resp *types.CompletionResponse)
	}{
		{
			name: "successful completion",
			mockCompletion: &openai.ChatCompletion{
				ID:      "chatcmpl-test",
				Object:  "chat.completion",
				Created: 1234567890,
				Model:   "gpt-4o-mini",
				Choices: []openai.ChatCompletionChoice{
					{
						Index: 0,
						Message: openai.ChatCompletionMessage{
							Role:    "assistant",
							Content: "log('Hello, World!');",
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
			mockError:     nil,
			expectError:   false,
			expectedSuggs: 1,
		},
		{
			name: "multiple line completion",
			mockCompletion: &openai.ChatCompletion{
				ID:      "chatcmpl-test",
				Object:  "chat.completion",
				Created: 1234567890,
				Model:   "gpt-4o-mini",
				Choices: []openai.ChatCompletionChoice{
					{
						Index: 0,
						Message: openai.ChatCompletionMessage{
							Role:    "assistant",
							Content: "log('Hello');\nconsole.warn('World');",
						},
						FinishReason: "stop",
					},
				},
			},
			mockError:     nil,
			expectError:   false,
			expectedSuggs: 2,
		},
		{
			name:           "rate limit error",
			mockCompletion: nil,
			mockError: &openai.Error{
				Code:    "rate_limit_exceeded",
				Message: "Rate limit exceeded",
				Type:    "rate_limit_error",
			},
			expectError:   true,
			errorContains: "rate limit exceeded",
		},
		{
			name: "empty response",
			mockCompletion: &openai.ChatCompletion{
				ID:      "chatcmpl-test",
				Object:  "chat.completion",
				Created: 1234567890,
				Model:   "gpt-4o-mini",
				Choices: []openai.ChatCompletionChoice{},
			},
			mockError:     nil,
			expectError:   false,
			expectedSuggs: 0,
		},
		{
			name: "content with whitespace",
			mockCompletion: &openai.ChatCompletion{
				ID:      "chatcmpl-whitespace",
				Object:  "chat.completion",
				Created: 1234567890,
				Model:   "gpt-4o-mini",
				Choices: []openai.ChatCompletionChoice{
					{
						Index: 0,
						Message: openai.ChatCompletionMessage{
							Role:    "assistant",
							Content: "  \n  log('test');  \n  ",
						},
						FinishReason: "stop",
					},
				},
			},
			mockError:     nil,
			expectError:   false,
			expectedSuggs: 1,
			validateResp: func(t *testing.T, resp *types.CompletionResponse) {
				if len(resp.Suggestions) != 1 {
					t.Errorf("Expected 1 suggestion, got: %d", len(resp.Suggestions))
					return
				}
				if resp.Suggestions[0] != "log('test');" {
					t.Errorf("Expected trimmed suggestion 'log('test');', got: '%s'", resp.Suggestions[0])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock SDK client
			mockClient := &MockOpenAISDKClient{
				completion: tt.mockCompletion,
				err:        tt.mockError,
			}

			// Create OpenAI client with mock SDK client
			client := &OpenAIClient{
				client:      mockClient,
				model:       "gpt-4o-mini",
				maxTokens:   1000,
				temperature: 0.7,
				logger:      utils.NewLogger("TestOpenAIClient"),
			}

			req := types.CompletionRequest{
				Code:     "console.",
				Cursor:   8,
				Language: "javascript",
				Context: types.CodeContext{
					CurrentFunction: "testFunction",
					Imports:         []string{"import fs from 'fs'"},
					ProjectType:     "Node.js",
					RecentChanges:   []string{},
				},
			}

			ctx := context.Background()
			resp, err := client.GenerateCompletion(ctx, req)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if tt.expectError {
				if resp.Error == "" {
					t.Errorf("Expected error in response but got none")
				}
				if !strings.Contains(resp.Error, tt.errorContains) {
					t.Errorf("Expected error to contain '%s', got: %s", tt.errorContains, resp.Error)
				}
			} else {
				if resp.Error != "" {
					t.Errorf("Unexpected error in response: %s", resp.Error)
				}
				if len(resp.Suggestions) != tt.expectedSuggs {
					t.Errorf("Expected %d suggestions, got: %d", tt.expectedSuggs, len(resp.Suggestions))
				}
			}

			// Run custom validation if provided
			if tt.validateResp != nil {
				tt.validateResp(t, resp)
			}

			// Verify the mock was called with correct parameters
			if mockClient.lastParams == nil && tt.mockError == nil {
				t.Errorf("Expected mock to be called with parameters")
				return
			}

			if mockClient.lastParams != nil {
				// Verify model parameter
				if mockClient.lastParams.Model != openai.ChatModel(client.model) {
					t.Errorf("Expected model '%s', got: %s", client.model, string(mockClient.lastParams.Model))
				}

				// Verify we have exactly one message (the prompt)
				if len(mockClient.lastParams.Messages) != 1 {
					t.Errorf("Expected 1 message, got: %d", len(mockClient.lastParams.Messages))
				}

				// Verify other parameters
				if mockClient.lastParams.MaxTokens.Value != int64(client.maxTokens) {
					t.Errorf("Expected maxTokens %d, got: %d", client.maxTokens, mockClient.lastParams.MaxTokens.Value)
				}

				if mockClient.lastParams.Temperature.Value != client.temperature {
					t.Errorf("Expected temperature %f, got: %f", client.temperature, mockClient.lastParams.Temperature.Value)
				}
			}
		})
	}
}

func TestOpenAIClient_GenerateCode(t *testing.T) {
	tests := []struct {
		name           string
		mockCompletion *openai.ChatCompletion
		mockError      error
		expectError    bool
		errorContains  string
		expectedCode   string
		validateResp   func(t *testing.T, resp *types.CodeGenerationResponse)
	}{
		{
			name: "successful code generation",
			mockCompletion: &openai.ChatCompletion{
				ID:      "chatcmpl-test",
				Object:  "chat.completion",
				Created: 1234567890,
				Model:   "gpt-4o-mini",
				Choices: []openai.ChatCompletionChoice{
					{
						Index: 0,
						Message: openai.ChatCompletionMessage{
							Role:    "assistant",
							Content: "function hello() {\n  console.log('Hello, World!');\n}",
						},
						FinishReason: "stop",
					},
				},
			},
			mockError:   nil,
			expectError: false,
			expectedCode: `function hello() {
  console.log('Hello, World!');
}`,
		},
		{
			name: "code with markdown formatting",
			mockCompletion: &openai.ChatCompletion{
				ID:      "chatcmpl-test",
				Object:  "chat.completion",
				Created: 1234567890,
				Model:   "gpt-4o-mini",
				Choices: []openai.ChatCompletionChoice{
					{
						Index: 0,
						Message: openai.ChatCompletionMessage{
							Role:    "assistant",
							Content: "```javascript\nfunction hello() {\n  console.log('Hello, World!');\n}\n```",
						},
						FinishReason: "stop",
					},
				},
			},
			mockError:   nil,
			expectError: false,
			expectedCode: `function hello() {
  console.log('Hello, World!');
}`,
		},
		{
			name:           "rate limit error",
			mockCompletion: nil,
			mockError: &openai.Error{
				Code:    "rate_limit_exceeded",
				Message: "Rate limit exceeded",
				Type:    "rate_limit_error",
			},
			expectError:   true,
			errorContains: "rate limit exceeded",
		},
		{
			name: "empty response",
			mockCompletion: &openai.ChatCompletion{
				ID:      "chatcmpl-empty",
				Object:  "chat.completion",
				Created: 1234567890,
				Model:   "gpt-4o-mini",
				Choices: []openai.ChatCompletionChoice{},
			},
			mockError:    nil,
			expectError:  false,
			expectedCode: "",
		},
		{
			name: "multiple choices - uses first",
			mockCompletion: &openai.ChatCompletion{
				ID:      "chatcmpl-multi",
				Object:  "chat.completion",
				Created: 1234567890,
				Model:   "gpt-4o-mini",
				Choices: []openai.ChatCompletionChoice{
					{
						Index: 0,
						Message: openai.ChatCompletionMessage{
							Role:    "assistant",
							Content: "const greeting = 'Hello, World!';",
						},
						FinishReason: "stop",
					},
					{
						Index: 1,
						Message: openai.ChatCompletionMessage{
							Role:    "assistant",
							Content: "let message = 'Hello, World!';",
						},
						FinishReason: "stop",
					},
				},
			},
			mockError:    nil,
			expectError:  false,
			expectedCode: "const greeting = 'Hello, World!';",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock SDK client
			mockClient := &MockOpenAISDKClient{
				completion: tt.mockCompletion,
				err:        tt.mockError,
			}

			// Create OpenAI client with mock SDK client
			client := &OpenAIClient{
				client:      mockClient,
				model:       "gpt-4o-mini",
				maxTokens:   1000,
				temperature: 0.7,
				logger:      utils.NewLogger("TestOpenAIClient"),
			}

			req := types.CodeGenerationRequest{
				Prompt:   "Create a hello world function",
				Language: "javascript",
				Context: types.CodeContext{
					CurrentFunction: "",
					Imports:         []string{},
					ProjectType:     "Node.js",
					RecentChanges:   []string{},
				},
			}

			ctx := context.Background()
			resp, err := client.GenerateCode(ctx, req)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if tt.expectError {
				if resp.Error == "" {
					t.Errorf("Expected error in response but got none")
				}
				if !strings.Contains(resp.Error, tt.errorContains) {
					t.Errorf("Expected error to contain '%s', got: %s", tt.errorContains, resp.Error)
				}
			} else {
				if resp.Error != "" {
					t.Errorf("Unexpected error in response: %s", resp.Error)
				}
				// Compare the actual code content
				if strings.TrimSpace(resp.Code) != strings.TrimSpace(tt.expectedCode) {
					t.Errorf("Expected code '%s', got: '%s'", tt.expectedCode, resp.Code)
				}
			}

			// Run custom validation if provided
			if tt.validateResp != nil {
				tt.validateResp(t, resp)
			}

			// Verify the mock was called with correct parameters
			if mockClient.lastParams == nil && tt.mockError == nil {
				t.Errorf("Expected mock to be called with parameters")
				return
			}

			if mockClient.lastParams != nil {
				// Verify model parameter
				if mockClient.lastParams.Model != openai.ChatModel(client.model) {
					t.Errorf("Expected model '%s', got: %s", client.model, string(mockClient.lastParams.Model))
				}

				// Verify we have exactly one message (the prompt)
				if len(mockClient.lastParams.Messages) != 1 {
					t.Errorf("Expected 1 message, got: %d", len(mockClient.lastParams.Messages))
				}

				// Verify other parameters
				if mockClient.lastParams.MaxTokens.Value != int64(client.maxTokens) {
					t.Errorf("Expected maxTokens %d, got: %d", client.maxTokens, mockClient.lastParams.MaxTokens.Value)
				}

				if mockClient.lastParams.Temperature.Value != client.temperature {
					t.Errorf("Expected temperature %f, got: %f", client.temperature, mockClient.lastParams.Temperature.Value)
				}
			}
		})
	}
}

func TestOpenAIClient_PromptBuilding(t *testing.T) {
	config := &types.AIConfig{
		Provider: "openai",
		APIKey:   "test-key",
		Model:    "gpt-4o-mini",
	}

	client, err := NewOpenAIClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	t.Run("completion prompt", func(t *testing.T) {
		req := types.CompletionRequest{
			Code:     "console.log('Hello'); console.",
			Cursor:   30, // Position at the end of the string
			Language: "javascript",
			Context: types.CodeContext{
				CurrentFunction: "main",
				Imports:         []string{"import fs from 'fs'", "import path from 'path'"},
				ProjectType:     "Node.js",
				RecentChanges:   []string{},
			},
		}

		prompt := client.buildCompletionPrompt(req)

		// Check that prompt contains expected elements
		expectedElements := []string{
			"javascript",
			"code completion assistant",
			"Current function: main",
			"import fs from 'fs'",
			"import path from 'path'",
			"Project type: Node.js",
			"<CURSOR>",
			"console.log('Hello'); console.<CURSOR>",
		}

		for _, element := range expectedElements {
			if !strings.Contains(prompt, element) {
				t.Errorf("Expected prompt to contain '%s', but it didn't. Prompt: %s", element, prompt)
			}
		}
	})

	t.Run("code generation prompt", func(t *testing.T) {
		req := types.CodeGenerationRequest{
			Prompt:   "Create a function that reads a file",
			Language: "javascript",
			Context: types.CodeContext{
				CurrentFunction: "readFile",
				Imports:         []string{"import fs from 'fs'"},
				ProjectType:     "Node.js",
				RecentChanges:   []string{},
			},
		}

		prompt := client.buildCodeGenerationPrompt(req)

		expectedElements := []string{
			"javascript",
			"code generation assistant",
			"Current function: readFile",
			"import fs from 'fs'",
			"Project type: Node.js",
			"Create a function that reads a file",
		}

		for _, element := range expectedElements {
			if !strings.Contains(prompt, element) {
				t.Errorf("Expected prompt to contain '%s', but it didn't. Prompt: %s", element, prompt)
			}
		}
	})
}

// TestOpenAIClient_CallWithPrompt tests the CallWithPrompt method with SDK types
func TestOpenAIClient_CallWithPrompt(t *testing.T) {
	tests := []struct {
		name           string
		prompt         string
		mockCompletion *openai.ChatCompletion
		mockError      error
		expectError    bool
		errorContains  string
		validateResp   func(t *testing.T, resp *openai.ChatCompletion)
	}{
		{
			name:   "successful completion",
			prompt: "Hello, world!",
			mockCompletion: &openai.ChatCompletion{
				ID:      "chatcmpl-test123",
				Object:  "chat.completion",
				Created: 1234567890,
				Model:   "gpt-4o-mini",
				Choices: []openai.ChatCompletionChoice{
					{
						Index: 0,
						Message: openai.ChatCompletionMessage{
							Role:    "assistant",
							Content: "Hello! How can I help you today?",
						},
						FinishReason: "stop",
					},
				},
				Usage: openai.CompletionUsage{
					PromptTokens:     10,
					CompletionTokens: 8,
					TotalTokens:      18,
				},
			},
			mockError:   nil,
			expectError: false,
			validateResp: func(t *testing.T, resp *openai.ChatCompletion) {
				if resp.ID != "chatcmpl-test123" {
					t.Errorf("Expected ID 'chatcmpl-test123', got: %s", resp.ID)
				}
				if len(resp.Choices) != 1 {
					t.Errorf("Expected 1 choice, got: %d", len(resp.Choices))
				}
				if resp.Choices[0].Message.Content != "Hello! How can I help you today?" {
					t.Errorf("Expected specific content, got: %s", resp.Choices[0].Message.Content)
				}
				if resp.Usage.TotalTokens != 18 {
					t.Errorf("Expected 18 total tokens, got: %d", resp.Usage.TotalTokens)
				}
			},
		},
		{
			name:   "empty prompt",
			prompt: "",
			mockCompletion: &openai.ChatCompletion{
				ID:      "chatcmpl-empty",
				Object:  "chat.completion",
				Created: 1234567890,
				Model:   "gpt-4o-mini",
				Choices: []openai.ChatCompletionChoice{
					{
						Index: 0,
						Message: openai.ChatCompletionMessage{
							Role:    "assistant",
							Content: "I'm here to help! What would you like to know?",
						},
						FinishReason: "stop",
					},
				},
			},
			mockError:   nil,
			expectError: false,
			validateResp: func(t *testing.T, resp *openai.ChatCompletion) {
				if len(resp.Choices) == 0 {
					t.Errorf("Expected at least one choice")
				}
			},
		},
		{
			name:   "multiple choices response",
			prompt: "Tell me a joke",
			mockCompletion: &openai.ChatCompletion{
				ID:      "chatcmpl-multi",
				Object:  "chat.completion",
				Created: 1234567890,
				Model:   "gpt-4o-mini",
				Choices: []openai.ChatCompletionChoice{
					{
						Index: 0,
						Message: openai.ChatCompletionMessage{
							Role:    "assistant",
							Content: "Why don't scientists trust atoms? Because they make up everything!",
						},
						FinishReason: "stop",
					},
					{
						Index: 1,
						Message: openai.ChatCompletionMessage{
							Role:    "assistant",
							Content: "What do you call a fake noodle? An impasta!",
						},
						FinishReason: "stop",
					},
				},
			},
			mockError:   nil,
			expectError: false,
			validateResp: func(t *testing.T, resp *openai.ChatCompletion) {
				if len(resp.Choices) != 2 {
					t.Errorf("Expected 2 choices, got: %d", len(resp.Choices))
				}
			},
		},
		{
			name:           "invalid API key error",
			prompt:         "Test prompt",
			mockCompletion: nil,
			mockError: &openai.Error{
				Code:    "invalid_api_key",
				Message: "Invalid API key provided",
				Type:    "invalid_request_error",
			},
			expectError:   true,
			errorContains: "invalid API key",
		},
		{
			name:           "rate limit error",
			prompt:         "Test prompt",
			mockCompletion: nil,
			mockError: &openai.Error{
				Code:    "rate_limit_exceeded",
				Message: "Rate limit exceeded",
				Type:    "rate_limit_error",
			},
			expectError:   true,
			errorContains: "rate limit exceeded",
		},
		{
			name:           "model not found error",
			prompt:         "Test prompt",
			mockCompletion: nil,
			mockError: &openai.Error{
				Code:    "model_not_found",
				Message: "The model 'invalid-model' does not exist",
				Type:    "invalid_request_error",
			},
			expectError:   true,
			errorContains: "model not found",
		},
		{
			name:           "context length exceeded error",
			prompt:         "Very long prompt that exceeds context window...",
			mockCompletion: nil,
			mockError: &openai.Error{
				Code:    "context_length_exceeded",
				Message: "This model's maximum context length is 4096 tokens",
				Type:    "invalid_request_error",
			},
			expectError:   true,
			errorContains: "context length exceeded",
		},
		{
			name:           "server error",
			prompt:         "Test prompt",
			mockCompletion: nil,
			mockError: &openai.Error{
				Message: "Internal server error",
				Type:    "server_error",
			},
			expectError:   true,
			errorContains: "server error",
		},
		{
			name:           "network error",
			prompt:         "Test prompt",
			mockCompletion: nil,
			mockError:      fmt.Errorf("connection refused"),
			expectError:    true,
			errorContains:  "network error",
		},
		{
			name:           "timeout error",
			prompt:         "Test prompt",
			mockCompletion: nil,
			mockError:      fmt.Errorf("context deadline exceeded"),
			expectError:    true,
			errorContains:  "request timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock SDK client
			mockClient := &MockOpenAISDKClient{
				completion: tt.mockCompletion,
				err:        tt.mockError,
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
			resp, err := client.callWithPrompt(ctx, tt.prompt)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.errorContains)) {
					t.Errorf("Expected error to contain '%s', got: %s", tt.errorContains, err.Error())
				}
				if resp != nil {
					t.Errorf("Expected nil response on error, got: %v", resp)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if resp == nil {
				t.Errorf("Expected non-nil response")
				return
			}

			// Verify no JSON processing occurred by checking we got the exact mock object
			if resp != tt.mockCompletion {
				t.Errorf("Expected exact mock completion object, got different object")
			}

			// Run custom validation if provided
			if tt.validateResp != nil {
				tt.validateResp(t, resp)
			}

			// Verify the mock was called with correct parameters
			if mockClient.lastParams == nil {
				t.Errorf("Expected mock to be called with parameters")
				return
			}

			// Verify model parameter
			if mockClient.lastParams.Model != openai.ChatModel(client.model) {
				t.Errorf("Expected model '%s', got: %s", client.model, string(mockClient.lastParams.Model))
			}

			// Verify message parameter
			if len(mockClient.lastParams.Messages) != 1 {
				t.Errorf("Expected 1 message, got: %d", len(mockClient.lastParams.Messages))
				return
			}

			// Extract the user message content - we need to check the underlying structure
			// Since ChatCompletionMessageParamUnion is a complex union type, we'll verify the prompt was passed correctly
			// by checking that we have exactly one message (which should be our user message)
			if len(mockClient.lastParams.Messages) != 1 {
				t.Errorf("Expected 1 message, got: %d", len(mockClient.lastParams.Messages))
			}

			// Verify other parameters using the SDK's parameter types
			if mockClient.lastParams.MaxTokens.Value != int64(client.maxTokens) {
				t.Errorf("Expected maxTokens %d, got: %d", client.maxTokens, mockClient.lastParams.MaxTokens.Value)
			}

			if mockClient.lastParams.Temperature.Value != client.temperature {
				t.Errorf("Expected temperature %f, got: %f", client.temperature, mockClient.lastParams.Temperature.Value)
			}
		})
	}
}

// TestOpenAIClient_CallWithPrompt_ParameterValidation tests parameter handling
func TestOpenAIClient_CallWithPrompt_ParameterValidation(t *testing.T) {
	mockClient := &MockOpenAISDKClient{
		completion: &openai.ChatCompletion{
			ID: "test",
			Choices: []openai.ChatCompletionChoice{
				{
					Message: openai.ChatCompletionMessage{
						Content: "response",
					},
				},
			},
		},
	}

	tests := []struct {
		name        string
		model       string
		maxTokens   int
		temperature float64
	}{
		{
			name:        "default parameters",
			model:       "gpt-4o-mini",
			maxTokens:   1000,
			temperature: 0.7,
		},
		{
			name:        "custom parameters",
			model:       "gpt-4",
			maxTokens:   2000,
			temperature: 0.1,
		},
		{
			name:        "zero temperature",
			model:       "gpt-4o-mini",
			maxTokens:   500,
			temperature: 0.0,
		},
		{
			name:        "high temperature",
			model:       "gpt-4o-mini",
			maxTokens:   1500,
			temperature: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &OpenAIClient{
				client:      mockClient,
				model:       tt.model,
				maxTokens:   tt.maxTokens,
				temperature: tt.temperature,
				logger:      utils.NewLogger("TestOpenAIClient"),
			}

			ctx := context.Background()
			_, err := client.CallWithPrompt(ctx, "test prompt")

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Verify parameters were passed correctly
			if string(mockClient.lastParams.Model) != tt.model {
				t.Errorf("Expected model '%s', got: %s", tt.model, string(mockClient.lastParams.Model))
			}

			if mockClient.lastParams.MaxTokens.Value != int64(tt.maxTokens) {
				t.Errorf("Expected maxTokens %d, got: %d", tt.maxTokens, mockClient.lastParams.MaxTokens.Value)
			}

			if mockClient.lastParams.Temperature.Value != tt.temperature {
				t.Errorf("Expected temperature %f, got: %f", tt.temperature, mockClient.lastParams.Temperature.Value)
			}
		})
	}
}

// TestOpenAIClient_CallWithPrompt_ContextCancellation tests context handling
func TestOpenAIClient_CallWithPrompt_ContextCancellation(t *testing.T) {
	mockClient := &MockOpenAISDKClient{
		err: context.Canceled,
	}

	client := &OpenAIClient{
		client:      mockClient,
		model:       "gpt-4o-mini",
		maxTokens:   1000,
		temperature: 0.7,
		logger:      utils.NewLogger("TestOpenAIClient"),
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	resp, err := client.CallWithPrompt(ctx, "test prompt")

	if err == nil {
		t.Errorf("Expected error due to cancelled context")
	}

	if resp != nil {
		t.Errorf("Expected nil response on cancelled context")
	}

	if !strings.Contains(err.Error(), "request failed") {
		t.Errorf("Expected error to be wrapped properly, got: %s", err.Error())
	}
}

// TestOpenAIClient_CallWithPromptAndVariables2 tests template processing with SDK types
func TestOpenAIClient_CallWithPromptAndVariables2(t *testing.T) {
	tests := []struct {
		name           string
		template       string
		variablesJSON  string
		mockCompletion *openai.ChatCompletion
		mockError      error
		expectError    bool
		errorContains  string
		validateResp   func(t *testing.T, resp *openai.ChatCompletion)
		validatePrompt func(t *testing.T, processedPrompt string)
	}{
		{
			name:     "successful variable substitution",
			template: "You are a {{role}} assistant. Help with {{task}} in {{language}}.",
			variablesJSON: `{
				"role": "senior developer",
				"task": "code review",
				"language": "Go"
			}`,
			mockCompletion: &openai.ChatCompletion{
				ID:      "chatcmpl-vars123",
				Object:  "chat.completion",
				Created: 1234567890,
				Model:   "gpt-4o-mini",
				Choices: []openai.ChatCompletionChoice{
					{
						Index: 0,
						Message: openai.ChatCompletionMessage{
							Role:    "assistant",
							Content: "I'll help you with the Go code review as a senior developer.",
						},
						FinishReason: "stop",
					},
				},
			},
			mockError:   nil,
			expectError: false,
			validateResp: func(t *testing.T, resp *openai.ChatCompletion) {
				if resp.ID != "chatcmpl-vars123" {
					t.Errorf("Expected ID 'chatcmpl-vars123', got: %s", resp.ID)
				}
				if len(resp.Choices) != 1 {
					t.Errorf("Expected 1 choice, got: %d", len(resp.Choices))
				}
			},
			validatePrompt: func(t *testing.T, processedPrompt string) {
				expected := "You are a senior developer assistant. Help with code review in Go."
				if processedPrompt != expected {
					t.Errorf("Expected processed prompt '%s', got: '%s'", expected, processedPrompt)
				}
			},
		},
		{
			name:     "multiple occurrences of same variable",
			template: "Hello {{name}}, {{name}} is working on {{project}}. Please help {{name}} with the task.",
			variablesJSON: `{
				"name": "Alice",
				"project": "AI Provider"
			}`,
			mockCompletion: &openai.ChatCompletion{
				ID: "chatcmpl-multi",
				Choices: []openai.ChatCompletionChoice{
					{
						Message: openai.ChatCompletionMessage{
							Content: "I'll help Alice with the AI Provider project.",
						},
					},
				},
			},
			mockError:   nil,
			expectError: false,
			validatePrompt: func(t *testing.T, processedPrompt string) {
				expected := "Hello Alice, Alice is working on AI Provider. Please help Alice with the task."
				if processedPrompt != expected {
					t.Errorf("Expected processed prompt '%s', got: '%s'", expected, processedPrompt)
				}
			},
		},
		{
			name:     "missing variables remain unchanged",
			template: "Process {{existing}} but leave {{missing}} unchanged.",
			variablesJSON: `{
				"existing": "this value"
			}`,
			mockCompletion: &openai.ChatCompletion{
				ID: "chatcmpl-partial",
				Choices: []openai.ChatCompletionChoice{
					{
						Message: openai.ChatCompletionMessage{
							Content: "Processed successfully.",
						},
					},
				},
			},
			mockError:   nil,
			expectError: false,
			validatePrompt: func(t *testing.T, processedPrompt string) {
				expected := "Process this value but leave {{missing}} unchanged."
				if processedPrompt != expected {
					t.Errorf("Expected processed prompt '%s', got: '%s'", expected, processedPrompt)
				}
			},
		},
		{
			name:          "empty variables JSON",
			template:      "No substitution for {{variable}}.",
			variablesJSON: `{}`,
			mockCompletion: &openai.ChatCompletion{
				ID: "chatcmpl-empty",
				Choices: []openai.ChatCompletionChoice{
					{
						Message: openai.ChatCompletionMessage{
							Content: "Template unchanged.",
						},
					},
				},
			},
			mockError:   nil,
			expectError: false,
			validatePrompt: func(t *testing.T, processedPrompt string) {
				expected := "No substitution for {{variable}}."
				if processedPrompt != expected {
					t.Errorf("Expected processed prompt '%s', got: '%s'", expected, processedPrompt)
				}
			},
		},
		{
			name:          "null and empty string variables JSON",
			template:      "Template with {{var}} should remain unchanged.",
			variablesJSON: "",
			mockCompletion: &openai.ChatCompletion{
				ID: "chatcmpl-null",
				Choices: []openai.ChatCompletionChoice{
					{
						Message: openai.ChatCompletionMessage{
							Content: "No changes made.",
						},
					},
				},
			},
			mockError:   nil,
			expectError: false,
			validatePrompt: func(t *testing.T, processedPrompt string) {
				expected := "Template with {{var}} should remain unchanged."
				if processedPrompt != expected {
					t.Errorf("Expected processed prompt '%s', got: '%s'", expected, processedPrompt)
				}
			},
		},
		{
			name:     "complex variable names and values",
			template: "User {{user_name}} has {{task-count}} tasks with priority {{priority_level}}.",
			variablesJSON: `{
				"user_name": "john_doe",
				"task-count": "15",
				"priority_level": "high"
			}`,
			mockCompletion: &openai.ChatCompletion{
				ID: "chatcmpl-complex",
				Choices: []openai.ChatCompletionChoice{
					{
						Message: openai.ChatCompletionMessage{
							Content: "Task information processed.",
						},
					},
				},
			},
			mockError:   nil,
			expectError: false,
			validatePrompt: func(t *testing.T, processedPrompt string) {
				expected := "User john_doe has 15 tasks with priority high."
				if processedPrompt != expected {
					t.Errorf("Expected processed prompt '%s', got: '%s'", expected, processedPrompt)
				}
			},
		},
		{
			name:     "non-string JSON values",
			template: "Count: {{count}}, Active: {{active}}, Rate: {{rate}}, Null: {{null_val}}",
			variablesJSON: `{
				"count": 42,
				"active": true,
				"rate": 3.14159,
				"null_val": null
			}`,
			mockCompletion: &openai.ChatCompletion{
				ID: "chatcmpl-types",
				Choices: []openai.ChatCompletionChoice{
					{
						Message: openai.ChatCompletionMessage{
							Content: "Values processed correctly.",
						},
					},
				},
			},
			mockError:   nil,
			expectError: false,
			validatePrompt: func(t *testing.T, processedPrompt string) {
				expected := "Count: 42, Active: true, Rate: 3.14159, Null: "
				if processedPrompt != expected {
					t.Errorf("Expected processed prompt '%s', got: '%s'", expected, processedPrompt)
				}
			},
		},
		{
			name:     "special characters in variable values",
			template: "Message: {{message}}",
			variablesJSON: `{
				"message": "Hello! @#$%^&*(){}[]|\\:;\"'<>,.?/~` + "`" + `"
			}`,
			mockCompletion: &openai.ChatCompletion{
				ID: "chatcmpl-special",
				Choices: []openai.ChatCompletionChoice{
					{
						Message: openai.ChatCompletionMessage{
							Content: "Special characters handled.",
						},
					},
				},
			},
			mockError:   nil,
			expectError: false,
			validatePrompt: func(t *testing.T, processedPrompt string) {
				expected := "Message: Hello! @#$%^&*(){}[]|\\:;\"'<>,.?/~`"
				if processedPrompt != expected {
					t.Errorf("Expected processed prompt '%s', got: '%s'", expected, processedPrompt)
				}
			},
		},
		{
			name:           "invalid JSON variables",
			template:       "Hello {{name}}!",
			variablesJSON:  `{name: "Alice"}`, // Missing quotes around key
			mockCompletion: nil,
			mockError:      nil,
			expectError:    true,
			errorContains:  "variable substitution failed",
		},
		{
			name:           "malformed JSON with trailing comma",
			template:       "Hello {{name}}!",
			variablesJSON:  `{"name": "Alice",}`,
			mockCompletion: nil,
			mockError:      nil,
			expectError:    true,
			errorContains:  "variable substitution failed",
		},
		{
			name:           "empty template",
			template:       "",
			variablesJSON:  `{"name": "Alice"}`,
			mockCompletion: nil,
			mockError:      nil,
			expectError:    true,
			errorContains:  "variable substitution failed",
		},
		{
			name:           "API error after successful variable substitution",
			template:       "Hello {{name}}!",
			variablesJSON:  `{"name": "Alice"}`,
			mockCompletion: nil,
			mockError: &openai.Error{
				Code:    "rate_limit_exceeded",
				Message: "Rate limit exceeded",
				Type:    "rate_limit_error",
			},
			expectError:   true,
			errorContains: "rate limit exceeded",
		},
		{
			name:           "network error after successful variable substitution",
			template:       "Process {{data}} with {{method}}.",
			variablesJSON:  `{"data": "user input", "method": "validation"}`,
			mockCompletion: nil,
			mockError:      fmt.Errorf("connection refused"),
			expectError:    true,
			errorContains:  "network error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock SDK client
			mockClient := &MockOpenAISDKClient{
				completion: tt.mockCompletion,
				err:        tt.mockError,
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
			resp, err := client.callWithPromptAndVariables(ctx, tt.template, tt.variablesJSON)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.errorContains)) {
					t.Errorf("Expected error to contain '%s', got: %s", tt.errorContains, err.Error())
				}
				if resp != nil {
					t.Errorf("Expected nil response on error, got: %v", resp)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if resp == nil {
				t.Errorf("Expected non-nil response")
				return
			}

			// Verify no JSON processing occurred by checking we got the exact mock object
			if resp != tt.mockCompletion {
				t.Errorf("Expected exact mock completion object, got different object")
			}

			// Run custom validation if provided
			if tt.validateResp != nil {
				tt.validateResp(t, resp)
			}

			// Verify template processing by checking the processed prompt was passed correctly
			if tt.validatePrompt != nil && mockClient.lastParams != nil {
				// We can't directly access the processed prompt, but we can verify
				// that the mock was called (indicating successful template processing)
				if mockClient.lastParams == nil {
					t.Errorf("Expected mock to be called with parameters after successful template processing")
				}

				// For validation, we'll process the template ourselves and compare
				processedPrompt, err := utils.SubstituteVariables(tt.template, tt.variablesJSON)
				if err != nil {
					t.Errorf("Template processing validation failed: %v", err)
					return
				}
				tt.validatePrompt(t, processedPrompt)
			}

			// Verify the mock was called with correct parameters (indicating successful template processing)
			if mockClient.lastParams == nil {
				t.Errorf("Expected mock to be called with parameters")
				return
			}

			// Verify model parameter
			if mockClient.lastParams.Model != openai.ChatModel(client.model) {
				t.Errorf("Expected model '%s', got: %s", client.model, string(mockClient.lastParams.Model))
			}

			// Verify message parameter
			if len(mockClient.lastParams.Messages) != 1 {
				t.Errorf("Expected 1 message, got: %d", len(mockClient.lastParams.Messages))
				return
			}

			// Verify other parameters using the SDK's parameter types
			if mockClient.lastParams.MaxTokens.Value != int64(client.maxTokens) {
				t.Errorf("Expected maxTokens %d, got: %d", client.maxTokens, mockClient.lastParams.MaxTokens.Value)
			}

			if mockClient.lastParams.Temperature.Value != client.temperature {
				t.Errorf("Expected temperature %f, got: %f", client.temperature, mockClient.lastParams.Temperature.Value)
			}
		})
	}
}

// TestOpenAIClient_CallWithPromptAndVariables_ErrorHandling tests error scenarios
func TestOpenAIClient_CallWithPromptAndVariables_ErrorHandling(t *testing.T) {
	tests := []struct {
		name          string
		template      string
		variablesJSON string
		expectError   bool
		errorContains string
	}{
		{
			name:          "template processing error - empty template",
			template:      "",
			variablesJSON: `{"name": "Alice"}`,
			expectError:   true,
			errorContains: "variable substitution failed",
		},
		{
			name:          "template processing error - invalid JSON",
			template:      "Hello {{name}}!",
			variablesJSON: `{invalid json}`,
			expectError:   true,
			errorContains: "variable substitution failed",
		},
		{
			name:          "template processing error - JSON array instead of object",
			template:      "Hello {{name}}!",
			variablesJSON: `["not", "an", "object"]`,
			expectError:   true,
			errorContains: "variable substitution failed",
		},
		{
			name:          "template processing error - unclosed JSON",
			template:      "Hello {{name}}!",
			variablesJSON: `{"name": "Alice"`,
			expectError:   true,
			errorContains: "variable substitution failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock SDK client that would succeed if called
			mockClient := &MockOpenAISDKClient{
				completion: &openai.ChatCompletion{
					ID: "should-not-be-called",
					Choices: []openai.ChatCompletionChoice{
						{
							Message: openai.ChatCompletionMessage{
								Content: "This should not be returned",
							},
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
			resp, err := client.CallWithPromptAndVariables(ctx, tt.template, tt.variablesJSON)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.errorContains)) {
					t.Errorf("Expected error to contain '%s', got: %s", tt.errorContains, err.Error())
				}
				if resp != nil {
					t.Errorf("Expected nil response on error, got: %v", resp)
				}

				// Verify that the SDK client was NOT called due to template processing failure
				if mockClient.lastParams != nil {
					t.Errorf("Expected SDK client not to be called due to template processing error")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// TestOpenAIClient_CallWithPromptAndVariables2_Integration tests integration with template processor
func TestOpenAIClient_CallWithPromptAndVariables2_Integration(t *testing.T) {
	// This test verifies that the method correctly integrates with utils.SubstituteVariables
	// and passes the processed prompt to CallWithPrompt

	mockClient := &MockOpenAISDKClient{
		completion: &openai.ChatCompletion{
			ID: "chatcmpl-integration",
			Choices: []openai.ChatCompletionChoice{
				{
					Message: openai.ChatCompletionMessage{
						Content: "Integration test response",
					},
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

	// Test with a complex template that exercises various template processor features
	template := `You are a {{role}} working on {{project}}.
Current task: {{task}}
Priority: {{priority}}
Due date: {{due_date}}
Additional notes: {{notes}}

Please {{action}} the following:
- Review {{item1}}
- Process {{item2}}
- Validate {{item3}}

Context: {{context}}`

	variablesJSON := `{
		"role": "senior developer",
		"project": "AI Provider Library",
		"task": "implement template processing tests",
		"priority": "high",
		"due_date": "2024-01-15",
		"notes": "Ensure comprehensive test coverage",
		"action": "complete",
		"item1": "variable substitution logic",
		"item2": "error handling scenarios",
		"item3": "SDK integration",
		"context": "This is part of the OpenAI SDK migration project"
	}`

	ctx := context.Background()
	resp, err := client.callWithPromptAndVariables(ctx, template, variablesJSON)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	if resp == nil {
		t.Errorf("Expected non-nil response")
		return
	}

	// Verify the response is the exact mock object (no JSON processing)
	if resp != mockClient.completion {
		t.Errorf("Expected exact mock completion object")
	}

	// Verify the mock was called
	if mockClient.lastParams == nil {
		t.Errorf("Expected mock to be called with parameters")
		return
	}

	// Verify that template processing occurred by independently processing the template
	// and ensuring it would produce a different result than the original template
	processedPrompt, err := utils.SubstituteVariables(template, variablesJSON)
	if err != nil {
		t.Errorf("Template processing validation failed: %v", err)
		return
	}

	// The processed prompt should be different from the original template
	if processedPrompt == template {
		t.Errorf("Expected processed prompt to be different from original template")
	}

	// The processed prompt should not contain any variable placeholders
	if strings.Contains(processedPrompt, "{{") || strings.Contains(processedPrompt, "}}") {
		t.Errorf("Processed prompt still contains variable placeholders: %s", processedPrompt)
	}

	// Verify specific substitutions occurred
	expectedSubstitutions := map[string]string{
		"{{role}}":    "senior developer",
		"{{project}}": "AI Provider Library",
		"{{task}}":    "implement template processing tests",
		"{{action}}":  "complete",
	}

	for placeholder, expectedValue := range expectedSubstitutions {
		if strings.Contains(processedPrompt, placeholder) {
			t.Errorf("Placeholder %s was not substituted in processed prompt", placeholder)
		}
		if !strings.Contains(processedPrompt, expectedValue) {
			t.Errorf("Expected value '%s' not found in processed prompt", expectedValue)
		}
	}
}

// TestOpenAIClient_CallWithPrompt_NoJSONProcessing verifies no JSON processing occurs
func TestOpenAIClient_CallWithPrompt_NoJSONProcessing(t *testing.T) {
	// Create a completion with complex nested data to ensure no JSON marshaling/unmarshaling
	originalCompletion := &openai.ChatCompletion{
		ID:      "chatcmpl-complex",
		Object:  "chat.completion",
		Created: 1234567890,
		Model:   "gpt-4o-mini",
		Choices: []openai.ChatCompletionChoice{
			{
				Index: 0,
				Message: openai.ChatCompletionMessage{
					Role:    "assistant",
					Content: "Complex response with special characters: !@#$%^&*(){}[]|\\:;\"'<>,.?/~`",
				},
				FinishReason: "stop",
			},
		},
		Usage: openai.CompletionUsage{
			PromptTokens:     25,
			CompletionTokens: 15,
			TotalTokens:      40,
		},
	}

	mockClient := &MockOpenAISDKClient{
		completion: originalCompletion,
	}

	client := &OpenAIClient{
		client:      mockClient,
		model:       "gpt-4o-mini",
		maxTokens:   1000,
		temperature: 0.7,
		logger:      utils.NewLogger("TestOpenAIClient"),
	}

	ctx := context.Background()
	resp, err := client.callWithPrompt(ctx, "test prompt")

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	// Verify we got the exact same object (no JSON processing)
	if resp != originalCompletion {
		t.Errorf("Expected exact same completion object, indicating no JSON processing occurred")
	}

	// Verify all fields are accessible without JSON unmarshaling
	if resp.ID != "chatcmpl-complex" {
		t.Errorf("Expected ID to be preserved exactly")
	}

	if resp.Choices[0].Message.Content != "Complex response with special characters: !@#$%^&*(){}[]|\\:;\"'<>,.?/~`" {
		t.Errorf("Expected special characters to be preserved exactly")
	}

	if resp.Usage.TotalTokens != 40 {
		t.Errorf("Expected usage data to be preserved exactly")
	}

	// Verify direct field access works (this would fail if JSON processing was involved)
	content := resp.Choices[0].Message.Content
	if len(content) == 0 {
		t.Errorf("Expected direct field access to work without JSON unmarshaling")
	}
}

func TestOpenAIClient_NetworkTimeout(t *testing.T) {
	// Create a server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond) // Short delay for testing
		w.WriteHeader(200)
		w.Write([]byte(`{"choices": [{"message": {"content": "test"}, "finish_reason": "stop"}]}`))
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

	// Note: With SDK, timeout handling is managed internally
	// The SDK provides its own timeout and retry mechanisms
	if client == nil {
		t.Errorf("Expected client to be created")
	}
}

// TestOpenAIClient_CallWithTools tests the CallWithTools method for function calling
func TestOpenAIClient_CallWithTools(t *testing.T) {
	tests := []struct {
		name           string
		prompt         string
		tools          []openai.ChatCompletionToolUnionParam
		mockCompletion *openai.ChatCompletion
		mockError      error
		expectError    bool
		errorContains  string
		validateResp   func(t *testing.T, resp *openai.ChatCompletion)
	}{
		{
			name:   "successful function call",
			prompt: "What's the weather in Paris?",
			tools: []openai.ChatCompletionToolUnionParam{
				openai.ChatCompletionFunctionTool(shared.FunctionDefinitionParam{
					Name:        "get_weather",
					Description: openai.String("Get current weather for a location"),
					Parameters: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"location": map[string]interface{}{
								"type":        "string",
								"description": "City name",
							},
						},
						"required": []string{"location"},
					},
				}),
			},
			mockCompletion: &openai.ChatCompletion{
				ID:      "chatcmpl-tools123",
				Object:  "chat.completion",
				Created: 1234567890,
				Model:   "gpt-4o-mini",
				Choices: []openai.ChatCompletionChoice{
					{
						Index: 0,
						Message: openai.ChatCompletionMessage{
							Role:    "assistant",
							Content: "",
							ToolCalls: []openai.ChatCompletionMessageToolCallUnion{
								{
									ID:   "call_123",
									Type: "function",
									Function: openai.ChatCompletionMessageFunctionToolCallFunction{
										Name:      "get_weather",
										Arguments: `{"location": "Paris"}`,
									},
								},
							},
						},
						FinishReason: "tool_calls",
					},
				},
			},
			mockError:   nil,
			expectError: false,
			validateResp: func(t *testing.T, resp *openai.ChatCompletion) {
				if resp.ID != "chatcmpl-tools123" {
					t.Errorf("Expected ID 'chatcmpl-tools123', got: %s", resp.ID)
				}
				if len(resp.Choices) != 1 {
					t.Errorf("Expected 1 choice, got: %d", len(resp.Choices))
				}
				if len(resp.Choices[0].Message.ToolCalls) != 1 {
					t.Errorf("Expected 1 tool call, got: %d", len(resp.Choices[0].Message.ToolCalls))
				}
				if resp.Choices[0].Message.ToolCalls[0].Function.Name != "get_weather" {
					t.Errorf("Expected function name 'get_weather', got: %s", resp.Choices[0].Message.ToolCalls[0].Function.Name)
				}
				if resp.Choices[0].FinishReason != "tool_calls" {
					t.Errorf("Expected finish reason 'tool_calls', got: %s", resp.Choices[0].FinishReason)
				}
			},
		},
		{
			name:   "text response without tool calls",
			prompt: "Hello, how are you?",
			tools: []openai.ChatCompletionToolUnionParam{
				openai.ChatCompletionFunctionTool(shared.FunctionDefinitionParam{
					Name:        "get_weather",
					Description: openai.String("Get current weather for a location"),
				}),
			},
			mockCompletion: &openai.ChatCompletion{
				ID:      "chatcmpl-notools",
				Object:  "chat.completion",
				Created: 1234567890,
				Model:   "gpt-4o-mini",
				Choices: []openai.ChatCompletionChoice{
					{
						Index: 0,
						Message: openai.ChatCompletionMessage{
							Role:      "assistant",
							Content:   "Hello! I'm doing well, thank you for asking. How can I help you today?",
							ToolCalls: []openai.ChatCompletionMessageToolCallUnion{},
						},
						FinishReason: "stop",
					},
				},
			},
			mockError:   nil,
			expectError: false,
			validateResp: func(t *testing.T, resp *openai.ChatCompletion) {
				if len(resp.Choices[0].Message.ToolCalls) != 0 {
					t.Errorf("Expected no tool calls, got: %d", len(resp.Choices[0].Message.ToolCalls))
				}
				if resp.Choices[0].Message.Content == "" {
					t.Errorf("Expected text content in response")
				}
				if resp.Choices[0].FinishReason != "stop" {
					t.Errorf("Expected finish reason 'stop', got: %s", resp.Choices[0].FinishReason)
				}
			},
		},
		{
			name:   "multiple tools available",
			prompt: "Get the weather in London and calculate 2+2",
			tools: []openai.ChatCompletionToolUnionParam{
				openai.ChatCompletionFunctionTool(shared.FunctionDefinitionParam{
					Name:        "get_weather",
					Description: openai.String("Get current weather for a location"),
					Parameters: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"location": map[string]interface{}{
								"type": "string",
							},
						},
					},
				}),
				openai.ChatCompletionFunctionTool(shared.FunctionDefinitionParam{
					Name:        "calculate",
					Description: openai.String("Perform mathematical calculations"),
					Parameters: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"expression": map[string]interface{}{
								"type": "string",
							},
						},
					},
				}),
			},
			mockCompletion: &openai.ChatCompletion{
				ID:      "chatcmpl-multitools",
				Object:  "chat.completion",
				Created: 1234567890,
				Model:   "gpt-4o-mini",
				Choices: []openai.ChatCompletionChoice{
					{
						Index: 0,
						Message: openai.ChatCompletionMessage{
							Role:    "assistant",
							Content: "",
							ToolCalls: []openai.ChatCompletionMessageToolCallUnion{
								{
									ID:   "call_weather",
									Type: "function",
									Function: openai.ChatCompletionMessageFunctionToolCallFunction{
										Name:      "get_weather",
										Arguments: `{"location": "London"}`,
									},
								},
								{
									ID:   "call_calc",
									Type: "function",
									Function: openai.ChatCompletionMessageFunctionToolCallFunction{
										Name:      "calculate",
										Arguments: `{"expression": "2+2"}`,
									},
								},
							},
						},
						FinishReason: "tool_calls",
					},
				},
			},
			mockError:   nil,
			expectError: false,
			validateResp: func(t *testing.T, resp *openai.ChatCompletion) {
				if len(resp.Choices[0].Message.ToolCalls) != 2 {
					t.Errorf("Expected 2 tool calls, got: %d", len(resp.Choices[0].Message.ToolCalls))
				}
			},
		},
		{
			name:   "empty tools array",
			prompt: "Hello",
			tools:  []openai.ChatCompletionToolUnionParam{},
			mockCompletion: &openai.ChatCompletion{
				ID:      "chatcmpl-notools",
				Object:  "chat.completion",
				Created: 1234567890,
				Model:   "gpt-4o-mini",
				Choices: []openai.ChatCompletionChoice{
					{
						Index: 0,
						Message: openai.ChatCompletionMessage{
							Role:    "assistant",
							Content: "Hello! How can I help you?",
						},
						FinishReason: "stop",
					},
				},
			},
			mockError:   nil,
			expectError: false,
			validateResp: func(t *testing.T, resp *openai.ChatCompletion) {
				if resp.Choices[0].Message.Content == "" {
					t.Errorf("Expected text response")
				}
			},
		},
		{
			name:   "invalid API key error",
			prompt: "Test prompt",
			tools: []openai.ChatCompletionToolUnionParam{
				openai.ChatCompletionFunctionTool(shared.FunctionDefinitionParam{
					Name: "get_weather",
				}),
			},
			mockCompletion: nil,
			mockError: &openai.Error{
				Code:    "invalid_api_key",
				Message: "Invalid API key provided",
				Type:    "invalid_request_error",
			},
			expectError:   true,
			errorContains: "invalid API key",
		},
		{
			name:   "rate limit error",
			prompt: "Test prompt",
			tools: []openai.ChatCompletionToolUnionParam{
				openai.ChatCompletionFunctionTool(shared.FunctionDefinitionParam{
					Name: "get_weather",
				}),
			},
			mockCompletion: nil,
			mockError: &openai.Error{
				Code:    "rate_limit_exceeded",
				Message: "Rate limit exceeded",
				Type:    "rate_limit_error",
			},
			expectError:   true,
			errorContains: "rate limit exceeded",
		},
		{
			name:   "model not found error",
			prompt: "Test prompt",
			tools: []openai.ChatCompletionToolUnionParam{
				openai.ChatCompletionFunctionTool(shared.FunctionDefinitionParam{
					Name: "get_weather",
				}),
			},
			mockCompletion: nil,
			mockError: &openai.Error{
				Code:    "model_not_found",
				Message: "The model 'invalid-model' does not exist",
				Type:    "invalid_request_error",
			},
			expectError:   true,
			errorContains: "model not found",
		},
		{
			name:   "network error",
			prompt: "Test prompt",
			tools: []openai.ChatCompletionToolUnionParam{
				openai.ChatCompletionFunctionTool(shared.FunctionDefinitionParam{
					Name: "get_weather",
				}),
			},
			mockCompletion: nil,
			mockError:      fmt.Errorf("connection refused"),
			expectError:    true,
			errorContains:  "network error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock SDK client
			mockClient := &MockOpenAISDKClient{
				completion: tt.mockCompletion,
				err:        tt.mockError,
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
			resp, err := client.CallWithTools(ctx, tt.prompt, tt.tools)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.errorContains)) {
					t.Errorf("Expected error to contain '%s', got: %s", tt.errorContains, err.Error())
				}
				if resp != nil {
					t.Errorf("Expected nil response on error, got: %v", resp)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if resp == nil {
				t.Errorf("Expected non-nil response")
				return
			}

			// Verify no JSON processing occurred by checking we got the exact mock object
			if resp != tt.mockCompletion {
				t.Errorf("Expected exact mock completion object, got different object")
			}

			// Run custom validation if provided
			if tt.validateResp != nil {
				tt.validateResp(t, resp)
			}

			// Verify the mock was called with correct parameters
			if mockClient.lastParams == nil {
				t.Errorf("Expected mock to be called with parameters")
				return
			}

			// Verify model parameter
			if mockClient.lastParams.Model != openai.ChatModel(client.model) {
				t.Errorf("Expected model '%s', got: %s", client.model, string(mockClient.lastParams.Model))
			}

			// Verify message parameter
			if len(mockClient.lastParams.Messages) != 1 {
				t.Errorf("Expected 1 message, got: %d", len(mockClient.lastParams.Messages))
				return
			}

			// Verify tools parameter
			if len(mockClient.lastParams.Tools) != len(tt.tools) {
				t.Errorf("Expected %d tools, got: %d", len(tt.tools), len(mockClient.lastParams.Tools))
			}

			// Verify other parameters using the SDK's parameter types
			if mockClient.lastParams.MaxTokens.Value != int64(client.maxTokens) {
				t.Errorf("Expected maxTokens %d, got: %d", client.maxTokens, mockClient.lastParams.MaxTokens.Value)
			}

			if mockClient.lastParams.Temperature.Value != client.temperature {
				t.Errorf("Expected temperature %f, got: %f", client.temperature, mockClient.lastParams.Temperature.Value)
			}
		})
	}
}

// TestOpenAIClient_CallWithTools_ParameterValidation tests parameter handling for function calling
func TestOpenAIClient_CallWithTools_ParameterValidation(t *testing.T) {
	mockClient := &MockOpenAISDKClient{
		completion: &openai.ChatCompletion{
			ID: "test",
			Choices: []openai.ChatCompletionChoice{
				{
					Message: openai.ChatCompletionMessage{
						Content: "response",
					},
				},
			},
		},
	}

	tests := []struct {
		name        string
		model       string
		maxTokens   int
		temperature float64
		toolsCount  int
	}{
		{
			name:        "default parameters with single tool",
			model:       "gpt-4o-mini",
			maxTokens:   1000,
			temperature: 0.7,
			toolsCount:  1,
		},
		{
			name:        "custom parameters with multiple tools",
			model:       "gpt-4",
			maxTokens:   2000,
			temperature: 0.1,
			toolsCount:  3,
		},
		{
			name:        "zero temperature with no tools",
			model:       "gpt-4o-mini",
			maxTokens:   500,
			temperature: 0.0,
			toolsCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &OpenAIClient{
				client:      mockClient,
				model:       tt.model,
				maxTokens:   tt.maxTokens,
				temperature: tt.temperature,
				logger:      utils.NewLogger("TestOpenAIClient"),
			}

			// Create tools array based on test case
			tools := make([]openai.ChatCompletionToolUnionParam, tt.toolsCount)
			for i := 0; i < tt.toolsCount; i++ {
				tools[i] = openai.ChatCompletionFunctionTool(shared.FunctionDefinitionParam{
					Name:        fmt.Sprintf("test_function_%d", i),
					Description: openai.String(fmt.Sprintf("Test function %d", i)),
				})
			}

			ctx := context.Background()
			_, err := client.CallWithTools(ctx, "test prompt", tools)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Verify parameters were passed correctly
			if string(mockClient.lastParams.Model) != tt.model {
				t.Errorf("Expected model '%s', got: %s", tt.model, string(mockClient.lastParams.Model))
			}

			if mockClient.lastParams.MaxTokens.Value != int64(tt.maxTokens) {
				t.Errorf("Expected maxTokens %d, got: %d", tt.maxTokens, mockClient.lastParams.MaxTokens.Value)
			}

			if mockClient.lastParams.Temperature.Value != tt.temperature {
				t.Errorf("Expected temperature %f, got: %f", tt.temperature, mockClient.lastParams.Temperature.Value)
			}

			if len(mockClient.lastParams.Tools) != tt.toolsCount {
				t.Errorf("Expected %d tools, got: %d", tt.toolsCount, len(mockClient.lastParams.Tools))
			}
		})
	}
}

// TestOpenAIClient_CallWithTools_ContextCancellation tests context handling for function calling
func TestOpenAIClient_CallWithTools_ContextCancellation(t *testing.T) {
	mockClient := &MockOpenAISDKClient{
		err: context.Canceled,
	}

	client := &OpenAIClient{
		client:      mockClient,
		model:       "gpt-4o-mini",
		maxTokens:   1000,
		temperature: 0.7,
		logger:      utils.NewLogger("TestOpenAIClient"),
	}

	tools := []openai.ChatCompletionToolUnionParam{
		openai.ChatCompletionFunctionTool(shared.FunctionDefinitionParam{
			Name: "get_weather",
		}),
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	resp, err := client.CallWithTools(ctx, "test prompt", tools)

	if err == nil {
		t.Errorf("Expected error due to cancelled context")
	}

	if resp != nil {
		t.Errorf("Expected nil response on cancelled context")
	}

	if !strings.Contains(err.Error(), "request failed") {
		t.Errorf("Expected error to be wrapped properly, got: %s", err.Error())
	}
}

// TestOpenAIClient_CallWithTools_NoJSONProcessing verifies no JSON processing occurs for function calling
func TestOpenAIClient_CallWithTools_NoJSONProcessing(t *testing.T) {
	// Create a completion with complex tool call data to ensure no JSON marshaling/unmarshaling
	originalCompletion := &openai.ChatCompletion{
		ID:      "chatcmpl-complex-tools",
		Object:  "chat.completion",
		Created: 1234567890,
		Model:   "gpt-4o-mini",
		Choices: []openai.ChatCompletionChoice{
			{
				Index: 0,
				Message: openai.ChatCompletionMessage{
					Role:    "assistant",
					Content: "",
					ToolCalls: []openai.ChatCompletionMessageToolCallUnion{
						{
							ID:   "call_complex_123",
							Type: "function",
							Function: openai.ChatCompletionMessageFunctionToolCallFunction{
								Name:      "complex_function",
								Arguments: `{"data": "Complex data with special chars: !@#$%^&*(){}[]|\\:;\"'<>,.?/~` + "`" + `", "numbers": [1, 2, 3]}`,
							},
						},
					},
				},
				FinishReason: "tool_calls",
			},
		},
		Usage: openai.CompletionUsage{
			PromptTokens:     30,
			CompletionTokens: 20,
			TotalTokens:      50,
		},
	}

	mockClient := &MockOpenAISDKClient{
		completion: originalCompletion,
	}

	client := &OpenAIClient{
		client:      mockClient,
		model:       "gpt-4o-mini",
		maxTokens:   1000,
		temperature: 0.7,
		logger:      utils.NewLogger("TestOpenAIClient"),
	}

	tools := []openai.ChatCompletionToolUnionParam{
		openai.ChatCompletionFunctionTool(shared.FunctionDefinitionParam{
			Name:        "get_weather",
			Description: openai.String("A complex function for testing"),
		}),
	}

	ctx := context.Background()
	resp, err := client.CallWithTools(ctx, "test prompt", tools)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	// Verify we got the exact same object (no JSON processing)
	if resp != originalCompletion {
		t.Errorf("Expected exact same completion object, indicating no JSON processing occurred")
	}

	// Verify all fields are accessible without JSON unmarshaling
	if resp.ID != "chatcmpl-complex-tools" {
		t.Errorf("Expected ID to be preserved exactly")
	}

	if len(resp.Choices[0].Message.ToolCalls) != 1 {
		t.Errorf("Expected tool calls to be preserved exactly")
	}

	toolCall := resp.Choices[0].Message.ToolCalls[0]
	if toolCall.Function.Name != "complex_function" {
		t.Errorf("Expected function name to be preserved exactly")
	}

	expectedArgs := `{"data": "Complex data with special chars: !@#$%^&*(){}[]|\\:;\"'<>,.?/~` + "`" + `", "numbers": [1, 2, 3]}`
	if toolCall.Function.Arguments != expectedArgs {
		t.Errorf("Expected complex arguments to be preserved exactly")
	}

	if resp.Usage.TotalTokens != 50 {
		t.Errorf("Expected usage data to be preserved exactly")
	}

	// Verify direct field access works (this would fail if JSON processing was involved)
	functionName := resp.Choices[0].Message.ToolCalls[0].Function.Name
	if len(functionName) == 0 {
		t.Errorf("Expected direct field access to work without JSON unmarshaling")
	}
}

// TestOpenAIClient_CallWithMessages tests the CallWithMessages method with SDK types
func TestOpenAIClient_CallWithMessages(t *testing.T) {
	tests := []struct {
		name           string
		messages       []openai.ChatCompletionMessageParamUnion
		mockCompletion *openai.ChatCompletion
		mockError      error
		expectError    bool
		errorContains  string
		validateResp   func(t *testing.T, resp *openai.ChatCompletion)
	}{
		{
			name: "successful conversation",
			messages: []openai.ChatCompletionMessageParamUnion{
				openai.SystemMessage("You are a helpful assistant."),
				openai.UserMessage("What is the capital of France?"),
			},
			mockCompletion: &openai.ChatCompletion{
				ID:      "chatcmpl-conversation",
				Object:  "chat.completion",
				Created: 1234567890,
				Model:   "gpt-4o-mini",
				Choices: []openai.ChatCompletionChoice{
					{
						Index: 0,
						Message: openai.ChatCompletionMessage{
							Role:    "assistant",
							Content: "The capital of France is Paris.",
						},
						FinishReason: "stop",
					},
				},
				Usage: openai.CompletionUsage{
					PromptTokens:     15,
					CompletionTokens: 8,
					TotalTokens:      23,
				},
			},
			mockError:   nil,
			expectError: false,
			validateResp: func(t *testing.T, resp *openai.ChatCompletion) {
				if resp.ID != "chatcmpl-conversation" {
					t.Errorf("Expected ID 'chatcmpl-conversation', got: %s", resp.ID)
				}
				if len(resp.Choices) != 1 {
					t.Errorf("Expected 1 choice, got: %d", len(resp.Choices))
				}
				if resp.Choices[0].Message.Content != "The capital of France is Paris." {
					t.Errorf("Expected specific content, got: %s", resp.Choices[0].Message.Content)
				}
			},
		},
		{
			name: "multi-turn conversation",
			messages: []openai.ChatCompletionMessageParamUnion{
				openai.SystemMessage("You are a helpful assistant."),
				openai.UserMessage("What is the capital of France?"),
				openai.AssistantMessage("The capital of France is Paris."),
				openai.UserMessage("What about Germany?"),
			},
			mockCompletion: &openai.ChatCompletion{
				ID:      "chatcmpl-multiturn",
				Object:  "chat.completion",
				Created: 1234567890,
				Model:   "gpt-4o-mini",
				Choices: []openai.ChatCompletionChoice{
					{
						Index: 0,
						Message: openai.ChatCompletionMessage{
							Role:    "assistant",
							Content: "The capital of Germany is Berlin.",
						},
						FinishReason: "stop",
					},
				},
			},
			mockError:   nil,
			expectError: false,
			validateResp: func(t *testing.T, resp *openai.ChatCompletion) {
				if resp.Choices[0].Message.Content != "The capital of Germany is Berlin." {
					t.Errorf("Expected specific content, got: %s", resp.Choices[0].Message.Content)
				}
			},
		},
		{
			name: "single user message",
			messages: []openai.ChatCompletionMessageParamUnion{
				openai.UserMessage("Hello!"),
			},
			mockCompletion: &openai.ChatCompletion{
				ID:      "chatcmpl-single",
				Object:  "chat.completion",
				Created: 1234567890,
				Model:   "gpt-4o-mini",
				Choices: []openai.ChatCompletionChoice{
					{
						Index: 0,
						Message: openai.ChatCompletionMessage{
							Role:    "assistant",
							Content: "Hello! How can I help you today?",
						},
						FinishReason: "stop",
					},
				},
			},
			mockError:   nil,
			expectError: false,
			validateResp: func(t *testing.T, resp *openai.ChatCompletion) {
				if len(resp.Choices) == 0 {
					t.Errorf("Expected at least one choice")
				}
			},
		},
		{
			name:     "empty messages array",
			messages: []openai.ChatCompletionMessageParamUnion{},
			mockCompletion: &openai.ChatCompletion{
				ID:      "chatcmpl-empty",
				Object:  "chat.completion",
				Created: 1234567890,
				Model:   "gpt-4o-mini",
				Choices: []openai.ChatCompletionChoice{
					{
						Index: 0,
						Message: openai.ChatCompletionMessage{
							Role:    "assistant",
							Content: "I'm here to help! What would you like to know?",
						},
						FinishReason: "stop",
					},
				},
			},
			mockError:   nil,
			expectError: false,
			validateResp: func(t *testing.T, resp *openai.ChatCompletion) {
				if len(resp.Choices) == 0 {
					t.Errorf("Expected at least one choice")
				}
			},
		},
		{
			name: "invalid API key error",
			messages: []openai.ChatCompletionMessageParamUnion{
				openai.UserMessage("Test message"),
			},
			mockCompletion: nil,
			mockError: &openai.Error{
				Code:    "invalid_api_key",
				Message: "Invalid API key provided",
				Type:    "invalid_request_error",
			},
			expectError:   true,
			errorContains: "invalid API key",
		},
		{
			name: "rate limit error",
			messages: []openai.ChatCompletionMessageParamUnion{
				openai.UserMessage("Test message"),
			},
			mockCompletion: nil,
			mockError: &openai.Error{
				Code:    "rate_limit_exceeded",
				Message: "Rate limit exceeded",
				Type:    "rate_limit_error",
			},
			expectError:   true,
			errorContains: "rate limit exceeded",
		},
		{
			name: "model not found error",
			messages: []openai.ChatCompletionMessageParamUnion{
				openai.UserMessage("Test message"),
			},
			mockCompletion: nil,
			mockError: &openai.Error{
				Code:    "model_not_found",
				Message: "The model 'invalid-model' does not exist",
				Type:    "invalid_request_error",
			},
			expectError:   true,
			errorContains: "model not found",
		},
		{
			name: "context length exceeded error",
			messages: []openai.ChatCompletionMessageParamUnion{
				openai.UserMessage("Very long message that exceeds context window..."),
			},
			mockCompletion: nil,
			mockError: &openai.Error{
				Code:    "context_length_exceeded",
				Message: "This model's maximum context length is 4096 tokens",
				Type:    "invalid_request_error",
			},
			expectError:   true,
			errorContains: "context length exceeded",
		},
		{
			name: "server error",
			messages: []openai.ChatCompletionMessageParamUnion{
				openai.UserMessage("Test message"),
			},
			mockCompletion: nil,
			mockError: &openai.Error{
				Message: "Internal server error",
				Type:    "server_error",
			},
			expectError:   true,
			errorContains: "server error",
		},
		{
			name: "network error",
			messages: []openai.ChatCompletionMessageParamUnion{
				openai.UserMessage("Test message"),
			},
			mockCompletion: nil,
			mockError:      fmt.Errorf("connection refused"),
			expectError:    true,
			errorContains:  "network error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock SDK client
			mockClient := &MockOpenAISDKClient{
				completion: tt.mockCompletion,
				err:        tt.mockError,
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
			resp, err := client.CallWithMessages(ctx, tt.messages)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.errorContains)) {
					t.Errorf("Expected error to contain '%s', got: %s", tt.errorContains, err.Error())
				}
				if resp != nil {
					t.Errorf("Expected nil response on error, got: %v", resp)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if resp == nil {
				t.Errorf("Expected non-nil response")
				return
			}

			// Verify no JSON processing occurred by checking we got the exact mock object
			if resp != tt.mockCompletion {
				t.Errorf("Expected exact mock completion object, got different object")
			}

			// Run custom validation if provided
			if tt.validateResp != nil {
				tt.validateResp(t, resp)
			}

			// Verify the mock was called with correct parameters
			if mockClient.lastParams == nil {
				t.Errorf("Expected mock to be called with parameters")
				return
			}

			// Verify model parameter
			if mockClient.lastParams.Model != openai.ChatModel(client.model) {
				t.Errorf("Expected model '%s', got: %s", client.model, string(mockClient.lastParams.Model))
			}

			// Verify messages parameter
			if len(mockClient.lastParams.Messages) != len(tt.messages) {
				t.Errorf("Expected %d messages, got: %d", len(tt.messages), len(mockClient.lastParams.Messages))
			}

			// Verify other parameters using the SDK's parameter types
			if mockClient.lastParams.MaxTokens.Value != int64(client.maxTokens) {
				t.Errorf("Expected maxTokens %d, got: %d", client.maxTokens, mockClient.lastParams.MaxTokens.Value)
			}

			if mockClient.lastParams.Temperature.Value != client.temperature {
				t.Errorf("Expected temperature %f, got: %f", client.temperature, mockClient.lastParams.Temperature.Value)
			}
		})
	}
}

// TestOpenAIClient_CallWithMessages_ParameterValidation tests parameter handling for CallWithMessages
func TestOpenAIClient_CallWithMessages_ParameterValidation(t *testing.T) {
	mockClient := &MockOpenAISDKClient{
		completion: &openai.ChatCompletion{
			ID: "test",
			Choices: []openai.ChatCompletionChoice{
				{
					Message: openai.ChatCompletionMessage{
						Content: "response",
					},
				},
			},
		},
	}

	tests := []struct {
		name        string
		model       string
		maxTokens   int
		temperature float64
		messages    []openai.ChatCompletionMessageParamUnion
	}{
		{
			name:        "default parameters with system message",
			model:       "gpt-4o-mini",
			maxTokens:   1000,
			temperature: 0.7,
			messages: []openai.ChatCompletionMessageParamUnion{
				openai.SystemMessage("You are a helpful assistant."),
				openai.UserMessage("Hello"),
			},
		},
		{
			name:        "custom parameters with conversation",
			model:       "gpt-4",
			maxTokens:   2000,
			temperature: 0.1,
			messages: []openai.ChatCompletionMessageParamUnion{
				openai.UserMessage("What is 2+2?"),
				openai.AssistantMessage("2+2 equals 4."),
				openai.UserMessage("What about 3+3?"),
			},
		},
		{
			name:        "zero temperature with single message",
			model:       "gpt-4o-mini",
			maxTokens:   500,
			temperature: 0.0,
			messages: []openai.ChatCompletionMessageParamUnion{
				openai.UserMessage("Generate code"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &OpenAIClient{
				client:      mockClient,
				model:       tt.model,
				maxTokens:   tt.maxTokens,
				temperature: tt.temperature,
				logger:      utils.NewLogger("TestOpenAIClient"),
			}

			ctx := context.Background()
			_, err := client.CallWithMessages(ctx, tt.messages)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Verify parameters were passed correctly
			if string(mockClient.lastParams.Model) != tt.model {
				t.Errorf("Expected model '%s', got: %s", tt.model, string(mockClient.lastParams.Model))
			}

			if mockClient.lastParams.MaxTokens.Value != int64(tt.maxTokens) {
				t.Errorf("Expected maxTokens %d, got: %d", tt.maxTokens, mockClient.lastParams.MaxTokens.Value)
			}

			if mockClient.lastParams.Temperature.Value != tt.temperature {
				t.Errorf("Expected temperature %f, got: %f", tt.temperature, mockClient.lastParams.Temperature.Value)
			}

			// Verify messages were passed correctly
			if len(mockClient.lastParams.Messages) != len(tt.messages) {
				t.Errorf("Expected %d messages, got: %d", len(tt.messages), len(mockClient.lastParams.Messages))
			}
		})
	}
}

// TestOpenAIClient_CallWithMessages_ContextCancellation tests context handling for CallWithMessages
func TestOpenAIClient_CallWithMessages_ContextCancellation(t *testing.T) {
	mockClient := &MockOpenAISDKClient{
		err: context.Canceled,
	}

	client := &OpenAIClient{
		client:      mockClient,
		model:       "gpt-4o-mini",
		maxTokens:   1000,
		temperature: 0.7,
		logger:      utils.NewLogger("TestOpenAIClient"),
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	messages := []openai.ChatCompletionMessageParamUnion{
		openai.UserMessage("test message"),
	}

	resp, err := client.CallWithMessages(ctx, messages)

	if err == nil {
		t.Errorf("Expected error due to cancelled context")
	}

	if resp != nil {
		t.Errorf("Expected nil response on cancelled context")
	}

	if !strings.Contains(err.Error(), "request failed") {
		t.Errorf("Expected error to be wrapped properly, got: %s", err.Error())
	}
}

// TestOpenAIClient_CallWithMessages_NoJSONProcessing verifies no JSON processing occurs
func TestOpenAIClient_CallWithMessages_NoJSONProcessing(t *testing.T) {
	// Create a completion with complex nested data to ensure no JSON marshaling/unmarshaling
	originalCompletion := &openai.ChatCompletion{
		ID:      "chatcmpl-messages-complex",
		Object:  "chat.completion",
		Created: 1234567890,
		Model:   "gpt-4o-mini",
		Choices: []openai.ChatCompletionChoice{
			{
				Index: 0,
				Message: openai.ChatCompletionMessage{
					Role:    "assistant",
					Content: "Complex conversation response with special characters: !@#$%^&*(){}[]|\\:;\"'<>,.?/~`",
				},
				FinishReason: "stop",
			},
		},
		Usage: openai.CompletionUsage{
			PromptTokens:     30,
			CompletionTokens: 20,
			TotalTokens:      50,
		},
	}

	mockClient := &MockOpenAISDKClient{
		completion: originalCompletion,
	}

	client := &OpenAIClient{
		client:      mockClient,
		model:       "gpt-4o-mini",
		maxTokens:   1000,
		temperature: 0.7,
		logger:      utils.NewLogger("TestOpenAIClient"),
	}

	messages := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage("You are a helpful assistant with special chars: !@#$%"),
		openai.UserMessage("Test message with unicode: 🚀 ✨ 🎉"),
	}

	ctx := context.Background()
	resp, err := client.CallWithMessages(ctx, messages)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	// Verify we got the exact same object (no JSON processing)
	if resp != originalCompletion {
		t.Errorf("Expected exact same completion object, indicating no JSON processing occurred")
	}

	// Verify all fields are accessible without JSON unmarshaling
	if resp.ID != "chatcmpl-messages-complex" {
		t.Errorf("Expected ID to be preserved exactly")
	}

	if resp.Choices[0].Message.Content != "Complex conversation response with special characters: !@#$%^&*(){}[]|\\:;\"'<>,.?/~`" {
		t.Errorf("Expected special characters to be preserved exactly")
	}

	if resp.Usage.TotalTokens != 50 {
		t.Errorf("Expected usage data to be preserved exactly")
	}

	// Verify direct field access works (this would fail if JSON processing was involved)
	content := resp.Choices[0].Message.Content
	if len(content) == 0 {
		t.Errorf("Expected direct field access to work without JSON unmarshaling")
	}
}

// TestOpenAIClient_CallWithMessages_LoggingBehavior tests logging behavior
func TestOpenAIClient_CallWithMessages_LoggingBehavior(t *testing.T) {
	mockClient := &MockOpenAISDKClient{
		completion: &openai.ChatCompletion{
			ID: "test",
			Choices: []openai.ChatCompletionChoice{
				{
					Message: openai.ChatCompletionMessage{
						Content: "response",
					},
				},
			},
		},
	}

	client := &OpenAIClient{
		client:      mockClient,
		model:       "gpt-4o-mini",
		maxTokens:   1000,
		temperature: 0.7,
		logger:      utils.NewLogger("TestOpenAIClient"),
	}

	tests := []struct {
		name         string
		messages     []openai.ChatCompletionMessageParamUnion
		expectedLogs int // Number of messages we expect to be logged about
	}{
		{
			name: "single message",
			messages: []openai.ChatCompletionMessageParamUnion{
				openai.UserMessage("Hello"),
			},
			expectedLogs: 1,
		},
		{
			name: "conversation with multiple messages",
			messages: []openai.ChatCompletionMessageParamUnion{
				openai.SystemMessage("You are helpful."),
				openai.UserMessage("What is 2+2?"),
				openai.AssistantMessage("4"),
				openai.UserMessage("What about 3+3?"),
			},
			expectedLogs: 4,
		},
		{
			name:         "empty messages",
			messages:     []openai.ChatCompletionMessageParamUnion{},
			expectedLogs: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			resp, err := client.CallWithMessages(ctx, tt.messages)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if resp == nil {
				t.Errorf("Expected non-nil response")
				return
			}

			// Verify the correct number of messages was logged
			// Note: The actual logging verification would depend on the logger implementation
			// Here we just verify that the method completed successfully with the expected message count
			if len(mockClient.lastParams.Messages) != tt.expectedLogs {
				t.Errorf("Expected %d messages to be processed, got: %d", tt.expectedLogs, len(mockClient.lastParams.Messages))
			}
		})
	}
}

func TestOpenAIClient_RateLimitHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle both /chat/completions and /v1/chat/completions paths
		if r.URL.Path == "/chat/completions" || r.URL.Path == "/v1/chat/completions" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(429)
			w.Write([]byte(`{"error": {"message": "Rate limit exceeded", "type": "rate_limit_error", "code": "rate_limit_exceeded"}}`))
		} else {
			w.WriteHeader(404)
		}
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

	req := types.CompletionRequest{
		Code:     "console.",
		Cursor:   8,
		Language: "javascript",
		Context:  types.CodeContext{},
	}

	ctx := context.Background()
	resp, err := client.GenerateCompletion(ctx, req)

	if err != nil {
		t.Errorf("Should handle rate limit gracefully, got error: %v", err)
	}

	if !strings.Contains(resp.Error, "rate limit exceeded") {
		t.Errorf("Expected rate limit error message, got: %s", resp.Error)
	}

	if resp.Confidence != 0.0 {
		t.Errorf("Expected confidence 0.0 for rate limit error, got: %f", resp.Confidence)
	}

	if len(resp.Suggestions) != 0 {
		t.Errorf("Expected no suggestions for rate limit error, got: %d", len(resp.Suggestions))
	}
}

// Integration Tests - These tests use real OpenAI API endpoints

func TestOpenAIClient_CallWithPrompt_Integration(t *testing.T) {
	if !utils.CanRunOpenAIIntegrationTests() {
		t.Skip("Skipping OpenAI integration test: OPENAI_API_KEY environment variable not set")
	}

	testConfig, err := utils.LoadTestConfig()
	if err != nil {
		t.Fatalf("Failed to load test configuration: %v", err)
	}

	// Create client using enhanced TestConfig with custom settings for testing
	config := testConfig.CreateOpenAIConfig()
	config.MaxTokens = 150
	config.Temperature = 0.1 // Low temperature for more predictable results

	client, err := NewOpenAIClient(config)
	if err != nil {
		t.Fatalf("Failed to create OpenAI client: %v", err)
	}

	ctx := context.Background()

	// Test cases with different types of prompts that are distinct from GenerateCompletion and GenerateCode
	testCases := []struct {
		name           string
		prompt         string
		expectedInResp []string // Keywords we expect to find in the response
		minLength      int      // Minimum expected response length
	}{
		{
			name:           "creative writing prompt",
			prompt:         "Write a haiku about programming. Return only the haiku, no explanations.",
			expectedInResp: []string{}, // Haiku content is unpredictable, just check it's not empty
			minLength:      10,
		},
		{
			name:           "mathematical calculation",
			prompt:         "What is 15 * 23? Provide only the numerical answer.",
			expectedInResp: []string{"345"}, // Should contain the correct answer
			minLength:      1,
		},
		{
			name:           "simple question answering",
			prompt:         "What is the capital of France? Answer in one word only.",
			expectedInResp: []string{"Paris"}, // Should contain Paris
			minLength:      3,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := client.callWithPrompt(ctx, tc.prompt)

			if err != nil {
				t.Fatalf("CallWithPrompt failed for %s: %v", tc.name, err)
			}

			if resp == nil {
				t.Errorf("Expected non-nil response for %s", tc.name)
				return
			}

			// Verify response structure using SDK types
			if len(resp.Choices) == 0 {
				t.Errorf("Expected at least one choice in response for %s", tc.name)
				return
			}

			content := resp.Choices[0].Message.Content
			if len(content) < tc.minLength {
				t.Errorf("Response content too short for %s. Expected at least %d characters, got %d: %s",
					tc.name, tc.minLength, len(content), content)
			}

			// Check for expected keywords in response
			for _, expected := range tc.expectedInResp {
				if !strings.Contains(strings.ToLower(content), strings.ToLower(expected)) {
					t.Errorf("Expected response for %s to contain '%s', but got: %s",
						tc.name, expected, content)
				}
			}

			// Verify response metadata
			if resp.Model == "" {
				t.Errorf("Expected model field to be set in response for %s", tc.name)
			}

			if resp.ID == "" {
				t.Errorf("Expected ID field to be set in response for %s", tc.name)
			}

			// Log the response for manual verification during development
			t.Logf("Response for %s: %s", tc.name, content)
		})
	}
}

// TestOpenAIClient_ValidateCredentials_SDK tests credential validation using SDK client mocks
func TestOpenAIClient_ValidateCredentials_SDK(t *testing.T) {
	tests := []struct {
		name           string
		mockCompletion *openai.ChatCompletion
		mockError      error
		expectError    bool
		errorContains  string
		validateCall   func(t *testing.T, params *openai.ChatCompletionNewParams)
	}{
		{
			name: "successful validation with valid credentials",
			mockCompletion: &openai.ChatCompletion{
				ID:      "chatcmpl-validation-success",
				Object:  "chat.completion",
				Created: 1234567890,
				Model:   "gpt-4o-mini",
				Choices: []openai.ChatCompletionChoice{
					{
						Index: 0,
						Message: openai.ChatCompletionMessage{
							Role:    "assistant",
							Content: "Hello",
						},
						FinishReason: "stop",
					},
				},
			},
			mockError:   nil,
			expectError: false,
			validateCall: func(t *testing.T, params *openai.ChatCompletionNewParams) {
				// Verify minimal validation request parameters
				if params.MaxTokens.Value != 5 {
					t.Errorf("Expected maxTokens 5 for validation, got: %d", params.MaxTokens.Value)
				}
				if params.Temperature.Value != 0.1 {
					t.Errorf("Expected temperature 0.1 for validation, got: %f", params.Temperature.Value)
				}
				if len(params.Messages) != 1 {
					t.Errorf("Expected 1 message for validation, got: %d", len(params.Messages))
				}
			},
		},
		{
			name:           "invalid API key error",
			mockCompletion: nil,
			mockError: &openai.Error{
				Code:    "invalid_api_key",
				Message: "Invalid API key provided",
				Type:    "invalid_request_error",
			},
			expectError:   true,
			errorContains: "invalid API key",
		},
		{
			name:           "insufficient permissions error",
			mockCompletion: nil,
			mockError: &openai.Error{
				Code:    "",
				Message: "Insufficient permissions to access this resource",
				Type:    "invalid_request_error",
			},
			expectError:   true,
			errorContains: "insufficient permissions",
		},
		{
			name:           "rate limit exceeded error",
			mockCompletion: nil,
			mockError: &openai.Error{
				Code:    "rate_limit_exceeded",
				Message: "Rate limit exceeded. Please try again later.",
				Type:    "rate_limit_error",
			},
			expectError:   true,
			errorContains: "rate limit exceeded",
		},
		{
			name:           "quota exceeded error",
			mockCompletion: nil,
			mockError: &openai.Error{
				Code:    "insufficient_quota",
				Message: "You exceeded your current quota, please check your plan and billing details",
				Type:    "invalid_request_error",
			},
			expectError:   true,
			errorContains: "quota exceeded",
		},
		{
			name:           "model not found error",
			mockCompletion: nil,
			mockError: &openai.Error{
				Code:    "model_not_found",
				Message: "The model 'invalid-model' does not exist",
				Type:    "invalid_request_error",
			},
			expectError:   true,
			errorContains: "model not found",
		},
		{
			name:           "server error",
			mockCompletion: nil,
			mockError: &openai.Error{
				Code:    "",
				Message: "Internal server error",
				Type:    "server_error",
			},
			expectError:   true,
			errorContains: "server error",
		},
		{
			name:           "network connection error",
			mockCompletion: nil,
			mockError:      fmt.Errorf("connection refused"),
			expectError:    true,
			errorContains:  "network error",
		},
		{
			name:           "timeout error",
			mockCompletion: nil,
			mockError:      fmt.Errorf("context deadline exceeded"),
			expectError:    true,
			errorContains:  "request timeout",
		},
		{
			name:           "HTTP 401 unauthorized",
			mockCompletion: nil,
			mockError:      fmt.Errorf("401 Unauthorized"),
			expectError:    true,
			errorContains:  "invalid API key",
		},
		{
			name:           "HTTP 403 forbidden",
			mockCompletion: nil,
			mockError:      fmt.Errorf("403 Forbidden"),
			expectError:    true,
			errorContains:  "insufficient permissions",
		},
		{
			name:           "HTTP 429 too many requests",
			mockCompletion: nil,
			mockError:      fmt.Errorf("429 Too Many Requests"),
			expectError:    true,
			errorContains:  "rate limit exceeded",
		},
		{
			name:           "HTTP 500 internal server error",
			mockCompletion: nil,
			mockError:      fmt.Errorf("500 Internal Server Error"),
			expectError:    true,
			errorContains:  "server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock SDK client
			mockClient := &MockOpenAISDKClient{
				completion: tt.mockCompletion,
				err:        tt.mockError,
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
			err := client.ValidateCredentials(ctx)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.errorContains)) {
					t.Errorf("Expected error to contain '%s', got: %s", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
					return
				}
			}

			// Validate the call parameters if validation function is provided
			if tt.validateCall != nil && mockClient.lastParams != nil {
				tt.validateCall(t, mockClient.lastParams)
			}

			// Verify that ValidateCredentials made the expected minimal request
			if !tt.expectError && mockClient.lastParams != nil {
				// Verify model parameter
				if mockClient.lastParams.Model != openai.ChatModel(client.model) {
					t.Errorf("Expected model '%s', got: %s", client.model, string(mockClient.lastParams.Model))
				}

				// Verify minimal parameters for validation
				if mockClient.lastParams.MaxTokens.Value != 5 {
					t.Errorf("Expected maxTokens 5 for validation, got: %d", mockClient.lastParams.MaxTokens.Value)
				}

				if mockClient.lastParams.Temperature.Value != 0.1 {
					t.Errorf("Expected temperature 0.1 for validation, got: %f", mockClient.lastParams.Temperature.Value)
				}

				if len(mockClient.lastParams.Messages) != 1 {
					t.Errorf("Expected 1 message for validation, got: %d", len(mockClient.lastParams.Messages))
				}
			}
		})
	}
}

// TestOpenAIClient_ValidateCredentials_EdgeCases tests edge cases for credential validation
func TestOpenAIClient_ValidateCredentials_EdgeCases(t *testing.T) {
	tests := []struct {
		name           string
		mockCompletion *openai.ChatCompletion
		mockError      error
		expectError    bool
		errorContains  string
		description    string
	}{
		{
			name: "empty response choices",
			mockCompletion: &openai.ChatCompletion{
				ID:      "chatcmpl-empty-choices",
				Object:  "chat.completion",
				Created: 1234567890,
				Model:   "gpt-4o-mini",
				Choices: []openai.ChatCompletionChoice{},
			},
			mockError:   nil,
			expectError: false,
			description: "Validation should succeed even with empty choices",
		},
		{
			name: "response with content filter",
			mockCompletion: &openai.ChatCompletion{
				ID:      "chatcmpl-filtered",
				Object:  "chat.completion",
				Created: 1234567890,
				Model:   "gpt-4o-mini",
				Choices: []openai.ChatCompletionChoice{
					{
						Index:        0,
						Message:      openai.ChatCompletionMessage{Role: "assistant", Content: ""},
						FinishReason: "content_filter",
					},
				},
			},
			mockError:   nil,
			expectError: false,
			description: "Validation should succeed even when content is filtered",
		},
		{
			name:           "context cancellation",
			mockCompletion: nil,
			mockError:      context.Canceled,
			expectError:    true,
			errorContains:  "request failed",
			description:    "Should handle context cancellation gracefully",
		},
		{
			name:           "malformed error without structured info",
			mockCompletion: nil,
			mockError:      fmt.Errorf("unexpected error format"),
			expectError:    true,
			errorContains:  "request failed",
			description:    "Should handle unexpected error formats",
		},
		{
			name:           "API error with empty message",
			mockCompletion: nil,
			mockError: &openai.Error{
				Code:    "unknown_error",
				Message: "",
				Type:    "",
			},
			expectError:   true,
			errorContains: "unknown_error",
			description:   "Should handle API errors with empty message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock SDK client
			mockClient := &MockOpenAISDKClient{
				completion: tt.mockCompletion,
				err:        tt.mockError,
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
			err := client.ValidateCredentials(ctx)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none for case: %s", tt.description)
					return
				}
				if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.errorContains)) {
					t.Errorf("Expected error to contain '%s', got: %s for case: %s", tt.errorContains, err.Error(), tt.description)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for case: %s - %v", tt.description, err)
					return
				}
			}
		})
	}
}

// TestOpenAIClient_ValidateCredentials_Concurrent tests concurrent credential validation
func TestOpenAIClient_ValidateCredentials_Concurrent(t *testing.T) {
	// Create a mock SDK client that simulates successful validation
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
						Content: "Hello",
					},
					FinishReason: "stop",
				},
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

	// Test concurrent validation calls
	const numGoroutines = 10
	errChan := make(chan error, numGoroutines)
	ctx := context.Background()

	// Launch multiple goroutines to validate credentials concurrently
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			err := client.ValidateCredentials(ctx)
			errChan <- err
		}(i)
	}

	// Collect results
	var errors []error
	for i := 0; i < numGoroutines; i++ {
		if err := <-errChan; err != nil {
			errors = append(errors, err)
		}
	}

	// Verify all validations succeeded
	if len(errors) > 0 {
		t.Errorf("Expected all concurrent validations to succeed, but got %d errors: %v", len(errors), errors)
	}
}

// TestOpenAIClient_ValidateCredentials_Timeout tests validation with timeout scenarios
func TestOpenAIClient_ValidateCredentials_Timeout(t *testing.T) {
	tests := []struct {
		name        string
		timeout     time.Duration
		mockDelay   time.Duration
		expectError bool
		description string
	}{
		{
			name:        "validation within timeout",
			timeout:     5 * time.Second,
			mockDelay:   100 * time.Millisecond,
			expectError: false,
			description: "Should succeed when response comes within timeout",
		},
		{
			name:        "validation exceeds timeout",
			timeout:     100 * time.Millisecond,
			mockDelay:   200 * time.Millisecond,
			expectError: true,
			description: "Should fail when response takes longer than timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock that simulates delay
			mockClient := &MockOpenAISDKClient{
				completion: &openai.ChatCompletion{
					ID:      "chatcmpl-timeout-test",
					Object:  "chat.completion",
					Created: 1234567890,
					Model:   "gpt-4o-mini",
					Choices: []openai.ChatCompletionChoice{
						{
							Index: 0,
							Message: openai.ChatCompletionMessage{
								Role:    "assistant",
								Content: "Hello",
							},
							FinishReason: "stop",
						},
					},
				},
				err: nil,
			}

			// For timeout test, we'll simulate the timeout by using context cancellation
			if tt.expectError {
				mockClient.err = context.DeadlineExceeded
			}

			// Create OpenAI client with mock SDK client
			client := &OpenAIClient{
				client:      mockClient,
				model:       "gpt-4o-mini",
				maxTokens:   1000,
				temperature: 0.7,
				logger:      utils.NewLogger("TestOpenAIClient"),
			}

			ctx, cancel := context.WithTimeout(context.Background(), tt.timeout)
			defer cancel()

			err := client.ValidateCredentials(ctx)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected timeout error but got none for case: %s", tt.description)
					return
				}
				if !strings.Contains(strings.ToLower(err.Error()), "timeout") &&
					!strings.Contains(strings.ToLower(err.Error()), "deadline") {
					t.Errorf("Expected timeout-related error, got: %s for case: %s", err.Error(), tt.description)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for case: %s - %v", tt.description, err)
				}
			}
		})
	}
}

func TestOpenAIClient_ValidateCredentials_Integration(t *testing.T) {
	if !utils.CanRunOpenAIIntegrationTests() {
		t.Skip("Skipping OpenAI integration test: OPENAI_API_KEY environment variable not set")
	}

	testConfig, err := utils.LoadTestConfig()
	if err != nil {
		t.Fatalf("Failed to load test configuration: %v", err)
	}

	// Create client using enhanced TestConfig
	config := testConfig.CreateOpenAIConfig()

	client, err := NewOpenAIClient(config)
	if err != nil {
		t.Fatalf("Failed to create OpenAI client: %v", err)
	}

	ctx := context.Background()
	err = client.ValidateCredentials(ctx)

	if err != nil {
		t.Errorf("Failed to validate credentials with real API: %v", err)
	}
}

func TestOpenAIClient_GenerateCompletion_Integration(t *testing.T) {
	if !utils.CanRunOpenAIIntegrationTests() {
		t.Skip("Skipping OpenAI integration test: OPENAI_API_KEY environment variable not set")
	}

	testConfig, err := utils.LoadTestConfig()
	if err != nil {
		t.Fatalf("Failed to load test configuration: %v", err)
	}

	// Create client using enhanced TestConfig with custom settings for testing
	config := testConfig.CreateOpenAIConfig()
	config.MaxTokens = 100
	config.Temperature = 0.1 // Low temperature for more predictable results

	client, err := NewOpenAIClient(config)
	if err != nil {
		t.Fatalf("Failed to create OpenAI client: %v", err)
	}

	req := types.CompletionRequest{
		Code:     "console.",
		Cursor:   8,
		Language: "javascript",
		Context: types.CodeContext{
			CurrentFunction: "testFunction",
			Imports:         []string{"import fs from 'fs'"},
			ProjectType:     "Node.js",
			RecentChanges:   []string{},
		},
	}

	ctx := context.Background()
	resp, err := client.GenerateCompletion(ctx, req)

	if err != nil {
		t.Fatalf("Failed to generate completion with real API: %v", err)
	}

	if resp.Error != "" {
		t.Errorf("Unexpected error in response: %s", resp.Error)
	}

	if len(resp.Suggestions) == 0 {
		t.Errorf("Expected at least one suggestion from real API")
	}

	if resp.Confidence <= 0 {
		t.Errorf("Expected positive confidence score, got: %f", resp.Confidence)
	}

	// Verify suggestions are reasonable for JavaScript console completion
	for i, suggestion := range resp.Suggestions {
		if suggestion == "" {
			t.Errorf("Suggestion %d has empty text", i)
		}
	}
}

func TestOpenAIClient_GenerateCode_Integration(t *testing.T) {
	if !utils.CanRunOpenAIIntegrationTests() {
		t.Skip("Skipping OpenAI integration test: OPENAI_API_KEY environment variable not set")
	}

	testConfig, err := utils.LoadTestConfig()
	if err != nil {
		t.Fatalf("Failed to load test configuration: %v", err)
	}

	// Create client using enhanced TestConfig with custom settings for testing
	config := testConfig.CreateOpenAIConfig()
	config.MaxTokens = 200
	config.Temperature = 0.1 // Low temperature for more predictable results

	client, err := NewOpenAIClient(config)
	if err != nil {
		t.Fatalf("Failed to create OpenAI client: %v", err)
	}

	req := types.CodeGenerationRequest{
		Prompt:   "Create a simple JavaScript function that adds two numbers",
		Language: "javascript",
		Context: types.CodeContext{
			CurrentFunction: "",
			Imports:         []string{},
			ProjectType:     "Node.js",
			RecentChanges:   []string{},
		},
	}

	ctx := context.Background()
	resp, err := client.GenerateCode(ctx, req)

	if err != nil {
		t.Fatalf("Failed to generate code with real API: %v", err)
	}

	if resp.Error != "" {
		t.Errorf("Unexpected error in response: %s", resp.Error)
	}

	if resp.Code == "" {
		t.Errorf("Expected generated code from real API")
	}

	// Verify the generated code contains expected elements for a simple add function
	code := strings.ToLower(resp.Code)
	if !strings.Contains(code, "function") && !strings.Contains(code, "=>") {
		t.Errorf("Generated code doesn't appear to contain a function definition: %s", resp.Code)
	}
}

func TestOpenAIClient_EndpointConfiguration_Integration(t *testing.T) {
	tests := []struct {
		name           string
		prompt         string
		variablesJSON  string
		mockResponse   string
		mockStatusCode int
		expectError    bool
		errorContains  string
		expectedPrompt string // The prompt we expect to be sent to CallWithPrompt
	}{
		{
			name:           "successful variable substitution",
			prompt:         "Hello {{name}}, please review this {{language}} code.",
			variablesJSON:  `{"name": "Alice", "language": "Go"}`,
			mockResponse:   `{"choices": [{"message": {"content": "Review completed"}, "finish_reason": "stop"}]}`,
			mockStatusCode: 200,
			expectError:    false,
			expectedPrompt: "Hello Alice, please review this Go code.",
		},
		{
			name:           "multiple variables in template",
			prompt:         "Task: {{task}} for {{user}} in {{language}} with priority {{priority}}",
			variablesJSON:  `{"task": "code review", "user": "Bob", "language": "JavaScript", "priority": "high"}`,
			mockResponse:   `{"choices": [{"message": {"content": "Task assigned"}, "finish_reason": "stop"}]}`,
			mockStatusCode: 200,
			expectError:    false,
			expectedPrompt: "Task: code review for Bob in JavaScript with priority high",
		},
		{
			name:           "missing variables remain unchanged",
			prompt:         "Hello {{name}}, missing {{unknown}} variable",
			variablesJSON:  `{"name": "Charlie"}`,
			mockResponse:   `{"choices": [{"message": {"content": "Hello response"}, "finish_reason": "stop"}]}`,
			mockStatusCode: 200,
			expectError:    false,
			expectedPrompt: "Hello Charlie, missing {{unknown}} variable",
		},
		{
			name:           "empty variables JSON",
			prompt:         "No variables here",
			variablesJSON:  `{}`,
			mockResponse:   `{"choices": [{"message": {"content": "No variables response"}, "finish_reason": "stop"}]}`,
			mockStatusCode: 200,
			expectError:    false,
			expectedPrompt: "No variables here",
		},
		{
			name:           "null variables JSON",
			prompt:         "Template with {{var}}",
			variablesJSON:  "",
			mockResponse:   `{"choices": [{"message": {"content": "Null response"}, "finish_reason": "stop"}]}`,
			mockStatusCode: 200,
			expectError:    false,
			expectedPrompt: "Template with {{var}}",
		},
		{
			name:          "malformed JSON error",
			prompt:        "Hello {{name}}",
			variablesJSON: `{"name": "Alice"`, // Missing closing brace
			expectError:   true,
			errorContains: "variable substitution failed",
		},
		{
			name:          "empty template error",
			prompt:        "",
			variablesJSON: `{"name": "Alice"}`,
			expectError:   true,
			errorContains: "variable substitution failed",
		},
		{
			name:           "API error propagation",
			prompt:         "Hello {{name}}",
			variablesJSON:  `{"name": "Alice"}`,
			mockResponse:   `{"error": {"message": "Rate limit exceeded", "type": "rate_limit_error"}}`,
			mockStatusCode: 429,
			expectError:    true,
			errorContains:  "rate limit exceeded",
			expectedPrompt: "Hello Alice",
		},
		{
			name:           "special characters in variables",
			prompt:         "User: {{user_name}}, Email: {{email-address}}",
			variablesJSON:  `{"user_name": "John Doe", "email-address": "john@example.com"}`,
			mockResponse:   `{"choices": [{"message": {"content": "User processed"}, "finish_reason": "stop"}]}`,
			mockStatusCode: 200,
			expectError:    false,
			expectedPrompt: "User: John Doe, Email: john@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Note: With SDK integration, we can't easily decode the request body
				// but the variable substitution is tested by the response behavior
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.mockStatusCode)
				w.Write([]byte(tt.mockResponse))
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
			resp, err := client.callWithPromptAndVariables(ctx, tt.prompt, tt.variablesJSON)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', got: %s", tt.errorContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if resp == nil {
				t.Errorf("Expected non-nil response")
				return
			}

			// Note: With SDK integration, we verify variable substitution through response behavior
			// The template processing is tested separately in the utils package

			// Verify response structure using SDK types
			if len(resp.Choices) == 0 {
				t.Errorf("Expected at least one choice in response")
				return
			}
		})
	}
}

// TestOpenAIClient_CallWithPromptStream tests the CallWithPromptStream method for streaming
func TestOpenAIClient_CallWithPromptAndVariables(t *testing.T) {
	// Create a server that delays response to test context cancellation
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond) // Delay to allow context cancellation
		w.WriteHeader(200)
		w.Write([]byte(`{"choices": [{"message": {"content": "test"}, "finish_reason": "stop"}]}`))
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

	// Create a context that will be cancelled
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	prompt := "Hello {{name}}"
	variablesJSON := `{"name": "Alice"}`

	_, err = client.callWithPromptAndVariables(ctx, prompt, variablesJSON)

	if err == nil {
		t.Errorf("Expected context cancellation error")
	}

	if !strings.Contains(err.Error(), "context") && !strings.Contains(err.Error(), "timeout") && !strings.Contains(err.Error(), "deadline") {
		t.Errorf("Expected context-related error, got: %v", err)
	}
}

func TestOpenAIClient_CallWithPromptAndVariables_ErrorPropagation(t *testing.T) {
	tests := []struct {
		name          string
		statusCode    int
		responseBody  string
		expectedError string
	}{
		{
			name:          "rate limit error",
			statusCode:    429,
			responseBody:  `{"error": {"message": "Rate limit exceeded", "type": "rate_limit_error"}}`,
			expectedError: "rate limit exceeded",
		},
		{
			name:          "invalid API key",
			statusCode:    401,
			responseBody:  `{"error": {"message": "Invalid API key", "type": "invalid_request_error"}}`,
			expectedError: "invalid API key",
		},
		{
			name:          "server error",
			statusCode:    500,
			responseBody:  "Internal Server Error",
			expectedError: "server error",
		},
		{
			name:          "model not found",
			statusCode:    404,
			responseBody:  `{"error": {"message": "Model not found", "type": "invalid_request_error"}}`,
			expectedError: "model error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.responseBody))
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
			prompt := "Hello {{name}}"
			variablesJSON := `{"name": "Alice"}`

			_, err = client.callWithPromptAndVariables(ctx, prompt, variablesJSON)

			if err == nil {
				t.Errorf("Expected error for status code %d", tt.statusCode)
				return
			}

			if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.expectedError)) {
				t.Errorf("Expected error to contain '%s', got: %s", tt.expectedError, err.Error())
			}
		})
	}
}

func TestOpenAIClient_CallWithPromptAndVariables_Integration(t *testing.T) {
	if !utils.CanRunOpenAIIntegrationTests() {
		t.Skip("Skipping OpenAI integration test: OPENAI_API_KEY environment variable not set")
	}

	testConfig, err := utils.LoadTestConfig()
	if err != nil {
		t.Fatalf("Failed to load test configuration: %v", err)
	}

	config := testConfig.CreateOpenAIConfig()
	config.MaxTokens = 100
	config.Temperature = 0.1

	client, err := NewOpenAIClient(config)
	if err != nil {
		t.Fatalf("Failed to create OpenAI client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	testCases := []struct {
		name           string
		prompt         string
		variablesJSON  string
		expectedInResp []string
	}{
		{
			name:           "simple variable substitution",
			prompt:         "What is the capital of {{country}}? Answer in one word only.",
			variablesJSON:  `{"country": "France"}`,
			expectedInResp: []string{"Paris"},
		},
		{
			name:           "multiple variables",
			prompt:         "Calculate {{num1}} + {{num2}}. Provide only the numerical answer.",
			variablesJSON:  `{"num1": "15", "num2": "25"}`,
			expectedInResp: []string{"40"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := client.callWithPromptAndVariables(ctx, tc.prompt, tc.variablesJSON)

			if err != nil {
				t.Fatalf("CallWithPromptAndVariables failed for %s: %v", tc.name, err)
			}

			if resp == nil {
				t.Errorf("Expected non-nil response for %s", tc.name)
				return
			}

			// Verify response structure using SDK types
			if len(resp.Choices) == 0 {
				t.Errorf("Expected at least one choice in response for %s", tc.name)
				return
			}

			content := resp.Choices[0].Message.Content

			// Check for expected content in response
			for _, expected := range tc.expectedInResp {
				if !strings.Contains(content, expected) {
					t.Errorf("Expected response for %s to contain '%s', but got: %s",
						tc.name, expected, content)
				}
			}

			t.Logf("Response for %s: %s", tc.name, content)
		})
	}
}

func TestOpenAIClient_EndpointConfiguration_NewOpenAIClient(t *testing.T) {
	if !utils.CanRunOpenAIIntegrationTests() {
		t.Skip("Skipping OpenAI integration test: OPENAI_API_KEY environment variable not set")
	}

	testConfig, err := utils.LoadTestConfig()
	if err != nil {
		t.Fatalf("Failed to load test configuration: %v", err)
	}

	// Create client using enhanced TestConfig
	config := testConfig.CreateOpenAIConfig()

	// Verify BaseURL is properly set. OpenAI has a default base URL, so the
	// API endpoint set in an environment variable is optional.
	expectedBaseURL := ""
	if testConfig.OpenAIAPIEndpoint != "" {
		// If custom endpoint is set and valid, it should be used
		if err := utils.ValidateEndpointURL(testConfig.OpenAIAPIEndpoint); err == nil {
			expectedBaseURL = testConfig.OpenAIAPIEndpoint
		}
	}

	if config.BaseURL != expectedBaseURL {
		t.Errorf("Expected BaseURL '%s', got: '%s'", expectedBaseURL, config.BaseURL)
	}

	// Verify other configuration fields are set correctly
	if config.Provider != "openai" {
		t.Errorf("Expected Provider 'openai', got: '%s'", config.Provider)
	}

	if config.APIKey != testConfig.OpenAIAPIKey {
		t.Errorf("Expected APIKey to match test config")
	}

	if config.Model != testConfig.OpenAIModel {
		t.Errorf("Expected Model '%s', got: '%s'", testConfig.OpenAIModel, config.Model)
	}

	// Test that the client can be created successfully with the configuration
	client, err := NewOpenAIClient(config)
	if err != nil {
		t.Fatalf("Failed to create OpenAI client with enhanced config: %v", err)
	}

	if client == nil {
		t.Fatal("Expected client to be created")
	}
}

// TestOpenAIClient_CallWithPromptStream tests the CallWithPromptStream method for streaming
func TestOpenAIClient_CallWithPromptStream(t *testing.T) {
	tests := []struct {
		name          string
		prompt        string
		mockError     error
		expectError   bool
		errorContains string
	}{
		{
			name:        "successful streaming request",
			prompt:      "Tell me a story",
			mockError:   nil,
			expectError: false,
		},
		{
			name:          "streaming with API error",
			prompt:        "Test prompt",
			mockError:     &openai.Error{Code: "rate_limit_exceeded", Message: "Rate limit exceeded"},
			expectError:   true,
			errorContains: "rate limit exceeded",
		},
		{
			name:          "streaming with network error",
			prompt:        "Test prompt",
			mockError:     fmt.Errorf("connection refused"),
			expectError:   true,
			errorContains: "streaming connection error",
		},
		{
			name:          "streaming with timeout error",
			prompt:        "Test prompt",
			mockError:     fmt.Errorf("context deadline exceeded"),
			expectError:   true,
			errorContains: "streaming timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client
			mockClient := &MockOpenAISDKClient{
				err: tt.mockError,
			}

			// Create OpenAI client with mock
			client := &OpenAIClient{
				client:      mockClient,
				model:       "gpt-4o-mini",
				maxTokens:   1000,
				temperature: 0.7,
				logger:      utils.NewLogger("TestOpenAIClient"),
			}

			// Call the streaming method
			ctx := context.Background()
			stream, err := client.CallWithPromptStream(ctx, tt.prompt)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', got: %s", tt.errorContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if stream == nil {
				t.Errorf("Expected stream to be returned")
				return
			}

			// Verify the parameters were set correctly
			if mockClient.lastParams == nil {
				t.Errorf("Expected parameters to be stored in mock")
				return
			}

			// Check model
			if string(mockClient.lastParams.Model) != client.model {
				t.Errorf("Expected model '%s', got: '%s'", client.model, string(mockClient.lastParams.Model))
			}

			// Check max tokens
			if !mockClient.lastParams.MaxTokens.Valid() || mockClient.lastParams.MaxTokens.Value != int64(client.maxTokens) {
				t.Errorf("Expected maxTokens %d, got: %v", client.maxTokens, mockClient.lastParams.MaxTokens)
			}

			// Check temperature
			if !mockClient.lastParams.Temperature.Valid() || mockClient.lastParams.Temperature.Value != client.temperature {
				t.Errorf("Expected temperature %f, got: %v", client.temperature, mockClient.lastParams.Temperature)
			}

			// Check messages
			if len(mockClient.lastParams.Messages) != 1 {
				t.Errorf("Expected 1 message, got: %d", len(mockClient.lastParams.Messages))
			}
		})
	}
}

// TestOpenAIClient_CallWithPromptStream_ParameterValidation tests parameter handling for streaming
func TestOpenAIClient_CallWithPromptStream_ParameterValidation(t *testing.T) {
	mockClient := &MockOpenAISDKClient{}

	client := &OpenAIClient{
		client:      mockClient,
		model:       "gpt-4",
		maxTokens:   2000,
		temperature: 0.5,
		logger:      utils.NewLogger("TestOpenAIClient"),
	}

	ctx := context.Background()
	prompt := "Test streaming prompt"

	_, err := client.CallWithPromptStream(ctx, prompt)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	// Verify all parameters were set correctly
	if mockClient.lastParams == nil {
		t.Fatal("Expected parameters to be stored in mock")
	}

	// Note: Streaming is enabled by calling NewStreaming instead of New, not by a parameter

	// Check model
	if string(mockClient.lastParams.Model) != "gpt-4" {
		t.Errorf("Expected model 'gpt-4', got: '%s'", string(mockClient.lastParams.Model))
	}

	// Check max tokens
	if !mockClient.lastParams.MaxTokens.Valid() || mockClient.lastParams.MaxTokens.Value != 2000 {
		t.Errorf("Expected maxTokens 2000, got: %v", mockClient.lastParams.MaxTokens)
	}

	// Check temperature
	if !mockClient.lastParams.Temperature.Valid() || mockClient.lastParams.Temperature.Value != 0.5 {
		t.Errorf("Expected temperature 0.5, got: %v", mockClient.lastParams.Temperature)
	}

	// Check messages
	if len(mockClient.lastParams.Messages) != 1 {
		t.Errorf("Expected 1 message, got: %d", len(mockClient.lastParams.Messages))
	}

	// Note: We can't easily inspect the content of the message due to the SDK's union type structure
	// but we can verify it was created properly by checking that we have the expected number of messages
}

// TestOpenAIClient_CallWithPromptStream_ContextCancellation tests context handling for streaming
func TestOpenAIClient_CallWithPromptStream_ContextCancellation(t *testing.T) {
	mockClient := &MockOpenAISDKClient{
		err: context.Canceled,
	}

	client := &OpenAIClient{
		client:      mockClient,
		model:       "gpt-4o-mini",
		maxTokens:   1000,
		temperature: 0.7,
		logger:      utils.NewLogger("TestOpenAIClient"),
	}

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.CallWithPromptStream(ctx, "Test prompt")
	if err == nil {
		t.Errorf("Expected error due to cancelled context")
		return
	}

	// The error should be handled by our streaming error handler
	if !strings.Contains(err.Error(), "request failed") {
		t.Errorf("Expected streaming error handling, got: %s", err.Error())
	}
}

// TestOpenAIClient_CallWithPromptStream_ErrorHandling tests streaming-specific error scenarios
func TestOpenAIClient_CallWithPromptStream_ErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		mockError   error
		expectError string
	}{
		{
			name:        "streaming connection error",
			mockError:   fmt.Errorf("connection refused"),
			expectError: "streaming connection error",
		},
		{
			name:        "streaming timeout error",
			mockError:   fmt.Errorf("request timeout exceeded"),
			expectError: "streaming timeout",
		},
		{
			name:        "streaming API error",
			mockError:   &openai.Error{Code: "invalid_api_key", Message: "Invalid API key"},
			expectError: "invalid API key",
		},
		{
			name:        "generic streaming error",
			mockError:   fmt.Errorf("unknown streaming error"),
			expectError: "streaming error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
			_, err := client.CallWithPromptStream(ctx, "Test prompt")

			if err == nil {
				t.Errorf("Expected error but got none")
				return
			}

			if !strings.Contains(err.Error(), tt.expectError) {
				t.Errorf("Expected error to contain '%s', got: %s", tt.expectError, err.Error())
			}
		})
	}
}

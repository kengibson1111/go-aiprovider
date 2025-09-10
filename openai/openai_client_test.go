package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kengibson1111/go-aiprovider/types"
	"github.com/kengibson1111/go-aiprovider/utils"
)

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
			if tt.config.Model == "" && client.model != "gpt-3.5-turbo" {
				t.Errorf("Expected default model 'gpt-3.5-turbo', got: %s", client.model)
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
				"model": "gpt-3.5-turbo",
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
				if r.URL.Path != "/v1/chat/completions" {
					t.Errorf("Expected path '/v1/chat/completions', got: %s", r.URL.Path)
				}
				if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
					t.Errorf("Expected Authorization header with Bearer token")
				}

				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			config := &types.AIConfig{
				Provider: "openai",
				APIKey:   "test-key",
				BaseURL:  server.URL,
				Model:    "gpt-3.5-turbo",
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
		name          string
		statusCode    int
		responseBody  string
		expectError   bool
		errorContains string
		expectedSuggs int
	}{
		{
			name:       "successful completion",
			statusCode: 200,
			responseBody: `{
				"id": "chatcmpl-test",
				"object": "chat.completion",
				"created": 1234567890,
				"model": "gpt-3.5-turbo",
				"choices": [
					{
						"index": 0,
						"message": {
							"role": "assistant",
							"content": "log('Hello, World!');"
						},
						"finish_reason": "stop"
					}
				],
				"usage": {
					"prompt_tokens": 10,
					"completion_tokens": 5,
					"total_tokens": 15
				}
			}`,
			expectError:   false,
			expectedSuggs: 1,
		},
		{
			name:       "multiple line completion",
			statusCode: 200,
			responseBody: `{
				"id": "chatcmpl-test",
				"object": "chat.completion",
				"created": 1234567890,
				"model": "gpt-3.5-turbo",
				"choices": [
					{
						"index": 0,
						"message": {
							"role": "assistant",
							"content": "log('Hello');\nconsole.warn('World');"
						},
						"finish_reason": "stop"
					}
				]
			}`,
			expectError:   false,
			expectedSuggs: 2,
		},
		{
			name:          "rate limit error",
			statusCode:    429,
			responseBody:  `{"error": {"message": "Rate limit exceeded", "type": "rate_limit_error"}}`,
			expectError:   true,
			errorContains: "Rate limit exceeded",
		},
		{
			name:       "empty response",
			statusCode: 200,
			responseBody: `{
				"id": "chatcmpl-test",
				"object": "chat.completion",
				"created": 1234567890,
				"model": "gpt-3.5-turbo",
				"choices": []
			}`,
			expectError:   false,
			expectedSuggs: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request format
				if r.Method != "POST" {
					t.Errorf("Expected POST request, got: %s", r.Method)
				}
				if r.URL.Path != "/v1/chat/completions" {
					t.Errorf("Expected path '/v1/chat/completions', got: %s", r.URL.Path)
				}

				// Verify request body structure
				var reqBody OpenAIRequest
				if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
					t.Errorf("Failed to decode request body: %v", err)
				}

				if reqBody.Model == "" {
					t.Errorf("Expected model to be set in request")
				}
				if len(reqBody.Messages) == 0 {
					t.Errorf("Expected messages to be set in request")
				}
				if reqBody.Stream != false {
					t.Errorf("Expected stream to be false")
				}

				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			config := &types.AIConfig{
				Provider: "openai",
				APIKey:   "test-key",
				BaseURL:  server.URL,
				Model:    "gpt-3.5-turbo",
			}

			client, err := NewOpenAIClient(config)
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}

			req := types.CompletionRequest{
				Code:     "console.",
				Cursor:   8,
				Language: "javascript",
				Context: utils.CodeContext{
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
		})
	}
}

func TestOpenAIClient_GenerateCode(t *testing.T) {
	tests := []struct {
		name          string
		statusCode    int
		responseBody  string
		expectError   bool
		errorContains string
		expectedCode  string
	}{
		{
			name:       "successful code generation",
			statusCode: 200,
			responseBody: `{
				"id": "chatcmpl-test",
				"object": "chat.completion",
				"created": 1234567890,
				"model": "gpt-3.5-turbo",
				"choices": [
					{
						"index": 0,
						"message": {
							"role": "assistant",
							"content": "function hello() {\\n  console.log('Hello, World!');\\n}"
						},
						"finish_reason": "stop"
					}
				]
			}`,
			expectError: false,
			expectedCode: `function hello() {
  console.log('Hello, World!');
}`,
		},
		{
			name:       "code with markdown formatting",
			statusCode: 200,
			responseBody: `{
				"id": "chatcmpl-test",
				"object": "chat.completion",
				"created": 1234567890,
				"model": "gpt-3.5-turbo",
				"choices": [
					{
						"index": 0,
						"message": {
							"role": "assistant",
							"content": "` + "```javascript\\nfunction hello() {\\n  console.log('Hello, World!');\\n}\\n```" + `"
						},
						"finish_reason": "stop"
					}
				]
			}`,
			expectError: false,
			expectedCode: `function hello() {
  console.log('Hello, World!');
}`,
		},
		{
			name:          "rate limit error",
			statusCode:    429,
			responseBody:  `{"error": {"message": "Rate limit exceeded", "type": "rate_limit_error"}}`,
			expectError:   true,
			errorContains: "Rate limit exceeded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			config := &types.AIConfig{
				Provider: "openai",
				APIKey:   "test-key",
				BaseURL:  server.URL,
				Model:    "gpt-3.5-turbo",
			}

			client, err := NewOpenAIClient(config)
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}

			req := types.CodeGenerationRequest{
				Prompt:   "Create a hello world function",
				Language: "javascript",
				Context: utils.CodeContext{
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
				if resp.Code != tt.expectedCode {
					t.Errorf("Expected code '%s', got: '%s'", tt.expectedCode, resp.Code)
				}
			}
		})
	}
}

func TestOpenAIClient_PromptBuilding(t *testing.T) {
	config := &types.AIConfig{
		Provider: "openai",
		APIKey:   "test-key",
		Model:    "gpt-3.5-turbo",
	}

	client, err := NewOpenAIClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	t.Run("completion prompt", func(t *testing.T) {
		req := types.CompletionRequest{
			Code:     "console.log('Hello'); console.",
			Cursor:   25,
			Language: "javascript",
			Context: utils.CodeContext{
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
			Context: utils.CodeContext{
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

func TestOpenAIClient_ConfidenceCalculation(t *testing.T) {
	config := &types.AIConfig{
		Provider: "openai",
		APIKey:   "test-key",
		Model:    "gpt-3.5-turbo",
	}

	client, err := NewOpenAIClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Create test responses manually to avoid struct literal issues
	stopResponse := OpenAIResponse{}
	stopResponse.Choices = make([]struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	}, 1)
	stopResponse.Choices[0].Message.Content = "This is a long response with more than fifty characters in it"
	stopResponse.Choices[0].FinishReason = "stop"

	lengthResponse := OpenAIResponse{}
	lengthResponse.Choices = make([]struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	}, 1)
	lengthResponse.Choices[0].Message.Content = "Short"
	lengthResponse.Choices[0].FinishReason = "length"

	filterResponse := OpenAIResponse{}
	filterResponse.Choices = make([]struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	}, 1)
	filterResponse.Choices[0].Message.Content = "Filtered content"
	filterResponse.Choices[0].FinishReason = "content_filter"

	emptyResponse := OpenAIResponse{}

	tests := []struct {
		name               string
		response           OpenAIResponse
		expectedConfidence float64
	}{
		{
			name:               "stop finish reason with long content",
			response:           stopResponse,
			expectedConfidence: 1.0, // 0.7 + 0.2 (stop) + 0.1 (long content)
		},
		{
			name:               "length finish reason",
			response:           lengthResponse,
			expectedConfidence: 0.6, // 0.7 - 0.1 (length)
		},
		{
			name:               "content_filter finish reason",
			response:           filterResponse,
			expectedConfidence: 0.4, // 0.7 - 0.3 (content_filter)
		},
		{
			name:               "empty choices",
			response:           emptyResponse,
			expectedConfidence: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			confidence := client.calculateConfidence(tt.response)
			if confidence != tt.expectedConfidence {
				t.Errorf("Expected confidence %f, got: %f", tt.expectedConfidence, confidence)
			}
		})
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
		Model:    "gpt-3.5-turbo",
	}

	client, err := NewOpenAIClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Set a very short timeout for testing
	client.BaseHTTPClient.HttpClient.Timeout = 50 * time.Millisecond

	req := types.CompletionRequest{
		Code:     "console.",
		Cursor:   8,
		Language: "javascript",
		Context:  utils.CodeContext{},
	}

	ctx := context.Background()
	resp, err := client.GenerateCompletion(ctx, req)

	// Should handle timeout gracefully
	if err != nil {
		t.Errorf("Should handle timeout gracefully, got error: %v", err)
	}

	if resp.Error == "" {
		t.Errorf("Expected error in response due to timeout")
	}
}

func TestOpenAIClient_RateLimitHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(429)
		w.Write([]byte(`{"error": {"message": "Rate limit exceeded", "type": "rate_limit_error"}}`))
	}))
	defer server.Close()

	config := &types.AIConfig{
		Provider: "openai",
		APIKey:   "test-key",
		BaseURL:  server.URL,
		Model:    "gpt-3.5-turbo",
	}

	client, err := NewOpenAIClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	req := types.CompletionRequest{
		Code:     "console.",
		Cursor:   8,
		Language: "javascript",
		Context:  utils.CodeContext{},
	}

	ctx := context.Background()
	resp, err := client.GenerateCompletion(ctx, req)

	if err != nil {
		t.Errorf("Should handle rate limit gracefully, got error: %v", err)
	}

	if !strings.Contains(resp.Error, "Rate limit exceeded") {
		t.Errorf("Expected rate limit error message, got: %s", resp.Error)
	}

	if resp.Confidence != 0.0 {
		t.Errorf("Expected confidence 0.0 for rate limit error, got: %f", resp.Confidence)
	}

	if len(resp.Suggestions) != 0 {
		t.Errorf("Expected no suggestions for rate limit error, got: %d", len(resp.Suggestions))
	}
}

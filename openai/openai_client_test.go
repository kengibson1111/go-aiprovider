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
			errorContains: "rate limit exceeded",
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
			errorContains: "rate limit exceeded",
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
				// Normalize whitespace for comparison
				expectedNormalized := strings.ReplaceAll(strings.TrimSpace(tt.expectedCode), "\n", "\\n")
				actualNormalized := strings.ReplaceAll(strings.TrimSpace(resp.Code), "\n", "\\n")
				if actualNormalized != expectedNormalized {
					t.Errorf("Expected code '%s', got: '%s'", expectedNormalized, actualNormalized)
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
			// Use a small epsilon for floating point comparison
			epsilon := 0.0001
			if confidence < tt.expectedConfidence-epsilon || confidence > tt.expectedConfidence+epsilon {
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
		Context:  types.CodeContext{},
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
			resp, err := client.CallWithPrompt(ctx, tc.prompt)

			if err != nil {
				t.Fatalf("CallWithPrompt failed for %s: %v", tc.name, err)
			}

			if len(resp) == 0 {
				t.Errorf("Expected non-empty response for %s", tc.name)
				return
			}

			// Parse the response to verify it's valid JSON
			var openaiResp OpenAIResponse
			if err := json.Unmarshal(resp, &openaiResp); err != nil {
				t.Errorf("Failed to unmarshal response for %s: %v", tc.name, err)
				return
			}

			// Verify response structure
			if len(openaiResp.Choices) == 0 {
				t.Errorf("Expected at least one choice in response for %s", tc.name)
				return
			}

			content := openaiResp.Choices[0].Message.Content
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
			if openaiResp.Model == "" {
				t.Errorf("Expected model field to be set in response for %s", tc.name)
			}

			if openaiResp.ID == "" {
				t.Errorf("Expected ID field to be set in response for %s", tc.name)
			}

			// Log the response for manual verification during development
			t.Logf("Response for %s: %s", tc.name, content)
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

func TestOpenAIClient_CallWithPromptAndVariables(t *testing.T) {
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
			var actualPromptSent string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Capture the prompt that was actually sent
				var reqBody OpenAIRequest
				if err := json.NewDecoder(r.Body).Decode(&reqBody); err == nil {
					if len(reqBody.Messages) > 0 {
						actualPromptSent = reqBody.Messages[0].Content
					}
				}

				w.WriteHeader(tt.mockStatusCode)
				w.Write([]byte(tt.mockResponse))
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
			resp, err := client.CallWithPromptAndVariables(ctx, tt.prompt, tt.variablesJSON)

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

			if len(resp) == 0 {
				t.Errorf("Expected non-empty response")
				return
			}

			// Verify the correct processed prompt was sent to the API
			if tt.expectedPrompt != "" && actualPromptSent != tt.expectedPrompt {
				t.Errorf("Expected prompt '%s' to be sent to API, but got '%s'", tt.expectedPrompt, actualPromptSent)
			}

			// Verify response is valid JSON
			var openaiResp OpenAIResponse
			if err := json.Unmarshal(resp, &openaiResp); err != nil {
				t.Errorf("Failed to unmarshal response: %v", err)
			}
		})
	}
}

func TestOpenAIClient_CallWithPromptAndVariables_ContextCancellation(t *testing.T) {
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
		Model:    "gpt-3.5-turbo",
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

	_, err = client.CallWithPromptAndVariables(ctx, prompt, variablesJSON)

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
			expectedError: "API error",
		},
		{
			name:          "server error",
			statusCode:    500,
			responseBody:  "Internal Server Error",
			expectedError: "API error",
		},
		{
			name:          "model not found",
			statusCode:    404,
			responseBody:  `{"error": {"message": "Model not found", "type": "invalid_request_error"}}`,
			expectedError: "API error",
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

			ctx := context.Background()
			prompt := "Hello {{name}}"
			variablesJSON := `{"name": "Alice"}`

			_, err = client.CallWithPromptAndVariables(ctx, prompt, variablesJSON)

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

	ctx := context.Background()

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
			resp, err := client.CallWithPromptAndVariables(ctx, tc.prompt, tc.variablesJSON)

			if err != nil {
				t.Fatalf("CallWithPromptAndVariables failed for %s: %v", tc.name, err)
			}

			if len(resp) == 0 {
				t.Errorf("Expected non-empty response for %s", tc.name)
				return
			}

			// Parse the response to verify it's valid JSON
			var openaiResp OpenAIResponse
			if err := json.Unmarshal(resp, &openaiResp); err != nil {
				t.Errorf("Failed to unmarshal response for %s: %v", tc.name, err)
				return
			}

			if len(openaiResp.Choices) == 0 {
				t.Errorf("Expected at least one choice in response for %s", tc.name)
				return
			}

			content := openaiResp.Choices[0].Message.Content

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

func TestOpenAIClient_EndpointConfiguration_Integration(t *testing.T) {
	if !utils.CanRunOpenAIIntegrationTests() {
		t.Skip("Skipping OpenAI integration test: OPENAI_API_KEY environment variable not set")
	}

	testConfig, err := utils.LoadTestConfig()
	if err != nil {
		t.Fatalf("Failed to load test configuration: %v", err)
	}

	// Create client using enhanced TestConfig
	config := testConfig.CreateOpenAIConfig()

	// Verify BaseURL is properly set
	expectedBaseURL := "https://api.openai.com"
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

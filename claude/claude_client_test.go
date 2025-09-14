package claude

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

func TestNewClaudeClient(t *testing.T) {
	tests := []struct {
		name        string
		config      *types.AIConfig
		expectError bool
	}{
		{
			name:        "nil config",
			config:      nil,
			expectError: true,
		},
		{
			name: "valid config with defaults",
			config: &types.AIConfig{
				Provider: "claude",
				APIKey:   "test-key",
			},
			expectError: false,
		},
		{
			name: "valid config with custom values",
			config: &types.AIConfig{
				Provider:    "claude",
				APIKey:      "test-key",
				BaseURL:     "https://custom.api.com",
				Model:       "claude-3-opus-20240229",
				MaxTokens:   2000,
				Temperature: 0.5,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClaudeClient(tt.config)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if client == nil {
				t.Errorf("Expected client but got nil")
				return
			}

			// Check defaults are set
			if client.model == "" {
				t.Errorf("Expected default model to be set")
			}
			if client.maxTokens == 0 {
				t.Errorf("Expected default maxTokens to be set")
			}
			if client.temperature == 0 {
				t.Errorf("Expected default temperature to be set")
			}
		})
	}
}

func TestBuildCompletionPrompt(t *testing.T) {
	client := &ClaudeClient{
		logger: utils.NewLogger("TestClaudeClient"),
	}

	req := types.CompletionRequest{
		Code:     "function hello() {\n  console.log('Hello');\n}",
		Cursor:   25,
		Language: "typescript",
		Context: types.CodeContext{
			CurrentFunction: "hello",
			Imports:         []string{"import React from 'react'"},
			ProjectType:     "React",
		},
	}

	prompt := client.buildCompletionPrompt(req)

	// Check that prompt contains expected elements
	if prompt == "" {
		t.Errorf("Expected non-empty prompt")
	}

	expectedElements := []string{
		"typescript",
		"Current function: hello",
		"import React from 'react'",
		"Project type: React",
		"<CURSOR>",
	}

	for _, element := range expectedElements {
		if !contains(prompt, element) {
			t.Errorf("Expected prompt to contain '%s'", element)
		}
	}
}

func TestBuildCodeGenerationPrompt(t *testing.T) {
	client := &ClaudeClient{
		logger: utils.NewLogger("TestClaudeClient"),
	}

	req := types.CodeGenerationRequest{
		Prompt:   "Create a function that adds two numbers",
		Language: "typescript",
		Context: types.CodeContext{
			CurrentFunction: "calculator",
			Imports:         []string{"import { Calculator } from './types'"},
			ProjectType:     "Node.js",
		},
	}

	prompt := client.buildCodeGenerationPrompt(req)

	// Check that prompt contains expected elements
	if prompt == "" {
		t.Errorf("Expected non-empty prompt")
	}

	expectedElements := []string{
		"typescript",
		"Current function: calculator",
		"import { Calculator } from './types'",
		"Project type: Node.js",
		"Create a function that adds two numbers",
	}

	for _, element := range expectedElements {
		if !contains(prompt, element) {
			t.Errorf("Expected prompt to contain '%s'", element)
		}
	}
}

func TestExtractCompletionSuggestions(t *testing.T) {
	client := &ClaudeClient{
		logger: utils.NewLogger("TestClaudeClient"),
	}

	tests := []struct {
		name     string
		response ClaudeResponse
		expected []string
	}{
		{
			name: "empty content",
			response: ClaudeResponse{
				Content: []struct {
					Type string `json:"type"`
					Text string `json:"text"`
				}{},
			},
			expected: []string{},
		},
		{
			name: "single line suggestion",
			response: ClaudeResponse{
				Content: []struct {
					Type string `json:"type"`
					Text string `json:"text"`
				}{
					{Type: "text", Text: "console.log('World');"},
				},
			},
			expected: []string{"console.log('World');"},
		},
		{
			name: "multi-line suggestion",
			response: ClaudeResponse{
				Content: []struct {
					Type string `json:"type"`
					Text string `json:"text"`
				}{
					{Type: "text", Text: "console.log('World');\nreturn true;"},
				},
			},
			expected: []string{"console.log('World');", "return true;"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestions := client.extractCompletionSuggestions(tt.response)

			if len(suggestions) != len(tt.expected) {
				t.Errorf("Expected %d suggestions, got %d", len(tt.expected), len(suggestions))
				return
			}

			for i, expected := range tt.expected {
				if suggestions[i] != expected {
					t.Errorf("Expected suggestion %d to be '%s', got '%s'", i, expected, suggestions[i])
				}
			}
		})
	}
}

func TestExtractGeneratedCode(t *testing.T) {
	client := &ClaudeClient{
		logger: utils.NewLogger("TestClaudeClient"),
	}

	tests := []struct {
		name     string
		response ClaudeResponse
		expected string
	}{
		{
			name: "empty content",
			response: ClaudeResponse{
				Content: []struct {
					Type string `json:"type"`
					Text string `json:"text"`
				}{},
			},
			expected: "",
		},
		{
			name: "plain code",
			response: ClaudeResponse{
				Content: []struct {
					Type string `json:"type"`
					Text string `json:"text"`
				}{
					{Type: "text", Text: "function add(a, b) { return a + b; }"},
				},
			},
			expected: "function add(a, b) { return a + b; }",
		},
		{
			name: "code with markdown",
			response: ClaudeResponse{
				Content: []struct {
					Type string `json:"type"`
					Text string `json:"text"`
				}{
					{Type: "text", Text: "```typescript\nfunction add(a: number, b: number): number { return a + b; }\n```"},
				},
			},
			expected: "function add(a: number, b: number): number { return a + b; }",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := client.extractGeneratedCode(tt.response)

			if code != tt.expected {
				t.Errorf("Expected code '%s', got '%s'", tt.expected, code)
			}
		})
	}
}

func TestCalculateConfidence(t *testing.T) {
	client := &ClaudeClient{
		logger: utils.NewLogger("TestClaudeClient"),
	}

	tests := []struct {
		name     string
		response ClaudeResponse
		minConf  float64
		maxConf  float64
	}{
		{
			name: "end_turn stop reason",
			response: ClaudeResponse{
				StopReason: "end_turn",
				Content: []struct {
					Type string `json:"type"`
					Text string `json:"text"`
				}{
					{Type: "text", Text: "function test() { return true; }"},
				},
			},
			minConf: 0.8,
			maxConf: 1.0,
		},
		{
			name: "max_tokens stop reason",
			response: ClaudeResponse{
				StopReason: "max_tokens",
				Content: []struct {
					Type string `json:"type"`
					Text string `json:"text"`
				}{
					{Type: "text", Text: "short"},
				},
			},
			minConf: 0.5,
			maxConf: 0.7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			confidence := client.calculateConfidence(tt.response)

			if confidence < tt.minConf || confidence > tt.maxConf {
				t.Errorf("Expected confidence between %f and %f, got %f", tt.minConf, tt.maxConf, confidence)
			}
		})
	}
}

// Integration Tests - These tests use real API endpoints

func TestClaudeClient_CallWithPrompt_Integration(t *testing.T) {
	if !utils.CanRunClaudeIntegrationTests() {
		t.Skip("Skipping Claude integration test: CLAUDE_API_KEY environment variable not set")
	}

	testConfig, err := utils.LoadTestConfig()
	if err != nil {
		t.Fatalf("Failed to load test configuration: %v", err)
	}

	// Create client using enhanced TestConfig with custom settings for testing
	config := testConfig.CreateClaudeConfig()
	config.MaxTokens = 150
	config.Temperature = 0.1 // Low temperature for more predictable results

	client, err := NewClaudeClient(config)
	if err != nil {
		t.Fatalf("Failed to create Claude client: %v", err)
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
			prompt:         "Write a short poem about coding. Return only the poem, no explanations.",
			expectedInResp: []string{}, // Poem content is unpredictable, just check it's not empty
			minLength:      10,
		},
		{
			name:           "mathematical calculation",
			prompt:         "What is 12 * 17? Provide only the numerical answer.",
			expectedInResp: []string{"204"}, // Should contain the correct answer
			minLength:      1,
		},
		{
			name:           "simple question answering",
			prompt:         "What is the capital of Japan? Answer in one word only.",
			expectedInResp: []string{"Tokyo"}, // Should contain Tokyo
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
			var claudeResp ClaudeResponse
			if err := json.Unmarshal(resp, &claudeResp); err != nil {
				t.Errorf("Failed to unmarshal response for %s: %v", tc.name, err)
				return
			}

			// Verify response structure
			if len(claudeResp.Content) == 0 {
				t.Errorf("Expected at least one content block in response for %s", tc.name)
				return
			}

			content := claudeResp.Content[0].Text
			if len(content) < tc.minLength {
				t.Errorf("Response content too short for %s. Expected at least %d characters, got %d: %s",
					tc.name, tc.minLength, len(content), content)
			}

			// Check for expected keywords in response
			for _, expected := range tc.expectedInResp {
				if !contains(content, expected) {
					t.Errorf("Expected response for %s to contain '%s', but got: %s",
						tc.name, expected, content)
				}
			}

			// Verify response metadata
			if claudeResp.Model == "" {
				t.Errorf("Expected model field to be set in response for %s", tc.name)
			}

			if claudeResp.ID == "" {
				t.Errorf("Expected ID field to be set in response for %s", tc.name)
			}

			// Log the response for manual verification during development
			t.Logf("Response for %s: %s", tc.name, content)
		})
	}
}

func TestClaudeIntegration_GenerateCompletion(t *testing.T) {
	if !utils.CanRunClaudeIntegrationTests() {
		t.Skip("Skipping Claude integration test: CLAUDE_API_KEY environment variable not set")
	}

	// Load test configuration
	testConfig, err := utils.LoadTestConfig()
	if err != nil {
		t.Fatalf("Failed to load test configuration: %v", err)
	}

	// Create client using enhanced TestConfig
	config := testConfig.CreateClaudeConfig()

	client, err := NewClaudeClient(config)
	if err != nil {
		t.Fatalf("Failed to create Claude client: %v", err)
	}

	// Test completion request
	req := types.CompletionRequest{
		Code:     "function greet(name) {\n  console.log('Hello, ' + ",
		Cursor:   42, // Position after the +
		Language: "javascript",
		Context: types.CodeContext{
			CurrentFunction: "greet",
			ProjectType:     "Node.js",
		},
	}

	ctx := context.Background()
	response, err := client.GenerateCompletion(ctx, req)
	if err != nil {
		t.Fatalf("GenerateCompletion failed: %v", err)
	}

	// Verify response structure
	if response == nil {
		t.Fatal("Expected non-nil response")
	}

	if len(response.Suggestions) == 0 {
		t.Error("Expected at least one suggestion")
	}

	if response.Confidence < 0 || response.Confidence > 1 {
		t.Errorf("Expected confidence between 0 and 1, got %f", response.Confidence)
	}

	// Verify suggestions are reasonable
	for i, suggestion := range response.Suggestions {
		if suggestion == "" {
			t.Errorf("Suggestion %d is empty", i)
		}
	}
}

func TestClaudeIntegration_GenerateCode(t *testing.T) {
	if !utils.CanRunClaudeIntegrationTests() {
		t.Skip("Skipping Claude integration test: CLAUDE_API_KEY environment variable not set")
	}

	// Load test configuration
	testConfig, err := utils.LoadTestConfig()
	if err != nil {
		t.Fatalf("Failed to load test configuration: %v", err)
	}

	// Create client using enhanced TestConfig
	config := testConfig.CreateClaudeConfig()

	client, err := NewClaudeClient(config)
	if err != nil {
		t.Fatalf("Failed to create Claude client: %v", err)
	}

	// Test code generation request
	req := types.CodeGenerationRequest{
		Prompt:   "Create a simple function that adds two numbers and returns the result",
		Language: "javascript",
		Context: types.CodeContext{
			ProjectType: "Node.js",
		},
	}

	ctx := context.Background()
	response, err := client.GenerateCode(ctx, req)
	if err != nil {
		t.Fatalf("GenerateCode failed: %v", err)
	}

	// Verify response structure
	if response == nil {
		t.Fatal("Expected non-nil response")
	}

	if response.Code == "" {
		t.Error("Expected generated code to be non-empty")
	}

	// Basic validation that the code looks reasonable
	if len(response.Code) < 10 {
		t.Error("Generated code seems too short to be a meaningful function")
	}
}

func TestClaudeIntegration_EndpointConfiguration(t *testing.T) {
	if !utils.CanRunClaudeIntegrationTests() {
		t.Skip("Skipping Claude integration test: CLAUDE_API_KEY environment variable not set")
	}

	// Load test configuration
	testConfig, err := utils.LoadTestConfig()
	if err != nil {
		t.Fatalf("Failed to load test configuration: %v", err)
	}

	// Create client using enhanced TestConfig
	config := testConfig.CreateClaudeConfig()

	// Verify BaseURL is properly set
	expectedBaseURL := "https://api.anthropic.com"
	if testConfig.ClaudeAPIEndpoint != "" {
		// If custom endpoint is set and valid, it should be used
		if err := utils.ValidateEndpointURL(testConfig.ClaudeAPIEndpoint); err == nil {
			expectedBaseURL = testConfig.ClaudeAPIEndpoint
		}
	}

	if config.BaseURL != expectedBaseURL {
		t.Errorf("Expected BaseURL '%s', got: '%s'", expectedBaseURL, config.BaseURL)
	}

	// Verify other configuration fields are set correctly
	if config.Provider != "claude" {
		t.Errorf("Expected Provider 'claude', got: '%s'", config.Provider)
	}

	if config.APIKey != testConfig.ClaudeAPIKey {
		t.Errorf("Expected APIKey to match test config")
	}

	if config.Model != testConfig.ClaudeModel {
		t.Errorf("Expected Model '%s', got: '%s'", testConfig.ClaudeModel, config.Model)
	}

	// Test that the client can be created successfully with the configuration
	client, err := NewClaudeClient(config)
	if err != nil {
		t.Fatalf("Failed to create Claude client with enhanced config: %v", err)
	}

	if client == nil {
		t.Fatal("Expected client to be created")
	}
}

func TestClaudeIntegration_ErrorHandling(t *testing.T) {
	if !utils.CanRunClaudeIntegrationTests() {
		t.Skip("Skipping Claude integration test: CLAUDE_API_KEY environment variable not set")
	}

	// Load test configuration
	testConfig, err := utils.LoadTestConfig()
	if err != nil {
		t.Fatalf("Failed to load test configuration: %v", err)
	}

	// Create client using enhanced TestConfig but override model for error testing
	config := testConfig.CreateClaudeConfig()
	config.Model = "invalid-model-name"

	client, err := NewClaudeClient(config)
	if err != nil {
		t.Fatalf("Failed to create Claude client: %v", err)
	}

	// Test that invalid model returns appropriate error
	req := types.CompletionRequest{
		Code:     "function test() {",
		Cursor:   16,
		Language: "javascript",
	}

	ctx := context.Background()
	response, err := client.GenerateCompletion(ctx, req)
	if err != nil {
		t.Fatalf("GenerateCompletion failed: %v", err)
	}

	// Check that the response contains an error due to invalid model
	if response.Error == "" {
		t.Error("Expected error in response for invalid model, but got none")
	}

	// Verify error response structure
	if len(response.Suggestions) != 0 {
		t.Error("Expected no suggestions for invalid model error")
	}

	if response.Confidence != 0.0 {
		t.Errorf("Expected confidence 0.0 for invalid model error, got: %f", response.Confidence)
	}
}

func TestClaudeClient_CallWithPromptAndVariables(t *testing.T) {
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
			mockResponse:   `{"id": "msg_test", "type": "message", "role": "assistant", "content": [{"type": "text", "text": "Review completed"}], "model": "claude-3-sonnet-20240229", "stop_reason": "end_turn"}`,
			mockStatusCode: 200,
			expectError:    false,
			expectedPrompt: "Hello Alice, please review this Go code.",
		},
		{
			name:           "multiple variables in template",
			prompt:         "Task: {{task}} for {{user}} in {{language}} with priority {{priority}}",
			variablesJSON:  `{"task": "code review", "user": "Bob", "language": "JavaScript", "priority": "high"}`,
			mockResponse:   `{"id": "msg_test", "type": "message", "role": "assistant", "content": [{"type": "text", "text": "Task assigned"}], "model": "claude-3-sonnet-20240229", "stop_reason": "end_turn"}`,
			mockStatusCode: 200,
			expectError:    false,
			expectedPrompt: "Task: code review for Bob in JavaScript with priority high",
		},
		{
			name:           "missing variables remain unchanged",
			prompt:         "Hello {{name}}, missing {{unknown}} variable",
			variablesJSON:  `{"name": "Charlie"}`,
			mockResponse:   `{"id": "msg_test", "type": "message", "role": "assistant", "content": [{"type": "text", "text": "Hello response"}], "model": "claude-3-sonnet-20240229", "stop_reason": "end_turn"}`,
			mockStatusCode: 200,
			expectError:    false,
			expectedPrompt: "Hello Charlie, missing {{unknown}} variable",
		},
		{
			name:           "empty variables JSON",
			prompt:         "No variables here",
			variablesJSON:  `{}`,
			mockResponse:   `{"id": "msg_test", "type": "message", "role": "assistant", "content": [{"type": "text", "text": "No variables response"}], "model": "claude-3-sonnet-20240229", "stop_reason": "end_turn"}`,
			mockStatusCode: 200,
			expectError:    false,
			expectedPrompt: "No variables here",
		},
		{
			name:           "null variables JSON",
			prompt:         "Template with {{var}}",
			variablesJSON:  "",
			mockResponse:   `{"id": "msg_test", "type": "message", "role": "assistant", "content": [{"type": "text", "text": "Null response"}], "model": "claude-3-sonnet-20240229", "stop_reason": "end_turn"}`,
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
			mockResponse:   `{"type": "error", "error": {"type": "rate_limit_error", "message": "Rate limit exceeded"}}`,
			mockStatusCode: 429,
			expectError:    true,
			errorContains:  "API error",
			expectedPrompt: "Hello Alice",
		},
		{
			name:           "special characters in variables",
			prompt:         "User: {{user_name}}, Email: {{email-address}}",
			variablesJSON:  `{"user_name": "John Doe", "email-address": "john@example.com"}`,
			mockResponse:   `{"id": "msg_test", "type": "message", "role": "assistant", "content": [{"type": "text", "text": "User processed"}], "model": "claude-3-sonnet-20240229", "stop_reason": "end_turn"}`,
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
				var reqBody ClaudeRequest
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
				Provider: "claude",
				APIKey:   "test-key",
				BaseURL:  server.URL,
				Model:    "claude-3-sonnet-20240229",
			}

			client, err := NewClaudeClient(config)
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
			var claudeResp ClaudeResponse
			if err := json.Unmarshal(resp, &claudeResp); err != nil {
				t.Errorf("Failed to unmarshal response: %v", err)
			}
		})
	}
}

func TestClaudeClient_CallWithPromptAndVariables_ContextCancellation(t *testing.T) {
	// Create a server that delays response to test context cancellation
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond) // Delay to allow context cancellation
		w.WriteHeader(200)
		w.Write([]byte(`{"id": "msg_test", "type": "message", "role": "assistant", "content": [{"type": "text", "text": "test"}], "model": "claude-3-sonnet-20240229", "stop_reason": "end_turn"}`))
	}))
	defer server.Close()

	config := &types.AIConfig{
		Provider: "claude",
		APIKey:   "test-key",
		BaseURL:  server.URL,
		Model:    "claude-3-sonnet-20240229",
	}

	client, err := NewClaudeClient(config)
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

func TestClaudeClient_CallWithPromptAndVariables_ErrorPropagation(t *testing.T) {
	tests := []struct {
		name          string
		statusCode    int
		responseBody  string
		expectedError string
	}{
		{
			name:          "rate limit error",
			statusCode:    429,
			responseBody:  `{"type": "error", "error": {"type": "rate_limit_error", "message": "Rate limit exceeded"}}`,
			expectedError: "API error",
		},
		{
			name:          "invalid API key",
			statusCode:    401,
			responseBody:  `{"type": "error", "error": {"type": "authentication_error", "message": "Invalid API key"}}`,
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
			responseBody:  `{"type": "error", "error": {"type": "not_found_error", "message": "Model not found"}}`,
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
				Provider: "claude",
				APIKey:   "test-key",
				BaseURL:  server.URL,
				Model:    "claude-3-sonnet-20240229",
			}

			client, err := NewClaudeClient(config)
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

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && containsAt(s, substr, 0)))
}

func containsAt(s, substr string, start int) bool {
	if start+len(substr) > len(s) {
		return false
	}
	for i := 0; i < len(substr); i++ {
		if s[start+i] != substr[i] {
			if start+1 < len(s) {
				return containsAt(s, substr, start+1)
			}
			return false
		}
	}
	return true
}

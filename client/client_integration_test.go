package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kengibson1111/go-aiprovider/claude"
	"github.com/kengibson1111/go-aiprovider/openai"
	"github.com/kengibson1111/go-aiprovider/types"
)

// TestCrossClientConsistency tests that both OpenAI and Claude clients handle identical inputs consistently
func TestCrossClientConsistency(t *testing.T) {
	testCases := []struct {
		name          string
		prompt        string
		variablesJSON string
		expectError   bool
		errorContains string
	}{
		{
			name:          "successful variable substitution",
			prompt:        "Hello {{name}}, please review this {{language}} code.",
			variablesJSON: `{"name": "Alice", "language": "Go"}`,
			expectError:   false,
		},
		{
			name:          "multiple variables",
			prompt:        "Task: {{task}} for {{user}} in {{language}} with priority {{priority}}",
			variablesJSON: `{"task": "code review", "user": "Bob", "language": "JavaScript", "priority": "high"}`,
			expectError:   false,
		},
		{
			name:          "missing variables remain unchanged",
			prompt:        "Hello {{name}}, missing {{unknown}} variable",
			variablesJSON: `{"name": "Charlie"}`,
			expectError:   false,
		},
		{
			name:          "empty variables JSON",
			prompt:        "No variables here",
			variablesJSON: `{}`,
			expectError:   false,
		},
		{
			name:          "null variables JSON",
			prompt:        "Template with {{var}}",
			variablesJSON: "",
			expectError:   false,
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
			name:          "special characters in variables",
			prompt:        "User: {{user_name}}, Email: {{email-address}}",
			variablesJSON: `{"user_name": "John Doe", "email-address": "john@example.com"}`,
			expectError:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test both clients with identical inputs
			openaiResult := testOpenAIClient(t, tc.prompt, tc.variablesJSON, tc.expectError, tc.errorContains)
			claudeResult := testClaudeClient(t, tc.prompt, tc.variablesJSON, tc.expectError, tc.errorContains)

			// Compare results for consistency
			if openaiResult.hasError != claudeResult.hasError {
				t.Errorf("Error state mismatch: OpenAI hasError=%v, Claude hasError=%v",
					openaiResult.hasError, claudeResult.hasError)
			}

			if openaiResult.hasError && claudeResult.hasError {
				// Both should have similar error types
				if !strings.Contains(openaiResult.errorMsg, "variable substitution failed") !=
					!strings.Contains(claudeResult.errorMsg, "variable substitution failed") {
					t.Errorf("Error type mismatch: OpenAI error='%s', Claude error='%s'",
						openaiResult.errorMsg, claudeResult.errorMsg)
				}
			}

			if !openaiResult.hasError && !claudeResult.hasError {
				// Both should have processed the same prompt
				if openaiResult.processedPrompt != claudeResult.processedPrompt {
					t.Errorf("Processed prompt mismatch: OpenAI='%s', Claude='%s'",
						openaiResult.processedPrompt, claudeResult.processedPrompt)
				}
			}
		})
	}
}

// TestInterfaceCompliance verifies that both clients properly implement the AIClient interface
func TestInterfaceCompliance(t *testing.T) {
	// Create mock servers for both clients
	openaiServer := createMockOpenAIServer()
	defer openaiServer.Close()

	claudeServer := createMockClaudeServer()
	defer claudeServer.Close()

	// Create client configurations
	openaiConfig := &types.AIConfig{
		Provider: "openai",
		APIKey:   "test-key",
		BaseURL:  openaiServer.URL,
		Model:    "gpt-3.5-turbo",
	}

	claudeConfig := &types.AIConfig{
		Provider: "claude",
		APIKey:   "test-key",
		BaseURL:  claudeServer.URL,
		Model:    "claude-3-sonnet-20240229",
	}

	// Create clients
	openaiClient, err := openai.NewOpenAIClient(openaiConfig)
	if err != nil {
		t.Fatalf("Failed to create OpenAI client: %v", err)
	}

	claudeClient, err := claude.NewClaudeClient(claudeConfig)
	if err != nil {
		t.Fatalf("Failed to create Claude client: %v", err)
	}

	// Test that both clients implement AIClient interface
	var clients []AIClient = []AIClient{openaiClient, claudeClient}

	ctx := context.Background()
	prompt := "Hello {{name}}"
	variablesJSON := `{"name": "World"}`

	for i, client := range clients {
		clientName := []string{"OpenAI", "Claude"}[i]

		t.Run(clientName+"_interface_compliance", func(t *testing.T) {
			// Test CallWithPromptAndVariables method exists and works
			resp, err := client.CallWithPromptAndVariables(ctx, prompt, variablesJSON)
			if err != nil {
				t.Errorf("%s CallWithPromptAndVariables failed: %v", clientName, err)
			}
			if len(resp) == 0 {
				t.Errorf("%s CallWithPromptAndVariables returned empty response", clientName)
			}

			// Test CallWithPrompt method exists and works
			resp, err = client.CallWithPrompt(ctx, "Hello World")
			if err != nil {
				t.Errorf("%s CallWithPrompt failed: %v", clientName, err)
			}
			if len(resp) == 0 {
				t.Errorf("%s CallWithPrompt returned empty response", clientName)
			}

			// Test ValidateCredentials method exists
			err = client.ValidateCredentials(ctx)
			if err != nil {
				t.Errorf("%s ValidateCredentials failed: %v", clientName, err)
			}

			// Test GenerateCompletion method exists
			req := types.CompletionRequest{
				Code:     "console.",
				Cursor:   8,
				Language: "javascript",
			}
			compResp, err := client.GenerateCompletion(ctx, req)
			if err != nil {
				t.Errorf("%s GenerateCompletion failed: %v", clientName, err)
			}
			if compResp == nil {
				t.Errorf("%s GenerateCompletion returned nil response", clientName)
			}

			// Test GenerateCode method exists
			codeReq := types.CodeGenerationRequest{
				Prompt:   "Create a hello function",
				Language: "javascript",
			}
			codeResp, err := client.GenerateCode(ctx, codeReq)
			if err != nil {
				t.Errorf("%s GenerateCode failed: %v", clientName, err)
			}
			if codeResp == nil {
				t.Errorf("%s GenerateCode returned nil response", clientName)
			}
		})
	}
}

// TestErrorMessageConsistency verifies that error messages are consistent across implementations
func TestErrorMessageConsistency(t *testing.T) {
	errorTestCases := []struct {
		name          string
		prompt        string
		variablesJSON string
		expectedError string
	}{
		{
			name:          "malformed JSON",
			prompt:        "Hello {{name}}",
			variablesJSON: `{"name": "Alice"`, // Missing closing brace
			expectedError: "variable substitution failed",
		},
		{
			name:          "empty template",
			prompt:        "",
			variablesJSON: `{"name": "Alice"}`,
			expectedError: "variable substitution failed",
		},
		{
			name:          "invalid JSON array",
			prompt:        "Hello {{name}}",
			variablesJSON: `["not", "an", "object"]`,
			expectedError: "variable substitution failed",
		},
	}

	for _, tc := range errorTestCases {
		t.Run(tc.name, func(t *testing.T) {
			openaiResult := testOpenAIClient(t, tc.prompt, tc.variablesJSON, true, tc.expectedError)
			claudeResult := testClaudeClient(t, tc.prompt, tc.variablesJSON, true, tc.expectedError)

			// Both should have errors
			if !openaiResult.hasError {
				t.Errorf("OpenAI should have error for %s", tc.name)
			}
			if !claudeResult.hasError {
				t.Errorf("Claude should have error for %s", tc.name)
			}

			// Error messages should contain the expected error text
			if !strings.Contains(openaiResult.errorMsg, tc.expectedError) {
				t.Errorf("OpenAI error message should contain '%s', got: %s", tc.expectedError, openaiResult.errorMsg)
			}
			if !strings.Contains(claudeResult.errorMsg, tc.expectedError) {
				t.Errorf("Claude error message should contain '%s', got: %s", tc.expectedError, claudeResult.errorMsg)
			}

			// Error messages should be similar in structure
			openaiHasSubstitution := strings.Contains(openaiResult.errorMsg, "variable substitution")
			claudeHasSubstitution := strings.Contains(claudeResult.errorMsg, "variable substitution")

			if openaiHasSubstitution != claudeHasSubstitution {
				t.Errorf("Error message structure mismatch for %s: OpenAI='%s', Claude='%s'",
					tc.name, openaiResult.errorMsg, claudeResult.errorMsg)
			}
		})
	}
}

// TestVariableSubstitutionBehavior verifies that variable substitution behavior is identical between clients
func TestVariableSubstitutionBehavior(t *testing.T) {
	substitutionTestCases := []struct {
		name           string
		prompt         string
		variablesJSON  string
		expectedPrompt string
	}{
		{
			name:           "single variable",
			prompt:         "Hello {{name}}!",
			variablesJSON:  `{"name": "Alice"}`,
			expectedPrompt: "Hello Alice!",
		},
		{
			name:           "multiple variables",
			prompt:         "{{greeting}} {{name}}, welcome to {{place}}!",
			variablesJSON:  `{"greeting": "Hi", "name": "Bob", "place": "our app"}`,
			expectedPrompt: "Hi Bob, welcome to our app!",
		},
		{
			name:           "missing variable unchanged",
			prompt:         "Hello {{name}}, your {{status}} is {{unknown}}",
			variablesJSON:  `{"name": "Charlie", "status": "active"}`,
			expectedPrompt: "Hello Charlie, your active is {{unknown}}",
		},
		{
			name:           "no variables",
			prompt:         "This has no variables",
			variablesJSON:  `{"unused": "value"}`,
			expectedPrompt: "This has no variables",
		},
		{
			name:           "empty variables",
			prompt:         "Hello {{name}}",
			variablesJSON:  `{}`,
			expectedPrompt: "Hello {{name}}",
		},
		{
			name:           "special characters",
			prompt:         "Email: {{email}}, Phone: {{phone-number}}",
			variablesJSON:  `{"email": "test@example.com", "phone-number": "+1-555-0123"}`,
			expectedPrompt: "Email: test@example.com, Phone: +1-555-0123",
		},
	}

	for _, tc := range substitutionTestCases {
		t.Run(tc.name, func(t *testing.T) {
			openaiResult := testOpenAIClient(t, tc.prompt, tc.variablesJSON, false, "")
			claudeResult := testClaudeClient(t, tc.prompt, tc.variablesJSON, false, "")

			// Both should succeed
			if openaiResult.hasError {
				t.Errorf("OpenAI should not have error for %s: %s", tc.name, openaiResult.errorMsg)
			}
			if claudeResult.hasError {
				t.Errorf("Claude should not have error for %s: %s", tc.name, claudeResult.errorMsg)
			}

			// Processed prompts should be identical
			if openaiResult.processedPrompt != claudeResult.processedPrompt {
				t.Errorf("Processed prompt mismatch for %s: OpenAI='%s', Claude='%s'",
					tc.name, openaiResult.processedPrompt, claudeResult.processedPrompt)
			}

			// Processed prompt should match expected
			if openaiResult.processedPrompt != tc.expectedPrompt {
				t.Errorf("OpenAI processed prompt mismatch for %s: expected='%s', got='%s'",
					tc.name, tc.expectedPrompt, openaiResult.processedPrompt)
			}
			if claudeResult.processedPrompt != tc.expectedPrompt {
				t.Errorf("Claude processed prompt mismatch for %s: expected='%s', got='%s'",
					tc.name, tc.expectedPrompt, claudeResult.processedPrompt)
			}
		})
	}
}

// clientTestResult holds the result of testing a client
type clientTestResult struct {
	hasError        bool
	errorMsg        string
	processedPrompt string
}

// testOpenAIClient tests the OpenAI client with given inputs
func testOpenAIClient(t *testing.T, prompt, variablesJSON string, expectError bool, errorContains string) clientTestResult {
	var actualPromptSent string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Capture the prompt that was actually sent
		var reqBody openai.OpenAIRequest
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err == nil {
			if len(reqBody.Messages) > 0 {
				actualPromptSent = reqBody.Messages[0].Content
			}
		}

		w.WriteHeader(200)
		w.Write([]byte(`{"choices": [{"message": {"content": "test response"}, "finish_reason": "stop"}]}`))
	}))
	defer server.Close()

	config := &types.AIConfig{
		Provider: "openai",
		APIKey:   "test-key",
		BaseURL:  server.URL,
		Model:    "gpt-3.5-turbo",
	}

	client, err := openai.NewOpenAIClient(config)
	if err != nil {
		t.Fatalf("Failed to create OpenAI client: %v", err)
	}

	ctx := context.Background()
	_, err = client.CallWithPromptAndVariables(ctx, prompt, variablesJSON)

	result := clientTestResult{
		hasError:        err != nil,
		processedPrompt: actualPromptSent,
	}

	if err != nil {
		result.errorMsg = err.Error()
	}

	return result
}

// testClaudeClient tests the Claude client with given inputs
func testClaudeClient(t *testing.T, prompt, variablesJSON string, expectError bool, errorContains string) clientTestResult {
	var actualPromptSent string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Capture the prompt that was actually sent
		var reqBody claude.ClaudeRequest
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err == nil {
			if len(reqBody.Messages) > 0 {
				actualPromptSent = reqBody.Messages[0].Content
			}
		}

		w.WriteHeader(200)
		w.Write([]byte(`{"id": "msg_test", "type": "message", "role": "assistant", "content": [{"type": "text", "text": "test response"}], "model": "claude-3-sonnet-20240229", "stop_reason": "end_turn"}`))
	}))
	defer server.Close()

	config := &types.AIConfig{
		Provider: "claude",
		APIKey:   "test-key",
		BaseURL:  server.URL,
		Model:    "claude-3-sonnet-20240229",
	}

	client, err := claude.NewClaudeClient(config)
	if err != nil {
		t.Fatalf("Failed to create Claude client: %v", err)
	}

	ctx := context.Background()
	_, err = client.CallWithPromptAndVariables(ctx, prompt, variablesJSON)

	result := clientTestResult{
		hasError:        err != nil,
		processedPrompt: actualPromptSent,
	}

	if err != nil {
		result.errorMsg = err.Error()
	}

	return result
}

// createMockOpenAIServer creates a mock server that responds like OpenAI API
func createMockOpenAIServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/chat/completions":
			w.WriteHeader(200)
			w.Write([]byte(`{
				"id": "chatcmpl-test",
				"object": "chat.completion",
				"created": 1234567890,
				"model": "gpt-3.5-turbo",
				"choices": [
					{
						"index": 0,
						"message": {
							"role": "assistant",
							"content": "Mock response"
						},
						"finish_reason": "stop"
					}
				]
			}`))
		default:
			w.WriteHeader(404)
		}
	}))
}

// createMockClaudeServer creates a mock server that responds like Claude API
func createMockClaudeServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/messages":
			w.WriteHeader(200)
			w.Write([]byte(`{
				"id": "msg_test",
				"type": "message",
				"role": "assistant",
				"content": [
					{
						"type": "text",
						"text": "Mock response"
					}
				],
				"model": "claude-3-sonnet-20240229",
				"stop_reason": "end_turn"
			}`))
		default:
			w.WriteHeader(404)
		}
	}))
}

// TestContextHandlingConsistency verifies that both clients handle context cancellation consistently
func TestContextHandlingConsistency(t *testing.T) {
	// Create servers that delay response to test context cancellation
	openaiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(200)
		w.Write([]byte(`{"choices": [{"message": {"content": "test"}, "finish_reason": "stop"}]}`))
	}))
	defer openaiServer.Close()

	claudeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(200)
		w.Write([]byte(`{"id": "msg_test", "type": "message", "role": "assistant", "content": [{"type": "text", "text": "test"}], "model": "claude-3-sonnet-20240229", "stop_reason": "end_turn"}`))
	}))
	defer claudeServer.Close()

	// Create clients
	openaiConfig := &types.AIConfig{
		Provider: "openai",
		APIKey:   "test-key",
		BaseURL:  openaiServer.URL,
		Model:    "gpt-3.5-turbo",
	}

	claudeConfig := &types.AIConfig{
		Provider: "claude",
		APIKey:   "test-key",
		BaseURL:  claudeServer.URL,
		Model:    "claude-3-sonnet-20240229",
	}

	openaiClient, err := openai.NewOpenAIClient(openaiConfig)
	if err != nil {
		t.Fatalf("Failed to create OpenAI client: %v", err)
	}

	claudeClient, err := claude.NewClaudeClient(claudeConfig)
	if err != nil {
		t.Fatalf("Failed to create Claude client: %v", err)
	}

	// Test context cancellation
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	prompt := "Hello {{name}}"
	variablesJSON := `{"name": "World"}`

	// Test OpenAI client
	_, openaiErr := openaiClient.CallWithPromptAndVariables(ctx, prompt, variablesJSON)

	// Test Claude client
	_, claudeErr := claudeClient.CallWithPromptAndVariables(ctx, prompt, variablesJSON)

	// Both should have context-related errors
	if openaiErr == nil {
		t.Errorf("Expected OpenAI client to have context cancellation error")
	}
	if claudeErr == nil {
		t.Errorf("Expected Claude client to have context cancellation error")
	}

	// Both errors should be context-related
	openaiIsContextError := strings.Contains(strings.ToLower(openaiErr.Error()), "context") ||
		strings.Contains(strings.ToLower(openaiErr.Error()), "timeout") ||
		strings.Contains(strings.ToLower(openaiErr.Error()), "deadline")

	claudeIsContextError := strings.Contains(strings.ToLower(claudeErr.Error()), "context") ||
		strings.Contains(strings.ToLower(claudeErr.Error()), "timeout") ||
		strings.Contains(strings.ToLower(claudeErr.Error()), "deadline")

	if !openaiIsContextError {
		t.Errorf("OpenAI error should be context-related, got: %v", openaiErr)
	}
	if !claudeIsContextError {
		t.Errorf("Claude error should be context-related, got: %v", claudeErr)
	}
}

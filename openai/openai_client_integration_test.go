package openai

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/kengibson1111/go-aiprovider/types"
	"github.com/openai/openai-go/v2"
)

// TestIntegration_OpenAIClient_RealAPI tests all OpenAI client methods against the real API
// This test requires valid OpenAI API credentials in environment variables
func TestIntegration_OpenAIClient_RealAPI(t *testing.T) {
	// Skip integration tests if running in CI or if API key is not available
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY environment variable not set, skipping integration tests")
	}

	// Create client with real API configuration
	config := &types.AIConfig{
		Provider:    "openai",
		APIKey:      apiKey,
		Model:       "gpt-4o-mini", // Use the most cost-effective model for testing
		MaxTokens:   100,           // Keep token usage low for cost efficiency
		Temperature: 0.1,           // Low temperature for consistent responses
	}

	// Use standard OpenAI endpoint (no custom endpoint needed)

	client, err := NewOpenAIClient(config)
	if err != nil {
		t.Fatalf("Failed to create OpenAI client: %v", err)
	}

	// Set a reasonable timeout for all tests
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("ValidateCredentials", func(t *testing.T) {
		err := client.ValidateCredentials(ctx)
		if err != nil {
			t.Errorf("Credential validation failed: %v", err)
		}
	})

	t.Run("CallWithPrompt", func(t *testing.T) {
		completion, err := client.callWithPrompt(ctx, "Say 'Hello, World!' and nothing else.")
		if err != nil {
			t.Errorf("CallWithPrompt failed: %v", err)
			return
		}

		// Verify response structure
		if completion == nil {
			t.Error("Expected non-nil completion")
			return
		}

		if len(completion.Choices) == 0 {
			t.Error("Expected at least one choice in completion")
			return
		}

		content := completion.Choices[0].Message.Content
		if content == "" {
			t.Error("Expected non-empty content in response")
		}

		// Verify the response contains expected greeting
		if !strings.Contains(strings.ToLower(content), "hello") {
			t.Errorf("Expected response to contain 'hello', got: %s", content)
		}

		// Verify usage information is present
		if completion.Usage.TotalTokens == 0 {
			t.Error("Expected non-zero token usage")
		}

		t.Logf("Response: %s", content)
		t.Logf("Tokens used: %d", completion.Usage.TotalTokens)
	})

	t.Run("CallWithMessages", func(t *testing.T) {
		messages := []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage("You are a helpful assistant that responds concisely."),
			openai.UserMessage("What is 2+2?"),
		}

		completion, err := client.CallWithMessages(ctx, messages)
		if err != nil {
			t.Errorf("CallWithMessages failed: %v", err)
			return
		}

		if completion == nil || len(completion.Choices) == 0 {
			t.Error("Expected valid completion with choices")
			return
		}

		content := completion.Choices[0].Message.Content
		if content == "" {
			t.Error("Expected non-empty content in response")
		}

		// Verify the response contains the answer
		if !strings.Contains(content, "4") {
			t.Errorf("Expected response to contain '4', got: %s", content)
		}

		t.Logf("Math response: %s", content)
	})

	t.Run("CallWithTools", func(t *testing.T) {
		// Create a simple function tool using the correct SDK types
		// We'll test that the client can handle function calling requests
		// even if the model doesn't actually call the function

		// For now, just test that CallWithTools method exists and works
		// by passing an empty tools slice
		tools := []openai.ChatCompletionToolUnionParam{}

		completion, err := client.CallWithTools(ctx, "Just respond with 'Hello' - no tools needed.", tools)
		if err != nil {
			t.Errorf("CallWithTools failed: %v", err)
			return
		}

		if completion == nil || len(completion.Choices) == 0 {
			t.Error("Expected valid completion with choices")
			return
		}

		content := completion.Choices[0].Message.Content
		if content == "" {
			t.Error("Expected non-empty content in response")
		}

		t.Logf("Function calling response: %s", content)
	})

	t.Run("CallWithPromptStream", func(t *testing.T) {
		stream, err := client.CallWithPromptStream(ctx, "Count from 1 to 5, one number per line.")
		if err != nil {
			t.Errorf("CallWithPromptStream failed: %v", err)
			return
		}

		var fullResponse strings.Builder
		chunkCount := 0

		// Process streaming chunks
		for stream.Next() {
			chunk := stream.Current()
			chunkCount++

			if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
				fullResponse.WriteString(chunk.Choices[0].Delta.Content)
			}

			// Prevent infinite loops in case of issues
			if chunkCount > 100 {
				t.Error("Too many chunks received, possible infinite loop")
				break
			}
		}

		// Check for streaming errors
		if err := stream.Err(); err != nil {
			t.Errorf("Streaming error: %v", err)
			return
		}

		response := fullResponse.String()
		if response == "" {
			t.Error("Expected non-empty streaming response")
		}

		// Verify we received multiple chunks
		if chunkCount < 2 {
			t.Errorf("Expected multiple chunks, got: %d", chunkCount)
		}

		t.Logf("Streaming response (%d chunks): %s", chunkCount, response)
	})

	t.Run("CallWithPromptAndVariables", func(t *testing.T) {
		prompt := "You are a {{role}} assistant. Respond to this {{task}} request: {{question}}"
		variables := `{
			"role": "helpful",
			"task": "math",
			"question": "What is 3 * 7?"
		}`

		completion, err := client.callWithPromptAndVariables(ctx, prompt, variables)
		if err != nil {
			t.Errorf("CallWithPromptAndVariables failed: %v", err)
			return
		}

		if completion == nil || len(completion.Choices) == 0 {
			t.Error("Expected valid completion with choices")
			return
		}

		content := completion.Choices[0].Message.Content
		if content == "" {
			t.Error("Expected non-empty content in response")
		}

		// Verify the response contains the answer
		if !strings.Contains(content, "21") {
			t.Errorf("Expected response to contain '21', got: %s", content)
		}

		t.Logf("Variable substitution response: %s", content)
	})

	t.Run("GenerateCompletion", func(t *testing.T) {
		req := types.CompletionRequest{
			Code:     "console.log('Hello'); console.",
			Cursor:   30, // Position after the dot
			Language: "javascript",
			Context: types.CodeContext{
				CurrentFunction: "main",
				Imports:         []string{"import fs from 'fs'"},
				ProjectType:     "Node.js",
				RecentChanges:   []string{},
			},
		}

		resp, err := client.GenerateCompletion(ctx, req)
		if err != nil {
			t.Errorf("GenerateCompletion failed: %v", err)
			return
		}

		if resp.Error != "" {
			t.Errorf("Unexpected error in completion response: %s", resp.Error)
			return
		}

		if len(resp.Suggestions) == 0 {
			t.Error("Expected at least one completion suggestion")
			return
		}

		// Verify suggestions are reasonable JavaScript completions
		for i, suggestion := range resp.Suggestions {
			if suggestion == "" {
				t.Errorf("Suggestion %d is empty", i)
			}
			t.Logf("Completion suggestion %d: %s", i, suggestion)
		}
	})

	t.Run("GenerateCode", func(t *testing.T) {
		req := types.CodeGenerationRequest{
			Prompt:   "Create a simple function that adds two numbers and returns the result",
			Language: "javascript",
			Context: types.CodeContext{
				CurrentFunction: "",
				Imports:         []string{},
				ProjectType:     "Node.js",
				RecentChanges:   []string{},
			},
		}

		resp, err := client.GenerateCode(ctx, req)
		if err != nil {
			t.Errorf("GenerateCode failed: %v", err)
			return
		}

		if resp.Error != "" {
			t.Errorf("Unexpected error in code generation response: %s", resp.Error)
			return
		}

		if resp.Code == "" {
			t.Error("Expected non-empty generated code")
			return
		}

		// Verify the code contains function-related keywords
		code := strings.ToLower(resp.Code)
		if !strings.Contains(code, "function") && !strings.Contains(code, "=>") {
			t.Errorf("Expected generated code to contain function definition, got: %s", resp.Code)
		}

		t.Logf("Generated code: %s", resp.Code)
	})

	// Test resource cleanup
	t.Run("CloseIdleConnections", func(t *testing.T) {
		// This should not panic or error
		client.CloseIdleConnections()

		// Verify we can still make requests after cleanup
		completion, err := client.callWithPrompt(ctx, "Say 'cleanup test'")
		if err != nil {
			t.Errorf("Failed to make request after CloseIdleConnections: %v", err)
			return
		}

		if completion == nil || len(completion.Choices) == 0 {
			t.Error("Expected valid response after connection cleanup")
		}
	})
}

// TestIntegration_CustomBaseURL tests the custom base URL functionality
// This test uses a mock server to verify the client can work with custom endpoints
func TestIntegration_CustomBaseURL(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	// Create a mock server that mimics OpenAI API responses
	server := createMockOpenAIServer(t)
	defer server.Close()

	// Create client with custom base URL pointing to our mock server
	config := &types.AIConfig{
		Provider:    "openai",
		APIKey:      "test-key-for-custom-endpoint",
		BaseURL:     server.URL,
		Model:       "gpt-4o-mini",
		MaxTokens:   100,
		Temperature: 0.1,
	}

	client, err := NewOpenAIClient(config)
	if err != nil {
		t.Fatalf("Failed to create OpenAI client with custom base URL: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	t.Run("CustomBaseURL_CallWithPrompt", func(t *testing.T) {
		completion, err := client.callWithPrompt(ctx, "Test prompt")
		if err != nil {
			t.Errorf("CallWithPrompt with custom base URL failed: %v", err)
			return
		}

		if completion == nil || len(completion.Choices) == 0 {
			t.Error("Expected valid completion from custom endpoint")
			return
		}

		content := completion.Choices[0].Message.Content
		if content == "" {
			t.Error("Expected non-empty content from custom endpoint")
		}

		t.Logf("Custom endpoint response: %s", content)
	})

	t.Run("CustomBaseURL_ValidateCredentials", func(t *testing.T) {
		err := client.ValidateCredentials(ctx)
		if err != nil {
			t.Errorf("Credential validation with custom base URL failed: %v", err)
		}
	})
}

// createMockOpenAIServer creates a test server that mimics OpenAI API responses
func createMockOpenAIServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request is properly formatted
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got: %s", r.Method)
		}

		if r.URL.Path != "/chat/completions" {
			t.Errorf("Expected path '/chat/completions', got: %s", r.URL.Path)
		}

		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			t.Errorf("Expected Authorization header with Bearer token, got: %s", authHeader)
		}

		// Return a mock OpenAI response
		response := `{
			"id": "chatcmpl-mock123",
			"object": "chat.completion",
			"created": 1234567890,
			"model": "gpt-4o-mini",
			"choices": [
				{
					"index": 0,
					"message": {
						"role": "assistant",
						"content": "Mock response from custom endpoint"
					},
					"finish_reason": "stop"
				}
			],
			"usage": {
				"prompt_tokens": 10,
				"completion_tokens": 8,
				"total_tokens": 18
			}
		}`

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(response))
	}))
}

// TestIntegration_AdvancedFeatures tests advanced OpenAI features comprehensively
func TestIntegration_AdvancedFeatures(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY environment variable not set, skipping integration tests")
	}

	// Create client with real API configuration
	config := &types.AIConfig{
		Provider:    "openai",
		APIKey:      apiKey,
		Model:       "gpt-4o-mini",
		MaxTokens:   150,
		Temperature: 0.1,
	}

	client, err := NewOpenAIClient(config)
	if err != nil {
		t.Fatalf("Failed to create OpenAI client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	t.Run("StreamingWithMultipleChunks", func(t *testing.T) {
		stream, err := client.CallWithPromptStream(ctx, "Write a short poem about programming, one line at a time.")
		if err != nil {
			t.Errorf("Streaming failed: %v", err)
			return
		}

		var chunks []string
		var fullResponse strings.Builder
		chunkCount := 0

		for stream.Next() {
			chunk := stream.Current()
			chunkCount++

			if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
				content := chunk.Choices[0].Delta.Content
				chunks = append(chunks, content)
				fullResponse.WriteString(content)
			}

			// Safety check
			if chunkCount > 200 {
				t.Error("Too many chunks, possible infinite loop")
				break
			}
		}

		if err := stream.Err(); err != nil {
			t.Errorf("Streaming error: %v", err)
			return
		}

		response := fullResponse.String()
		if response == "" {
			t.Error("Expected non-empty streaming response")
		}

		if chunkCount < 3 {
			t.Errorf("Expected multiple chunks for streaming, got: %d", chunkCount)
		}

		if len(chunks) == 0 {
			t.Error("Expected content chunks from streaming")
		}

		t.Logf("Streaming completed with %d chunks, response length: %d", chunkCount, len(response))

		// Log first few chunks
		maxChunks := 3
		if len(chunks) < maxChunks {
			maxChunks = len(chunks)
		}
		t.Logf("First few chunks: %v", chunks[:maxChunks])
	})

	t.Run("MultiTurnConversation", func(t *testing.T) {
		// Test a multi-turn conversation
		messages := []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage("You are a helpful math tutor. Keep responses concise."),
			openai.UserMessage("What is 15 + 27?"),
		}

		// First turn
		completion1, err := client.CallWithMessages(ctx, messages)
		if err != nil {
			t.Errorf("First turn failed: %v", err)
			return
		}

		if len(completion1.Choices) == 0 {
			t.Error("Expected response in first turn")
			return
		}

		firstResponse := completion1.Choices[0].Message.Content
		if !strings.Contains(firstResponse, "42") {
			t.Errorf("Expected first response to contain '42', got: %s", firstResponse)
		}

		// Add assistant response and continue conversation
		messages = append(messages, openai.AssistantMessage(firstResponse))
		messages = append(messages, openai.UserMessage("Now multiply that by 2"))

		// Second turn
		completion2, err := client.CallWithMessages(ctx, messages)
		if err != nil {
			t.Errorf("Second turn failed: %v", err)
			return
		}

		if len(completion2.Choices) == 0 {
			t.Error("Expected response in second turn")
			return
		}

		secondResponse := completion2.Choices[0].Message.Content
		if !strings.Contains(secondResponse, "84") {
			t.Errorf("Expected second response to contain '84', got: %s", secondResponse)
		}

		t.Logf("Multi-turn conversation:")
		t.Logf("Turn 1: %s", firstResponse)
		t.Logf("Turn 2: %s", secondResponse)
	})

	t.Run("VariableSubstitutionComplexity", func(t *testing.T) {
		// Test complex variable substitution
		prompt := `You are a {{role}} working on {{project_type}} development. 
		Your task is to {{action}} for the {{language}} programming language.
		The context is: {{context}}.
		Please provide a {{output_type}} response.`

		variables := `{
			"role": "senior software engineer",
			"project_type": "web application",
			"action": "write a simple function",
			"language": "JavaScript",
			"context": "building a user authentication system",
			"output_type": "concise"
		}`

		completion, err := client.callWithPromptAndVariables(ctx, prompt, variables)
		if err != nil {
			t.Errorf("Complex variable substitution failed: %v", err)
			return
		}

		if len(completion.Choices) == 0 {
			t.Error("Expected response from variable substitution")
			return
		}

		response := completion.Choices[0].Message.Content
		if response == "" {
			t.Error("Expected non-empty response")
		}

		// Verify the response is contextually appropriate
		responseLower := strings.ToLower(response)
		if !strings.Contains(responseLower, "function") {
			t.Errorf("Expected response to mention 'function', got: %s", response)
		}

		t.Logf("Complex variable substitution response: %s", response)
	})

	t.Run("ErrorHandlingAndRecovery", func(t *testing.T) {
		// Test with an intentionally problematic request to verify error handling
		longPrompt := strings.Repeat("This is a very long prompt that might cause issues. ", 100)

		completion, err := client.callWithPrompt(ctx, longPrompt)

		// This should either succeed or fail gracefully
		if err != nil {
			// Verify error is handled gracefully
			if !strings.Contains(err.Error(), "context length") &&
				!strings.Contains(err.Error(), "token") &&
				!strings.Contains(err.Error(), "limit") {
				t.Errorf("Expected context length related error, got: %v", err)
			}
			t.Logf("Gracefully handled error: %v", err)
		} else {
			// If it succeeded, verify we got a response
			if len(completion.Choices) == 0 {
				t.Error("Expected response or error for long prompt")
			} else {
				t.Logf("Long prompt succeeded with response length: %d", len(completion.Choices[0].Message.Content))
			}
		}
	})

	t.Run("ResourceManagement", func(t *testing.T) {
		// Test resource cleanup and management
		initialClient := client

		// Make several requests to establish connections
		for i := 0; i < 3; i++ {
			_, err := client.callWithPrompt(ctx, fmt.Sprintf("Test request %d", i+1))
			if err != nil {
				t.Errorf("Request %d failed: %v", i+1, err)
			}
		}

		// Test connection cleanup
		client.CloseIdleConnections()

		// Verify client still works after cleanup
		completion, err := client.callWithPrompt(ctx, "Test after cleanup")
		if err != nil {
			t.Errorf("Request after cleanup failed: %v", err)
		}

		if len(completion.Choices) == 0 {
			t.Error("Expected response after cleanup")
		}

		// Verify we're still using the same client instance
		if client != initialClient {
			t.Error("Client instance should remain the same")
		}

		t.Log("Resource management test completed successfully")
	})
}

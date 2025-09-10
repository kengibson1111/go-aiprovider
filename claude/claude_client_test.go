package claude

import (
	"testing"

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
		Context: utils.CodeContext{
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
		Context: utils.CodeContext{
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

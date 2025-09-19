package client

import (
	"context"
	"testing"

	"github.com/kengibson1111/go-aiprovider/types"
)

// MockAIClient implements AIClient interface for testing
type MockAIClient struct {
	generateCompletionFunc  func(ctx context.Context, req types.CompletionRequest) (*types.CompletionResponse, error)
	generateCodeFunc        func(ctx context.Context, req types.CodeGenerationRequest) (*types.CodeGenerationResponse, error)
	validateCredentialsFunc func(ctx context.Context) error
}

func (m *MockAIClient) GenerateCompletion(ctx context.Context, req types.CompletionRequest) (*types.CompletionResponse, error) {
	if m.generateCompletionFunc != nil {
		return m.generateCompletionFunc(ctx, req)
	}
	return &types.CompletionResponse{}, nil
}

func (m *MockAIClient) GenerateCode(ctx context.Context, req types.CodeGenerationRequest) (*types.CodeGenerationResponse, error) {
	if m.generateCodeFunc != nil {
		return m.generateCodeFunc(ctx, req)
	}
	return &types.CodeGenerationResponse{}, nil
}

func (m *MockAIClient) ValidateCredentials(ctx context.Context) error {
	if m.validateCredentialsFunc != nil {
		return m.validateCredentialsFunc(ctx)
	}
	return nil
}

// Test helper functions
func createValidClaudeConfig() *types.AIConfig {
	return &types.AIConfig{
		Provider:    "claude",
		APIKey:      "test-api-key",
		BaseURL:     "https://api.anthropic.com",
		Model:       "claude-3-sonnet-20240229",
		MaxTokens:   1000,
		Temperature: 0.7,
	}
}

func createValidOpenAIConfig() *types.AIConfig {
	return &types.AIConfig{
		Provider:    "openai",
		APIKey:      "test-api-key",
		BaseURL:     "https://api.openai.com",
		Model:       "gpt-4o-mini",
		MaxTokens:   1000,
		Temperature: 0.7,
	}
}

func TestNewClientFactory(t *testing.T) {
	factory := NewClientFactory()

	if factory == nil {
		t.Fatal("NewClientFactory() returned nil")
	}

	if factory.logger == nil {
		t.Error("ClientFactory logger is nil")
	}
}

func TestClientFactory_CreateClient_NilConfig(t *testing.T) {
	factory := NewClientFactory()

	client, err := factory.CreateClient(nil)

	if client != nil {
		t.Error("Expected nil client for nil config")
	}

	if err == nil {
		t.Error("Expected error for nil config")
	}

	expectedError := "configuration is required"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestClientFactory_CreateClient_ClaudeProvider(t *testing.T) {
	factory := NewClientFactory()
	config := createValidClaudeConfig()

	client, err := factory.CreateClient(config)

	if err != nil {
		t.Fatalf("Unexpected error creating Claude client: %v", err)
	}

	if client == nil {
		t.Fatal("Expected non-nil client for valid Claude config")
	}

	// Verify the client implements the AIClient interface
	_, ok := client.(AIClient)
	if !ok {
		t.Error("Created client does not implement AIClient interface")
	}
}

func TestClientFactory_CreateClient_OpenAIProvider(t *testing.T) {
	factory := NewClientFactory()
	config := createValidOpenAIConfig()

	client, err := factory.CreateClient(config)

	if err != nil {
		t.Fatalf("Unexpected error creating OpenAI client: %v", err)
	}

	if client == nil {
		t.Fatal("Expected non-nil client for valid OpenAI config")
	}

	// Verify the client implements the AIClient interface
	_, ok := client.(AIClient)
	if !ok {
		t.Error("Created client does not implement AIClient interface")
	}
}

func TestClientFactory_CreateClient_UnsupportedProvider(t *testing.T) {
	factory := NewClientFactory()
	config := &types.AIConfig{
		Provider:    "unsupported",
		APIKey:      "test-api-key",
		Model:       "test-model",
		MaxTokens:   1000,
		Temperature: 0.7,
	}

	client, err := factory.CreateClient(config)

	if client != nil {
		t.Error("Expected nil client for unsupported provider")
	}

	if err == nil {
		t.Error("Expected error for unsupported provider")
	}

	expectedError := "unsupported provider: unsupported"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestClientFactory_CreateClient_ProviderCaseInsensitive(t *testing.T) {
	factory := NewClientFactory()

	testCases := []struct {
		name     string
		provider string
		valid    bool
	}{
		{"Claude uppercase", "CLAUDE", true},
		{"Claude mixed case", "Claude", true},
		{"Claude lowercase", "claude", true},
		{"OpenAI uppercase", "OPENAI", true},
		{"OpenAI mixed case", "OpenAI", true},
		{"OpenAI lowercase", "openai", true},
		{"Invalid provider", "invalid", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := &types.AIConfig{
				Provider:    tc.provider,
				APIKey:      "test-api-key",
				Model:       "test-model",
				MaxTokens:   1000,
				Temperature: 0.7,
			}

			client, err := factory.CreateClient(config)

			if tc.valid {
				if err != nil {
					t.Errorf("Expected no error for provider '%s', got: %v", tc.provider, err)
				}
				if client == nil {
					t.Errorf("Expected non-nil client for provider '%s'", tc.provider)
				}
			} else {
				if err == nil {
					t.Errorf("Expected error for invalid provider '%s'", tc.provider)
				}
				if client != nil {
					t.Errorf("Expected nil client for invalid provider '%s'", tc.provider)
				}
			}
		})
	}
}

func TestClientFactory_CreateClient_EmptyProvider(t *testing.T) {
	factory := NewClientFactory()
	config := &types.AIConfig{
		Provider:    "",
		APIKey:      "test-api-key",
		Model:       "test-model",
		MaxTokens:   1000,
		Temperature: 0.7,
	}

	client, err := factory.CreateClient(config)

	if client != nil {
		t.Error("Expected nil client for empty provider")
	}

	if err == nil {
		t.Error("Expected error for empty provider")
	}

	expectedError := "unsupported provider: "
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestClientFactory_CreateClient_ConfigValidation(t *testing.T) {
	factory := NewClientFactory()

	testCases := []struct {
		name   string
		config *types.AIConfig
		valid  bool
	}{
		{
			name: "Valid Claude config",
			config: &types.AIConfig{
				Provider:    "claude",
				APIKey:      "test-key",
				Model:       "claude-3-sonnet",
				MaxTokens:   1000,
				Temperature: 0.7,
			},
			valid: true,
		},
		{
			name: "Valid OpenAI config",
			config: &types.AIConfig{
				Provider:    "openai",
				APIKey:      "test-key",
				Model:       "gpt-4o-mini",
				MaxTokens:   1000,
				Temperature: 0.7,
			},
			valid: true,
		},
		{
			name: "Config with empty API key",
			config: &types.AIConfig{
				Provider:    "claude",
				APIKey:      "",
				Model:       "claude-3-sonnet",
				MaxTokens:   1000,
				Temperature: 0.7,
			},
			valid: true, // Client creation should succeed, validation happens later
		},
		{
			name: "Config with empty model",
			config: &types.AIConfig{
				Provider:    "openai",
				APIKey:      "test-key",
				Model:       "",
				MaxTokens:   1000,
				Temperature: 0.7,
			},
			valid: true, // Client creation should succeed, defaults will be applied
		},
		{
			name: "Config with zero max tokens",
			config: &types.AIConfig{
				Provider:    "claude",
				APIKey:      "test-key",
				Model:       "claude-3-sonnet",
				MaxTokens:   0,
				Temperature: 0.7,
			},
			valid: true, // Client creation should succeed, defaults will be applied
		},
		{
			name: "Config with zero temperature",
			config: &types.AIConfig{
				Provider:    "openai",
				APIKey:      "test-key",
				Model:       "gpt-4o-mini",
				MaxTokens:   1000,
				Temperature: 0,
			},
			valid: true, // Client creation should succeed, defaults will be applied
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client, err := factory.CreateClient(tc.config)

			if tc.valid {
				if err != nil {
					t.Errorf("Expected no error for valid config, got: %v", err)
				}
				if client == nil {
					t.Error("Expected non-nil client for valid config")
				}
			} else {
				if err == nil {
					t.Error("Expected error for invalid config")
				}
				if client != nil {
					t.Error("Expected nil client for invalid config")
				}
			}
		})
	}
}

func TestClientFactory_CreateClient_LoggerIntegration(t *testing.T) {
	// This test verifies that the logger is properly integrated
	// We can't easily test log output without modifying the logger,
	// but we can verify the factory has a logger and doesn't panic
	factory := NewClientFactory()

	if factory.logger == nil {
		t.Fatal("ClientFactory logger should not be nil")
	}

	// Test that creating clients doesn't panic when logging
	config := createValidClaudeConfig()
	client, err := factory.CreateClient(config)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if client == nil {
		t.Fatal("Expected non-nil client")
	}
}

// Benchmark tests for performance
func BenchmarkClientFactory_CreateClient_Claude(b *testing.B) {
	factory := NewClientFactory()
	config := createValidClaudeConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client, err := factory.CreateClient(config)
		if err != nil {
			b.Fatalf("Unexpected error: %v", err)
		}
		if client == nil {
			b.Fatal("Expected non-nil client")
		}
	}
}

func BenchmarkClientFactory_CreateClient_OpenAI(b *testing.B) {
	factory := NewClientFactory()
	config := createValidOpenAIConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client, err := factory.CreateClient(config)
		if err != nil {
			b.Fatalf("Unexpected error: %v", err)
		}
		if client == nil {
			b.Fatal("Expected non-nil client")
		}
	}
}

// Table-driven test for comprehensive provider testing
func TestClientFactory_CreateClient_AllProviders(t *testing.T) {
	factory := NewClientFactory()

	providers := []struct {
		name     string
		provider string
		baseURL  string
		model    string
	}{
		{"Claude", "claude", "https://api.anthropic.com", "claude-3-sonnet-20240229"},
		{"OpenAI", "openai", "https://api.openai.com", "gpt-4o-mini"},
	}

	for _, p := range providers {
		t.Run(p.name, func(t *testing.T) {
			config := &types.AIConfig{
				Provider:    p.provider,
				APIKey:      "test-api-key",
				BaseURL:     p.baseURL,
				Model:       p.model,
				MaxTokens:   1000,
				Temperature: 0.7,
			}

			client, err := factory.CreateClient(config)

			if err != nil {
				t.Errorf("Failed to create %s client: %v", p.name, err)
			}

			if client == nil {
				t.Errorf("Expected non-nil %s client", p.name)
			}

			// Verify the client implements all required methods
			_, ok := client.(AIClient)
			if !ok {
				t.Errorf("%s client does not implement AIClient interface", p.name)
			}
		})
	}
}

// Test error handling with various edge cases
func TestClientFactory_CreateClient_EdgeCases(t *testing.T) {
	factory := NewClientFactory()

	testCases := []struct {
		name        string
		config      *types.AIConfig
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Nil config",
			config:      nil,
			expectError: true,
			errorMsg:    "configuration is required",
		},
		{
			name: "Provider with whitespace",
			config: &types.AIConfig{
				Provider: "  claude  ",
				APIKey:   "test-key",
				Model:    "test-model",
			},
			expectError: true,
			errorMsg:    "unsupported provider:   claude  ",
		},
		{
			name: "Provider with special characters",
			config: &types.AIConfig{
				Provider: "claude@#$",
				APIKey:   "test-key",
				Model:    "test-model",
			},
			expectError: true,
			errorMsg:    "unsupported provider: claude@#$",
		},
		{
			name: "Very long provider name",
			config: &types.AIConfig{
				Provider: "verylongprovidernamethatdoesnotexist",
				APIKey:   "test-key",
				Model:    "test-model",
			},
			expectError: true,
			errorMsg:    "unsupported provider: verylongprovidernamethatdoesnotexist",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client, err := factory.CreateClient(tc.config)

			if tc.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if err.Error() != tc.errorMsg {
					t.Errorf("Expected error '%s', got '%s'", tc.errorMsg, err.Error())
				}
				if client != nil {
					t.Error("Expected nil client when error occurs")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if client == nil {
					t.Error("Expected non-nil client")
				}
			}
		})
	}
}

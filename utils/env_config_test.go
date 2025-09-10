package utils

import (
	"os"
	"testing"
)

func TestLoadEnvConfig(t *testing.T) {
	// Test loading when .env file doesn't exist
	err := LoadEnvConfig()
	if err != nil {
		t.Errorf("LoadEnvConfig() should not error when .env file doesn't exist, got: %v", err)
	}
}

func TestGetEnvVar(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		envValue     string
		expected     string
	}{
		{
			name:         "returns environment value when set",
			key:          "TEST_VAR",
			defaultValue: "default",
			envValue:     "env_value",
			expected:     "env_value",
		},
		{
			name:         "returns default when env var not set",
			key:          "NONEXISTENT_VAR",
			defaultValue: "default",
			envValue:     "",
			expected:     "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment variable if needed
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			}

			result := GetEnvVar(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("GetEnvVar(%s, %s) = %s, want %s", tt.key, tt.defaultValue, result, tt.expected)
			}
		})
	}
}

func TestEnvironmentConstants(t *testing.T) {
	// Test that constants are defined correctly
	constants := []string{
		ClaudeAPIKeyEnv,
		ClaudeModelEnv,
		OpenAIAPIKeyEnv,
		OpenAIModelEnv,
	}

	expected := []string{
		"CLAUDE_API_KEY",
		"CLAUDE_MODEL",
		"OPENAI_API_KEY",
		"OPENAI_MODEL",
	}

	for i, constant := range constants {
		if constant != expected[i] {
			t.Errorf("Expected constant %s, got %s", expected[i], constant)
		}
	}
}

func TestLoadTestConfig(t *testing.T) {
	tests := []struct {
		name           string
		claudeAPIKey   string
		claudeModel    string
		openaiAPIKey   string
		openaiModel    string
		expectedConfig *TestConfig
	}{
		{
			name:         "loads config with all environment variables set",
			claudeAPIKey: "test-claude-key",
			claudeModel:  "claude-3-opus-20240229",
			openaiAPIKey: "test-openai-key",
			openaiModel:  "gpt-4",
			expectedConfig: &TestConfig{
				ClaudeAPIKey: "test-claude-key",
				ClaudeModel:  "claude-3-opus-20240229",
				OpenAIAPIKey: "test-openai-key",
				OpenAIModel:  "gpt-4",
			},
		},
		{
			name:         "uses default models when not specified",
			claudeAPIKey: "test-claude-key",
			openaiAPIKey: "test-openai-key",
			expectedConfig: &TestConfig{
				ClaudeAPIKey: "test-claude-key",
				ClaudeModel:  "claude-3-sonnet-20240229",
				OpenAIAPIKey: "test-openai-key",
				OpenAIModel:  "gpt-3.5-turbo",
			},
		},
		{
			name: "loads config with empty API keys",
			expectedConfig: &TestConfig{
				ClaudeAPIKey: "",
				ClaudeModel:  "claude-3-sonnet-20240229",
				OpenAIAPIKey: "",
				OpenAIModel:  "gpt-3.5-turbo",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up environment variables
			defer func() {
				os.Unsetenv(ClaudeAPIKeyEnv)
				os.Unsetenv(ClaudeModelEnv)
				os.Unsetenv(OpenAIAPIKeyEnv)
				os.Unsetenv(OpenAIModelEnv)
			}()

			// Set up test environment variables
			if tt.claudeAPIKey != "" {
				os.Setenv(ClaudeAPIKeyEnv, tt.claudeAPIKey)
			}
			if tt.claudeModel != "" {
				os.Setenv(ClaudeModelEnv, tt.claudeModel)
			}
			if tt.openaiAPIKey != "" {
				os.Setenv(OpenAIAPIKeyEnv, tt.openaiAPIKey)
			}
			if tt.openaiModel != "" {
				os.Setenv(OpenAIModelEnv, tt.openaiModel)
			}

			config, err := LoadTestConfig()
			if err != nil {
				t.Errorf("LoadTestConfig() returned error: %v", err)
				return
			}

			if config.ClaudeAPIKey != tt.expectedConfig.ClaudeAPIKey {
				t.Errorf("ClaudeAPIKey = %s, want %s", config.ClaudeAPIKey, tt.expectedConfig.ClaudeAPIKey)
			}
			if config.ClaudeModel != tt.expectedConfig.ClaudeModel {
				t.Errorf("ClaudeModel = %s, want %s", config.ClaudeModel, tt.expectedConfig.ClaudeModel)
			}
			if config.OpenAIAPIKey != tt.expectedConfig.OpenAIAPIKey {
				t.Errorf("OpenAIAPIKey = %s, want %s", config.OpenAIAPIKey, tt.expectedConfig.OpenAIAPIKey)
			}
			if config.OpenAIModel != tt.expectedConfig.OpenAIModel {
				t.Errorf("OpenAIModel = %s, want %s", config.OpenAIModel, tt.expectedConfig.OpenAIModel)
			}
		})
	}
}

func TestCanRunClaudeIntegrationTests(t *testing.T) {
	tests := []struct {
		name         string
		claudeAPIKey string
		expected     bool
	}{
		{
			name:         "returns true when Claude API key is set",
			claudeAPIKey: "test-claude-key",
			expected:     true,
		},
		{
			name:         "returns false when Claude API key is empty",
			claudeAPIKey: "",
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up environment variable
			defer os.Unsetenv(ClaudeAPIKeyEnv)

			if tt.claudeAPIKey != "" {
				os.Setenv(ClaudeAPIKeyEnv, tt.claudeAPIKey)
			}

			result := CanRunClaudeIntegrationTests()
			if result != tt.expected {
				t.Errorf("CanRunClaudeIntegrationTests() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCanRunOpenAIIntegrationTests(t *testing.T) {
	tests := []struct {
		name         string
		openaiAPIKey string
		expected     bool
	}{
		{
			name:         "returns true when OpenAI API key is set",
			openaiAPIKey: "test-openai-key",
			expected:     true,
		},
		{
			name:         "returns false when OpenAI API key is empty",
			openaiAPIKey: "",
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up environment variable
			defer os.Unsetenv(OpenAIAPIKeyEnv)

			if tt.openaiAPIKey != "" {
				os.Setenv(OpenAIAPIKeyEnv, tt.openaiAPIKey)
			}

			result := CanRunOpenAIIntegrationTests()
			if result != tt.expected {
				t.Errorf("CanRunOpenAIIntegrationTests() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCanRunIntegrationTests(t *testing.T) {
	tests := []struct {
		name         string
		claudeAPIKey string
		openaiAPIKey string
		expected     bool
	}{
		{
			name:         "returns true when Claude API key is set",
			claudeAPIKey: "test-claude-key",
			expected:     true,
		},
		{
			name:         "returns true when OpenAI API key is set",
			openaiAPIKey: "test-openai-key",
			expected:     true,
		},
		{
			name:         "returns true when both API keys are set",
			claudeAPIKey: "test-claude-key",
			openaiAPIKey: "test-openai-key",
			expected:     true,
		},
		{
			name:     "returns false when no API keys are set",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up environment variables
			defer func() {
				os.Unsetenv(ClaudeAPIKeyEnv)
				os.Unsetenv(OpenAIAPIKeyEnv)
			}()

			if tt.claudeAPIKey != "" {
				os.Setenv(ClaudeAPIKeyEnv, tt.claudeAPIKey)
			}
			if tt.openaiAPIKey != "" {
				os.Setenv(OpenAIAPIKeyEnv, tt.openaiAPIKey)
			}

			result := CanRunIntegrationTests()
			if result != tt.expected {
				t.Errorf("CanRunIntegrationTests() = %v, want %v", result, tt.expected)
			}
		})
	}
}

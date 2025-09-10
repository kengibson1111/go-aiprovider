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

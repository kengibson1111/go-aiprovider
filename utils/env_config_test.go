package utils

import (
	"os"
	"strings"
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
		ClaudeAPIEndpointEnv,
		OpenAIAPIKeyEnv,
		OpenAIModelEnv,
		OpenAIAPIEndpointEnv,
	}

	expected := []string{
		"CLAUDE_API_KEY",
		"CLAUDE_MODEL",
		"CLAUDE_API_ENDPOINT",
		"OPENAI_API_KEY",
		"OPENAI_MODEL",
		"OPENAI_API_ENDPOINT",
	}

	for i, constant := range constants {
		if constant != expected[i] {
			t.Errorf("Expected constant %s, got %s", expected[i], constant)
		}
	}
}

func TestLoadTestConfig(t *testing.T) {
	tests := []struct {
		name              string
		claudeAPIKey      string
		claudeModel       string
		claudeAPIEndpoint string
		openaiAPIKey      string
		openaiModel       string
		openaiAPIEndpoint string
		expectedConfig    *TestConfig
	}{
		{
			name:         "loads config with all environment variables set",
			claudeAPIKey: "test-claude-key",
			claudeModel:  "claude-3-opus-20240229",
			openaiAPIKey: "test-openai-key",
			openaiModel:  "gpt-4",
			expectedConfig: &TestConfig{
				ClaudeAPIKey:      "test-claude-key",
				ClaudeModel:       "claude-3-opus-20240229",
				ClaudeAPIEndpoint: "",
				OpenAIAPIKey:      "test-openai-key",
				OpenAIModel:       "gpt-4",
				OpenAIAPIEndpoint: "",
			},
		},
		{
			name:         "uses default models when not specified",
			claudeAPIKey: "test-claude-key",
			openaiAPIKey: "test-openai-key",
			expectedConfig: &TestConfig{
				ClaudeAPIKey:      "test-claude-key",
				ClaudeModel:       "claude-3-sonnet-20240229",
				ClaudeAPIEndpoint: "",
				OpenAIAPIKey:      "test-openai-key",
				OpenAIModel:       "gpt-3.5-turbo",
				OpenAIAPIEndpoint: "",
			},
		},
		{
			name: "loads config with empty API keys",
			expectedConfig: &TestConfig{
				ClaudeAPIKey:      "",
				ClaudeModel:       "claude-3-sonnet-20240229",
				ClaudeAPIEndpoint: "",
				OpenAIAPIKey:      "",
				OpenAIModel:       "gpt-3.5-turbo",
				OpenAIAPIEndpoint: "",
			},
		},
		{
			name:              "loads config with custom endpoints",
			claudeAPIKey:      "test-claude-key",
			claudeAPIEndpoint: "https://custom-claude.example.com",
			openaiAPIKey:      "test-openai-key",
			openaiAPIEndpoint: "https://custom-openai.example.com",
			expectedConfig: &TestConfig{
				ClaudeAPIKey:      "test-claude-key",
				ClaudeModel:       "claude-3-sonnet-20240229",
				ClaudeAPIEndpoint: "https://custom-claude.example.com",
				OpenAIAPIKey:      "test-openai-key",
				OpenAIModel:       "gpt-3.5-turbo",
				OpenAIAPIEndpoint: "https://custom-openai.example.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up environment variables
			defer func() {
				os.Unsetenv(ClaudeAPIKeyEnv)
				os.Unsetenv(ClaudeModelEnv)
				os.Unsetenv(ClaudeAPIEndpointEnv)
				os.Unsetenv(OpenAIAPIKeyEnv)
				os.Unsetenv(OpenAIModelEnv)
				os.Unsetenv(OpenAIAPIEndpointEnv)
			}()

			// Set up test environment variables
			if tt.claudeAPIKey != "" {
				os.Setenv(ClaudeAPIKeyEnv, tt.claudeAPIKey)
			}
			if tt.claudeModel != "" {
				os.Setenv(ClaudeModelEnv, tt.claudeModel)
			}
			if tt.claudeAPIEndpoint != "" {
				os.Setenv(ClaudeAPIEndpointEnv, tt.claudeAPIEndpoint)
			}
			if tt.openaiAPIKey != "" {
				os.Setenv(OpenAIAPIKeyEnv, tt.openaiAPIKey)
			}
			if tt.openaiModel != "" {
				os.Setenv(OpenAIModelEnv, tt.openaiModel)
			}
			if tt.openaiAPIEndpoint != "" {
				os.Setenv(OpenAIAPIEndpointEnv, tt.openaiAPIEndpoint)
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
			if config.ClaudeAPIEndpoint != tt.expectedConfig.ClaudeAPIEndpoint {
				t.Errorf("ClaudeAPIEndpoint = %s, want %s", config.ClaudeAPIEndpoint, tt.expectedConfig.ClaudeAPIEndpoint)
			}
			if config.OpenAIAPIKey != tt.expectedConfig.OpenAIAPIKey {
				t.Errorf("OpenAIAPIKey = %s, want %s", config.OpenAIAPIKey, tt.expectedConfig.OpenAIAPIKey)
			}
			if config.OpenAIModel != tt.expectedConfig.OpenAIModel {
				t.Errorf("OpenAIModel = %s, want %s", config.OpenAIModel, tt.expectedConfig.OpenAIModel)
			}
			if config.OpenAIAPIEndpoint != tt.expectedConfig.OpenAIAPIEndpoint {
				t.Errorf("OpenAIAPIEndpoint = %s, want %s", config.OpenAIAPIEndpoint, tt.expectedConfig.OpenAIAPIEndpoint)
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

func TestValidateEndpointURL(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		expectError bool
		errorMsg    string
	}{
		// Valid URLs
		{
			name:        "empty URL is valid",
			url:         "",
			expectError: false,
		},
		{
			name:        "valid https URL",
			url:         "https://api.example.com",
			expectError: false,
		},
		{
			name:        "valid http URL",
			url:         "http://api.example.com",
			expectError: false,
		},
		{
			name:        "valid URL with port",
			url:         "https://api.example.com:8080",
			expectError: false,
		},
		{
			name:        "valid URL with path",
			url:         "https://api.example.com/v1",
			expectError: false,
		},
		{
			name:        "valid URL with path and port",
			url:         "https://api.example.com:8080/v1/api",
			expectError: false,
		},
		{
			name:        "valid localhost URL",
			url:         "http://localhost:3000",
			expectError: false,
		},
		{
			name:        "valid IP address URL",
			url:         "https://192.168.1.1:8080",
			expectError: false,
		},

		// Invalid URLs - Missing protocol
		{
			name:        "missing protocol scheme",
			url:         "api.example.com",
			expectError: true,
			errorMsg:    "URL must include protocol scheme (http:// or https://)",
		},
		{
			name:        "missing protocol with path",
			url:         "api.example.com/v1",
			expectError: true,
			errorMsg:    "URL must include protocol scheme (http:// or https://)",
		},

		// Invalid URLs - Wrong protocol
		{
			name:        "ftp protocol not allowed",
			url:         "ftp://api.example.com",
			expectError: true,
			errorMsg:    "URL protocol must be http or https, got: ftp",
		},
		{
			name:        "ws protocol not allowed",
			url:         "ws://api.example.com",
			expectError: true,
			errorMsg:    "URL protocol must be http or https, got: ws",
		},

		// Invalid URLs - Missing hostname
		{
			name:        "missing hostname",
			url:         "https://",
			expectError: true,
			errorMsg:    "URL must include a hostname",
		},
		{
			name:        "protocol only",
			url:         "http://",
			expectError: true,
			errorMsg:    "URL must include a hostname",
		},

		// Invalid URLs - Query parameters
		{
			name:        "URL with query parameters",
			url:         "https://api.example.com?param=value",
			expectError: true,
			errorMsg:    "URL must not contain query parameters, found: ?param=value",
		},
		{
			name:        "URL with multiple query parameters",
			url:         "https://api.example.com?param1=value1&param2=value2",
			expectError: true,
			errorMsg:    "URL must not contain query parameters, found: ?param1=value1&param2=value2",
		},

		// Invalid URLs - Malformed
		{
			name:        "malformed URL with space in hostname",
			url:         "https://api example.com",
			expectError: true,
			errorMsg:    "invalid URL format",
		},
		{
			name:        "invalid URL format",
			url:         "not-a-url",
			expectError: true,
			errorMsg:    "URL must include protocol scheme (http:// or https://)",
		},
		{
			name:        "URL with invalid characters in hostname",
			url:         "https://api.exam ple.com",
			expectError: true,
			errorMsg:    "invalid URL format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateEndpointURL(tt.url)

			if tt.expectError {
				if err == nil {
					t.Errorf("validateEndpointURL(%s) expected error but got nil", tt.url)
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("validateEndpointURL(%s) error = %v, want error containing %s", tt.url, err, tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("validateEndpointURL(%s) unexpected error: %v", tt.url, err)
				}
			}
		})
	}
}

func TestCreateClaudeConfig(t *testing.T) {
	tests := []struct {
		name             string
		testConfig       *TestConfig
		expectedBaseURL  string
		expectedProvider string
	}{
		{
			name: "uses default endpoint when custom endpoint is empty",
			testConfig: &TestConfig{
				ClaudeAPIKey:      "test-key",
				ClaudeModel:       "claude-3-sonnet-20240229",
				ClaudeAPIEndpoint: "",
			},
			expectedBaseURL:  "https://api.anthropic.com",
			expectedProvider: "claude",
		},
		{
			name: "uses custom endpoint when valid",
			testConfig: &TestConfig{
				ClaudeAPIKey:      "test-key",
				ClaudeModel:       "claude-3-sonnet-20240229",
				ClaudeAPIEndpoint: "https://custom-claude.example.com",
			},
			expectedBaseURL:  "https://custom-claude.example.com",
			expectedProvider: "claude",
		},
		{
			name: "falls back to default when custom endpoint is invalid",
			testConfig: &TestConfig{
				ClaudeAPIKey:      "test-key",
				ClaudeModel:       "claude-3-sonnet-20240229",
				ClaudeAPIEndpoint: "invalid-url",
			},
			expectedBaseURL:  "https://api.anthropic.com",
			expectedProvider: "claude",
		},
		{
			name: "falls back to default when custom endpoint has query parameters",
			testConfig: &TestConfig{
				ClaudeAPIKey:      "test-key",
				ClaudeModel:       "claude-3-sonnet-20240229",
				ClaudeAPIEndpoint: "https://api.example.com?param=value",
			},
			expectedBaseURL:  "https://api.anthropic.com",
			expectedProvider: "claude",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := tt.testConfig.CreateClaudeConfig()

			if config.BaseURL != tt.expectedBaseURL {
				t.Errorf("CreateClaudeConfig() BaseURL = %s, want %s", config.BaseURL, tt.expectedBaseURL)
			}
			if config.Provider != tt.expectedProvider {
				t.Errorf("CreateClaudeConfig() Provider = %s, want %s", config.Provider, tt.expectedProvider)
			}
			if config.APIKey != tt.testConfig.ClaudeAPIKey {
				t.Errorf("CreateClaudeConfig() APIKey = %s, want %s", config.APIKey, tt.testConfig.ClaudeAPIKey)
			}
			if config.Model != tt.testConfig.ClaudeModel {
				t.Errorf("CreateClaudeConfig() Model = %s, want %s", config.Model, tt.testConfig.ClaudeModel)
			}
		})
	}
}

func TestCreateOpenAIConfig(t *testing.T) {
	tests := []struct {
		name             string
		testConfig       *TestConfig
		expectedBaseURL  string
		expectedProvider string
	}{
		{
			name: "uses default endpoint when custom endpoint is empty",
			testConfig: &TestConfig{
				OpenAIAPIKey:      "test-key",
				OpenAIModel:       "gpt-3.5-turbo",
				OpenAIAPIEndpoint: "",
			},
			expectedBaseURL:  "https://api.openai.com",
			expectedProvider: "openai",
		},
		{
			name: "uses custom endpoint when valid",
			testConfig: &TestConfig{
				OpenAIAPIKey:      "test-key",
				OpenAIModel:       "gpt-3.5-turbo",
				OpenAIAPIEndpoint: "https://custom-openai.example.com",
			},
			expectedBaseURL:  "https://custom-openai.example.com",
			expectedProvider: "openai",
		},
		{
			name: "falls back to default when custom endpoint is invalid",
			testConfig: &TestConfig{
				OpenAIAPIKey:      "test-key",
				OpenAIModel:       "gpt-3.5-turbo",
				OpenAIAPIEndpoint: "invalid-url",
			},
			expectedBaseURL:  "https://api.openai.com",
			expectedProvider: "openai",
		},
		{
			name: "falls back to default when custom endpoint has query parameters",
			testConfig: &TestConfig{
				OpenAIAPIKey:      "test-key",
				OpenAIModel:       "gpt-3.5-turbo",
				OpenAIAPIEndpoint: "https://api.example.com?param=value",
			},
			expectedBaseURL:  "https://api.openai.com",
			expectedProvider: "openai",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := tt.testConfig.CreateOpenAIConfig()

			if config.BaseURL != tt.expectedBaseURL {
				t.Errorf("CreateOpenAIConfig() BaseURL = %s, want %s", config.BaseURL, tt.expectedBaseURL)
			}
			if config.Provider != tt.expectedProvider {
				t.Errorf("CreateOpenAIConfig() Provider = %s, want %s", config.Provider, tt.expectedProvider)
			}
			if config.APIKey != tt.testConfig.OpenAIAPIKey {
				t.Errorf("CreateOpenAIConfig() APIKey = %s, want %s", config.APIKey, tt.testConfig.OpenAIAPIKey)
			}
			if config.Model != tt.testConfig.OpenAIModel {
				t.Errorf("CreateOpenAIConfig() Model = %s, want %s", config.Model, tt.testConfig.OpenAIModel)
			}
		})
	}
}

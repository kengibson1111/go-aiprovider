package utils

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestLoadEnvConfig(t *testing.T) {
	tests := []struct {
		name           string
		setupEnvFile   bool
		envFileContent string
		envFileName    string
		expectError    bool
		errorContains  string
		testEnvVar     string
		expectedValue  string
	}{
		{
			name:         "no .env file exists - should not error",
			setupEnvFile: false,
			expectError:  false,
		},
		{
			name:           "valid .env file in current directory",
			setupEnvFile:   true,
			envFileContent: "TEST_VAR=test_value\nANOTHER_VAR=another_value\n",
			envFileName:    ".env",
			expectError:    false,
			testEnvVar:     "TEST_VAR",
			expectedValue:  "test_value",
		},
		{
			name:           "valid .env file in parent directory",
			setupEnvFile:   true,
			envFileContent: "PARENT_TEST_VAR=parent_value\n",
			envFileName:    "../.env",
			expectError:    false,
			testEnvVar:     "PARENT_TEST_VAR",
			expectedValue:  "parent_value",
		},
		{
			name:           "empty .env file",
			setupEnvFile:   true,
			envFileContent: "",
			envFileName:    ".env",
			expectError:    false,
		},
		{
			name:           ".env file with comments and empty lines",
			setupEnvFile:   true,
			envFileContent: "# This is a comment\nTEST_VAR=test_value\n\n# Another comment\nANOTHER_VAR=another_value\n",
			envFileName:    ".env",
			expectError:    false,
			testEnvVar:     "TEST_VAR",
			expectedValue:  "test_value",
		},
		{
			name:           ".env file with special characters in values",
			setupEnvFile:   true,
			envFileContent: "SPECIAL_VAR=value with spaces and symbols!@#$%\nURL_VAR=https://example.com/path?param=value\n",
			envFileName:    ".env",
			expectError:    false,
			testEnvVar:     "SPECIAL_VAR",
			expectedValue:  "value with spaces and symbols!@#$%",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up any existing test environment variables
			if tt.testEnvVar != "" {
				defer os.Unsetenv(tt.testEnvVar)
			}

			// Set up test .env file if needed
			if tt.setupEnvFile {
				err := os.WriteFile(tt.envFileName, []byte(tt.envFileContent), 0644)
				if err != nil {
					t.Fatalf("Failed to create test .env file: %v", err)
				}
				defer os.Remove(tt.envFileName)
			}

			// Test LoadEnvConfig
			err := LoadEnvConfig()

			if tt.expectError {
				if err == nil {
					t.Errorf("LoadEnvConfig() expected error but got nil")
					return
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("LoadEnvConfig() error = %v, want error containing %s", err, tt.errorContains)
				}
			} else {
				if err != nil {
					t.Errorf("LoadEnvConfig() unexpected error: %v", err)
					return
				}

				// If we expect a specific environment variable to be set, check it
				if tt.testEnvVar != "" && tt.expectedValue != "" {
					actualValue := os.Getenv(tt.testEnvVar)
					if actualValue != tt.expectedValue {
						t.Errorf("After LoadEnvConfig(), %s = %s, want %s", tt.testEnvVar, actualValue, tt.expectedValue)
					}
				}
			}
		})
	}
}

func TestGetEnvVar(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		envValue     string
		expected     string
		description  string
	}{
		{
			name:         "returns environment value when set",
			key:          "TEST_VAR",
			defaultValue: "default",
			envValue:     "env_value",
			expected:     "env_value",
			description:  "Should return the environment variable value when it exists",
		},
		{
			name:         "returns default when env var not set",
			key:          "NONEXISTENT_VAR",
			defaultValue: "default",
			envValue:     "",
			expected:     "default",
			description:  "Should return the default value when environment variable doesn't exist",
		},
		{
			name:         "returns environment value when set to empty string",
			key:          "EMPTY_VAR",
			defaultValue: "default",
			envValue:     "",
			expected:     "default",
			description:  "Should return default when environment variable is set but empty",
		},
		{
			name:         "returns empty default when specified",
			key:          "NONEXISTENT_VAR2",
			defaultValue: "",
			envValue:     "",
			expected:     "",
			description:  "Should return empty string when default is empty and env var doesn't exist",
		},
		{
			name:         "handles special characters in environment value",
			key:          "SPECIAL_CHAR_VAR",
			defaultValue: "default",
			envValue:     "value with spaces, symbols!@#$%, and unicode: 你好",
			expected:     "value with spaces, symbols!@#$%, and unicode: 你好",
			description:  "Should handle special characters and unicode in environment values",
		},
		{
			name:         "handles special characters in default value",
			key:          "NONEXISTENT_SPECIAL",
			defaultValue: "default with spaces, symbols!@#$%, and unicode: 你好",
			envValue:     "",
			expected:     "default with spaces, symbols!@#$%, and unicode: 你好",
			description:  "Should handle special characters and unicode in default values",
		},
		{
			name:         "handles very long environment value",
			key:          "LONG_VAR",
			defaultValue: "short_default",
			envValue:     strings.Repeat("a", 1000),
			expected:     strings.Repeat("a", 1000),
			description:  "Should handle very long environment variable values",
		},
		{
			name:         "handles very long default value",
			key:          "NONEXISTENT_LONG",
			defaultValue: strings.Repeat("b", 1000),
			envValue:     "",
			expected:     strings.Repeat("b", 1000),
			description:  "Should handle very long default values",
		},
		{
			name:         "handles whitespace in environment value",
			key:          "WHITESPACE_VAR",
			defaultValue: "default",
			envValue:     "  value with leading and trailing spaces  ",
			expected:     "  value with leading and trailing spaces  ",
			description:  "Should preserve whitespace in environment variable values",
		},
		{
			name:         "handles newlines in environment value",
			key:          "NEWLINE_VAR",
			defaultValue: "default",
			envValue:     "line1\nline2\nline3",
			expected:     "line1\nline2\nline3",
			description:  "Should handle newlines in environment variable values",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up environment variable
			defer os.Unsetenv(tt.key)

			// Set up environment variable if needed
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
			}

			result := GetEnvVar(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("GetEnvVar(%s, %s) = %s, want %s\nDescription: %s",
					tt.key, tt.defaultValue, result, tt.expected, tt.description)
			}
		})
	}
}

// TestLoadEnvConfigErrorHandling tests error scenarios for LoadEnvConfig
func TestLoadEnvConfigErrorHandling(t *testing.T) {
	tests := []struct {
		name          string
		setupFunc     func(t *testing.T) (cleanup func())
		expectError   bool
		errorContains string
		description   string
	}{
		{
			name: "handles malformed .env file with error",
			setupFunc: func(t *testing.T) func() {
				// Create a malformed .env file that godotenv will reject
				content := "MALFORMED_LINE_WITHOUT_EQUALS\nVALID_VAR=valid_value\n"
				err := os.WriteFile(".env", []byte(content), 0644)
				if err != nil {
					t.Fatalf("Failed to create test .env file: %v", err)
				}
				return func() { os.Remove(".env") }
			},
			expectError:   true,
			errorContains: "unexpected character",
			description:   "Should return error for malformed .env files",
		},
		{
			name: "handles .env file with invalid UTF-8 with error",
			setupFunc: func(t *testing.T) func() {
				// Create a file with invalid UTF-8 bytes that will cause parsing issues
				invalidUTF8 := []byte{0xff, 0xfe, 0xfd}
				content := append([]byte("VALID_VAR=value\n"), invalidUTF8...)
				content = append(content, []byte("\nANOTHER_VAR=another_value\n")...)
				err := os.WriteFile(".env", content, 0644)
				if err != nil {
					t.Fatalf("Failed to create test .env file: %v", err)
				}
				return func() { os.Remove(".env") }
			},
			expectError:   true,
			errorContains: "unexpected character",
			description:   "Should return error for .env files with invalid UTF-8 sequences",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := tt.setupFunc(t)
			defer cleanup()

			err := LoadEnvConfig()

			if tt.expectError {
				if err == nil {
					t.Errorf("LoadEnvConfig() expected error but got nil. %s", tt.description)
					return
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("LoadEnvConfig() error = %v, want error containing %s. %s",
						err, tt.errorContains, tt.description)
				}
			} else {
				if err != nil {
					t.Errorf("LoadEnvConfig() unexpected error: %v. %s", err, tt.description)
				}
			}
		})
	}
}

// TestLoadEnvConfigFileSystemScenarios tests various file system scenarios
func TestLoadEnvConfigFileSystemScenarios(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(t *testing.T) (cleanup func())
		expectError bool
		description string
	}{
		{
			name: "handles directory named .env",
			setupFunc: func(t *testing.T) func() {
				// Create a directory named .env instead of a file
				err := os.Mkdir(".env", 0755)
				if err != nil {
					t.Fatalf("Failed to create .env directory: %v", err)
				}
				return func() { os.RemoveAll(".env") }
			},
			expectError: true, // Should error when trying to read a directory as a file
			description: "Should handle the case where .env is a directory, not a file",
		},
		{
			name: "handles very large .env file",
			setupFunc: func(t *testing.T) func() {
				// Create a large .env file
				var content strings.Builder
				for i := 0; i < 1000; i++ {
					content.WriteString(fmt.Sprintf("VAR_%d=value_%d\n", i, i))
				}
				err := os.WriteFile(".env", []byte(content.String()), 0644)
				if err != nil {
					t.Fatalf("Failed to create large .env file: %v", err)
				}
				return func() { os.Remove(".env") }
			},
			expectError: false,
			description: "Should handle very large .env files",
		},
		{
			name: "handles .env file in subdirectory scenario",
			setupFunc: func(t *testing.T) func() {
				// Create a subdirectory and test from there
				err := os.Mkdir("testdir", 0755)
				if err != nil {
					t.Fatalf("Failed to create test directory: %v", err)
				}

				// Create .env in parent (current) directory
				err = os.WriteFile(".env", []byte("PARENT_VAR=parent_value\n"), 0644)
				if err != nil {
					t.Fatalf("Failed to create .env file: %v", err)
				}

				// Change to subdirectory
				originalDir, _ := os.Getwd()
				err = os.Chdir("testdir")
				if err != nil {
					t.Fatalf("Failed to change to test directory: %v", err)
				}

				return func() {
					os.Chdir(originalDir)
					os.RemoveAll("testdir")
					os.Remove(".env")
				}
			},
			expectError: false,
			description: "Should find .env file in parent directory when run from subdirectory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := tt.setupFunc(t)
			defer cleanup()

			err := LoadEnvConfig()

			if tt.expectError {
				if err == nil {
					t.Errorf("LoadEnvConfig() expected error but got nil. %s", tt.description)
				}
			} else {
				if err != nil {
					t.Errorf("LoadEnvConfig() unexpected error: %v. %s", err, tt.description)
				}
			}
		})
	}
}

// TestGetEnvVarEdgeCases tests edge cases for GetEnvVar function
func TestGetEnvVarEdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		setupFunc    func()
		cleanupFunc  func()
		expected     string
		description  string
	}{
		{
			name:         "handles empty key",
			key:          "",
			defaultValue: "default",
			setupFunc:    func() {},
			cleanupFunc:  func() {},
			expected:     "default",
			description:  "Should return default when key is empty string",
		},
		{
			name:         "handles key with special characters",
			key:          "KEY_WITH_SPECIAL_CHARS!@#$%",
			defaultValue: "default",
			setupFunc: func() {
				os.Setenv("KEY_WITH_SPECIAL_CHARS!@#$%", "special_value")
			},
			cleanupFunc: func() {
				os.Unsetenv("KEY_WITH_SPECIAL_CHARS!@#$%")
			},
			expected:    "special_value",
			description: "Should handle environment variable keys with special characters",
		},
		{
			name:         "handles environment variable set to space",
			key:          "SPACE_VAR",
			defaultValue: "default",
			setupFunc: func() {
				os.Setenv("SPACE_VAR", " ")
			},
			cleanupFunc: func() {
				os.Unsetenv("SPACE_VAR")
			},
			expected:    " ",
			description: "Should return single space when environment variable is set to space",
		},
		{
			name:         "handles environment variable set to zero",
			key:          "ZERO_VAR",
			defaultValue: "default",
			setupFunc: func() {
				os.Setenv("ZERO_VAR", "0")
			},
			cleanupFunc: func() {
				os.Unsetenv("ZERO_VAR")
			},
			expected:    "0",
			description: "Should return '0' when environment variable is set to zero",
		},
		{
			name:         "handles environment variable set to false",
			key:          "FALSE_VAR",
			defaultValue: "default",
			setupFunc: func() {
				os.Setenv("FALSE_VAR", "false")
			},
			cleanupFunc: func() {
				os.Unsetenv("FALSE_VAR")
			},
			expected:    "false",
			description: "Should return 'false' when environment variable is set to false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupFunc()
			defer tt.cleanupFunc()

			result := GetEnvVar(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("GetEnvVar(%s, %s) = %s, want %s. %s",
					tt.key, tt.defaultValue, result, tt.expected, tt.description)
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
			err := ValidateEndpointURL(tt.url)

			if tt.expectError {
				if err == nil {
					t.Errorf("ValidateEndpointURL(%s) expected error but got nil", tt.url)
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("ValidateEndpointURL(%s) error = %v, want error containing %s", tt.url, err, tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateEndpointURL(%s) unexpected error: %v", tt.url, err)
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

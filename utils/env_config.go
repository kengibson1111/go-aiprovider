package utils

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/kengibson1111/go-aiprovider/types"
)

// Environment variable names for API configuration
const (
	// Claude API configuration
	ClaudeAPIKeyEnv      = "CLAUDE_API_KEY"      // Claude API authentication key
	ClaudeModelEnv       = "CLAUDE_MODEL"        // Claude model name to use
	ClaudeAPIEndpointEnv = "CLAUDE_API_ENDPOINT" // Custom Claude API base URL (optional)

	// OpenAI API configuration
	OpenAIAPIKeyEnv      = "OPENAI_API_KEY"      // OpenAI API authentication key
	OpenAIModelEnv       = "OPENAI_MODEL"        // OpenAI model name to use
	OpenAIAPIEndpointEnv = "OPENAI_API_ENDPOINT" // Custom OpenAI API base URL (optional)
)

// ValidateEndpointURL validates that a URL is properly formatted for API endpoints
// Returns error if URL is invalid, nil if valid or empty
func ValidateEndpointURL(endpoint string) error {
	// Empty URLs are allowed (will use defaults)
	if endpoint == "" {
		return nil
	}

	// Parse the URL
	parsedURL, err := url.Parse(endpoint)
	if err != nil {
		return fmt.Errorf("invalid URL format: %v", err)
	}

	// Check for required protocol scheme
	if parsedURL.Scheme == "" {
		return fmt.Errorf("URL must include protocol scheme (http:// or https://)")
	}

	// Only allow http and https schemes
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("URL protocol must be http or https, got: %s", parsedURL.Scheme)
	}

	// Check for hostname
	if parsedURL.Host == "" {
		return fmt.Errorf("URL must include a hostname")
	}

	// Reject URLs with query parameters to prevent configuration errors
	if parsedURL.RawQuery != "" {
		return fmt.Errorf("URL must not contain query parameters, found: ?%s", parsedURL.RawQuery)
	}

	// Additional validation for malformed hostnames
	hostname := parsedURL.Hostname()
	if hostname == "" {
		return fmt.Errorf("URL contains invalid hostname")
	}

	// Check for obviously malformed hostnames
	if strings.Contains(hostname, " ") {
		return fmt.Errorf("hostname cannot contain spaces")
	}

	return nil
}

// LoadEnvConfig loads environment variables from .env file if it exists.
//
// This function is part of the public API and provides a convenient way for
// module consumers to load environment configurations from .env files.
//
// Behavior:
//   - First checks for .env file in the current working directory
//   - If not found, checks for .env file in the parent directory (useful for tests)
//   - If no .env file is found, returns nil (uses system environment variables)
//   - If .env file exists but contains malformed content, returns an error
//
// Usage Example:
//
//	import "github.com/kengibson1111/go-aiprovider/utils"
//
//	// Load environment variables from .env file
//	if err := utils.LoadEnvConfig(); err != nil {
//	    log.Fatalf("Failed to load environment: %v", err)
//	}
//
// Returns:
//   - nil if successful or if .env file doesn't exist
//   - error if .env file exists but cannot be parsed or read
func LoadEnvConfig() error {
	// Check if .env file exists in current directory
	if _, err := os.Stat(".env"); err == nil {
		// Load .env file from current directory
		return godotenv.Load()
	}

	// Check if .env file exists in parent directory (for tests run from subdirectories)
	if _, err := os.Stat("../.env"); err == nil {
		// Load .env file from parent directory
		return godotenv.Load("../.env")
	}

	// .env file doesn't exist in current or parent directory, which is fine - use system environment variables
	return nil
}

// GetEnvVar gets an environment variable with an optional default value.
//
// This function is part of the public API and provides a convenient way for
// module consumers to retrieve environment variables with fallback defaults.
//
// Behavior:
//   - Returns the environment variable value if it exists and is non-empty
//   - Returns the defaultValue if the environment variable doesn't exist or is empty
//   - Preserves whitespace and special characters in both environment values and defaults
//
// Parameters:
//   - key: The name of the environment variable to retrieve
//   - defaultValue: The value to return if the environment variable is not set or empty
//
// Usage Examples:
//
//	import "github.com/kengibson1111/go-aiprovider/utils"
//
//	// Get API key with default
//	apiKey := utils.GetEnvVar("API_KEY", "default-key")
//
//	// Get port with default
//	port := utils.GetEnvVar("PORT", "8080")
//
//	// Get optional configuration
//	debug := utils.GetEnvVar("DEBUG", "false")
//
// Returns:
//   - The environment variable value if set and non-empty
//   - The defaultValue if the environment variable is not set or empty
func GetEnvVar(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// TestConfig provides configuration for integration tests
type TestConfig struct {
	ClaudeAPIKey      string
	ClaudeModel       string
	ClaudeAPIEndpoint string
	OpenAIAPIKey      string
	OpenAIModel       string
	OpenAIAPIEndpoint string
}

// LoadTestConfig loads configuration for integration tests
// Returns error if required environment variables are missing
func LoadTestConfig() (*TestConfig, error) {
	// First load environment variables from .env file if it exists
	if err := LoadEnvConfig(); err != nil {
		return nil, err
	}

	config := &TestConfig{
		ClaudeAPIKey:      os.Getenv(ClaudeAPIKeyEnv),
		ClaudeModel:       GetEnvVar(ClaudeModelEnv, "claude-3-sonnet-20240229"),
		ClaudeAPIEndpoint: os.Getenv(ClaudeAPIEndpointEnv),
		OpenAIAPIKey:      os.Getenv(OpenAIAPIKeyEnv),
		OpenAIModel:       GetEnvVar(OpenAIModelEnv, "gpt-3.5-turbo"),
		OpenAIAPIEndpoint: os.Getenv(OpenAIAPIEndpointEnv),
	}

	return config, nil
}

// CanRunClaudeIntegrationTests checks if Claude integration tests can run
// Returns true if Claude API key is available
func CanRunClaudeIntegrationTests() bool {
	LoadEnvConfig() // Load .env if available, ignore errors
	return os.Getenv(ClaudeAPIKeyEnv) != ""
}

// CanRunOpenAIIntegrationTests checks if OpenAI integration tests can run
// Returns true if OpenAI API key is available
func CanRunOpenAIIntegrationTests() bool {
	LoadEnvConfig() // Load .env if available, ignore errors
	return os.Getenv(OpenAIAPIKeyEnv) != ""
}

// CanRunIntegrationTests checks if any integration tests can run
// Returns true if at least one API key is available
func CanRunIntegrationTests() bool {
	return CanRunClaudeIntegrationTests() || CanRunOpenAIIntegrationTests()
}

// CreateClaudeConfig creates an AIConfig for Claude from TestConfig
// Uses ClaudeAPIEndpoint if valid, otherwise falls back to default endpoint
func (tc *TestConfig) CreateClaudeConfig() *types.AIConfig {
	baseURL := "https://api.anthropic.com"

	// Use custom endpoint if it's valid
	if tc.ClaudeAPIEndpoint != "" && ValidateEndpointURL(tc.ClaudeAPIEndpoint) == nil {
		baseURL = tc.ClaudeAPIEndpoint
	}

	return &types.AIConfig{
		Provider:    "claude",
		APIKey:      tc.ClaudeAPIKey,
		BaseURL:     baseURL,
		Model:       tc.ClaudeModel,
		MaxTokens:   4096,
		Temperature: 0.7,
	}
}

// CreateOpenAIConfig creates an AIConfig for OpenAI from TestConfig
// Uses OpenAIAPIEndpoint if valid, otherwise falls back to default endpoint
func (tc *TestConfig) CreateOpenAIConfig() *types.AIConfig {
	baseURL := "https://api.openai.com"

	// Use custom endpoint if it's valid
	if tc.OpenAIAPIEndpoint != "" && ValidateEndpointURL(tc.OpenAIAPIEndpoint) == nil {
		baseURL = tc.OpenAIAPIEndpoint
	}

	return &types.AIConfig{
		Provider:    "openai",
		APIKey:      tc.OpenAIAPIKey,
		BaseURL:     baseURL,
		Model:       tc.OpenAIModel,
		MaxTokens:   4096,
		Temperature: 0.7,
	}
}

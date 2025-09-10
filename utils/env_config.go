package utils

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/joho/godotenv"
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

// validateEndpointURL validates that a URL is properly formatted for API endpoints
// Returns error if URL is invalid, nil if valid or empty
func validateEndpointURL(endpoint string) error {
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

// LoadEnvConfig loads environment variables from .env file if it exists
// Returns nil if successful or if .env file doesn't exist
func LoadEnvConfig() error {
	// Check if .env file exists
	if _, err := os.Stat(".env"); os.IsNotExist(err) {
		// .env file doesn't exist, which is fine - use system environment variables
		return nil
	}

	// Load .env file
	return godotenv.Load()
}

// GetEnvVar gets an environment variable with an optional default value
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

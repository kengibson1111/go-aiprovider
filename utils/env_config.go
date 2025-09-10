package utils

import (
	"os"

	"github.com/joho/godotenv"
)

// Environment variable names
const (
	ClaudeAPIKeyEnv = "CLAUDE_API_KEY"
	ClaudeModelEnv  = "CLAUDE_MODEL"
	OpenAIAPIKeyEnv = "OPENAI_API_KEY"
	OpenAIModelEnv  = "OPENAI_MODEL"
)

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

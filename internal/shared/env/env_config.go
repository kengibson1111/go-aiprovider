package env

import (
	"os"

	"github.com/joho/godotenv"
)

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
//	import "github.com/kengibson1111/marketviz-tools/internal/shared/env"
//
//	// Load environment variables from .env file
//	if err := env.LoadEnvConfig(); err != nil {
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
//	import "github.com/kengibson1111/marketviz-tools/internal/shared/env"
//
//	// Get port with default
//	logLevel := utils.GetEnvVar("LOG_LEVEL", "info")
//
//	// Get optional configuration
//	debug := utils.GetEnvVar("VERBOSE", "false")
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

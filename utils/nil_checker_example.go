package utils

import "fmt"

// ExampleCheckNilValue demonstrates how to use the CheckNilValue utility function
func ExampleCheckNilValue() {
	logger := NewLogger("example")

	// Example 1: Check a required pointer value
	var userPtr *string
	if err := CheckNilValue("user_pointer", userPtr, true, logger); err != nil {
		fmt.Printf("Error: %v\n", err)
		// Output: Error: required value 'user_pointer' is nil
	}

	// Example 2: Check an optional value
	var configPtr *string
	if err := CheckNilValue("config_pointer", configPtr, false, logger); err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Println("Optional value check passed")
	}

	// Example 3: Check a valid pointer
	validValue := "hello world"
	if err := CheckNilValue("valid_string", &validValue, true, logger); err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Println("Valid pointer check passed")
	}

	// Example 4: Invalid name parameter
	if err := CheckNilValue("", "some_value", false, logger); err != nil {
		fmt.Printf("Error: %v\n", err)
		// Output: Error: name parameter must be a valid non-empty string
	}
}

// Example usage of CheckNilValue in a typical function
func ProcessUserData(userID *string, config *map[string]string, logger *Logger) error {
	// Check required user ID
	if err := CheckNilValue("userID", userID, true, logger); err != nil {
		return err
	}

	// Check optional config (not required)
	if err := CheckNilValue("config", config, false, logger); err != nil {
		return err
	}

	// Process the data...
	logger.Info("Processing user data for ID: %s", *userID)

	if config != nil {
		logger.Debug("Using provided configuration with %d settings", len(*config))
	} else {
		logger.Debug("Using default configuration")
	}

	return nil
}

// Example with struct fields
type APIClientTemp struct {
	BaseURL *string
	APIKey  *string
	Logger  *Logger
}

func (c *APIClientTemp) Validate() error {
	// Validate required fields
	if err := CheckNilValue("BaseURL", c.BaseURL, true, c.Logger); err != nil {
		return err
	}

	if err := CheckNilValue("APIKey", c.APIKey, true, c.Logger); err != nil {
		return err
	}

	if err := CheckNilValue("Logger", c.Logger, true, c.Logger); err != nil {
		// Note: This would fail if Logger is nil, but demonstrates the pattern
		return err
	}

	return nil
}

// Example usage of CheckStringValue in a typical function
func ProcessConfiguration(configName string, apiEndpoint string, logger *Logger) error {
	// Check required configuration name
	if err := CheckStringValue("configName", configName, true, logger); err != nil {
		return err
	}

	// Check optional API endpoint
	if err := CheckStringValue("apiEndpoint", apiEndpoint, false, logger); err != nil {
		return err
	}

	logger.Info("Processing configuration: %s", configName)

	if apiEndpoint != "" {
		logger.Debug("Using custom API endpoint: %s", apiEndpoint)
	} else {
		logger.Debug("Using default API endpoint")
	}

	return nil
}

// Example with string pointers
type DatabaseConfig struct {
	Host     *string
	Database *string
	Username *string
	Password *string // optional
	Logger   *Logger
}

func (c *DatabaseConfig) Validate() error {
	// Validate required string fields
	if err := CheckStringPointerValue("Host", c.Host, true, c.Logger); err != nil {
		return err
	}

	if err := CheckStringPointerValue("Database", c.Database, true, c.Logger); err != nil {
		return err
	}

	if err := CheckStringPointerValue("Username", c.Username, true, c.Logger); err != nil {
		return err
	}

	// Password is optional
	if err := CheckStringPointerValue("Password", c.Password, false, c.Logger); err != nil {
		return err
	}

	return nil
}

// Example showing combined nil and string validation
func SetupAPIClient(baseURL *string, apiKey *string, timeout *string, logger *Logger) error {
	// Check required pointer values first (nil check)
	if err := CheckNilValue("baseURL", baseURL, true, logger); err != nil {
		return err
	}

	if err := CheckNilValue("apiKey", apiKey, true, logger); err != nil {
		return err
	}

	// Then check if the strings they point to are empty
	if err := CheckStringPointerValue("baseURL", baseURL, true, logger); err != nil {
		return err
	}

	if err := CheckStringPointerValue("apiKey", apiKey, true, logger); err != nil {
		return err
	}

	// Timeout is optional - can be nil or empty
	if err := CheckStringPointerValue("timeout", timeout, false, logger); err != nil {
		return err
	}

	logger.Info("API client configured successfully")
	return nil
}

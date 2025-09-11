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

// ExampleCheckArrayValue demonstrates how to use the CheckArrayValue utility function
func ExampleCheckArrayValue() {
	logger := NewLogger("array_example")

	// Example 1: Check a required slice that is nil
	var userIDs []string
	if err := CheckArrayValue("user_ids", userIDs, true, logger); err != nil {
		fmt.Printf("Error: %v\n", err)
		// Output: Error: required value 'user_ids' is nil
	}

	// Example 2: Check a required slice that is empty
	emptySlice := []string{}
	if err := CheckArrayValue("empty_slice", emptySlice, true, logger); err != nil {
		fmt.Printf("Error: %v\n", err)
		// Output: Error: required array 'empty_slice' is empty
	}

	// Example 3: Check an optional empty slice
	if err := CheckArrayValue("optional_slice", emptySlice, false, logger); err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Println("Optional empty slice check passed")
	}

	// Example 4: Check a valid slice
	validSlice := []string{"item1", "item2", "item3"}
	if err := CheckArrayValue("valid_slice", validSlice, true, logger); err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Println("Valid slice check passed")
	}

	// Example 5: Check different array types
	intArray := [3]int{1, 2, 3}
	if err := CheckArrayValue("int_array", intArray, true, logger); err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Println("Valid array check passed")
	}

	// Example 6: Invalid type (not an array or slice)
	if err := CheckArrayValue("not_array", "string_value", true, logger); err != nil {
		fmt.Printf("Error: %v\n", err)
		// Output: Error: value 'not_array' is not an array or slice, got string
	}
}

// Example usage of CheckArrayValue in a typical function
func ProcessBatchData(userIDs []string, tags []string, logger *Logger) error {
	// Check required user IDs
	if err := CheckArrayValue("userIDs", userIDs, true, logger); err != nil {
		return err
	}

	// Check optional tags (can be empty or nil)
	if err := CheckArrayValue("tags", tags, false, logger); err != nil {
		return err
	}

	logger.Info("Processing batch data for %d users", len(userIDs))

	if len(tags) > 0 {
		logger.Debug("Using %d tags for filtering", len(tags))
	} else {
		logger.Debug("No tags provided, processing all data")
	}

	return nil
}

// Example with struct containing arrays
type BatchProcessingConfig struct {
	RequiredFields []string
	OptionalFields []string
	Filters        []string
	Logger         *Logger
}

func (c *BatchProcessingConfig) Validate() error {
	// Validate required array fields
	if err := CheckArrayValue("RequiredFields", c.RequiredFields, true, c.Logger); err != nil {
		return err
	}

	// Optional fields can be empty
	if err := CheckArrayValue("OptionalFields", c.OptionalFields, false, c.Logger); err != nil {
		return err
	}

	// Filters are optional
	if err := CheckArrayValue("Filters", c.Filters, false, c.Logger); err != nil {
		return err
	}

	return nil
}

// Example showing different array types
func ProcessMultiTypeArrays(
	stringSlice []string,
	intSlice []int,
	byteSlice []byte,
	interfaceSlice []interface{},
	logger *Logger,
) error {
	// Check all different types of arrays/slices
	if err := CheckArrayValue("stringSlice", stringSlice, true, logger); err != nil {
		return err
	}

	if err := CheckArrayValue("intSlice", intSlice, true, logger); err != nil {
		return err
	}

	if err := CheckArrayValue("byteSlice", byteSlice, false, logger); err != nil {
		return err
	}

	if err := CheckArrayValue("interfaceSlice", interfaceSlice, false, logger); err != nil {
		return err
	}

	logger.Info("All array validations passed")
	return nil
}

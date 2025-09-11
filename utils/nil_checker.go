package utils

import (
	"errors"
	"fmt"
	"reflect"
)

// CheckNilValue logs a debug statement and checks if a value is nil.
// If the value is nil and required is true, returns an error.
//
// Parameters:
//   - name: A valid string used for logging (must not be empty)
//   - value: The value to check for nil (typically a pointer)
//   - required: If true, returns an error when value is nil
//   - logger: Logger instance for debug output
//
// Returns:
//   - error: nil if validation passes, error if name is invalid or value is nil when required
func CheckNilValue(name string, value interface{}, required bool, logger *Logger) error {
	// Validate name parameter
	if name == "" {
		return errors.New("name parameter must be a valid non-empty string")
	}

	// Log debug statement
	logger.Debug("Checking value for '%s', required: %t", name, required)

	// Check if value is nil
	if value == nil {
		if required {
			return fmt.Errorf("required value '%s' is nil", name)
		}
		logger.Debug("Value '%s' is nil but not required", name)
		return nil
	}

	// Handle pointer types and interface values that might contain nil
	rv := reflect.ValueOf(value)
	if rv.Kind() == reflect.Ptr || rv.Kind() == reflect.Interface || rv.Kind() == reflect.Slice || rv.Kind() == reflect.Map || rv.Kind() == reflect.Chan || rv.Kind() == reflect.Func {
		if rv.IsNil() {
			if required {
				return fmt.Errorf("required value '%s' is nil", name)
			}
			logger.Debug("Value '%s' is nil but not required", name)
			return nil
		}
	}

	logger.Debug("Value '%s' is not nil", name)
	return nil
}

// CheckStringValue logs a debug statement and checks if a string value is empty.
// If the string is empty and required is true, returns an error.
//
// Parameters:
//   - name: A valid string used for logging (must not be empty)
//   - value: The string value to check for emptiness
//   - required: If true, returns an error when value is empty
//   - logger: Logger instance for debug output
//
// Returns:
//   - error: nil if validation passes, error if name is invalid or string is empty when required
func CheckStringValue(name string, value string, required bool, logger *Logger) error {
	// Validate name parameter
	if name == "" {
		return errors.New("name parameter must be a valid non-empty string")
	}

	// Log debug statement
	logger.Debug("Checking string value for '%s', required: %t, length: %d", name, required, len(value))

	// Check if string is empty
	if value == "" {
		if required {
			return fmt.Errorf("required string '%s' is empty", name)
		}
		logger.Debug("String '%s' is empty but not required", name)
		return nil
	}

	logger.Debug("String '%s' is not empty (value: '%s')", name, value)
	return nil
}

// CheckStringPointerValue logs a debug statement and checks if a string pointer is nil or points to an empty string.
// If the pointer is nil or points to empty string and required is true, returns an error.
//
// Parameters:
//   - name: A valid string used for logging (must not be empty)
//   - value: The string pointer to check
//   - required: If true, returns an error when value is nil or points to empty string
//   - logger: Logger instance for debug output
//
// Returns:
//   - error: nil if validation passes, error if name is invalid or string pointer is nil/empty when required
func CheckStringPointerValue(name string, value *string, required bool, logger *Logger) error {
	// Validate name parameter
	if name == "" {
		return errors.New("name parameter must be a valid non-empty string")
	}

	// First check if pointer is nil
	if err := CheckNilValue(name, value, required, logger); err != nil {
		return err
	}

	// If pointer is not nil, check the string value it points to
	if value != nil {
		return CheckStringValue(name, *value, required, logger)
	}

	// If we get here, pointer was nil but not required
	return nil
}

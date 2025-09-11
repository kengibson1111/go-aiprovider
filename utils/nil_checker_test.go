package utils

import (
	"strings"
	"testing"
)

func TestCheckNilValue(t *testing.T) {
	logger := NewLogger("test")

	tests := []struct {
		name        string
		valueName   string
		value       interface{}
		required    bool
		expectError bool
		errorMsg    string
	}{
		{
			name:        "empty name parameter",
			valueName:   "",
			value:       "test",
			required:    false,
			expectError: true,
			errorMsg:    "name parameter must be a valid non-empty string",
		},
		{
			name:        "nil value not required",
			valueName:   "test_value",
			value:       nil,
			required:    false,
			expectError: false,
		},
		{
			name:        "nil value required",
			valueName:   "test_value",
			value:       nil,
			required:    true,
			expectError: true,
			errorMsg:    "required value 'test_value' is nil",
		},
		{
			name:        "nil pointer not required",
			valueName:   "test_pointer",
			value:       (*string)(nil),
			required:    false,
			expectError: false,
		},
		{
			name:        "nil pointer required",
			valueName:   "test_pointer",
			value:       (*string)(nil),
			required:    true,
			expectError: true,
			errorMsg:    "required value 'test_pointer' is nil",
		},
		{
			name:        "valid pointer not required",
			valueName:   "valid_pointer",
			value:       &[]string{"test"},
			required:    false,
			expectError: false,
		},
		{
			name:        "valid pointer required",
			valueName:   "valid_pointer",
			value:       &[]string{"test"},
			required:    true,
			expectError: false,
		},
		{
			name:        "non-nil value not required",
			valueName:   "string_value",
			value:       "hello",
			required:    false,
			expectError: false,
		},
		{
			name:        "non-nil value required",
			valueName:   "string_value",
			value:       "hello",
			required:    true,
			expectError: false,
		},
		{
			name:        "nil slice required",
			valueName:   "slice_value",
			value:       ([]string)(nil),
			required:    true,
			expectError: true,
			errorMsg:    "required value 'slice_value' is nil",
		},
		{
			name:        "nil map required",
			valueName:   "map_value",
			value:       (map[string]string)(nil),
			required:    true,
			expectError: true,
			errorMsg:    "required value 'map_value' is nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckNilValue(tt.valueName, tt.value, tt.required, logger)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got nil")
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error message to contain '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestCheckNilValueWithDifferentTypes(t *testing.T) {
	logger := NewLogger("type_test")

	// Test with different pointer types
	var intPtr *int
	var stringPtr *string
	var structPtr *struct{ Name string }

	tests := []struct {
		name     string
		value    interface{}
		required bool
		isNil    bool
	}{
		{"nil int pointer", intPtr, true, true},
		{"nil string pointer", stringPtr, true, true},
		{"nil struct pointer", structPtr, true, true},
		{"valid int pointer", &[]int{42}[0], true, false},
		{"valid string pointer", &[]string{"test"}[0], true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckNilValue(tt.name, tt.value, tt.required, logger)

			if tt.isNil && tt.required {
				if err == nil {
					t.Errorf("Expected error for nil required value")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}
func TestCheckStringValue(t *testing.T) {
	logger := NewLogger("string_test")

	tests := []struct {
		name        string
		valueName   string
		value       string
		required    bool
		expectError bool
		errorMsg    string
	}{
		{
			name:        "empty name parameter",
			valueName:   "",
			value:       "test",
			required:    false,
			expectError: true,
			errorMsg:    "name parameter must be a valid non-empty string",
		},
		{
			name:        "empty string not required",
			valueName:   "test_string",
			value:       "",
			required:    false,
			expectError: false,
		},
		{
			name:        "empty string required",
			valueName:   "test_string",
			value:       "",
			required:    true,
			expectError: true,
			errorMsg:    "required string 'test_string' is empty",
		},
		{
			name:        "non-empty string not required",
			valueName:   "test_string",
			value:       "hello",
			required:    false,
			expectError: false,
		},
		{
			name:        "non-empty string required",
			valueName:   "test_string",
			value:       "hello",
			required:    true,
			expectError: false,
		},
		{
			name:        "whitespace only string required",
			valueName:   "whitespace_string",
			value:       "   ",
			required:    true,
			expectError: false, // whitespace is not considered empty
		},
		{
			name:        "single character string required",
			valueName:   "single_char",
			value:       "a",
			required:    true,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckStringValue(tt.valueName, tt.value, tt.required, logger)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got nil")
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error message to contain '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestCheckStringPointerValue(t *testing.T) {
	logger := NewLogger("string_ptr_test")

	emptyString := ""
	validString := "hello world"
	whitespaceString := "   "

	tests := []struct {
		name        string
		valueName   string
		value       *string
		required    bool
		expectError bool
		errorMsg    string
	}{
		{
			name:        "empty name parameter",
			valueName:   "",
			value:       &validString,
			required:    false,
			expectError: true,
			errorMsg:    "name parameter must be a valid non-empty string",
		},
		{
			name:        "nil pointer not required",
			valueName:   "test_ptr",
			value:       nil,
			required:    false,
			expectError: false,
		},
		{
			name:        "nil pointer required",
			valueName:   "test_ptr",
			value:       nil,
			required:    true,
			expectError: true,
			errorMsg:    "required value 'test_ptr' is nil",
		},
		{
			name:        "pointer to empty string not required",
			valueName:   "empty_ptr",
			value:       &emptyString,
			required:    false,
			expectError: false,
		},
		{
			name:        "pointer to empty string required",
			valueName:   "empty_ptr",
			value:       &emptyString,
			required:    true,
			expectError: true,
			errorMsg:    "required string 'empty_ptr' is empty",
		},
		{
			name:        "pointer to valid string not required",
			valueName:   "valid_ptr",
			value:       &validString,
			required:    false,
			expectError: false,
		},
		{
			name:        "pointer to valid string required",
			valueName:   "valid_ptr",
			value:       &validString,
			required:    true,
			expectError: false,
		},
		{
			name:        "pointer to whitespace string required",
			valueName:   "whitespace_ptr",
			value:       &whitespaceString,
			required:    true,
			expectError: false, // whitespace is not considered empty
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckStringPointerValue(tt.valueName, tt.value, tt.required, logger)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got nil")
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error message to contain '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}
func TestCheckNilValueWithArrays(t *testing.T) {
	logger := NewLogger("array_test")

	// Test with arrays (which cannot be nil)
	var intArray [3]int
	var stringArray [2]string

	tests := []struct {
		name     string
		value    interface{}
		required bool
		isNil    bool
	}{
		{"int array", intArray, true, false},
		{"string array", stringArray, true, false},
		{"array pointer", &intArray, true, false}, // pointer to array is not nil
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This should not panic and should pass validation
			err := CheckNilValue(tt.name, tt.value, tt.required, logger)

			if tt.isNil && tt.required {
				if err == nil {
					t.Errorf("Expected error for nil required value")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestCheckArrayValue(t *testing.T) {
	logger := NewLogger("array_test")

	tests := []struct {
		name        string
		valueName   string
		value       any
		required    bool
		expectError bool
		errorMsg    string
	}{
		{
			name:        "empty name parameter",
			valueName:   "",
			value:       []string{"test"},
			required:    false,
			expectError: true,
			errorMsg:    "name parameter must be a valid non-empty string",
		},
		{
			name:        "nil slice not required",
			valueName:   "test_slice",
			value:       ([]string)(nil),
			required:    false,
			expectError: false,
		},
		{
			name:        "nil slice required",
			valueName:   "test_slice",
			value:       ([]string)(nil),
			required:    true,
			expectError: true,
			errorMsg:    "required value 'test_slice' is nil",
		},
		{
			name:        "empty slice not required",
			valueName:   "empty_slice",
			value:       []string{},
			required:    false,
			expectError: false,
		},
		{
			name:        "empty slice required",
			valueName:   "empty_slice",
			value:       []string{},
			required:    true,
			expectError: true,
			errorMsg:    "required array 'empty_slice' is empty",
		},
		{
			name:        "non-empty slice not required",
			valueName:   "valid_slice",
			value:       []string{"hello", "world"},
			required:    false,
			expectError: false,
		},
		{
			name:        "non-empty slice required",
			valueName:   "valid_slice",
			value:       []string{"hello", "world"},
			required:    true,
			expectError: false,
		},
		{
			name:        "empty int slice required",
			valueName:   "empty_int_slice",
			value:       []int{},
			required:    true,
			expectError: true,
			errorMsg:    "required array 'empty_int_slice' is empty",
		},
		{
			name:        "non-empty int slice required",
			valueName:   "valid_int_slice",
			value:       []int{1, 2, 3},
			required:    true,
			expectError: false,
		},
		{
			name:        "empty array required",
			valueName:   "empty_array",
			value:       [0]string{},
			required:    true,
			expectError: true,
			errorMsg:    "required array 'empty_array' is empty",
		},
		{
			name:        "non-empty array required",
			valueName:   "valid_array",
			value:       [2]string{"hello", "world"},
			required:    true,
			expectError: false,
		},
		{
			name:        "non-array type",
			valueName:   "not_array",
			value:       "string_value",
			required:    true,
			expectError: true,
			errorMsg:    "value 'not_array' is not an array or slice",
		},
		{
			name:        "map type",
			valueName:   "map_value",
			value:       map[string]int{"key": 1},
			required:    true,
			expectError: true,
			errorMsg:    "value 'map_value' is not an array or slice",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckArrayValue(tt.valueName, tt.value, tt.required, logger)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got nil")
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error message to contain '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestCheckArrayValueWithDifferentTypes(t *testing.T) {
	logger := NewLogger("array_type_test")

	tests := []struct {
		name     string
		value    any
		required bool
		isEmpty  bool
	}{
		{"string slice", []string{"a", "b"}, true, false},
		{"int slice", []int{1, 2, 3}, true, false},
		{"empty string slice", []string{}, true, true},
		{"empty int slice", []int{}, true, true},
		{"byte slice", []byte("hello"), true, false},
		{"empty byte slice", []byte{}, true, true},
		{"interface slice", []interface{}{1, "hello", true}, true, false},
		{"empty interface slice", []interface{}{}, true, true},
		{"string array", [3]string{"a", "b", "c"}, true, false},
		{"int array", [2]int{1, 2}, true, false},
		{"empty string array", [0]string{}, true, true},
		{"empty int array", [0]int{}, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckArrayValue(tt.name, tt.value, tt.required, logger)

			if tt.isEmpty && tt.required {
				if err == nil {
					t.Errorf("Expected error for empty required array")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

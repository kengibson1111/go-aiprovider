package utils

import (
	"strings"
	"testing"
)

func TestSubstituteVariables(t *testing.T) {
	tests := []struct {
		name        string
		template    string
		variables   string
		expected    string
		expectError bool
		errorType   error
	}{
		// Valid substitution scenarios
		{
			name:        "Single variable substitution",
			template:    "Hello {{name}}!",
			variables:   `{"name": "Alice"}`,
			expected:    "Hello Alice!",
			expectError: false,
		},
		{
			name:        "Multiple variables in single template",
			template:    "Hello {{name}}, please review this {{language}} code.",
			variables:   `{"name": "Alice", "language": "Go"}`,
			expected:    "Hello Alice, please review this Go code.",
			expectError: false,
		},
		{
			name:        "Multiple occurrences of same variable",
			template:    "{{name}} said hello to {{name}} again.",
			variables:   `{"name": "Bob"}`,
			expected:    "Bob said hello to Bob again.",
			expectError: false,
		},
		{
			name:        "No variables in template",
			template:    "This is a plain template with no variables.",
			variables:   `{"unused": "value"}`,
			expected:    "This is a plain template with no variables.",
			expectError: false,
		},
		{
			name:        "Missing variables remain unchanged",
			template:    "Hello {{name}}, your {{unknown}} is ready.",
			variables:   `{"name": "Charlie"}`,
			expected:    "Hello Charlie, your {{unknown}} is ready.",
			expectError: false,
		},
		{
			name:        "Empty variables JSON",
			template:    "Hello {{name}}!",
			variables:   `{}`,
			expected:    "Hello {{name}}!",
			expectError: false,
		},
		{
			name:        "Empty variables string",
			template:    "Hello {{name}}!",
			variables:   "",
			expected:    "Hello {{name}}!",
			expectError: false,
		},
		{
			name:        "Null variables string",
			template:    "Hello {{name}}!",
			variables:   "null",
			expected:    "Hello {{name}}!",
			expectError: false,
		},
		{
			name:        "Variable names with underscores and hyphens",
			template:    "{{user_name}} has {{task-count}} tasks.",
			variables:   `{"user_name": "David", "task-count": "5"}`,
			expected:    "David has 5 tasks.",
			expectError: false,
		},
		{
			name:        "Variable values with special characters",
			template:    "Message: {{message}}",
			variables:   `{"message": "Hello, world! @#$%^&*()"}`,
			expected:    "Message: Hello, world! @#$%^&*()",
			expectError: false,
		},
		{
			name:        "Non-string JSON values converted to strings",
			template:    "Count: {{count}}, Active: {{active}}, Rate: {{rate}}",
			variables:   `{"count": 42, "active": true, "rate": 3.14}`,
			expected:    "Count: 42, Active: true, Rate: 3.14",
			expectError: false,
		},
		{
			name:        "Null JSON values become empty strings",
			template:    "Value: '{{value}}'",
			variables:   `{"value": null}`,
			expected:    "Value: ''",
			expectError: false,
		},

		// Error scenarios
		{
			name:        "Empty template",
			template:    "",
			variables:   `{"name": "Alice"}`,
			expected:    "",
			expectError: true,
			errorType:   ErrEmptyTemplate,
		},
		{
			name:        "Malformed JSON - missing quotes",
			template:    "Hello {{name}}!",
			variables:   `{name: "Alice"}`,
			expected:    "",
			expectError: true,
			errorType:   ErrInvalidJSON,
		},
		{
			name:        "Malformed JSON - trailing comma",
			template:    "Hello {{name}}!",
			variables:   `{"name": "Alice",}`,
			expected:    "",
			expectError: true,
			errorType:   ErrInvalidJSON,
		},
		{
			name:        "Malformed JSON - unclosed brace",
			template:    "Hello {{name}}!",
			variables:   `{"name": "Alice"`,
			expected:    "",
			expectError: true,
			errorType:   ErrInvalidJSON,
		},
		{
			name:        "Invalid JSON - not an object",
			template:    "Hello {{name}}!",
			variables:   `["not", "an", "object"]`,
			expected:    "",
			expectError: true,
			errorType:   ErrInvalidJSON,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SubstituteVariables(tt.template, tt.variables)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if tt.errorType != nil && !strings.Contains(err.Error(), tt.errorType.Error()) {
					t.Errorf("Expected error type %v, got %v", tt.errorType, err)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestSubstituteVariablesEdgeCases(t *testing.T) {
	// Test edge cases that might cause issues

	t.Run("Large template with many variables", func(t *testing.T) {
		// Create a template with many variables
		template := strings.Repeat("{{var}} ", 100)
		variables := `{"var": "test"}`
		expected := strings.Repeat("test ", 100)

		result, err := SubstituteVariables(template, variables)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if result != expected {
			t.Errorf("Large template substitution failed")
		}
	})

	t.Run("Nested braces in template", func(t *testing.T) {
		// Variables with nested braces should not be matched
		template := "This {{{invalid}}} should not match {{valid}}"
		variables := `{"valid": "works", "invalid": "broken"}`
		expected := "This {{{invalid}}} should not match works"

		result, err := SubstituteVariables(template, variables)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if result != expected {
			t.Errorf("Expected %q, got %q", expected, result)
		}
	})

	t.Run("Variable names with numbers", func(t *testing.T) {
		template := "{{var1}} and {{var2}} and {{123var}}"
		variables := `{"var1": "first", "var2": "second", "123var": "third"}`
		expected := "first and second and third"

		result, err := SubstituteVariables(template, variables)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if result != expected {
			t.Errorf("Expected %q, got %q", expected, result)
		}
	})

	t.Run("Empty variable name should not match", func(t *testing.T) {
		template := "This {{}} should not be replaced"
		variables := `{"": "empty"}`
		expected := "This {{}} should not be replaced"

		result, err := SubstituteVariables(template, variables)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if result != expected {
			t.Errorf("Expected %q, got %q", expected, result)
		}
	})

	t.Run("Variable with spaces should not match", func(t *testing.T) {
		template := "This {{var name}} should not be replaced"
		variables := `{"var name": "spaced"}`
		expected := "This {{var name}} should not be replaced"

		result, err := SubstituteVariables(template, variables)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if result != expected {
			t.Errorf("Expected %q, got %q", expected, result)
		}
	})
}

func TestVariablePatternRegex(t *testing.T) {
	// Test the regex pattern directly
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "Single variable",
			input:    "{{name}}",
			expected: []string{"name"},
		},
		{
			name:     "Multiple variables",
			input:    "{{first}} and {{second}}",
			expected: []string{"first", "second"},
		},
		{
			name:     "Variable with underscores",
			input:    "{{user_name}}",
			expected: []string{"user_name"},
		},
		{
			name:     "Variable with hyphens",
			input:    "{{task-id}}",
			expected: []string{"task-id"},
		},
		{
			name:     "Variable with numbers",
			input:    "{{var123}}",
			expected: []string{"var123"},
		},
		{
			name:     "No variables",
			input:    "No variables here",
			expected: []string{},
		},
		{
			name:     "Invalid variable with spaces",
			input:    "{{var name}}",
			expected: []string{},
		},
		{
			name:     "Invalid variable with special chars",
			input:    "{{var@name}}",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := variablePattern.FindAllStringSubmatch(tt.input, -1)
			var result []string
			for _, match := range matches {
				if len(match) > 1 {
					result = append(result, match[1])
				}
			}

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d matches, got %d", len(tt.expected), len(result))
				return
			}

			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("Expected match %d to be %q, got %q", i, expected, result[i])
				}
			}
		})
	}
}

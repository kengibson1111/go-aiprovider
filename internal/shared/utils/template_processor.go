package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
)

// Template processing errors define specific error conditions for variable substitution
var (
	// ErrInvalidJSON is returned when the variables JSON string cannot be parsed
	ErrInvalidJSON = errors.New("invalid JSON format in variables")

	// ErrEmptyTemplate is returned when an empty template string is provided
	ErrEmptyTemplate = errors.New("template cannot be empty")
)

// variablePattern is a compiled regular expression that matches variable placeholders
// in the format {{variable_name}}. The pattern captures:
//   - Opening double braces: {{
//   - Variable name: ([a-zA-Z0-9_-]+) - letters, numbers, underscores, hyphens
//   - Closing double braces: }}
//
// The captured group (parentheses) extracts just the variable name without braces,
// enabling easy replacement of the entire {{variable_name}} with its value.
var variablePattern = regexp.MustCompile(`\{\{([a-zA-Z0-9_-]+)\}\}`)

// SubstituteVariables replaces variables in template with values from JSON string.
//
// This function enables prompt template functionality by substituting placeholder variables
// with actual values. Variables in the template must use the {{variable_name}} format.
//
// Variable Format:
//   - Variables must be enclosed in double curly braces: {{variable_name}}
//   - Variable names can contain letters, numbers, underscores, and hyphens
//   - Variable names are case-sensitive
//   - Nested braces (e.g., {{{variable}}}) are not supported and will be ignored
//
// Variables JSON Format:
//   - Must be a valid JSON object with string keys matching variable names
//   - Values can be strings, numbers, booleans, or null (all converted to strings)
//   - Empty object {} is valid (no substitutions performed)
//   - null or empty string results in no substitutions
//
// Behavior:
//   - Variables with matching JSON keys are replaced with their values
//   - Variables without matching keys remain unchanged in the template
//   - All JSON values are converted to their string representation
//   - Processing is done in reverse order to handle overlapping replacements correctly
//
// Parameters:
//   - template: The template string containing variables in {{variable_name}} format
//   - variablesJSON: JSON string containing variable name-value pairs
//
// Returns:
//   - Processed template string with variables substituted
//   - Error if template is empty or JSON is malformed
//
// Example:
//
//	template := "Hello {{name}}, welcome to {{platform}}!"
//	variables := `{"name": "Alice", "platform": "Go AI Provider"}`
//	result, err := SubstituteVariables(template, variables)
//	// result: "Hello Alice, welcome to Go AI Provider!"
func SubstituteVariables(template string, variablesJSON string) (string, error) {
	// Handle empty template
	if template == "" {
		return "", ErrEmptyTemplate
	}

	// Handle empty or null variables JSON - return template unchanged
	if variablesJSON == "" || variablesJSON == "null" {
		return template, nil
	}

	// Parse variables JSON
	var variables map[string]any
	if err := json.Unmarshal([]byte(variablesJSON), &variables); err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidJSON, err)
	}

	// If no variables provided, return template unchanged
	if len(variables) == 0 {
		return template, nil
	}

	// Find all variable matches in template and their positions
	result := template
	matches := variablePattern.FindAllStringSubmatchIndex(template, -1)

	// Process matches in reverse order to avoid position shifts during replacement
	for i := len(matches) - 1; i >= 0; i-- {
		match := matches[i]
		if len(match) < 4 {
			continue
		}

		// Extract the full match and variable name
		fullMatchStart := match[0]
		fullMatchEnd := match[1]
		variableNameStart := match[2]
		variableNameEnd := match[3]

		variableName := template[variableNameStart:variableNameEnd] // Captured group: variable_name

		// Check for nested braces - skip if there are extra braces around our match
		if fullMatchStart > 0 && template[fullMatchStart-1] == '{' {
			continue // Skip {{{variable}}} patterns
		}
		if fullMatchEnd < len(template) && template[fullMatchEnd] == '}' {
			continue // Skip {{variable}}} patterns
		}

		// Check if variable exists in provided values
		if value, exists := variables[variableName]; exists {
			// Convert value to string
			var stringValue string
			switch v := value.(type) {
			case string:
				stringValue = v
			case nil:
				stringValue = ""
			default:
				// Convert other types to string representation
				stringValue = fmt.Sprintf("%v", v)
			}

			// Replace this specific occurrence
			result = result[:fullMatchStart] + stringValue + result[fullMatchEnd:]
		}
		// If variable doesn't exist in values, leave placeholder unchanged
	}

	return result, nil
}

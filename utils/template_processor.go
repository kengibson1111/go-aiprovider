package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
)

// Template processing errors
var (
	ErrInvalidJSON   = errors.New("invalid JSON format in variables")
	ErrEmptyTemplate = errors.New("template cannot be empty")
)

// variablePattern matches {{variable_name}} format
// Variable names can contain letters, numbers, underscores, and hyphens
var variablePattern = regexp.MustCompile(`\{\{([a-zA-Z0-9_-]+)\}\}`)

// SubstituteVariables replaces variables in template with values from JSON string
// Variables in template should be in format {{variable_name}}
// Variables JSON should be a valid JSON object with string keys and values
// Returns processed template string or error if JSON is malformed
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

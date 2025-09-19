package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/kengibson1111/go-aiprovider/client"
	"github.com/kengibson1111/go-aiprovider/types"
	"github.com/kengibson1111/go-aiprovider/utils"
)

// Example demonstrating prompt template variable substitution functionality
func main() {
	fmt.Println("=== Prompt Template Variables Example ===")
	fmt.Println("")

	// Example 1: Basic Variable Substitution Utility
	basicVariableSubstitutionExample()

	// Example 2: Using CallWithPromptAndVariables with OpenAI
	openAIVariableExample()

	// Example 3: Using CallWithPromptAndVariables with Claude
	claudeVariableExample()

	// Example 4: Advanced Variable Patterns
	advancedVariableExample()

	// Example 5: Error Handling Examples
	errorHandlingExample()
}

// basicVariableSubstitutionExample demonstrates the standalone utility function
func basicVariableSubstitutionExample() {
	fmt.Println("1. Basic Variable Substitution Utility")
	fmt.Println("=====================================")

	// Template with variables in {{variable_name}} format
	template := "Hello {{name}}, please review this {{language}} code for {{task_type}}."

	// Variables as JSON string
	variablesJSON := `{
		"name": "Alice",
		"language": "Go",
		"task_type": "performance optimization"
	}`

	// Substitute variables
	result, err := utils.SubstituteVariables(template, variablesJSON)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Printf("Template: %s\n", template)
	fmt.Printf("Variables: %s\n", variablesJSON)
	fmt.Printf("Result: %s\n\n", result)
}

// openAIVariableExample demonstrates CallWithPromptAndVariables with OpenAI
func openAIVariableExample() {
	fmt.Println("2. OpenAI Client with Variable Substitution")
	fmt.Println("==========================================")

	// Skip if no API key is available
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("Skipping OpenAI example - no API key found in OPENAI_API_KEY environment variable")
		fmt.Println("")
		return
	}

	// Create client factory and OpenAI client
	factory := client.NewClientFactory()
	config := &types.AIConfig{
		Provider:    "openai",
		APIKey:      apiKey,
		Model:       "gpt-4o-mini",
		MaxTokens:   150,
		Temperature: 0.7,
	}

	aiClient, err := factory.CreateClient(config)
	if err != nil {
		log.Printf("Failed to create OpenAI client: %v", err)
		return
	}

	// Template prompt with variables
	promptTemplate := "You are a {{role}} assistant. Help me write a {{language}} function that {{task}}. The function should be optimized for {{optimization_target}}."

	// Variables for the prompt
	variablesJSON := `{
		"role": "senior software engineer",
		"language": "Go",
		"task": "calculates the factorial of a number",
		"optimization_target": "performance and readability"
	}`

	fmt.Printf("Prompt Template: %s\n", promptTemplate)
	fmt.Printf("Variables: %s\n", variablesJSON)

	// Call with prompt and variables
	ctx := context.Background()
	response, err := aiClient.CallWithPromptAndVariables(ctx, promptTemplate, variablesJSON)
	if err != nil {
		log.Printf("Error calling OpenAI with variables: %v", err)
		return
	}

	fmt.Printf("OpenAI Response: %s\n\n", string(response))
}

// claudeVariableExample demonstrates CallWithPromptAndVariables with Claude
func claudeVariableExample() {
	fmt.Println("3. Claude Client with Variable Substitution")
	fmt.Println("==========================================")

	// Skip if no API key is available
	apiKey := os.Getenv("CLAUDE_API_KEY")
	if apiKey == "" {
		fmt.Println("Skipping Claude example - no API key found in CLAUDE_API_KEY environment variable")
		fmt.Println("")
		return
	}

	// Create client factory and Claude client
	factory := client.NewClientFactory()
	config := &types.AIConfig{
		Provider:    "claude",
		APIKey:      apiKey,
		Model:       "claude-3-sonnet-20240229",
		MaxTokens:   150,
		Temperature: 0.5,
	}

	aiClient, err := factory.CreateClient(config)
	if err != nil {
		log.Printf("Failed to create Claude client: %v", err)
		return
	}

	// Template prompt with variables
	promptTemplate := "As a {{expertise}} expert, explain {{concept}} in {{language}} with a practical example. Focus on {{aspect}} and provide {{detail_level}} explanations."

	// Variables for the prompt
	variablesJSON := `{
		"expertise": "concurrency",
		"concept": "goroutines and channels",
		"language": "Go",
		"aspect": "best practices",
		"detail_level": "intermediate"
	}`

	fmt.Printf("Prompt Template: %s\n", promptTemplate)
	fmt.Printf("Variables: %s\n", variablesJSON)

	// Call with prompt and variables
	ctx := context.Background()
	response, err := aiClient.CallWithPromptAndVariables(ctx, promptTemplate, variablesJSON)
	if err != nil {
		log.Printf("Error calling Claude with variables: %v", err)
		return
	}

	fmt.Printf("Claude Response: %s\n\n", string(response))
}

// advancedVariableExample demonstrates advanced variable patterns and edge cases
func advancedVariableExample() {
	fmt.Println("4. Advanced Variable Patterns")
	fmt.Println("=============================")

	examples := []struct {
		name        string
		template    string
		variables   string
		description string
	}{
		{
			name:        "Multiple Variables",
			template:    "Create a {{type}} in {{language}} that handles {{operation}} with {{error_handling}} error handling.",
			variables:   `{"type": "REST API", "language": "Go", "operation": "user authentication", "error_handling": "comprehensive"}`,
			description: "Multiple variables in a single template",
		},
		{
			name:        "Missing Variables",
			template:    "Implement {{feature}} with {{missing_var}} and {{technology}}.",
			variables:   `{"feature": "caching", "technology": "Redis"}`,
			description: "Template with missing variables ({{missing_var}} will remain unchanged)",
		},
		{
			name:        "Special Characters",
			template:    "Generate {{file_type}} for {{project-name}} using {{tech_stack}}.",
			variables:   `{"file_type": "Dockerfile", "project-name": "my-web-app", "tech_stack": "Node.js & Express"}`,
			description: "Variables with hyphens and special characters in values",
		},
		{
			name:        "Empty Variables",
			template:    "This template has {{no_vars}} but empty JSON.",
			variables:   `{}`,
			description: "Empty variables JSON (template remains unchanged)",
		},
		{
			name:        "Numeric Values",
			template:    "Set timeout to {{timeout}} seconds and retry {{max_retries}} times.",
			variables:   `{"timeout": 30, "max_retries": 3}`,
			description: "Numeric values in JSON (converted to strings)",
		},
	}

	for i, example := range examples {
		fmt.Printf("Example %d: %s\n", i+1, example.name)
		fmt.Printf("Description: %s\n", example.description)
		fmt.Printf("Template: %s\n", example.template)
		fmt.Printf("Variables: %s\n", example.variables)

		result, err := utils.SubstituteVariables(example.template, example.variables)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		} else {
			fmt.Printf("Result: %s\n", result)
		}
		fmt.Println()
	}
}

// errorHandlingExample demonstrates error handling scenarios
func errorHandlingExample() {
	fmt.Println("5. Error Handling Examples")
	fmt.Println("==========================")

	errorCases := []struct {
		name        string
		template    string
		variables   string
		description string
	}{
		{
			name:        "Malformed JSON",
			template:    "Hello {{name}}",
			variables:   `{"name": "Alice"`,
			description: "Invalid JSON format (missing closing brace)",
		},
		{
			name:        "Empty Template",
			template:    "",
			variables:   `{"name": "Alice"}`,
			description: "Empty template string",
		},
		{
			name:        "Invalid JSON Structure",
			template:    "Hello {{name}}",
			variables:   `["not", "an", "object"]`,
			description: "JSON array instead of object",
		},
		{
			name:        "Null Variables",
			template:    "Hello {{name}}",
			variables:   "null",
			description: "Null variables (template returned unchanged)",
		},
	}

	for i, errorCase := range errorCases {
		fmt.Printf("Error Case %d: %s\n", i+1, errorCase.name)
		fmt.Printf("Description: %s\n", errorCase.description)
		fmt.Printf("Template: %q\n", errorCase.template)
		fmt.Printf("Variables: %s\n", errorCase.variables)

		result, err := utils.SubstituteVariables(errorCase.template, errorCase.variables)
		if err != nil {
			fmt.Printf("Expected Error: %v\n", err)
		} else {
			fmt.Printf("Result: %s\n", result)
		}
		fmt.Println()
	}
}

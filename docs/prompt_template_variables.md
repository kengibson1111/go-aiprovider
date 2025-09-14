# Prompt Template Variables

The Go AI Provider library supports prompt template variables, allowing you to create reusable prompt templates with placeholder variables that can be dynamically substituted with actual values.

## Overview

Prompt template variables enable you to:
- Create reusable prompt templates for common use cases
- Dynamically customize prompts without string concatenation
- Maintain clean separation between prompt structure and variable data
- Ensure consistent prompt formatting across your application

## Variable Format

Variables in prompt templates must follow the `{{variable_name}}` format:

```
{{variable_name}}
```

### Variable Name Rules

- **Allowed characters**: Letters (a-z, A-Z), numbers (0-9), underscores (_), and hyphens (-)
- **Case sensitivity**: Variable names are case-sensitive (`{{Name}}` ≠ `{{name}}`)
- **No spaces**: Variable names cannot contain spaces
- **No nested braces**: Patterns like `{{{variable}}}` are not supported

### Valid Examples

```
{{name}}
{{user_id}}
{{project-name}}
{{API_KEY}}
{{version2}}
```

### Invalid Examples

```
{{user name}}        // Contains space
{{user.name}}        // Contains dot
{{{variable}}}       // Nested braces
{{}}                 // Empty variable name
```

## Variables JSON Format

Variables are provided as a JSON string containing key-value pairs:

```json
{
  "variable_name": "value",
  "another_var": "another value"
}
```

### JSON Value Types

All JSON value types are supported and automatically converted to strings:

```json
{
  "name": "Alice",           // String
  "age": 30,                 // Number → "30"
  "active": true,            // Boolean → "true"
  "description": null,       // Null → ""
  "score": 95.5             // Float → "95.5"
}
```

### Special Cases

- **Empty object**: `{}` - No substitutions performed, template returned unchanged
- **Null/empty string**: `null` or `""` - No substitutions performed
- **Missing variables**: Variables in template without matching JSON keys remain unchanged

## Usage Methods

### 1. AIClient Interface Method

Use `CallWithPromptAndVariables` with any AI provider client:

```go
// Create client (OpenAI or Claude)
client, err := factory.CreateClient(config)
if err != nil {
    log.Fatal(err)
}

// Template with variables
promptTemplate := "You are a {{role}} assistant. Help with {{task}} in {{language}}."

// Variables as JSON
variablesJSON := `{
    "role": "senior developer",
    "task": "code review", 
    "language": "Go"
}`

// Call with template and variables
response, err := client.CallWithPromptAndVariables(ctx, promptTemplate, variablesJSON)
```

### 2. Standalone Utility Function

Use `utils.SubstituteVariables` for direct template processing:

```go
import "github.com/kengibson1111/go-aiprovider/utils"

template := "Hello {{name}}, welcome to {{platform}}!"
variables := `{"name": "Alice", "platform": "Go AI Provider"}`

result, err := utils.SubstituteVariables(template, variables)
if err != nil {
    log.Fatal(err)
}
// result: "Hello Alice, welcome to Go AI Provider!"
```

## Examples

### Basic Substitution

```go
template := "Create a {{type}} in {{language}} for {{purpose}}."
variables := `{
    "type": "REST API",
    "language": "Go",
    "purpose": "user management"
}`

result, _ := utils.SubstituteVariables(template, variables)
// Result: "Create a REST API in Go for user management."
```

### Multiple Variables

```go
template := "Generate {{count}} {{item_type}} examples for {{audience}} using {{technology}}."
variables := `{
    "count": 5,
    "item_type": "unit test",
    "audience": "junior developers", 
    "technology": "Go testing framework"
}`

result, _ := utils.SubstituteVariables(template, variables)
// Result: "Generate 5 unit test examples for junior developers using Go testing framework."
```

### Missing Variables

```go
template := "Process {{input}} and generate {{output}} with {{missing_var}}."
variables := `{
    "input": "user data",
    "output": "report"
}`

result, _ := utils.SubstituteVariables(template, variables)
// Result: "Process user data and generate report with {{missing_var}}."
```

### Complex Prompt Template

```go
template := `You are a {{expertise}} expert working on a {{project_type}} project.

Task: {{task_description}}
Requirements:
- Use {{primary_language}} as the main language
- Follow {{coding_standard}} coding standards  
- Optimize for {{optimization_target}}
- Include {{test_type}} tests

Context:
- Team size: {{team_size}}
- Timeline: {{timeline}}
- Experience level: {{experience_level}}

Please provide a detailed implementation plan.`

variables := `{
    "expertise": "backend development",
    "project_type": "microservices",
    "task_description": "implement user authentication service",
    "primary_language": "Go",
    "coding_standard": "Google Go",
    "optimization_target": "performance and security",
    "test_type": "unit and integration",
    "team_size": "4 developers",
    "timeline": "2 weeks",
    "experience_level": "intermediate"
}`
```

## Error Handling

The library provides specific error types for different failure scenarios:

### Template Processing Errors

```go
// Empty template
result, err := utils.SubstituteVariables("", `{"name": "Alice"}`)
// err: "template cannot be empty"

// Malformed JSON
result, err := utils.SubstituteVariables("Hello {{name}}", `{"name": "Alice"`)
// err: "invalid JSON format in variables: unexpected end of JSON input"

// Invalid JSON structure  
result, err := utils.SubstituteVariables("Hello {{name}}", `["not", "an", "object"]`)
// err: "invalid JSON format in variables: json: cannot unmarshal array into Go value of type map[string]interface {}"
```

### Client Method Errors

```go
response, err := client.CallWithPromptAndVariables(ctx, template, invalidJSON)
if err != nil {
    // Handle variable substitution errors
    if strings.Contains(err.Error(), "variable substitution failed") {
        log.Printf("Template processing error: %v", err)
        return
    }
    
    // Handle AI provider errors (passed through from CallWithPrompt)
    log.Printf("AI provider error: %v", err)
}
```

## Best Practices

### 1. Template Design

- **Use descriptive variable names**: `{{user_role}}` instead of `{{r}}`
- **Group related variables**: Keep similar variables together in JSON
- **Validate templates**: Test templates with sample data before production use

### 2. Variable Management

- **Centralize variable definitions**: Store common variables in configuration
- **Type safety**: Validate variable values before JSON serialization
- **Default values**: Handle missing variables gracefully in your application logic

### 3. Error Handling

- **Validate JSON**: Check JSON validity before calling substitution methods
- **Handle missing variables**: Decide whether missing variables should cause errors
- **Log substitution details**: Log template and variables for debugging

### 4. Performance Considerations

- **Cache compiled templates**: Reuse template strings when possible
- **Minimize JSON parsing**: Prepare variables JSON once for multiple uses
- **Batch processing**: Process multiple templates with same variables efficiently

## Integration Examples

### Configuration-Driven Templates

```go
type PromptConfig struct {
    Template  string            `json:"template"`
    Variables map[string]string `json:"variables"`
}

func processPromptConfig(config PromptConfig, client AIClient) ([]byte, error) {
    variablesJSON, err := json.Marshal(config.Variables)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal variables: %w", err)
    }
    
    return client.CallWithPromptAndVariables(ctx, config.Template, string(variablesJSON))
}
```

### Template Library

```go
type TemplateLibrary struct {
    templates map[string]string
}

func (tl *TemplateLibrary) GetCodeReviewTemplate() string {
    return `You are a {{seniority}} {{language}} developer. 
Review this code for {{focus_areas}}. 
Provide {{feedback_style}} feedback.`
}

func (tl *TemplateLibrary) GetDocumentationTemplate() string {
    return `Generate {{doc_type}} documentation for {{component}} 
in {{format}} format. Target audience: {{audience}}.`
}
```

### Dynamic Variable Building

```go
func buildUserContextVariables(user User, project Project) (string, error) {
    variables := map[string]interface{}{
        "user_name":       user.Name,
        "user_role":       user.Role,
        "experience":      user.ExperienceLevel,
        "project_name":    project.Name,
        "project_type":    project.Type,
        "tech_stack":      strings.Join(project.Technologies, ", "),
        "deadline":        project.Deadline.Format("2006-01-02"),
    }
    
    variablesJSON, err := json.Marshal(variables)
    if err != nil {
        return "", fmt.Errorf("failed to build variables: %w", err)
    }
    
    return string(variablesJSON), nil
}
```

## Comparison with Alternatives

### Before: String Concatenation

```go
// Fragile and hard to maintain
prompt := "You are a " + role + " assistant. Help with " + task + " in " + language + "."
```

### Before: fmt.Sprintf

```go
// Better but still coupled
prompt := fmt.Sprintf("You are a %s assistant. Help with %s in %s.", role, task, language)
```

### After: Template Variables

```go
// Clean, reusable, and maintainable
template := "You are a {{role}} assistant. Help with {{task}} in {{language}}."
variables := `{"role": "senior developer", "task": "code review", "language": "Go"}`
```

## Troubleshooting

### Common Issues

1. **Variables not substituted**
   - Check variable name spelling and case sensitivity
   - Verify JSON format is valid
   - Ensure variable names match exactly between template and JSON

2. **JSON parsing errors**
   - Validate JSON syntax using a JSON validator
   - Check for trailing commas or missing quotes
   - Ensure proper escaping of special characters

3. **Unexpected results**
   - Verify variable names don't contain invalid characters
   - Check for nested braces or malformed variable syntax
   - Test with simple examples first

### Debugging Tips

```go
// Log template and variables for debugging
log.Printf("Template: %s", template)
log.Printf("Variables: %s", variablesJSON)

result, err := utils.SubstituteVariables(template, variablesJSON)
if err != nil {
    log.Printf("Substitution error: %v", err)
} else {
    log.Printf("Result: %s", result)
}
```

## See Also

- [Examples](../examples/prompt_template_variables_example.go) - Complete working examples
- [API Reference](../README.md#api-reference) - Full API documentation
- [Testing Guide](../TESTING.md) - Testing prompt template functionality
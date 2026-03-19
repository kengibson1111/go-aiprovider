# Examples

This directory contains example code demonstrating various features of the Go AI Provider library.

## Available Examples

### Prompt Template Variables Example

**File**: `prompt_template_variables_example.go`

Demonstrates the prompt template variables feature with comprehensive examples including:

- Basic variable substitution using the standalone utility
- Using `CallWithPromptAndVariables` with OpenAI client
- Using `CallWithPromptAndVariables` with Claude client  
- Advanced variable patterns and edge cases
- Error handling scenarios

**Prerequisites**:
- Set up `.env` file with API keys (see main README.md)
- Ensure `OPENAI_API_KEY` and/or `CLAUDE_API_KEY` environment variables are set

**Running the example**:

```bash
# Run the complete example
go run examples/prompt_template_variables_example.go

# Or run from the examples directory
cd examples
go run prompt_template_variables_example.go
```

**What you'll see**:
1. Basic variable substitution with the utility function
2. Real API calls to OpenAI and Claude (if API keys are configured)
3. Advanced variable patterns including missing variables and special characters
4. Error handling demonstrations with malformed JSON and other edge cases

**Note**: The example will skip OpenAI or Claude sections if the respective API keys are not found in environment variables, allowing you to test with just one provider if needed.

## Adding New Examples

When adding new examples:

1. Create a new `.go` file in this directory
2. Include comprehensive comments explaining the functionality
3. Add error handling and graceful degradation for missing API keys
4. Update this README with a description of the new example
5. Follow the existing code style and structure

## Environment Setup

Before running examples, ensure you have:

1. **API Keys**: Set up your `.env` file or environment variables
   ```bash
   export OPENAI_API_KEY="your_openai_key_here"
   export CLAUDE_API_KEY="your_claude_key_here"
   ```

2. **Dependencies**: Install required Go modules
   ```bash
   go mod tidy
   ```

3. **Network Access**: Ensure you can reach the AI provider APIs

## Example Output

The prompt template variables example produces output similar to:

```
=== Prompt Template Variables Example ===

1. Basic Variable Substitution Utility
=====================================
Template: Hello {{name}}, please review this {{language}} code for {{task_type}}.
Variables: {"name": "Alice", "language": "Go", "task_type": "performance optimization"}
Result: Hello Alice, please review this Go code for performance optimization.

2. OpenAI Client with Variable Substitution
==========================================
Prompt Template: You are a {{role}} assistant. Help me write a {{language}} function that {{task}}...
Variables: {"role": "senior software engineer", "language": "Go", "task": "calculates the factorial of a number"...}
OpenAI Response: [AI-generated response about factorial function implementation]

[Additional examples continue...]
```
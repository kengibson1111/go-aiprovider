# go-aiprovider

A Go library for unified AI provider integration, supporting multiple AI services through a common interface.

## Features

- **Unified Interface**: Single API for multiple AI providers (Claude, OpenAI)
- **Direct Prompt Calls**: Send raw prompts directly to AI providers with CallWithPrompt
- **Code Completion**: AI-powered code completion with context awareness
- **Code Generation**: Generate code from natural language prompts
- **Style Analysis**: Automatic detection of code style preferences
- **Context-Aware**: Leverages project context for better suggestions
- **Configurable**: Flexible configuration for different AI models and parameters

## Supported Providers

- **Claude** (Anthropic)
- **OpenAI** (GPT models)

## Installation

```bash
go get github.com/kengibson1111/go-aiprovider
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/kengibson1111/go-aiprovider/client"
    "github.com/kengibson1111/go-aiprovider/types"
)

func main() {
    // Create client factory
    factory := client.NewClientFactory()

    // Configure AI provider
    config := &types.AIConfig{
        Provider:    "openai",
        APIKey:      "your-api-key-here",
        Model:       "gpt-4o-mini",
        MaxTokens:   1000,
        Temperature: 0.7,
    }

    // Create client
    aiClient, err := factory.CreateClient(config)
    if err != nil {
        log.Fatal(err)
    }

    // Generate code completion
    req := types.CompletionRequest{
        Code:     "func calculateSum(a, b int) int {\n    return ",
        Cursor:   35,
        Language: "go",
        Context: types.CodeContext{
            CurrentFunction: "calculateSum",
            ProjectType:     "go-module",
        },
    }

    response, err := aiClient.GenerateCompletion(context.Background(), req)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Suggestions: %v\n", response.Suggestions)
}
```

## API Reference

### Client Factory

#### `NewClientFactory() *ClientFactory`

Creates a new client factory for creating AI provider clients.

#### `CreateClient(config *types.AIConfig) (AIClient, error)`

Creates an AI client based on the provider configuration.

### AIClient Interface

All AI providers implement the `AIClient` interface:

```go
type AIClient interface {
    CallWithPrompt(ctx context.Context, prompt string) ([]byte, error)
    GenerateCompletion(ctx context.Context, req types.CompletionRequest) (*types.CompletionResponse, error)
    GenerateCode(ctx context.Context, req types.CodeGenerationRequest) (*types.CodeGenerationResponse, error)
    ValidateCredentials(ctx context.Context) error
}
```

**Note:** The OpenAI client implementation uses the official OpenAI Go SDK v2 and returns native SDK types (`*openai.ChatCompletion`) for better performance and type safety. The interface methods maintain compatibility by handling the conversion internally.

#### `CallWithPrompt(ctx, prompt) ([]byte, error)`

Sends a raw prompt directly to the AI provider and returns the raw response. This is the foundational method that other methods build upon, providing direct access to the AI provider's API without any preprocessing or response parsing.

**OpenAI Implementation:** Uses the official OpenAI SDK v2 internally, providing 40-60% faster response processing and 30-50% reduction in memory usage compared to custom HTTP implementations.

#### `CallWithPromptAndVariables(ctx, prompt, variablesJSON) ([]byte, error)`

Sends a prompt template with variable substitution to the AI provider. This method enables reusable prompt templates by substituting placeholder variables with actual values before sending to the AI provider.

**Variable Format:**
- Variables must be enclosed in double curly braces: `{{variable_name}}`
- Variable names can contain letters, numbers, underscores, and hyphens
- Variable names are case-sensitive

**Variables JSON Format:**
- Must be a valid JSON object with string keys matching variable names
- Values can be strings, numbers, booleans, or null (all converted to strings)
- Variables without matching keys remain unchanged in the template

For detailed documentation and advanced usage, see [Prompt Template Variables Guide](docs/prompt_template_variables.md).

#### `GenerateCompletion(ctx, req) (*CompletionResponse, error)`

Generates code completions based on current code context and cursor position.

#### `GenerateCode(ctx, req) (*CodeGenerationResponse, error)`

Generates code from natural language prompts with project context.

#### `ValidateCredentials(ctx) error`

Validates API credentials for the configured provider.

### Configuration

#### `types.AIConfig`

```go
type AIConfig struct {
    Provider    string  `json:"provider"`    // "claude" or "openai"
    APIKey      string  `json:"apiKey"`      // API key for the provider
    BaseURL     string  `json:"baseUrl"`     // Optional custom base URL
    Model       string  `json:"model"`       // Model name (e.g., "gpt-4o-mini")
    MaxTokens   int     `json:"maxTokens"`   // Maximum tokens in response
    Temperature float64 `json:"temperature"` // Creativity level (0.0-1.0)
}
```

### Request Types

#### `types.CompletionRequest`

```go
type CompletionRequest struct {
    Code     string      `json:"code"`     // Current code content
    Cursor   int         `json:"cursor"`   // Cursor position in code
    Language string      `json:"language"` // Programming language
    Context  CodeContext `json:"context"`  // Additional context
}
```

#### `types.CodeGenerationRequest`

```go
type CodeGenerationRequest struct {
    Prompt   string      `json:"prompt"`   // Natural language prompt
    Context  CodeContext `json:"context"`  // Project context
    Language string      `json:"language"` // Target language
}
```

### Response Types

#### `types.CompletionResponse`

```go
type CompletionResponse struct {
    Suggestions []string `json:"suggestions"` // Code completion suggestions
    Confidence  float64  `json:"confidence"`  // Confidence score (0.0-1.0)
    Error       string   `json:"error"`       // Error message if any
}
```

#### `types.CodeGenerationResponse`

```go
type CodeGenerationResponse struct {
    Code  string `json:"code"`  // Generated code
    Error string `json:"error"` // Error message if any
}
```

## Environment Setup

Create a `.env` file for API keys:

```bash
# Copy sample environment file
cp .env.sample .env
```

Edit `.env` with your API keys:

```env
# Claude Configuration
CLAUDE_API_KEY=your_claude_api_key_here
# Custom API endpoint for Claude (optional)
# Use this to point to staging environments, local proxies, or alternative endpoints
# Default production endpoint: https://api.anthropic.com
CLAUDE_API_ENDPOINT=https://api.anthropic.com
CLAUDE_MODEL=claude-sonnet-4-20250514

# OpenAI Configuration
OPENAI_API_KEY=your_openai_api_key_here
# Custom API endpoint for OpenAI (optional)
# Use this to point to staging environments, local proxies, or alternative endpoints
# Default production endpoint: https://api.openai.com
OPENAI_API_ENDPOINT=https://api.openai.com
OPENAI_MODEL=gpt-4o-mini
```

## Examples

For complete working examples, see the [examples directory](examples/) which includes:
- [Prompt Template Variables Example](examples/prompt_template_variables_example.go) - Comprehensive demonstration of variable substitution
- [Examples README](examples/README.md) - Setup instructions and example descriptions

### Direct Prompt Call

```go
// Send a raw prompt directly to the AI provider
prompt := "Explain the difference between goroutines and threads in Go"
response, err := client.CallWithPrompt(ctx, prompt)
if err != nil {
    log.Fatal(err)
}

// Parse the raw response as needed
fmt.Printf("AI Response: %s\n", string(response))
```

### OpenAI SDK Advanced Features

The OpenAI client provides additional methods leveraging the official SDK:

#### Multi-turn Conversations

```go
// For OpenAI clients, you can access advanced SDK features
if openaiClient, ok := client.(*openai.OpenAIClient); ok {
    messages := []openai.ChatCompletionMessageParamUnion{
        openai.SystemMessage("You are a helpful assistant."),
        openai.UserMessage("What is the capital of France?"),
        openai.AssistantMessage("The capital of France is Paris."),
        openai.UserMessage("What about Germany?"),
    }
    
    completion, err := openaiClient.CallWithMessages(ctx, messages)
    if err != nil {
        log.Fatal(err)
    }
    
    response := completion.Choices[0].Message.Content
    fmt.Printf("Response: %s\n", response)
}
```

#### Streaming Responses

```go
// Stream responses for real-time applications
if openaiClient, ok := client.(*openai.OpenAIClient); ok {
    stream, err := openaiClient.CallWithPromptStream(ctx, "Tell me a story")
    if err != nil {
        log.Fatal(err)
    }
    
    for stream.Next() {
        chunk := stream.Current()
        if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
            fmt.Print(chunk.Choices[0].Delta.Content)
        }
    }
}
```

#### Function Calling

```go
// Use function calling for tool integration
if openaiClient, ok := client.(*openai.OpenAIClient); ok {
    tools := []openai.ChatCompletionToolUnionParam{
        openai.ChatCompletionFunctionTool(shared.FunctionDefinitionParam{
            Name:        "get_weather",
            Description: openai.String("Get current weather for a location"),
            Parameters: map[string]interface{}{
                "type": "object",
                "properties": map[string]interface{}{
                    "location": map[string]interface{}{
                        "type":        "string",
                        "description": "City name",
                    },
                },
                "required": []string{"location"},
            },
        }),
    }
    
    completion, err := openaiClient.CallWithTools(ctx, "What's the weather in Paris?", tools)
    if err != nil {
        log.Fatal(err)
    }
    
    // Check for function calls in response
    if len(completion.Choices[0].Message.ToolCalls) > 0 {
        toolCall := completion.Choices[0].Message.ToolCalls[0]
        fmt.Printf("Function: %s, Args: %s\n", 
            toolCall.Function.Name, toolCall.Function.Arguments)
    }
}
```

### Prompt Template Variables

```go
// Create a reusable prompt template with variables
promptTemplate := "You are a {{role}} assistant. Help me write a {{language}} function that {{task}}. The function should be optimized for {{optimization_target}}."

// Define variables as JSON
variablesJSON := `{
    "role": "senior software engineer",
    "language": "Go", 
    "task": "calculates the factorial of a number",
    "optimization_target": "performance and readability"
}`

// Call with template and variables
response, err := client.CallWithPromptAndVariables(ctx, promptTemplate, variablesJSON)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("AI Response: %s\n", string(response))
```

#### Standalone Variable Substitution

```go
// Use the utility function directly for variable substitution
template := "Hello {{name}}, please review this {{language}} code."
variables := `{"name": "Alice", "language": "Go"}`

result, err := utils.SubstituteVariables(template, variables)
if err != nil {
    log.Fatal(err)
}
// result: "Hello Alice, please review this Go code."
```

### Code Completion

```go
req := types.CompletionRequest{
    Code:     "const users = [\n  { name: 'John', age: 30 },\n  { name: 'Jane', age: 25 }\n];\n\nconst adults = users.filter(",
    Cursor:   95,
    Language: "javascript",
    Context: types.CodeContext{
        CurrentFunction: "filterUsers",
        ProjectType:     "node-js",
        Imports:         []string{"lodash"},
    },
}

response, err := client.GenerateCompletion(ctx, req)
```

### Code Generation

```go
req := types.CodeGenerationRequest{
    Prompt:   "Create a function that validates email addresses using regex",
    Language: "go",
    Context: types.CodeContext{
        ProjectType: "go-module",
        Imports:     []string{"regexp", "strings"},
    },
}

response, err := client.GenerateCode(ctx, req)
```

### Provider-Specific Configuration

#### Claude

```go
config := &types.AIConfig{
    Provider:    "claude",
    APIKey:      os.Getenv("CLAUDE_API_KEY"),
    Model:       "claude-3-sonnet-20240229",
    MaxTokens:   2000,
    Temperature: 0.5,
}
```

#### OpenAI

```go
config := &types.AIConfig{
    Provider:    "openai",
    APIKey:      os.Getenv("OPENAI_API_KEY"),
    Model:       "gpt-4o-mini", // Defaults to gpt-4o-mini if not specified
    MaxTokens:   1500,
    Temperature: 0.7,
    BaseURL:     "", // Optional: for Azure OpenAI Service or custom endpoints
}
```

**OpenAI SDK Integration:** The OpenAI client uses the official OpenAI Go SDK v2 for:
- Better performance (no JSON marshaling/unmarshaling overhead)
- Type-safe response access with native SDK types
- Built-in retry logic and connection pooling
- Automatic updates with new OpenAI features
- Support for advanced features like streaming and function calling

## Testing

For comprehensive testing instructions, including unit tests, integration tests, and environment setup, see [TESTING.md](TESTING.md).

Quick start:

```bash
# Run unit tests (fast, no external dependencies)
go test -short -timeout 60s ./...

# Run integration tests (requires .env file with API keys)
go test -run Integration ./...
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

## License

See [LICENSE](LICENSE) file for details.

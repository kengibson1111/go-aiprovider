# go-aiprovider

A Go library for unified AI provider integration, supporting multiple AI services through a common interface.

## Features

- **Unified Interface**: Single API for multiple AI providers (Claude, OpenAI)
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
        Model:       "gpt-3.5-turbo",
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
    GenerateCompletion(ctx context.Context, req types.CompletionRequest) (*types.CompletionResponse, error)
    GenerateCode(ctx context.Context, req types.CodeGenerationRequest) (*types.CodeGenerationResponse, error)
    ValidateCredentials(ctx context.Context) error
}
```

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
    Model       string  `json:"model"`       // Model name (e.g., "gpt-3.5-turbo")
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
    Model:       "gpt-4",
    MaxTokens:   1500,
    Temperature: 0.7,
}
```

## Testing

For comprehensive testing instructions, including unit tests, integration tests, and environment setup, see [TESTING.md](TESTING.md).

Quick start:

```bash
# Run unit tests (fast, no external dependencies)
go test -short ./...

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

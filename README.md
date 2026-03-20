# go-aiprovider

A Go library for unified AI provider integration, supporting multiple AI services through a common interface.

## Features

- Unified interface for multiple AI providers (Claude, OpenAI)
- Direct prompt calls via `CallWithPrompt`
- Prompt template variable substitution via `CallWithPromptAndVariables`
- OpenAI SDK v2 integration with streaming, function calling, and multi-turn conversations
- Configurable logging with environment-based log levels
- Shared HTTP client with retry logic and network-aware backoff
- Optimized connection pooling for the OpenAI client

## Supported Providers

- Claude (Anthropic) — custom HTTP client implementation
- OpenAI (GPT models) — official OpenAI Go SDK v2

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
    factory := client.NewClientFactory()

    config := &types.AIConfig{
        Provider:    "openai",
        APIKey:      "your-api-key-here",
        Model:       "gpt-5.4-mini",
        MaxTokens:   1000,
        Temperature: 0.7,
    }

    aiClient, err := factory.CreateClient(config)
    if err != nil {
        log.Fatal(err)
    }

    response, err := aiClient.CallWithPrompt(context.Background(), "Explain goroutines in Go")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Response: %s\n", string(response))
}
```

## API Reference

### Client Factory

`NewClientFactory()` creates a new factory for building AI provider clients.

`CreateClient(config *types.AIConfig)` returns an `AIClient` for the configured provider.

### AIClient Interface

All providers implement the `AIClient` interface:

```go
type AIClient interface {
    CallWithPrompt(ctx context.Context, prompt string) ([]byte, error)
    CallWithPromptAndVariables(ctx context.Context, prompt string, variablesJSON string) ([]byte, error)
    ValidateCredentials(ctx context.Context) error
}
```

#### CallWithPrompt

Sends a raw prompt to the AI provider and returns the raw response bytes.

#### CallWithPromptAndVariables

Sends a prompt template with variable substitution. Variables use `{{variable_name}}` format, and `variablesJSON` is a JSON object mapping variable names to values.

```go
prompt := "You are a {{role}} assistant. Help me with {{task}}."
variables := `{"role": "senior engineer", "task": "code review"}`
response, err := aiClient.CallWithPromptAndVariables(ctx, prompt, variables)
```

For detailed documentation, see the [Prompt Template Variables Guide](docs/prompt_template_variables.md).

#### ValidateCredentials

Validates API credentials for the configured provider.

### OpenAI-Specific Methods

The OpenAI client exposes additional methods beyond the shared interface when accessed directly via type assertion:

```go
if openaiClient, ok := aiClient.(*openaiclient.OpenAIClient); ok {
    // Multi-turn conversations
    completion, err := openaiClient.CallWithMessages(ctx, messages)

    // Streaming responses
    stream, err := openaiClient.CallWithPromptStream(ctx, "Tell me a story")

    // Function calling
    completion, err := openaiClient.CallWithTools(ctx, "What's the weather?", tools)
}
```

See the [OpenAI SDK examples](examples/openai_sdk_examples/) for full working code.

### Configuration

```go
type AIConfig struct {
    Provider    string  `json:"provider"`    // "claude" or "openai"
    APIKey      string  `json:"apiKey"`      // API key for the provider
    BaseURL     string  `json:"baseUrl"`     // Optional custom base URL
    Model       string  `json:"model"`       // Model name (e.g., "gpt-5.4-mini")
    MaxTokens   int     `json:"maxTokens"`   // Maximum tokens in response
    Temperature float64 `json:"temperature"` // Creativity level (0.0-1.0)
}
```

## Environment Setup

Copy the sample environment file and fill in your API keys:

```bash
cp .env.sample .env
```

```env
# Claude Configuration
CLAUDE_API_KEY=your_claude_api_key_here
CLAUDE_API_ENDPOINT=https://api.anthropic.com
CLAUDE_MODEL=claude-sonnet-4-6

# OpenAI Configuration
OPENAI_API_KEY=your_openai_api_key_here
OPENAI_API_ENDPOINT=
OPENAI_MODEL=gpt-5.4-mini

# Logging Configuration
LOG_LEVEL=info
VERBOSE=true
```

## Project Structure

```text
go-aiprovider/
├── client/              # AIClient interface and ClientFactory
├── claudeclient/        # Claude (Anthropic) provider implementation
├── openaiclient/        # OpenAI provider implementation (SDK v2)
├── types/               # Shared types (AIConfig, ErrorResponse)
├── internal/
│   ├── shared/
│   │   ├── logging/     # Configurable logger
│   │   ├── utils/       # HTTP client, template processor
│   │   └── testutil/    # Test setup helpers
│   └── examples/        # Internal example code
├── examples/
│   └── openai_sdk_examples/  # Runnable OpenAI SDK examples
│       ├── basic_usage/
│       ├── advanced_features/
│       └── best_practices/
└── docs/                # Documentation
```

## Examples

Working examples are in the [examples/openai_sdk_examples](examples/openai_sdk_examples/) directory:

- [basic_usage](examples/openai_sdk_examples/basic_usage/) — client setup, configuration, simple prompts
- [advanced_features](examples/openai_sdk_examples/advanced_features/) — streaming, function calling, multi-turn conversations
- [best_practices](examples/openai_sdk_examples/best_practices/) — connection reuse, retry logic, concurrency, error handling

An internal [prompt template variables example](internal/examples/prompt_template_variables/) demonstrates the `CallWithPromptAndVariables` feature.

## Testing

```bash
# Run unit tests
go test -short -timeout 120s ./...

# Run integration tests (requires .env with API keys)
go test -run Integration -timeout 120s ./...
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

## License

See [LICENSE](LICENSE) file for details.

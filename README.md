# go-aiprovider

A Go library for unified AI provider integration, supporting multiple AI services through a common interface.

## Features

- Unified `AIClient` interface across all providers
- Direct prompt calls via `CallWithPrompt`
- Prompt template variable substitution via `CallWithPromptAndVariables`
- Credential validation via `ValidateCredentials`
- OpenAI SDK v2 integration with streaming, function calling, and multi-turn conversations
- Configurable logging with environment-based log levels
- Shared HTTP client with retry logic and network-aware backoff
- Optimized connection pooling for OpenAI and Azure OpenAI clients

## Supported Providers

| Provider | Config value | Authentication | Implementation |
| --- | --- | --- | --- |
| Claude (Anthropic) | `claude` | API key | Custom HTTP client |
| Claude via Amazon Bedrock | `claude-bedrock` | AWS credential chain | AWS SDK v2 |
| OpenAI | `openai` | API key | OpenAI Go SDK v2 |
| Azure OpenAI Service | `openai-azure` | Microsoft Entra ID | OpenAI Go SDK v2 + Azure identity |

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
        APIKey:      "your-api-key",
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

- `CallWithPrompt` — sends a raw prompt and returns the raw JSON response bytes.
- `CallWithPromptAndVariables` — substitutes `{{variable_name}}` placeholders in a prompt template before sending. `variablesJSON` is a JSON object mapping names to values.
- `ValidateCredentials` — makes a minimal API call to verify credentials are valid.

```go
prompt := "You are a {{role}} assistant. Help me with {{task}}."
variables := `{"role": "senior engineer", "task": "code review"}`
response, err := aiClient.CallWithPromptAndVariables(ctx, prompt, variables)
```

### Configuration

```go
type AIConfig struct {
    Provider    string  `json:"provider"`    // "claude", "claude-bedrock", "openai", or "openai-azure"
    APIKey      string  `json:"apiKey"`      // API key (not needed for claude-bedrock or openai-azure)
    BaseURL     string  `json:"baseUrl"`     // Optional custom endpoint
    Model       string  `json:"model"`       // Model or deployment name
    MaxTokens   int     `json:"maxTokens"`   // Max tokens in response (default: 1000)
    Temperature float64 `json:"temperature"` // Creativity level 0.0-1.0 (default: 0.7)
}
```

## Provider Setup

### Claude (Anthropic)

Set your API key and optional endpoint in `.env`:

```env
CLAUDE_API_KEY=your_claude_api_key_here
CLAUDE_API_ENDPOINT=https://api.anthropic.com
CLAUDE_MODEL=claude-sonnet-4-6
```

```go
config := &types.AIConfig{
    Provider: "claude",
    APIKey:   os.Getenv("CLAUDE_API_KEY"),
    BaseURL:  os.Getenv("CLAUDE_API_ENDPOINT"),
    Model:    "claude-sonnet-4-6",
}
```

### Claude via Amazon Bedrock

Uses the AWS default credential chain (env vars, `~/.aws/credentials`, SSO, IAM role). No API key needed.

```env
CLAUDE_BEDROCK_REGION=us-east-1
CLAUDE_BEDROCK_MODEL=us.anthropic.claude-sonnet-4-20250514-v1:0
```

```go
config := &types.AIConfig{
    Provider: "claude-bedrock",
    Model:    "us.anthropic.claude-sonnet-4-20250514-v1:0",
}
```

See [docs/claude_bedrock_setup.md](docs/claude_bedrock_setup.md) for IAM policy, model access, and troubleshooting.

### OpenAI

```env
OPENAI_API_KEY=your_openai_api_key_here
OPENAI_API_ENDPOINT=
OPENAI_MODEL=gpt-5.4-mini
```

```go
config := &types.AIConfig{
    Provider: "openai",
    APIKey:   os.Getenv("OPENAI_API_KEY"),
    Model:    "gpt-5.4-mini",
}
```

### Azure OpenAI Service

Uses Microsoft Entra ID (service principal) authentication. No API key needed.

```env
OPENAI_AZURE_ENDPOINT=https://your-resource.openai.azure.com
OPENAI_AZURE_API_VERSION=2024-12-01-preview
OPENAI_AZURE_MODEL=gpt-4o-mini
OPENAI_AZURE_SP_TENANT_ID=your_tenant_id
OPENAI_AZURE_SP_CLIENT_ID=your_client_id
OPENAI_AZURE_SP_CLIENT_SECRET=your_client_secret
```

```go
config := &types.AIConfig{
    Provider: "openai-azure",
}
```

See [docs/openai_azure_setup.md](docs/openai_azure_setup.md) for resource creation, RBAC, and troubleshooting.

## Environment Setup

Copy the sample file and fill in the values for the providers you need:

```powershell
Copy-Item .env.sample .env
```

See `.env.sample` for the full list of configuration variables.

## Project Structure

```text
go-aiprovider/
├── client/                        # AIClient interface, ClientFactory, integration tests
├── types/                         # Shared types (AIConfig, ErrorResponse)
├── internal/
│   ├── claudeclient/              # Claude and Claude Bedrock provider implementations
│   ├── openaiclient/              # OpenAI and Azure OpenAI provider implementations
│   └── shared/
│       ├── env/                   # Environment configuration loading
│       ├── logging/               # Configurable logger
│       ├── testutil/              # Test setup helpers
│       └── utils/                 # HTTP client, template processor
├── examples/
│   ├── claude_client/             # Runnable Claude (Anthropic) example
│   ├── claude_bedrock/            # Runnable Claude via Amazon Bedrock example
│   ├── openai_client/             # Runnable OpenAI example
│   └── openai_azure/             # Runnable Azure OpenAI example
└── docs/                          # Provider setup guides
```

## Examples

Each example directory contains a `main.go` with five patterns: client creation, prompt calls, template variables, error handling, and credential validation.

| Example | Provider | Auth | Run command |
| --- | --- | --- | --- |
| [claude_client](examples/claude_client/) | Claude (Anthropic) | API key | `cd examples\claude_client; go run main.go` |
| [claude_bedrock](examples/claude_bedrock/) | Claude via Bedrock | AWS credential chain | `cd examples\claude_bedrock; go run main.go` |
| [openai_client](examples/openai_client/) | OpenAI | API key | `cd examples\openai_client; go run main.go` |
| [openai_azure](examples/openai_azure/) | Azure OpenAI | Microsoft Entra ID | `cd examples\openai_azure; go run main.go` |

## Testing

Unit and integration tests are separated by build tags. Run them per-package with appropriate timeouts.

### Unit Tests

```powershell
go test ./internal/shared/env -v
go test ./internal/shared/logging -v
go test ./internal/shared/utils -v
```

### Integration Tests

Requires a configured `.env` file with valid credentials for the providers under test.

```powershell
go test ./client -v -tags=integration -timeout 5m
go test ./internal/claudeclient -v -tags=integration -timeout 5m
go test ./internal/openaiclient -v -tags=integration -timeout 5m
go test ./internal/shared/logging -v -tags=integration -timeout 5m
go test ./internal/shared/utils -v -tags=integration -timeout 5m
```

Run a specific provider's tests:

```powershell
go test ./client -v -tags=integration -timeout 5m -run "ClaudeBedrock"
go test ./internal/claudeclient -v -tags=integration -timeout 5m -run "TestClaudeBedrockIntegrationTestSuite"
```

## License

See [LICENSE](LICENSE) file for details.

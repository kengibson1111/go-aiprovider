# OpenAI Client Examples

Basic examples showing how to use the `go-aiprovider` client with OpenAI.

## Prerequisites

1. Copy `.env.sample` to `.env` at the repo root and set your OpenAI API key:

   ```text
   OPENAI_API_KEY=your_api_key_here
   ```

2. Install dependencies:

   ```powershell
   go mod tidy
   ```

## Running

From the repo root:

```powershell
cd examples\openai_client
go run main.go
```

## Examples

The `main.go` file contains five examples:

- **BasicUsageExample** — Create a client and make a simple prompt call.
- **TimeoutExample** — Use `context.WithTimeout` for request cancellation.
- **TemplateVariablesExample** — Use `{{variable}}` prompt templates with JSON variable substitution.
- **ErrorHandlingExample** — Inspect structured `*types.ErrorResponse` errors using `errors.As`.
- **ValidateCredentialsExample** — Validate API credentials before making calls.

## Key Patterns

### Client creation

```go
factory := client.NewClientFactory()
aiClient, err := factory.CreateClient(&types.AIConfig{
    Provider:    "openai",
    APIKey:      os.Getenv("OPENAI_API_KEY"),
    Model:       "gpt-4o-mini",
    MaxTokens:   1000,
    Temperature: 0.7,
})
```

### Error handling

All API errors are returned as `*types.ErrorResponse`, which you can unwrap with `errors.As`:

```go
var apiErr *types.ErrorResponse
if errors.As(err, &apiErr) {
    fmt.Printf("Code: %s, Message: %s\n", apiErr.Code, apiErr.Message)
}
```

### Prompt templates

```go
prompt := "You are a {{role}}. Summarize {{topic}}."
variables := `{"role": "teacher", "topic": "concurrency"}`
response, err := aiClient.CallWithPromptAndVariables(ctx, prompt, variables)
```

# Claude Bedrock Examples

Examples showing how to use the `go-aiprovider` client with Claude via Amazon Bedrock.

## Prerequisites

1. Set up AWS credentials and Bedrock model access. See [Claude Bedrock Setup Guide](../../docs/claude_bedrock_setup.md) for full instructions.

2. Copy `.env.sample` to `.env` at the repo root and configure the Bedrock variables:

   ```text
   CLAUDE_BEDROCK_REGION=us-east-1
   CLAUDE_BEDROCK_MODEL=us.anthropic.claude-sonnet-4-20250514-v1:0
   ```

   No API key is needed — AWS credentials handle authentication via the default credential chain.

3. Install dependencies:

   ```powershell
   go mod tidy
   ```

## Running

From the repo root:

```powershell
cd examples\claude_bedrock
go run main.go
```

## Examples

The `main.go` file contains five examples:

- **BasicUsageExample** — Create a Bedrock client and make a simple prompt call.
- **TimeoutExample** — Use `context.WithTimeout` for request cancellation.
- **TemplateVariablesExample** — Use `{{variable}}` prompt templates with JSON variable substitution.
- **ErrorHandlingExample** — Inspect structured `*types.ErrorResponse` errors using `errors.As`.
- **ValidateCredentialsExample** — Validate AWS credentials and model access before making calls.

## Key Patterns

### Client creation

```go
factory := client.NewClientFactory()
aiClient, err := factory.CreateClient(&types.AIConfig{
    Provider:    "claude-bedrock",
    Model:       os.Getenv("CLAUDE_BEDROCK_MODEL"),
    MaxTokens:   1000,
    Temperature: 0.7,
})
```

### Authentication

The `claude-bedrock` provider uses the AWS SDK v2 default credential chain (in order):

1. Environment variables (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`)
2. Shared credentials file (`~/.aws/credentials`)
3. SSO / AWS Identity Center
4. IAM instance role (EC2, ECS, Lambda)

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

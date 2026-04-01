# Azure OpenAI Client Examples (UsernamePassword)

Basic examples showing how to use the `go-aiprovider` client with Azure OpenAI Service using UsernamePasswordCredential (ROPC flow) authentication.

## Prerequisites

1. Copy `.env.sample` to `.env` at the repo root and set your Azure OpenAI environment variables:

   ```text
   OPENAI_AZURE_ENDPOINT=https://your-resource.openai.azure.com
   OPENAI_AZURE_API_VERSION=2024-12-01-preview
   OPENAI_AZURE_MODEL=gpt-4o-mini
   OPENAI_AZURE_TENANT_ID=your_tenant_id
   OPENAI_AZURE_CLIENT_ID=your_client_id
   OPENAI_AZURE_UP_USERNAME=user@yourtenant.onmicrosoft.com
   OPENAI_AZURE_UP_PASSWORD=your_password
   ```

2. Install dependencies:

   ```powershell
   go mod tidy
   ```

## Running

From the repo root:

```powershell
cd examples\openai_azure_up
go run main.go
```

## Examples

The `main.go` file contains five examples:

- **BasicUsageExample** — Create an Azure OpenAI UP client and make a simple prompt call.
- **TimeoutExample** — Use `context.WithTimeout` for request cancellation.
- **TemplateVariablesExample** — Use `{{variable}}` prompt templates with JSON variable substitution.
- **ErrorHandlingExample** — Inspect structured `*types.ErrorResponse` errors using `errors.As`.
- **ValidateCredentialsExample** — Validate Azure UP credentials before making calls.

## Key Patterns

### Client creation

The Azure UP client uses UsernamePasswordCredential (ROPC) authentication instead of a service principal secret:

```go
factory := client.NewClientFactory()
aiClient, err := factory.CreateClient(&types.AIConfig{
    Provider:    "openai-azure-up",
    BaseURL:     os.Getenv("OPENAI_AZURE_ENDPOINT"),
    Model:       os.Getenv("OPENAI_AZURE_MODEL"),
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

## Authentication

This example uses UsernamePasswordCredential (ROPC flow) authentication via `azidentity.NewUsernamePasswordCredential`. The `OPENAI_AZURE_TENANT_ID`, `OPENAI_AZURE_CLIENT_ID`, `OPENAI_AZURE_UP_USERNAME`, and `OPENAI_AZURE_UP_PASSWORD` environment variables are used to authenticate. Note that this flow is deprecated by Microsoft but useful in environments where interactive login or service principal secrets are not available. See [Azure OpenAI UP Setup](../../docs/openai_azure_up_setup.md) for detailed configuration instructions.

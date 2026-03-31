package client

import (
	"context"
	"fmt"
	"strings"

	"github.com/kengibson1111/go-aiprovider/internal/claudeclient"
	"github.com/kengibson1111/go-aiprovider/internal/openaiclient"
	"github.com/kengibson1111/go-aiprovider/internal/shared/logging"
	"github.com/kengibson1111/go-aiprovider/internal/shared/testutil"
	"github.com/kengibson1111/go-aiprovider/types"
)

// SetupEnvironment loads the .env file from the given repoRoot directory so that
// environment variables (API keys, endpoints, etc.) are available to the process.
// Panics on failure. This is a convenience wrapper around the internal testutil package
// for use in examples and non-test programs.
//
// repoRoot should be the relative path from the caller's working directory to the repo root.
// When running from the repo root, use "./".
func SetupEnvironment(repoRoot string) {
	testutil.SetupExampleEnvironment(repoRoot)
}

// SetupCurrentDirectory changes the working directory to repoRoot and returns a
// cleanup function that restores the original directory. Panics on failure.
// This is a convenience wrapper around the internal testutil package for use in
// examples and non-test programs.
//
// repoRoot should be the relative path from the caller's working directory to the repo root.
// When running from the repo root, use "./".
func SetupCurrentDirectory(repoRoot string) func() {
	return testutil.SetupExampleCurrentDirectory(repoRoot)
}

// AIClient defines the interface for AI service clients
type AIClient interface {
	// CallWithPrompt sends a raw prompt directly to the AI provider and returns the raw response.
	// This is the foundational method that other methods build upon, providing direct access
	// to the AI provider's API without any preprocessing or response parsing.
	CallWithPrompt(ctx context.Context, prompt string) ([]byte, error)

	// CallWithPromptAndVariables sends a prompt template with variable substitution to the AI provider.
	// Variables in the prompt template should use {{variable_name}} format, and variablesJSON
	// should contain a JSON object with variable name-value pairs. The method substitutes
	// variables in the prompt before sending it to the AI provider using CallWithPrompt.
	//
	// Example:
	//   prompt := "Hello {{name}}, please review this {{language}} code."
	//   variables := `{"name": "Alice", "language": "Go"}`
	//   response, err := client.CallWithPromptAndVariables(ctx, prompt, variables)
	CallWithPromptAndVariables(ctx context.Context, prompt string, variablesJSON string) ([]byte, error)

	// ValidateCredentials validates API credentials for the configured provider.
	ValidateCredentials(ctx context.Context) error
}

// ClientFactory creates AI clients based on provider configuration
type ClientFactory struct {
	logger *logging.DefaultLogger
}

// NewClientFactory creates a new client factory
func NewClientFactory() *ClientFactory {
	return &ClientFactory{
		logger: logging.NewDefaultLogger(),
	}
}

// CreateClient creates an AI client based on the provider configuration
func (f *ClientFactory) CreateClient(config *types.AIConfig) (AIClient, error) {
	if config == nil {
		return nil, fmt.Errorf("configuration is required")
	}

	f.logger.Info("Creating AI client for provider: %s", config.Provider)

	switch strings.ToLower(config.Provider) {
	case types.ProviderClaude:
		return claudeclient.NewClaudeClient(config)
	case types.ProviderClaudeBedrock:
		return claudeclient.NewClaudeBedrockClient(config)
	case types.ProviderOpenAI:
		return openaiclient.NewOpenAIClient(config)
	case types.ProviderOpenAIAzure:
		return openaiclient.NewOpenAIAzureClient(config)
	case types.ProviderOpenAIAzureUP:
		return openaiclient.NewOpenAIAzureUPClient(config)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", config.Provider)
	}
}

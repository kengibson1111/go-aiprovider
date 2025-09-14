package client

import (
	"context"
	"fmt"
	"strings"

	"github.com/kengibson1111/go-aiprovider/claude"
	"github.com/kengibson1111/go-aiprovider/openai"
	"github.com/kengibson1111/go-aiprovider/types"
	"github.com/kengibson1111/go-aiprovider/utils"
)

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

	// GenerateCompletion generates code completions based on current code context and cursor position.
	GenerateCompletion(ctx context.Context, req types.CompletionRequest) (*types.CompletionResponse, error)

	// GenerateCode generates code from natural language prompts with project context.
	GenerateCode(ctx context.Context, req types.CodeGenerationRequest) (*types.CodeGenerationResponse, error)

	// ValidateCredentials validates API credentials for the configured provider.
	ValidateCredentials(ctx context.Context) error
}

// ClientFactory creates AI clients based on provider configuration
type ClientFactory struct {
	logger *utils.Logger
}

// NewClientFactory creates a new client factory
func NewClientFactory() *ClientFactory {
	return &ClientFactory{
		logger: utils.NewLogger("ClientFactory"),
	}
}

// CreateClient creates an AI client based on the provider configuration
func (f *ClientFactory) CreateClient(config *types.AIConfig) (AIClient, error) {
	if config == nil {
		return nil, fmt.Errorf("configuration is required")
	}

	f.logger.Info("Creating AI client for provider: %s", config.Provider)

	switch strings.ToLower(config.Provider) {
	case "claude":
		return claude.NewClaudeClient(config)
	case "openai":
		return openai.NewOpenAIClient(config)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", config.Provider)
	}
}

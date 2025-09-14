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
	CallWithPrompt(ctx context.Context, prompt string) ([]byte, error)
	CallWithPromptAndVariables(ctx context.Context, prompt string, variablesJSON string) ([]byte, error)
	GenerateCompletion(ctx context.Context, req types.CompletionRequest) (*types.CompletionResponse, error)
	GenerateCode(ctx context.Context, req types.CodeGenerationRequest) (*types.CodeGenerationResponse, error)
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

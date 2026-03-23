package types

import (
	"fmt"
)

// Provider constants for AIConfig.Provider
const (
	ProviderClaude        = "claude"
	ProviderClaudeBedrock = "claude-bedrock"
	ProviderOpenAI        = "openai"
	ProviderOpenAIAzure   = "openai-azure"
)

// ErrorResponse represents a structured error response.
// It implements the error interface so it can be used with errors.As.
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
	Retry   bool   `json:"retry"`
}

// Error implements the error interface for ErrorResponse.
func (e *ErrorResponse) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("%s: %s (%s)", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// AIConfig represents the AI service configuration
type AIConfig struct {
	Provider    string  `json:"provider"`
	APIKey      string  `json:"apiKey"`
	BaseURL     string  `json:"baseUrl,omitempty"`
	Model       string  `json:"model"`
	MaxTokens   int     `json:"maxTokens"`
	Temperature float64 `json:"temperature"`
}

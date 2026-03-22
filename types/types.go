package types

// Provider constants for AIConfig.Provider
const (
	ProviderClaude        = "claude"
	ProviderClaudeBedrock = "claude-bedrock"
	ProviderOpenAI        = "openai"
	ProviderOpenAIAzure   = "openai-azure"
)

// ErrorResponse represents a structured error response
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
	Retry   bool   `json:"retry"`
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

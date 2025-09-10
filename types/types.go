package types

import (
	"github.com/kengibson1111/go-aiprovider/utils"
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

// CompletionRequest represents a code completion request
type CompletionRequest struct {
	Code     string            `json:"code"`
	Cursor   int               `json:"cursor"`
	Language string            `json:"language"`
	Context  utils.CodeContext `json:"context"`
}

// CompletionResponse represents a code completion response
type CompletionResponse struct {
	Suggestions []string `json:"suggestions"`
	Confidence  float64  `json:"confidence"`
	Error       string   `json:"error,omitempty"`
}

// CodeGenerationRequest represents a manual code generation request
type CodeGenerationRequest struct {
	Prompt   string            `json:"prompt"`
	Context  utils.CodeContext `json:"context"`
	Language string            `json:"language"`
}

// CodeGenerationResponse represents a code generation response
type CodeGenerationResponse struct {
	Code  string `json:"code"`
	Error string `json:"error,omitempty"`
}

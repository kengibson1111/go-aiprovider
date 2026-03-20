package claudeclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/kengibson1111/go-aiprovider/internal/shared/logging"
	"github.com/kengibson1111/go-aiprovider/internal/shared/utils"
	"github.com/kengibson1111/go-aiprovider/types"
)

// ClaudeClient implements the AIClient interface for Claude API
type ClaudeClient struct {
	*utils.BaseHTTPClient
	model       string
	maxTokens   int
	temperature float64
	logger      *logging.DefaultLogger
}

// ClaudeMessage represents a message in Claude API format
type ClaudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ClaudeRequest represents a request to Claude API
type ClaudeRequest struct {
	Model       string          `json:"model"`
	MaxTokens   int             `json:"max_tokens"`
	Temperature float64         `json:"temperature"`
	Messages    []ClaudeMessage `json:"messages"`
}

// ClaudeResponse represents a response from Claude API
type ClaudeResponse struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Role    string `json:"role"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Model        string `json:"model"`
	StopReason   string `json:"stop_reason"`
	StopSequence string `json:"stop_sequence"`
	Usage        struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// ClaudeErrorResponse represents an error response from Claude API
type ClaudeErrorResponse struct {
	Type  string `json:"type"`
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// NewClaudeClient creates a new Claude API client
func NewClaudeClient(config *types.AIConfig) (*ClaudeClient, error) {
	if config == nil {
		return nil, fmt.Errorf("configuration is required")
	}

	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}

	timeout := 30 * time.Second
	baseClient := utils.NewBaseHTTPClient(baseURL, config.APIKey, timeout)

	client := &ClaudeClient{
		BaseHTTPClient: baseClient,
		model:          config.Model,
		maxTokens:      config.MaxTokens,
		temperature:    config.Temperature,
		logger:         logging.NewDefaultLogger(),
	}

	// Set default model if not specified
	if client.model == "" {
		client.model = "claude-3-sonnet-20240229"
	}

	// Set default max tokens if not specified
	if client.maxTokens == 0 {
		client.maxTokens = 1000
	}

	// Set default temperature if not specified
	if client.temperature == 0 {
		client.temperature = 0.7
	}

	client.logger.Info("Claude client created with model: %s", client.model)
	return client, nil
}

// ValidateCredentials validates the Claude API credentials
func (c *ClaudeClient) ValidateCredentials(ctx context.Context) error {
	c.logger.Info("Validating Claude API credentials")

	// Create a simple test request
	messages := []ClaudeMessage{
		{
			Role:    "user",
			Content: "Hello",
		},
	}

	claudeReq := ClaudeRequest{
		Model:       c.model,
		MaxTokens:   10,
		Temperature: 0.1,
		Messages:    messages,
	}

	reqBody, err := json.Marshal(claudeReq)
	if err != nil {
		return fmt.Errorf("failed to marshal validation request: %w", err)
	}

	headers := map[string]string{
		"x-api-key":         c.ApiKey,
		"anthropic-version": "2023-06-01",
	}

	httpReq := utils.HTTPRequest{
		Method:  "POST",
		Path:    "/v1/messages",
		Headers: headers,
		Body:    bytes.NewReader(reqBody),
	}

	resp, err := c.DoRequest(ctx, httpReq)
	if err != nil {
		c.logger.Error("Credential validation request failed: %v", err)
		return fmt.Errorf("credential validation failed: %w", err)
	}

	if resp.StatusCode == 401 {
		return fmt.Errorf("invalid API key")
	}

	if resp.StatusCode == 403 {
		return fmt.Errorf("API key does not have required permissions")
	}

	if resp.StatusCode >= 400 {
		var errorResp ClaudeErrorResponse
		if err := json.Unmarshal(resp.Body, &errorResp); err == nil {
			return fmt.Errorf("API error: %s", errorResp.Error.Message)
		}
		return fmt.Errorf("API error: HTTP %d", resp.StatusCode)
	}

	c.logger.Info("Claude API credentials validated successfully")
	return nil
}

// CallWithPromptAndVariables calls the Claude API with variable substitution.
//
// This method implements the prompt template functionality by:
// 1. Substituting variables in the prompt template using utils.SubstituteVariables
// 2. Calling the existing CallWithPrompt method with the processed prompt
// 3. Returning the same response format as CallWithPrompt
//
// The implementation is identical to the OpenAI client to ensure consistent
// behavior across all AI provider implementations.
//
// Parameters:
//   - ctx: Context for request cancellation and timeouts
//   - prompt: Template string with variables in {{variable_name}} format
//   - variablesJSON: JSON string containing variable name-value pairs
//
// Returns:
//   - Raw response bytes from Claude API
//   - Error if variable substitution fails or API call fails
//
// Example:
//
//	prompt := "As a {{expertise}} expert, explain {{concept}} in {{language}}."
//	variables := `{"expertise": "concurrency", "concept": "goroutines", "language": "Go"}`
//	response, err := client.CallWithPromptAndVariables(ctx, prompt, variables)
func (c *ClaudeClient) CallWithPromptAndVariables(ctx context.Context, prompt string, variablesJSON string) ([]byte, error) {
	c.logger.Info("Processing prompt with variables for Claude API")

	// Substitute variables in the prompt using the template processor utility
	processedPrompt, err := utils.SubstituteVariables(prompt, variablesJSON)
	if err != nil {
		c.logger.Error("Variable substitution failed: %v", err)
		return nil, fmt.Errorf("variable substitution failed: %w", err)
	}

	c.logger.Debug("Variables substituted successfully, calling Claude API")

	// Call the existing CallWithPrompt method with the processed prompt
	// This ensures consistent behavior with direct prompt calls
	return c.CallWithPrompt(ctx, processedPrompt)
}

// CallWithPrompt calls the Claude API
func (c *ClaudeClient) CallWithPrompt(ctx context.Context, prompt string) ([]byte, error) {
	messages := []ClaudeMessage{
		{
			Role:    "user",
			Content: prompt,
		},
	}

	claudeReq := ClaudeRequest{
		Model:       c.model,
		MaxTokens:   c.maxTokens,
		Temperature: c.temperature,
		Messages:    messages,
	}

	reqBody, err := json.Marshal(claudeReq)
	if err != nil {
		c.logger.Error("Failed to marshal completion request: %v", err)
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	headers := map[string]string{
		"x-api-key":         c.ApiKey,
		"anthropic-version": "2023-06-01",
	}

	httpReq := utils.HTTPRequest{
		Method:  "POST",
		Path:    "/v1/messages",
		Headers: headers,
		Body:    bytes.NewReader(reqBody),
	}

	resp, err := c.DoRequest(ctx, httpReq)
	if err != nil {
		c.logger.Error("Completion request failed: %v", err)
		return []byte{}, fmt.Errorf("request failed: %v", err)
	}

	if err := c.ValidateResponse(resp); err != nil {
		c.logger.Error("Invalid response: %v", err)
		return []byte{}, fmt.Errorf("API error: %v", err)
	}

	return resp.Body, nil
}

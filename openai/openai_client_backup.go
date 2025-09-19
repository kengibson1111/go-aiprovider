package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/kengibson1111/go-aiprovider/types"
	"github.com/kengibson1111/go-aiprovider/utils"
)

// OpenAIClientBackup implements the AIClient interface for OpenAI API (BACKUP VERSION)
// This is a backup of the original implementation before SDK migration
type OpenAIClientBackup struct {
	*utils.BaseHTTPClient
	model       string
	maxTokens   int
	temperature float64
	logger      *utils.Logger
}

// OpenAIMessageBackup represents a message in OpenAI API format (BACKUP VERSION)
type OpenAIMessageBackup struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenAIRequestBackup represents a request to OpenAI API (BACKUP VERSION)
type OpenAIRequestBackup struct {
	Model       string                `json:"model"`
	Messages    []OpenAIMessageBackup `json:"messages"`
	MaxTokens   int                   `json:"max_tokens"`
	Temperature float64               `json:"temperature"`
	Stream      bool                  `json:"stream"`
}

// OpenAIResponseBackup represents a response from OpenAI API (BACKUP VERSION)
type OpenAIResponseBackup struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// OpenAIErrorResponseBackup represents an error response from OpenAI API (BACKUP VERSION)
type OpenAIErrorResponseBackup struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Param   string `json:"param"`
		Code    string `json:"code"`
	} `json:"error"`
}

// NewOpenAIClientBackup creates a new OpenAI API client (BACKUP VERSION)
func NewOpenAIClientBackup(config *types.AIConfig) (*OpenAIClientBackup, error) {
	if config == nil {
		return nil, fmt.Errorf("configuration is required")
	}

	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://api.openai.com"
	}

	timeout := 30 * time.Second
	baseClient := utils.NewBaseHTTPClient(baseURL, config.APIKey, timeout)

	client := &OpenAIClientBackup{
		BaseHTTPClient: baseClient,
		model:          config.Model,
		maxTokens:      config.MaxTokens,
		temperature:    config.Temperature,
		logger:         utils.NewLogger("OpenAIClientBackup"),
	}

	// Set default model if not specified
	if client.model == "" {
		client.model = "gpt-3.5-turbo"
	}

	// Set default max tokens if not specified
	if client.maxTokens == 0 {
		client.maxTokens = 1000
	}

	// Set default temperature if not specified
	if client.temperature == 0 {
		client.temperature = 0.7
	}

	client.logger.Info("OpenAI backup client created with model: %s", client.model)
	return client, nil
}

// ValidateCredentialsBackup validates the OpenAI API credentials (BACKUP VERSION)
func (c *OpenAIClientBackup) ValidateCredentialsBackup(ctx context.Context) error {
	c.logger.Info("Validating OpenAI API credentials (backup)")

	// Create a simple test request
	messages := []OpenAIMessageBackup{
		{
			Role:    "user",
			Content: "Hello",
		},
	}

	openaiReq := OpenAIRequestBackup{
		Model:       c.model,
		Messages:    messages,
		MaxTokens:   10,
		Temperature: 0.1,
		Stream:      false,
	}

	reqBody, err := json.Marshal(openaiReq)
	if err != nil {
		return fmt.Errorf("failed to marshal validation request: %w", err)
	}

	headers := map[string]string{
		"Authorization": "Bearer " + c.ApiKey,
	}

	httpReq := utils.HTTPRequest{
		Method:  "POST",
		Path:    "/v1/chat/completions",
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

	if resp.StatusCode == 429 {
		return fmt.Errorf("rate limit exceeded")
	}

	if resp.StatusCode >= 400 {
		var errorResp OpenAIErrorResponseBackup
		if err := json.Unmarshal(resp.Body, &errorResp); err == nil {
			return fmt.Errorf("API error: %s", errorResp.Error.Message)
		}
		return fmt.Errorf("API error: HTTP %d", resp.StatusCode)
	}

	c.logger.Info("OpenAI API credentials validated successfully (backup)")
	return nil
}

// CallWithPromptBackup calls the OpenAI API (BACKUP VERSION)
func (c *OpenAIClientBackup) CallWithPromptBackup(ctx context.Context, prompt string) ([]byte, error) {
	messages := []OpenAIMessageBackup{
		{
			Role:    "user",
			Content: prompt,
		},
	}

	openaiReq := OpenAIRequestBackup{
		Model:       c.model,
		Messages:    messages,
		MaxTokens:   c.maxTokens,
		Temperature: c.temperature,
		Stream:      false,
	}

	reqBody, err := json.Marshal(openaiReq)
	if err != nil {
		c.logger.Error("Failed to marshal completion request: %v", err)
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	headers := map[string]string{
		"Authorization": "Bearer " + c.ApiKey,
	}

	httpReq := utils.HTTPRequest{
		Method:  "POST",
		Path:    "/v1/chat/completions",
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

		// Handle rate limiting specifically
		if resp.StatusCode == 429 {
			return []byte{}, fmt.Errorf("rate limit exceeded. Please try again later: %v", err)
		}

		return []byte{}, fmt.Errorf("API error: %v", err)
	}

	return resp.Body, nil
}

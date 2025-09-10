package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/kengibson1111/go-aiprovider/types"
	"github.com/kengibson1111/go-aiprovider/utils"
)

// OpenAIClient implements the AIClient interface for OpenAI API
type OpenAIClient struct {
	*utils.BaseHTTPClient
	model       string
	maxTokens   int
	temperature float64
	logger      *utils.Logger
}

// OpenAIMessage represents a message in OpenAI API format
type OpenAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenAIRequest represents a request to OpenAI API
type OpenAIRequest struct {
	Model       string          `json:"model"`
	Messages    []OpenAIMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens"`
	Temperature float64         `json:"temperature"`
	Stream      bool            `json:"stream"`
}

// OpenAIResponse represents a response from OpenAI API
type OpenAIResponse struct {
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

// OpenAIErrorResponse represents an error response from OpenAI API
type OpenAIErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Param   string `json:"param"`
		Code    string `json:"code"`
	} `json:"error"`
}

// NewOpenAIClient creates a new OpenAI API client
func NewOpenAIClient(config *types.AIConfig) (*OpenAIClient, error) {
	if config == nil {
		return nil, fmt.Errorf("configuration is required")
	}

	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://api.openai.com"
	}

	timeout := 30 * time.Second
	baseClient := utils.NewBaseHTTPClient(baseURL, config.APIKey, timeout)

	client := &OpenAIClient{
		BaseHTTPClient: baseClient,
		model:          config.Model,
		maxTokens:      config.MaxTokens,
		temperature:    config.Temperature,
		logger:         utils.NewLogger("OpenAIClient"),
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

	client.logger.Info("OpenAI client created with model: %s", client.model)
	return client, nil
}

// ValidateCredentials validates the OpenAI API credentials
func (c *OpenAIClient) ValidateCredentials(ctx context.Context) error {
	c.logger.Info("Validating OpenAI API credentials")

	// Create a simple test request
	messages := []OpenAIMessage{
		{
			Role:    "user",
			Content: "Hello",
		},
	}

	openaiReq := OpenAIRequest{
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
		var errorResp OpenAIErrorResponse
		if err := json.Unmarshal(resp.Body, &errorResp); err == nil {
			return fmt.Errorf("API error: %s", errorResp.Error.Message)
		}
		return fmt.Errorf("API error: HTTP %d", resp.StatusCode)
	}

	c.logger.Info("OpenAI API credentials validated successfully")
	return nil
}

// GenerateCompletion generates code completion using OpenAI API
func (c *OpenAIClient) GenerateCompletion(ctx context.Context, req types.CompletionRequest) (*types.CompletionResponse, error) {
	c.logger.Info("Generating completion for language: %s", req.Language)

	// Build context-aware prompt
	prompt := c.buildCompletionPrompt(req)

	messages := []OpenAIMessage{
		{
			Role:    "user",
			Content: prompt,
		},
	}

	openaiReq := OpenAIRequest{
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
		return &types.CompletionResponse{
			Suggestions: []string{},
			Confidence:  0.0,
			Error:       fmt.Sprintf("Request failed: %v", err),
		}, nil
	}

	if err := c.ValidateResponse(resp); err != nil {
		c.logger.Error("Invalid response: %v", err)

		// Handle rate limiting specifically
		if resp.StatusCode == 429 {
			return &types.CompletionResponse{
				Suggestions: []string{},
				Confidence:  0.0,
				Error:       "Rate limit exceeded. Please try again later.",
			}, nil
		}

		return &types.CompletionResponse{
			Suggestions: []string{},
			Confidence:  0.0,
			Error:       fmt.Sprintf("API error: %v", err),
		}, nil
	}

	var openaiResp OpenAIResponse
	if err := json.Unmarshal(resp.Body, &openaiResp); err != nil {
		c.logger.Error("Failed to unmarshal response: %v", err)
		return &types.CompletionResponse{
			Suggestions: []string{},
			Confidence:  0.0,
			Error:       "Failed to parse response",
		}, nil
	}

	// Extract suggestions from OpenAI response
	suggestions := c.extractCompletionSuggestions(openaiResp)
	confidence := c.calculateConfidence(openaiResp)

	c.logger.Info("Generated %d completion suggestions", len(suggestions))

	return &types.CompletionResponse{
		Suggestions: suggestions,
		Confidence:  confidence,
	}, nil
}

// GenerateCode generates code using OpenAI API
func (c *OpenAIClient) GenerateCode(ctx context.Context, req types.CodeGenerationRequest) (*types.CodeGenerationResponse, error) {
	c.logger.Info("Generating code for language: %s", req.Language)

	// Build context-aware prompt
	prompt := c.buildCodeGenerationPrompt(req)

	messages := []OpenAIMessage{
		{
			Role:    "user",
			Content: prompt,
		},
	}

	openaiReq := OpenAIRequest{
		Model:       c.model,
		Messages:    messages,
		MaxTokens:   c.maxTokens,
		Temperature: c.temperature,
		Stream:      false,
	}

	reqBody, err := json.Marshal(openaiReq)
	if err != nil {
		c.logger.Error("Failed to marshal code generation request: %v", err)
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
		c.logger.Error("Code generation request failed: %v", err)
		return &types.CodeGenerationResponse{
			Code:  "",
			Error: fmt.Sprintf("Request failed: %v", err),
		}, nil
	}

	if err := c.ValidateResponse(resp); err != nil {
		c.logger.Error("Invalid response: %v", err)

		// Handle rate limiting specifically
		if resp.StatusCode == 429 {
			return &types.CodeGenerationResponse{
				Code:  "",
				Error: "Rate limit exceeded. Please try again later.",
			}, nil
		}

		return &types.CodeGenerationResponse{
			Code:  "",
			Error: fmt.Sprintf("API error: %v", err),
		}, nil
	}

	var openaiResp OpenAIResponse
	if err := json.Unmarshal(resp.Body, &openaiResp); err != nil {
		c.logger.Error("Failed to unmarshal response: %v", err)
		return &types.CodeGenerationResponse{
			Code:  "",
			Error: "Failed to parse response",
		}, nil
	}

	// Extract generated code from OpenAI response
	code := c.extractGeneratedCode(openaiResp)

	c.logger.Info("Generated code with %d characters", len(code))

	return &types.CodeGenerationResponse{
		Code: code,
	}, nil
}

// buildCompletionPrompt builds a context-aware prompt for code completion
func (c *OpenAIClient) buildCompletionPrompt(req types.CompletionRequest) string {
	var prompt strings.Builder

	prompt.WriteString(fmt.Sprintf("You are a code completion assistant for %s. ", req.Language))
	prompt.WriteString("Provide code completions that continue from the cursor position. ")
	prompt.WriteString("Return only the completion text without explanations or markdown formatting.\n\n")

	// Add context information
	if req.Context.CurrentFunction != "" {
		prompt.WriteString(fmt.Sprintf("Current function: %s\n", req.Context.CurrentFunction))
	}

	if len(req.Context.Imports) > 0 {
		prompt.WriteString("Imports:\n")
		for _, imp := range req.Context.Imports {
			prompt.WriteString(fmt.Sprintf("- %s\n", imp))
		}
	}

	if req.Context.ProjectType != "" {
		prompt.WriteString(fmt.Sprintf("Project type: %s\n", req.Context.ProjectType))
	}

	prompt.WriteString("\nCode to complete:\n")

	// Add code before cursor
	beforeCursor := req.Code[:req.Cursor]
	afterCursor := req.Code[req.Cursor:]

	prompt.WriteString(beforeCursor)
	prompt.WriteString("<CURSOR>")
	prompt.WriteString(afterCursor)

	prompt.WriteString("\n\nProvide the completion for <CURSOR> position:")

	return prompt.String()
}

// buildCodeGenerationPrompt builds a context-aware prompt for code generation
func (c *OpenAIClient) buildCodeGenerationPrompt(req types.CodeGenerationRequest) string {
	var prompt strings.Builder

	prompt.WriteString(fmt.Sprintf("You are a code generation assistant for %s. ", req.Language))
	prompt.WriteString("Generate code based on the following prompt. ")
	prompt.WriteString("Return only the code without explanations or markdown formatting.\n\n")

	// Add context information
	if req.Context.CurrentFunction != "" {
		prompt.WriteString(fmt.Sprintf("Current function: %s\n", req.Context.CurrentFunction))
	}

	if len(req.Context.Imports) > 0 {
		prompt.WriteString("Available imports:\n")
		for _, imp := range req.Context.Imports {
			prompt.WriteString(fmt.Sprintf("- %s\n", imp))
		}
	}

	if req.Context.ProjectType != "" {
		prompt.WriteString(fmt.Sprintf("Project type: %s\n", req.Context.ProjectType))
	}

	prompt.WriteString("\nGenerate code for:\n")
	prompt.WriteString(req.Prompt)

	return prompt.String()
}

// extractCompletionSuggestions extracts completion suggestions from OpenAI response
func (c *OpenAIClient) extractCompletionSuggestions(resp OpenAIResponse) []string {
	if len(resp.Choices) == 0 {
		return []string{}
	}

	// Get the text content from the first choice
	text := resp.Choices[0].Message.Content
	if text == "" {
		return []string{}
	}

	// Clean up the response text
	text = strings.TrimSpace(text)

	// Split by lines and filter out empty lines
	lines := strings.Split(text, "\n")
	var suggestions []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			suggestions = append(suggestions, line)
		}
	}

	// If we have multiple lines, treat each as a separate suggestion
	// Otherwise, return the single suggestion
	if len(suggestions) == 0 {
		return []string{text}
	}

	return suggestions
}

// extractGeneratedCode extracts generated code from OpenAI response
func (c *OpenAIClient) extractGeneratedCode(resp OpenAIResponse) string {
	if len(resp.Choices) == 0 {
		return ""
	}

	// Get the text content from the first choice
	text := resp.Choices[0].Message.Content

	// Remove markdown code block formatting if present
	text = strings.TrimPrefix(text, "```")
	if strings.HasPrefix(text, "typescript") || strings.HasPrefix(text, "javascript") ||
		strings.HasPrefix(text, "python") || strings.HasPrefix(text, "go") {
		lines := strings.Split(text, "\n")
		if len(lines) > 1 {
			text = strings.Join(lines[1:], "\n")
		}
	}
	text = strings.TrimSuffix(text, "```")

	return strings.TrimSpace(text)
}

// calculateConfidence calculates confidence score based on OpenAI response
func (c *OpenAIClient) calculateConfidence(resp OpenAIResponse) float64 {
	if len(resp.Choices) == 0 {
		return 0.0
	}

	// Base confidence
	confidence := 0.7

	choice := resp.Choices[0]

	// Adjust based on finish reason
	switch choice.FinishReason {
	case "stop":
		confidence += 0.2
	case "length":
		confidence -= 0.1
	case "content_filter":
		confidence -= 0.3
	}

	// Adjust based on response length
	if len(choice.Message.Content) > 50 {
		confidence += 0.1
	}

	// Ensure confidence is within bounds
	if confidence > 1.0 {
		confidence = 1.0
	}
	if confidence < 0.0 {
		confidence = 0.0
	}

	return confidence
}

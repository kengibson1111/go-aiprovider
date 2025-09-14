package claude

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

// ClaudeClient implements the AIClient interface for Claude API
type ClaudeClient struct {
	*utils.BaseHTTPClient
	model       string
	maxTokens   int
	temperature float64
	logger      *utils.Logger
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
		logger:         utils.NewLogger("ClaudeClient"),
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

// GenerateCompletion generates code completion using Claude API
func (c *ClaudeClient) GenerateCompletion(ctx context.Context, req types.CompletionRequest) (*types.CompletionResponse, error) {
	c.logger.Info("Generating completion for language: %s", req.Language)

	// Build context-aware prompt
	prompt := c.buildCompletionPrompt(req)
	resp, err := c.CallWithPrompt(ctx, prompt)
	if err != nil {
		return &types.CompletionResponse{
			Suggestions: []string{},
			Confidence:  0.0,
			Error:       fmt.Sprintf("ERROR: %v", err),
		}, nil
	}

	var claudeResp ClaudeResponse
	if err := json.Unmarshal(resp, &claudeResp); err != nil {
		c.logger.Error("Failed to unmarshal response: %v", err)
		return &types.CompletionResponse{
			Suggestions: []string{},
			Confidence:  0.0,
			Error:       "Failed to parse response",
		}, nil
	}

	// Extract suggestions from Claude response
	suggestions := c.extractCompletionSuggestions(claudeResp)
	confidence := c.calculateConfidence(claudeResp)

	c.logger.Info("Generated %d completion suggestions", len(suggestions))

	return &types.CompletionResponse{
		Suggestions: suggestions,
		Confidence:  confidence,
	}, nil
}

// GenerateCode generates code using Claude API
func (c *ClaudeClient) GenerateCode(ctx context.Context, req types.CodeGenerationRequest) (*types.CodeGenerationResponse, error) {
	c.logger.Info("Generating code for language: %s", req.Language)

	// Build context-aware prompt
	prompt := c.buildCodeGenerationPrompt(req)
	resp, err := c.CallWithPrompt(ctx, prompt)
	if err != nil {
		return &types.CodeGenerationResponse{
			Code:  "",
			Error: fmt.Sprintf("ERROR: %v", err),
		}, nil
	}

	var claudeResp ClaudeResponse
	if err := json.Unmarshal(resp, &claudeResp); err != nil {
		c.logger.Error("Failed to unmarshal response: %v", err)
		return &types.CodeGenerationResponse{
			Code:  "",
			Error: "Failed to parse response",
		}, nil
	}

	// Extract generated code from Claude response
	code := c.extractGeneratedCode(claudeResp)

	c.logger.Info("Generated code with %d characters", len(code))

	return &types.CodeGenerationResponse{
		Code: code,
	}, nil
}

// buildCompletionPrompt builds a context-aware prompt for code completion
func (c *ClaudeClient) buildCompletionPrompt(req types.CompletionRequest) string {
	var prompt strings.Builder

	prompt.WriteString(fmt.Sprintf("You are a code completion assistant for %s. ", req.Language))
	prompt.WriteString("Provide code completions that continue from the cursor position. ")
	prompt.WriteString("Return only the completion text without explanations.\n\n")

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
func (c *ClaudeClient) buildCodeGenerationPrompt(req types.CodeGenerationRequest) string {
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

// extractCompletionSuggestions extracts completion suggestions from Claude response
func (c *ClaudeClient) extractCompletionSuggestions(resp ClaudeResponse) []string {
	if len(resp.Content) == 0 {
		return []string{}
	}

	// Get the text content from the first content block
	text := resp.Content[0].Text
	if text == "" {
		return []string{}
	}

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

// extractGeneratedCode extracts generated code from Claude response
func (c *ClaudeClient) extractGeneratedCode(resp ClaudeResponse) string {
	if len(resp.Content) == 0 {
		return ""
	}

	// Get the text content from the first content block
	text := resp.Content[0].Text

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

// calculateConfidence calculates confidence score based on Claude response
func (c *ClaudeClient) calculateConfidence(resp ClaudeResponse) float64 {
	// Base confidence
	confidence := 0.7

	// Adjust based on stop reason
	switch resp.StopReason {
	case "end_turn":
		confidence += 0.2
	case "max_tokens":
		confidence -= 0.1
	case "stop_sequence":
		confidence += 0.1
	}

	// Adjust based on response length
	if len(resp.Content) > 0 && len(resp.Content[0].Text) > 50 {
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

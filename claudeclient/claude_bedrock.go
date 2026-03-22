// Package claudeclient provides AWS Bedrock connectivity for Claude models.
//
// This file provides NewClaudeBedrockClient which creates a ClaudeClient configured
// for Amazon Bedrock using the AWS SDK v2. All completion methods (CallWithPrompt,
// CallWithPromptAndVariables, ValidateCredentials) are inherited from ClaudeClient
// in claude_client.go via a custom HTTP round-tripper that handles SigV4 signing
// and Bedrock request/response translation.
//
// # Authentication
//
// AWS credentials are resolved via the default credential chain
// (aws-sdk-go-v2/config.LoadDefaultConfig), which checks in order:
//
//  1. Environment variables (AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, AWS_SESSION_TOKEN)
//  2. Shared credentials file (~/.aws/credentials)
//  3. SSO / AWS Identity Center (via ~/.aws/config)
//  4. IAM instance role (EC2, ECS, Lambda)
//
// # Required Configuration
//
//   - CLAUDE_BEDROCK_REGION: AWS region (e.g., us-east-1)
//   - CLAUDE_BEDROCK_MODEL: Bedrock model ID (e.g., anthropic.claude-sonnet-4-20250514-v1:0)
//
// # Optional Configuration
//
//   - CLAUDE_BEDROCK_ENDPOINT: Custom Bedrock runtime endpoint (defaults to standard regional endpoint)
package claudeclient

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/kengibson1111/go-aiprovider/internal/shared/logging"
	"github.com/kengibson1111/go-aiprovider/internal/shared/utils"
	"github.com/kengibson1111/go-aiprovider/types"
)

// ClaudeBedrockClient wraps the AWS Bedrock runtime client and reuses
// the ClaudeRequest/ClaudeResponse types from claude_client.go.
type ClaudeBedrockClient struct {
	bedrockClient *bedrockruntime.Client
	model         string
	maxTokens     int
	temperature   float64
	logger        *logging.DefaultLogger
}

// BedrockRequest is the request body format expected by Bedrock's Claude models.
// It mirrors ClaudeRequest but omits the "model" field (Bedrock passes model ID
// separately in the InvokeModel call).
type BedrockRequest struct {
	MaxTokens        int             `json:"max_tokens"`
	Temperature      float64         `json:"temperature"`
	Messages         []ClaudeMessage `json:"messages"`
	AnthropicVersion string          `json:"anthropic_version"`
}

// NewClaudeBedrockClient creates a Claude client backed by Amazon Bedrock.
//
// Authentication uses the AWS default credential chain (~/.aws/credentials,
// environment variables, SSO, or IAM role). No API key is needed.
//
// Parameters:
//   - config: AIConfig with Model set to the Bedrock model ID.
//     BaseURL is optional and overrides the Bedrock endpoint.
//
// Environment variables:
//   - CLAUDE_BEDROCK_REGION (required): AWS region
//   - CLAUDE_BEDROCK_MODEL (required if config.Model is empty): Bedrock model ID
//   - CLAUDE_BEDROCK_ENDPOINT (optional): Custom Bedrock runtime endpoint
func NewClaudeBedrockClient(aiConfig *types.AIConfig) (*ClaudeBedrockClient, error) {
	if aiConfig == nil {
		return nil, fmt.Errorf("configuration is required")
	}

	region := strings.TrimSpace(os.Getenv("CLAUDE_BEDROCK_REGION"))
	if region == "" {
		return nil, fmt.Errorf("CLAUDE_BEDROCK_REGION environment variable is required")
	}

	model := aiConfig.Model
	if model == "" {
		model = strings.TrimSpace(os.Getenv("CLAUDE_BEDROCK_MODEL"))
	}
	if model == "" {
		return nil, fmt.Errorf("CLAUDE_BEDROCK_MODEL environment variable is required (or set AIConfig.Model)")
	}

	logger := logging.NewDefaultLogger()

	// Load AWS config using the default credential chain
	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Build Bedrock client options
	var brOpts []func(*bedrockruntime.Options)

	// Optional: override the default regional endpoint
	if endpoint := strings.TrimSpace(os.Getenv("CLAUDE_BEDROCK_ENDPOINT")); endpoint != "" {
		brOpts = append(brOpts, func(o *bedrockruntime.Options) {
			o.BaseEndpoint = aws.String(endpoint)
		})
	}

	brClient := bedrockruntime.NewFromConfig(cfg, brOpts...)

	maxTokens := aiConfig.MaxTokens
	if maxTokens == 0 {
		maxTokens = 1000
	}

	temperature := aiConfig.Temperature
	if temperature == 0.0 {
		temperature = 0.7
	}

	client := &ClaudeBedrockClient{
		bedrockClient: brClient,
		model:         model,
		maxTokens:     maxTokens,
		temperature:   temperature,
		logger:        logger,
	}

	logger.Info("Claude Bedrock client created with model: %s, region: %s", model, region)
	return client, nil
}

// ValidateCredentials validates AWS credentials and Bedrock model access
// by sending a minimal prompt to the model.
func (c *ClaudeBedrockClient) ValidateCredentials(ctx context.Context) error {
	c.logger.Info("Validating Claude Bedrock credentials")

	_, err := c.invokeModel(ctx, []ClaudeMessage{
		{Role: "user", Content: "Hello"},
	}, 10, 0.1)
	if err != nil {
		c.logger.Error("Credential validation failed: %v", err)
		return fmt.Errorf("credential validation failed: %w", err)
	}

	c.logger.Info("Claude Bedrock credentials validated successfully")
	return nil
}

// CallWithPrompt sends a prompt to Claude via Bedrock and returns the raw response.
func (c *ClaudeBedrockClient) CallWithPrompt(ctx context.Context, prompt string) ([]byte, error) {
	messages := []ClaudeMessage{
		{Role: "user", Content: prompt},
	}

	return c.invokeModel(ctx, messages, c.maxTokens, c.temperature)
}

// CallWithPromptAndVariables sends a prompt template with variable substitution
// to Claude via Bedrock. Reuses the same template processing as the direct
// Claude client.
func (c *ClaudeBedrockClient) CallWithPromptAndVariables(ctx context.Context, prompt string, variablesJSON string) ([]byte, error) {
	c.logger.Info("Processing prompt with variables for Claude Bedrock")

	processedPrompt, err := utils.SubstituteVariables(prompt, variablesJSON)
	if err != nil {
		c.logger.Error("Variable substitution failed: %v", err)
		return nil, fmt.Errorf("variable substitution failed: %w", err)
	}

	c.logger.Debug("Variables substituted successfully, calling Claude Bedrock")
	return c.CallWithPrompt(ctx, processedPrompt)
}

// invokeModel is the shared implementation that calls Bedrock's InvokeModel API.
// It builds the Bedrock-specific request body, invokes the model, and returns
// the raw response bytes (same ClaudeResponse JSON format).
func (c *ClaudeBedrockClient) invokeModel(ctx context.Context, messages []ClaudeMessage, maxTokens int, temperature float64) ([]byte, error) {
	reqBody := BedrockRequest{
		MaxTokens:        maxTokens,
		Temperature:      temperature,
		Messages:         messages,
		AnthropicVersion: "bedrock-2023-05-31",
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		c.logger.Error("Failed to marshal Bedrock request: %v", err)
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	output, err := c.bedrockClient.InvokeModel(ctx, &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String(c.model),
		ContentType: aws.String("application/json"),
		Accept:      aws.String("application/json"),
		Body:        bodyBytes,
	})
	if err != nil {
		c.logger.Error("Bedrock InvokeModel failed: %v", err)
		return nil, fmt.Errorf("bedrock request failed: %w", err)
	}

	return output.Body, nil
}

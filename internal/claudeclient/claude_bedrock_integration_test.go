//go:build integration

package claudeclient

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/kengibson1111/go-aiprovider/internal/shared/testutil"
	"github.com/kengibson1111/go-aiprovider/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// ClaudeBedrockIntegrationTestSuite tests the Claude Bedrock client against the real AWS Bedrock API
type ClaudeBedrockIntegrationTestSuite struct {
	suite.Suite
	cleanupCwd func()
	client     *ClaudeBedrockClient
}

func TestClaudeBedrockIntegrationTestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	suite.Run(t, new(ClaudeBedrockIntegrationTestSuite))
}

func (s *ClaudeBedrockIntegrationTestSuite) SetupSuite() {
	testutil.SetupEnvironment(s.T(), "../../")
	s.cleanupCwd = testutil.SetupCurrentDirectory(s.T(), "../../")

	region := os.Getenv("CLAUDE_BEDROCK_REGION")
	if region == "" {
		s.T().Skip("CLAUDE_BEDROCK_REGION not set, skipping Claude Bedrock integration tests")
	}

	model := os.Getenv("CLAUDE_BEDROCK_MODEL")
	if model == "" {
		s.T().Skip("CLAUDE_BEDROCK_MODEL not set, skipping Claude Bedrock integration tests")
	}

	config := &types.AIConfig{
		Provider: types.ProviderClaudeBedrock,
		Model:    model,
	}

	client, err := NewClaudeBedrockClient(config)
	require.NoError(s.T(), err, "Failed to create Claude Bedrock client")
	s.client = client
}

func (s *ClaudeBedrockIntegrationTestSuite) TearDownSuite() {
	if s.cleanupCwd != nil {
		s.cleanupCwd()
	}
}

// TestValidateCredentials verifies that valid AWS credentials and Bedrock model access work
func (s *ClaudeBedrockIntegrationTestSuite) TestValidateCredentials() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := s.client.ValidateCredentials(ctx)
	assert.NoError(s.T(), err, "Valid AWS credentials should pass validation")
}

// TestCallWithPrompt verifies a basic prompt call returns a valid JSON response
func (s *ClaudeBedrockIntegrationTestSuite) TestCallWithPrompt() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	response, err := s.client.CallWithPrompt(ctx, "Reply with only the word 'hello'.")
	require.NoError(s.T(), err, "CallWithPrompt should succeed")
	require.NotNil(s.T(), response, "Response should not be nil")

	var result ClaudeResponse
	err = json.Unmarshal(response, &result)
	assert.NoError(s.T(), err, "Response should be valid JSON matching ClaudeResponse")

	assert.NotEmpty(s.T(), result.ID, "Response should contain an ID")
	assert.NotEmpty(s.T(), result.Content, "Response should contain content")
	assert.NotEmpty(s.T(), result.Model, "Response should contain model")
	assert.Equal(s.T(), "assistant", result.Role, "Response role should be assistant")
}

// TestCallWithPrompt_ResponseContent verifies the response content structure
func (s *ClaudeBedrockIntegrationTestSuite) TestCallWithPrompt_ResponseContent() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	response, err := s.client.CallWithPrompt(ctx, "What is 2+2? Reply with only the number.")
	require.NoError(s.T(), err)

	var result ClaudeResponse
	err = json.Unmarshal(response, &result)
	require.NoError(s.T(), err)

	require.NotEmpty(s.T(), result.Content, "Response should have content blocks")
	assert.Equal(s.T(), "text", result.Content[0].Type, "Content block should be of type text")
	assert.Contains(s.T(), result.Content[0].Text, "4", "Response should contain the answer '4'")
}

// TestCallWithPrompt_UsageTracking verifies that usage information is returned
func (s *ClaudeBedrockIntegrationTestSuite) TestCallWithPrompt_UsageTracking() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	response, err := s.client.CallWithPrompt(ctx, "Say 'hi'.")
	require.NoError(s.T(), err)

	var result ClaudeResponse
	err = json.Unmarshal(response, &result)
	require.NoError(s.T(), err)

	assert.Greater(s.T(), result.Usage.InputTokens, 0,
		"Input tokens should be greater than 0")
	assert.Greater(s.T(), result.Usage.OutputTokens, 0,
		"Output tokens should be greater than 0")
}

// TestCallWithPromptAndVariables verifies template variable substitution
func (s *ClaudeBedrockIntegrationTestSuite) TestCallWithPromptAndVariables() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	prompt := "You are a {{role}}. Reply with only: I am a {{role}}."
	variables := `{"role": "translator"}`

	response, err := s.client.CallWithPromptAndVariables(ctx, prompt, variables)
	require.NoError(s.T(), err, "CallWithPromptAndVariables should succeed")
	require.NotNil(s.T(), response, "Response should not be nil")

	var result ClaudeResponse
	err = json.Unmarshal(response, &result)
	assert.NoError(s.T(), err, "Response should be valid JSON")
	assert.NotEmpty(s.T(), result.Content, "Response should contain content")
}

// TestCallWithPromptAndVariables_InvalidJSON verifies error on bad variable JSON
func (s *ClaudeBedrockIntegrationTestSuite) TestCallWithPromptAndVariables_InvalidJSON() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	prompt := "Hello {{name}}"
	variables := `{invalid json}`

	_, err := s.client.CallWithPromptAndVariables(ctx, prompt, variables)
	assert.Error(s.T(), err, "Should fail with invalid JSON variables")
	assert.Contains(s.T(), err.Error(), "variable substitution failed",
		"Error should indicate variable substitution failure")
}

// TestCallWithPrompt_ContextCancellation verifies that cancelled contexts are handled
func (s *ClaudeBedrockIntegrationTestSuite) TestCallWithPrompt_ContextCancellation() {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := s.client.CallWithPrompt(ctx, "This should not complete")
	assert.Error(s.T(), err, "Cancelled context should produce an error")
}

// TestNewClaudeBedrockClient_Defaults verifies default values are applied correctly
func (s *ClaudeBedrockIntegrationTestSuite) TestNewClaudeBedrockClient_Defaults() {
	config := &types.AIConfig{
		Provider: types.ProviderClaudeBedrock,
	}

	client, err := NewClaudeBedrockClient(config)
	require.NoError(s.T(), err, "Client creation with defaults should succeed")

	assert.Equal(s.T(), os.Getenv("CLAUDE_BEDROCK_MODEL"), client.model,
		"Model should come from CLAUDE_BEDROCK_MODEL env var")
	assert.Equal(s.T(), 1000, client.maxTokens,
		"Default maxTokens should be 1000")
	assert.InDelta(s.T(), 0.7, client.temperature, 0.001,
		"Default temperature should be 0.7")
}

// TestNewClaudeBedrockClient_CustomConfig verifies custom config values are respected
func (s *ClaudeBedrockIntegrationTestSuite) TestNewClaudeBedrockClient_CustomConfig() {
	config := &types.AIConfig{
		Provider:    types.ProviderClaudeBedrock,
		Model:       os.Getenv("CLAUDE_BEDROCK_MODEL"),
		MaxTokens:   2000,
		Temperature: 0.5,
	}

	client, err := NewClaudeBedrockClient(config)
	require.NoError(s.T(), err, "Client creation with custom config should succeed")

	assert.Equal(s.T(), os.Getenv("CLAUDE_BEDROCK_MODEL"), client.model)
	assert.Equal(s.T(), 2000, client.maxTokens)
	assert.InDelta(s.T(), 0.5, client.temperature, 0.001)
}

// TestNewClaudeBedrockClient_NilConfig verifies nil config is rejected
func (s *ClaudeBedrockIntegrationTestSuite) TestNewClaudeBedrockClient_NilConfig() {
	_, err := NewClaudeBedrockClient(nil)
	assert.Error(s.T(), err, "Nil config should produce an error")
	assert.Contains(s.T(), err.Error(), "configuration is required")
}

// TestCallWithPrompt_StopReason verifies the response includes a stop reason
func (s *ClaudeBedrockIntegrationTestSuite) TestCallWithPrompt_StopReason() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	response, err := s.client.CallWithPrompt(ctx, "Say 'done'.")
	require.NoError(s.T(), err)

	var result ClaudeResponse
	err = json.Unmarshal(response, &result)
	require.NoError(s.T(), err)

	assert.NotEmpty(s.T(), result.StopReason, "Response should have a stop reason")
	assert.Equal(s.T(), "end_turn", result.StopReason,
		"Stop reason should be end_turn for a complete response")
}

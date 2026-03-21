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

// ClaudeClientIntegrationTestSuite tests the Claude client against the real Claude API
type ClaudeClientIntegrationTestSuite struct {
	suite.Suite
	cleanupCwd func()
	client     *ClaudeClient
}

func TestClaudeClientIntegrationTestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	suite.Run(t, new(ClaudeClientIntegrationTestSuite))
}

func (s *ClaudeClientIntegrationTestSuite) SetupSuite() {
	testutil.SetupEnvironment(s.T(), "../")
	s.cleanupCwd = testutil.SetupCurrentDirectory(s.T(), "../")

	apiKey := os.Getenv("CLAUDE_API_KEY")
	if apiKey == "" {
		s.T().Skip("CLAUDE_API_KEY not set, skipping Claude integration tests")
	}

	model := os.Getenv("CLAUDE_MODEL")
	if model == "" {
		model = "claude-sonnet-4-6"
	}

	config := &types.AIConfig{
		Provider: "claude",
		APIKey:   apiKey,
		BaseURL:  os.Getenv("CLAUDE_API_ENDPOINT"),
		Model:    model,
	}

	client, err := NewClaudeClient(config)
	require.NoError(s.T(), err, "Failed to create Claude client")
	s.client = client
}

func (s *ClaudeClientIntegrationTestSuite) TearDownSuite() {
	if s.cleanupCwd != nil {
		s.cleanupCwd()
	}
}

// TestValidateCredentials verifies that valid API credentials pass validation
func (s *ClaudeClientIntegrationTestSuite) TestValidateCredentials() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := s.client.ValidateCredentials(ctx)
	assert.NoError(s.T(), err, "Valid credentials should pass validation")
}

// TestValidateCredentials_InvalidKey verifies that an invalid API key is rejected
func (s *ClaudeClientIntegrationTestSuite) TestValidateCredentials_InvalidKey() {
	config := &types.AIConfig{
		Provider: "claude",
		APIKey:   "sk-ant-invalid-key-for-testing",
		Model:    s.client.model,
	}
	invalidClient, err := NewClaudeClient(config)
	require.NoError(s.T(), err, "Client creation should succeed even with invalid key")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = invalidClient.ValidateCredentials(ctx)
	assert.Error(s.T(), err, "Invalid credentials should fail validation")
}

// TestCallWithPrompt verifies a basic prompt call returns a valid JSON response
func (s *ClaudeClientIntegrationTestSuite) TestCallWithPrompt() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	response, err := s.client.CallWithPrompt(ctx, "Reply with only the word 'hello'.")
	require.NoError(s.T(), err, "CallWithPrompt should succeed")
	require.NotNil(s.T(), response, "Response should not be nil")

	// Verify the response is valid JSON
	var result ClaudeResponse
	err = json.Unmarshal(response, &result)
	assert.NoError(s.T(), err, "Response should be valid JSON matching ClaudeResponse")

	// Verify expected fields exist
	assert.NotEmpty(s.T(), result.ID, "Response should contain an ID")
	assert.NotEmpty(s.T(), result.Content, "Response should contain content")
	assert.NotEmpty(s.T(), result.Model, "Response should contain model")
	assert.Equal(s.T(), "assistant", result.Role, "Response role should be assistant")
}

// TestCallWithPrompt_ResponseContent verifies the response content structure
func (s *ClaudeClientIntegrationTestSuite) TestCallWithPrompt_ResponseContent() {
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
func (s *ClaudeClientIntegrationTestSuite) TestCallWithPrompt_UsageTracking() {
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
func (s *ClaudeClientIntegrationTestSuite) TestCallWithPromptAndVariables() {
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
func (s *ClaudeClientIntegrationTestSuite) TestCallWithPromptAndVariables_InvalidJSON() {
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
func (s *ClaudeClientIntegrationTestSuite) TestCallWithPrompt_ContextCancellation() {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := s.client.CallWithPrompt(ctx, "This should not complete")
	assert.Error(s.T(), err, "Cancelled context should produce an error")
}

// TestNewClaudeClient_Defaults verifies default values are applied correctly
func (s *ClaudeClientIntegrationTestSuite) TestNewClaudeClient_Defaults() {
	apiKey := os.Getenv("CLAUDE_API_KEY")

	config := &types.AIConfig{
		Provider: "claude",
		APIKey:   apiKey,
		BaseURL:  os.Getenv("CLAUDE_API_ENDPOINT"),
	}

	client, err := NewClaudeClient(config)
	require.NoError(s.T(), err, "Client creation with defaults should succeed")

	assert.Equal(s.T(), "claude-sonnet-4-6", client.model,
		"Default model should be claude-sonnet-4-6")
	assert.Equal(s.T(), 1000, client.maxTokens,
		"Default maxTokens should be 1000")
	assert.InDelta(s.T(), 0.7, client.temperature, 0.001,
		"Default temperature should be 0.7")
}

// TestNewClaudeClient_CustomConfig verifies custom config values are respected
func (s *ClaudeClientIntegrationTestSuite) TestNewClaudeClient_CustomConfig() {
	apiKey := os.Getenv("CLAUDE_API_KEY")

	config := &types.AIConfig{
		Provider:    "claude",
		APIKey:      apiKey,
		BaseURL:     os.Getenv("CLAUDE_API_ENDPOINT"),
		Model:       "claude-sonnet-4-6",
		MaxTokens:   2000,
		Temperature: 0.5,
	}

	client, err := NewClaudeClient(config)
	require.NoError(s.T(), err, "Client creation with custom config should succeed")

	assert.Equal(s.T(), "claude-sonnet-4-6", client.model)
	assert.Equal(s.T(), 2000, client.maxTokens)
	assert.InDelta(s.T(), 0.5, client.temperature, 0.001)
}

// TestNewClaudeClient_NilConfig verifies nil config is rejected
func (s *ClaudeClientIntegrationTestSuite) TestNewClaudeClient_NilConfig() {
	_, err := NewClaudeClient(nil)
	assert.Error(s.T(), err, "Nil config should produce an error")
	assert.Contains(s.T(), err.Error(), "configuration is required")
}

// TestNewClaudeClient_CustomBaseURL verifies custom base URL is accepted without error
func (s *ClaudeClientIntegrationTestSuite) TestNewClaudeClient_CustomBaseURL() {
	apiKey := os.Getenv("CLAUDE_API_KEY")

	config := &types.AIConfig{
		Provider: "claude",
		APIKey:   apiKey,
		BaseURL:  "https://custom.anthropic.com",
	}

	client, err := NewClaudeClient(config)
	require.NoError(s.T(), err, "Client creation with custom base URL should succeed")
	assert.NotNil(s.T(), client, "Client should not be nil")
}

// TestCallWithPrompt_StopReason verifies the response includes a stop reason
func (s *ClaudeClientIntegrationTestSuite) TestCallWithPrompt_StopReason() {
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

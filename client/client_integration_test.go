//go:build integration
// +build integration

package client

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

// ClientFactoryIntegrationTestSuite tests the ClientFactory against real AI provider APIs
type ClientFactoryIntegrationTestSuite struct {
	suite.Suite
	cleanupCwd func()
	factory    *ClientFactory
}

func TestClientFactoryIntegrationTestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	suite.Run(t, new(ClientFactoryIntegrationTestSuite))
}

func (s *ClientFactoryIntegrationTestSuite) SetupSuite() {
	testutil.SetupEnvironment(s.T(), "../")
	s.cleanupCwd = testutil.SetupCurrentDirectory(s.T(), "../")
	s.factory = NewClientFactory()
}

func (s *ClientFactoryIntegrationTestSuite) TearDownSuite() {
	if s.cleanupCwd != nil {
		s.cleanupCwd()
	}
}

// --- Factory Creation Tests ---

// TestNewClientFactory verifies the factory is created successfully
func (s *ClientFactoryIntegrationTestSuite) TestNewClientFactory() {
	factory := NewClientFactory()
	assert.NotNil(s.T(), factory, "Factory should not be nil")
}

// TestCreateClient_NilConfig verifies nil config is rejected
func (s *ClientFactoryIntegrationTestSuite) TestCreateClient_NilConfig() {
	_, err := s.factory.CreateClient(nil)
	assert.Error(s.T(), err, "Nil config should produce an error")
	assert.Contains(s.T(), err.Error(), "configuration is required")
}

// TestCreateClient_UnsupportedProvider verifies unsupported providers are rejected
func (s *ClientFactoryIntegrationTestSuite) TestCreateClient_UnsupportedProvider() {
	config := &types.AIConfig{
		Provider: "unsupported-provider",
		APIKey:   "some-key",
	}

	_, err := s.factory.CreateClient(config)
	assert.Error(s.T(), err, "Unsupported provider should produce an error")
	assert.Contains(s.T(), err.Error(), "unsupported provider")
}

// TestCreateClient_CaseInsensitiveProvider verifies provider name is case-insensitive
func (s *ClientFactoryIntegrationTestSuite) TestCreateClient_CaseInsensitiveProvider() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		s.T().Skip("OPENAI_API_KEY not set, skipping test")
	}

	testCases := []struct {
		name     string
		provider string
	}{
		{"lowercase", "openai"},
		{"uppercase", "OPENAI"},
		{"mixed_case", "OpenAI"},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			config := &types.AIConfig{
				Provider: tc.provider,
				APIKey:   apiKey,
				BaseURL:  os.Getenv("OPENAI_API_ENDPOINT"),
			}
			client, err := s.factory.CreateClient(config)
			require.NoError(s.T(), err, "Provider %q should be accepted", tc.provider)
			assert.NotNil(s.T(), client, "Client should not be nil for provider %q", tc.provider)
		})
	}
}

// --- Claude Provider Tests ---

// TestCreateClient_Claude verifies a Claude client can be created and used
func (s *ClientFactoryIntegrationTestSuite) TestCreateClient_Claude() {
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

	client, err := s.factory.CreateClient(config)
	require.NoError(s.T(), err, "CreateClient for Claude should succeed")
	require.NotNil(s.T(), client, "Claude client should not be nil")
}

// TestCreateClient_Claude_ValidateCredentials verifies credential validation through the factory
func (s *ClientFactoryIntegrationTestSuite) TestCreateClient_Claude_ValidateCredentials() {
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

	client, err := s.factory.CreateClient(config)
	require.NoError(s.T(), err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = client.ValidateCredentials(ctx)
	assert.NoError(s.T(), err, "Valid Claude credentials should pass validation")
}

// TestCreateClient_Claude_CallWithPrompt verifies a prompt call through the factory-created client
func (s *ClientFactoryIntegrationTestSuite) TestCreateClient_Claude_CallWithPrompt() {
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

	client, err := s.factory.CreateClient(config)
	require.NoError(s.T(), err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	response, err := client.CallWithPrompt(ctx, "Reply with only the word 'hello'.")
	require.NoError(s.T(), err, "CallWithPrompt should succeed")
	require.NotNil(s.T(), response, "Response should not be nil")

	// Verify the response is valid JSON
	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	assert.NoError(s.T(), err, "Response should be valid JSON")
	assert.Contains(s.T(), result, "content", "Claude response should contain content")
	assert.Contains(s.T(), result, "model", "Claude response should contain model")
}

// TestCreateClient_Claude_CallWithPromptAndVariables verifies template variable substitution
func (s *ClientFactoryIntegrationTestSuite) TestCreateClient_Claude_CallWithPromptAndVariables() {
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

	client, err := s.factory.CreateClient(config)
	require.NoError(s.T(), err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	prompt := "You are a {{role}}. Reply with only: I am a {{role}}."
	variables := `{"role": "translator"}`

	response, err := client.CallWithPromptAndVariables(ctx, prompt, variables)
	require.NoError(s.T(), err, "CallWithPromptAndVariables should succeed")
	require.NotNil(s.T(), response, "Response should not be nil")

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	assert.NoError(s.T(), err, "Response should be valid JSON")
	assert.Contains(s.T(), result, "content", "Response should contain content")
}

// TestCreateClient_Claude_InvalidCredentials verifies invalid credentials are rejected
func (s *ClientFactoryIntegrationTestSuite) TestCreateClient_Claude_InvalidCredentials() {
	config := &types.AIConfig{
		Provider: "claude",
		APIKey:   "sk-ant-invalid-key-for-testing",
		Model:    "claude-sonnet-4-6",
	}

	client, err := s.factory.CreateClient(config)
	require.NoError(s.T(), err, "Client creation should succeed even with invalid key")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = client.ValidateCredentials(ctx)
	assert.Error(s.T(), err, "Invalid credentials should fail validation")
}

// TestCreateClient_Claude_ContextCancellation verifies cancelled contexts are handled
func (s *ClientFactoryIntegrationTestSuite) TestCreateClient_Claude_ContextCancellation() {
	apiKey := os.Getenv("CLAUDE_API_KEY")
	if apiKey == "" {
		s.T().Skip("CLAUDE_API_KEY not set, skipping Claude integration tests")
	}

	config := &types.AIConfig{
		Provider: "claude",
		APIKey:   apiKey,
		BaseURL:  os.Getenv("CLAUDE_API_ENDPOINT"),
		Model:    "claude-sonnet-4-6",
	}

	client, err := s.factory.CreateClient(config)
	require.NoError(s.T(), err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err = client.CallWithPrompt(ctx, "This should not complete")
	assert.Error(s.T(), err, "Cancelled context should produce an error")
}

// --- OpenAI Provider Tests ---

// TestCreateClient_OpenAI verifies an OpenAI client can be created and used
func (s *ClientFactoryIntegrationTestSuite) TestCreateClient_OpenAI() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		s.T().Skip("OPENAI_API_KEY not set, skipping OpenAI integration tests")
	}

	config := &types.AIConfig{
		Provider: "openai",
		APIKey:   apiKey,
		BaseURL:  os.Getenv("OPENAI_API_ENDPOINT"),
		Model:    "gpt-5.4-mini",
	}

	client, err := s.factory.CreateClient(config)
	require.NoError(s.T(), err, "CreateClient for OpenAI should succeed")
	require.NotNil(s.T(), client, "OpenAI client should not be nil")
}

// TestCreateClient_OpenAI_ValidateCredentials verifies credential validation through the factory
func (s *ClientFactoryIntegrationTestSuite) TestCreateClient_OpenAI_ValidateCredentials() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		s.T().Skip("OPENAI_API_KEY not set, skipping OpenAI integration tests")
	}

	config := &types.AIConfig{
		Provider: "openai",
		APIKey:   apiKey,
		BaseURL:  os.Getenv("OPENAI_API_ENDPOINT"),
		Model:    "gpt-5.4-mini",
	}

	client, err := s.factory.CreateClient(config)
	require.NoError(s.T(), err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = client.ValidateCredentials(ctx)
	assert.NoError(s.T(), err, "Valid OpenAI credentials should pass validation")
}

// TestCreateClient_OpenAI_CallWithPrompt verifies a prompt call through the factory-created client
func (s *ClientFactoryIntegrationTestSuite) TestCreateClient_OpenAI_CallWithPrompt() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		s.T().Skip("OPENAI_API_KEY not set, skipping OpenAI integration tests")
	}

	config := &types.AIConfig{
		Provider: "openai",
		APIKey:   apiKey,
		BaseURL:  os.Getenv("OPENAI_API_ENDPOINT"),
		Model:    "gpt-5.4-mini",
	}

	client, err := s.factory.CreateClient(config)
	require.NoError(s.T(), err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	response, err := client.CallWithPrompt(ctx, "Reply with only the word 'hello'.")
	require.NoError(s.T(), err, "CallWithPrompt should succeed")
	require.NotNil(s.T(), response, "Response should not be nil")

	// Verify the response is valid JSON
	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	assert.NoError(s.T(), err, "Response should be valid JSON")
	assert.Contains(s.T(), result, "choices", "OpenAI response should contain choices")
	assert.Contains(s.T(), result, "model", "OpenAI response should contain model")
	assert.Contains(s.T(), result, "usage", "OpenAI response should contain usage")
}

// TestCreateClient_OpenAI_CallWithPromptAndVariables verifies template variable substitution
func (s *ClientFactoryIntegrationTestSuite) TestCreateClient_OpenAI_CallWithPromptAndVariables() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		s.T().Skip("OPENAI_API_KEY not set, skipping OpenAI integration tests")
	}

	config := &types.AIConfig{
		Provider: "openai",
		APIKey:   apiKey,
		BaseURL:  os.Getenv("OPENAI_API_ENDPOINT"),
		Model:    "gpt-5.4-mini",
	}

	client, err := s.factory.CreateClient(config)
	require.NoError(s.T(), err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	prompt := "You are a {{role}}. Reply with only: I am a {{role}}."
	variables := `{"role": "translator"}`

	response, err := client.CallWithPromptAndVariables(ctx, prompt, variables)
	require.NoError(s.T(), err, "CallWithPromptAndVariables should succeed")
	require.NotNil(s.T(), response, "Response should not be nil")

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	assert.NoError(s.T(), err, "Response should be valid JSON")
	assert.Contains(s.T(), result, "choices", "Response should contain choices")
}

// TestCreateClient_OpenAI_InvalidCredentials verifies invalid credentials are rejected
func (s *ClientFactoryIntegrationTestSuite) TestCreateClient_OpenAI_InvalidCredentials() {
	config := &types.AIConfig{
		Provider: "openai",
		APIKey:   "sk-invalid-key-for-testing",
		Model:    "gpt-5.4-mini",
	}

	client, err := s.factory.CreateClient(config)
	require.NoError(s.T(), err, "Client creation should succeed even with invalid key")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = client.ValidateCredentials(ctx)
	assert.Error(s.T(), err, "Invalid credentials should fail validation")
}

// TestCreateClient_OpenAI_ContextCancellation verifies cancelled contexts are handled
func (s *ClientFactoryIntegrationTestSuite) TestCreateClient_OpenAI_ContextCancellation() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		s.T().Skip("OPENAI_API_KEY not set, skipping OpenAI integration tests")
	}

	config := &types.AIConfig{
		Provider: "openai",
		APIKey:   apiKey,
		BaseURL:  os.Getenv("OPENAI_API_ENDPOINT"),
		Model:    "gpt-5.4-mini",
	}

	client, err := s.factory.CreateClient(config)
	require.NoError(s.T(), err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err = client.CallWithPrompt(ctx, "This should not complete")
	assert.Error(s.T(), err, "Cancelled context should produce an error")
}

// --- OpenAI Azure Provider Tests ---

// TestCreateClient_OpenAIAzure verifies an OpenAI Azure client can be created and used
func (s *ClientFactoryIntegrationTestSuite) TestCreateClient_OpenAIAzure() {
	endpoint := os.Getenv("OPENAI_AZURE_ENDPOINT")
	if endpoint == "" {
		s.T().Skip("OPENAI_AZURE_ENDPOINT not set, skipping OpenAI Azure integration tests")
	}

	config := &types.AIConfig{
		Provider: "openai-azure",
	}

	client, err := s.factory.CreateClient(config)
	require.NoError(s.T(), err, "CreateClient for OpenAI Azure should succeed")
	require.NotNil(s.T(), client, "OpenAI Azure client should not be nil")
}

// TestCreateClient_OpenAIAzure_ValidateCredentials verifies credential validation through the factory
func (s *ClientFactoryIntegrationTestSuite) TestCreateClient_OpenAIAzure_ValidateCredentials() {
	endpoint := os.Getenv("OPENAI_AZURE_ENDPOINT")
	if endpoint == "" {
		s.T().Skip("OPENAI_AZURE_ENDPOINT not set, skipping OpenAI Azure integration tests")
	}

	config := &types.AIConfig{
		Provider: "openai-azure",
	}

	client, err := s.factory.CreateClient(config)
	require.NoError(s.T(), err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = client.ValidateCredentials(ctx)
	assert.NoError(s.T(), err, "Valid Azure credentials should pass validation")
}

// TestCreateClient_OpenAIAzure_CallWithPrompt verifies a prompt call through the factory-created client
func (s *ClientFactoryIntegrationTestSuite) TestCreateClient_OpenAIAzure_CallWithPrompt() {
	endpoint := os.Getenv("OPENAI_AZURE_ENDPOINT")
	if endpoint == "" {
		s.T().Skip("OPENAI_AZURE_ENDPOINT not set, skipping OpenAI Azure integration tests")
	}

	config := &types.AIConfig{
		Provider: "openai-azure",
	}

	client, err := s.factory.CreateClient(config)
	require.NoError(s.T(), err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	response, err := client.CallWithPrompt(ctx, "Reply with only the word 'hello'.")
	require.NoError(s.T(), err, "CallWithPrompt should succeed")
	require.NotNil(s.T(), response, "Response should not be nil")

	// Verify the response is valid JSON
	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	assert.NoError(s.T(), err, "Response should be valid JSON")
	assert.Contains(s.T(), result, "choices", "OpenAI Azure response should contain choices")
	assert.Contains(s.T(), result, "model", "OpenAI Azure response should contain model")
	assert.Contains(s.T(), result, "usage", "OpenAI Azure response should contain usage")
}

// TestCreateClient_OpenAIAzure_CallWithPromptAndVariables verifies template variable substitution
func (s *ClientFactoryIntegrationTestSuite) TestCreateClient_OpenAIAzure_CallWithPromptAndVariables() {
	endpoint := os.Getenv("OPENAI_AZURE_ENDPOINT")
	if endpoint == "" {
		s.T().Skip("OPENAI_AZURE_ENDPOINT not set, skipping OpenAI Azure integration tests")
	}

	config := &types.AIConfig{
		Provider: "openai-azure",
	}

	client, err := s.factory.CreateClient(config)
	require.NoError(s.T(), err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	prompt := "You are a {{role}}. Reply with only: I am a {{role}}."
	variables := `{"role": "translator"}`

	response, err := client.CallWithPromptAndVariables(ctx, prompt, variables)
	require.NoError(s.T(), err, "CallWithPromptAndVariables should succeed")
	require.NotNil(s.T(), response, "Response should not be nil")

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	assert.NoError(s.T(), err, "Response should be valid JSON")
	assert.Contains(s.T(), result, "choices", "Response should contain choices")
}

// TestCreateClient_OpenAIAzure_ContextCancellation verifies cancelled contexts are handled
func (s *ClientFactoryIntegrationTestSuite) TestCreateClient_OpenAIAzure_ContextCancellation() {
	endpoint := os.Getenv("OPENAI_AZURE_ENDPOINT")
	if endpoint == "" {
		s.T().Skip("OPENAI_AZURE_ENDPOINT not set, skipping OpenAI Azure integration tests")
	}

	config := &types.AIConfig{
		Provider: "openai-azure",
	}

	client, err := s.factory.CreateClient(config)
	require.NoError(s.T(), err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err = client.CallWithPrompt(ctx, "This should not complete")
	assert.Error(s.T(), err, "Cancelled context should produce an error")
}

// --- Shared Error Tests ---

// TestCallWithPromptAndVariables_InvalidJSON verifies error on bad variable JSON for both providers
func (s *ClientFactoryIntegrationTestSuite) TestCallWithPromptAndVariables_InvalidJSON() {
	// Test with whichever provider is available
	var client AIClient
	var err error

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey != "" {
		client, err = s.factory.CreateClient(&types.AIConfig{
			Provider: "openai",
			APIKey:   apiKey,
			BaseURL:  os.Getenv("OPENAI_API_ENDPOINT"),
			Model:    "gpt-5.4-mini",
		})
		require.NoError(s.T(), err)
	} else {
		apiKey = os.Getenv("CLAUDE_API_KEY")
		if apiKey == "" {
			s.T().Skip("No API keys set, skipping test")
		}
		client, err = s.factory.CreateClient(&types.AIConfig{
			Provider: "claude",
			APIKey:   apiKey,
			BaseURL:  os.Getenv("CLAUDE_API_ENDPOINT"),
		})
		require.NoError(s.T(), err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	prompt := "Hello {{name}}"
	variables := `{invalid json}`

	_, err = client.CallWithPromptAndVariables(ctx, prompt, variables)
	assert.Error(s.T(), err, "Should fail with invalid JSON variables")
	assert.Contains(s.T(), err.Error(), "variable substitution failed",
		"Error should indicate variable substitution failure")
}

// TestCreateClient_EmptyProvider verifies empty provider string is rejected
func (s *ClientFactoryIntegrationTestSuite) TestCreateClient_EmptyProvider() {
	config := &types.AIConfig{
		Provider: "",
		APIKey:   "some-key",
	}

	_, err := s.factory.CreateClient(config)
	assert.Error(s.T(), err, "Empty provider should produce an error")
	assert.Contains(s.T(), err.Error(), "unsupported provider")
}

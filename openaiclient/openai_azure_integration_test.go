//go:build integration

package openaiclient

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/kengibson1111/go-aiprovider/internal/shared/testutil"
	"github.com/kengibson1111/go-aiprovider/types"
	"github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// OpenAIAzureIntegrationTestSuite tests the OpenAI client against Azure OpenAI Service
type OpenAIAzureIntegrationTestSuite struct {
	suite.Suite
	cleanupCwd func()
	client     *OpenAIClient
}

func TestOpenAIAzureIntegrationTestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	suite.Run(t, new(OpenAIAzureIntegrationTestSuite))
}

func (s *OpenAIAzureIntegrationTestSuite) SetupSuite() {
	testutil.SetupEnvironment(s.T(), "../")
	s.cleanupCwd = testutil.SetupCurrentDirectory(s.T(), "../")

	// Skip if required Azure env vars are not set
	endpoint := os.Getenv("OPENAI_AZURE_ENDPOINT")
	if endpoint == "" {
		s.T().Skip("OPENAI_AZURE_ENDPOINT not set, skipping Azure OpenAI integration tests")
	}

	config := &types.AIConfig{
		Provider: types.ProviderOpenAIAzure,
	}

	client, err := NewOpenAIAzureClient(config)
	require.NoError(s.T(), err, "Failed to create Azure OpenAI client")
	s.client = client
}

func (s *OpenAIAzureIntegrationTestSuite) TearDownSuite() {
	if s.client != nil {
		s.client.CloseIdleConnections()
	}
	if s.cleanupCwd != nil {
		s.cleanupCwd()
	}
}

// TestValidateCredentials verifies that valid Azure credentials pass validation
func (s *OpenAIAzureIntegrationTestSuite) TestValidateCredentials() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := s.client.ValidateCredentials(ctx)
	assert.NoError(s.T(), err, "Valid Azure credentials should pass validation")
}

// TestCallWithPrompt verifies a basic prompt call returns a valid JSON response
func (s *OpenAIAzureIntegrationTestSuite) TestCallWithPrompt() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	response, err := s.client.CallWithPrompt(ctx, "Reply with only the word 'hello'.")
	require.NoError(s.T(), err, "CallWithPrompt should succeed")
	require.NotNil(s.T(), response, "Response should not be nil")

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	assert.NoError(s.T(), err, "Response should be valid JSON")

	assert.Contains(s.T(), result, "choices", "Response should contain choices")
	assert.Contains(s.T(), result, "model", "Response should contain model")
	assert.Contains(s.T(), result, "usage", "Response should contain usage")
}

// TestCallWithMessages verifies multi-turn conversation support
func (s *OpenAIAzureIntegrationTestSuite) TestCallWithMessages() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	messages := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage("You are a helpful assistant. Reply as briefly as possible."),
		openai.UserMessage("What is 2+2?"),
	}

	completion, err := s.client.CallWithMessages(ctx, messages)
	require.NoError(s.T(), err, "CallWithMessages should succeed")
	require.NotNil(s.T(), completion, "Completion should not be nil")
	require.NotEmpty(s.T(), completion.Choices, "Completion should have at least one choice")

	content := completion.Choices[0].Message.Content
	assert.NotEmpty(s.T(), content, "Response content should not be empty")
	assert.Contains(s.T(), content, "4", "Response should contain the answer '4'")
}

// TestCallWithMessages_MultiTurn verifies a multi-turn conversation with context retention
func (s *OpenAIAzureIntegrationTestSuite) TestCallWithMessages_MultiTurn() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	messages := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage("You are a helpful assistant. Reply as briefly as possible."),
		openai.UserMessage("My name is TestUser."),
		openai.AssistantMessage("Nice to meet you, TestUser."),
		openai.UserMessage("What is my name?"),
	}

	completion, err := s.client.CallWithMessages(ctx, messages)
	require.NoError(s.T(), err, "Multi-turn CallWithMessages should succeed")
	require.NotEmpty(s.T(), completion.Choices, "Should have at least one choice")

	content := completion.Choices[0].Message.Content
	assert.Contains(s.T(), content, "TestUser",
		"Model should recall the name from conversation context")
}

// TestCallWithTools verifies function calling capabilities
func (s *OpenAIAzureIntegrationTestSuite) TestCallWithTools() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tools := []openai.ChatCompletionToolUnionParam{
		openai.ChatCompletionFunctionTool(shared.FunctionDefinitionParam{
			Name:        "get_weather",
			Description: openai.String("Get current weather for a location"),
			Parameters: shared.FunctionParameters{
				"type": "object",
				"properties": map[string]interface{}{
					"location": map[string]interface{}{
						"type":        "string",
						"description": "City name",
					},
				},
				"required": []string{"location"},
			},
		}),
	}

	completion, err := s.client.CallWithTools(ctx, "What is the weather in Paris?", tools)
	require.NoError(s.T(), err, "CallWithTools should succeed")
	require.NotNil(s.T(), completion, "Completion should not be nil")
	require.NotEmpty(s.T(), completion.Choices, "Should have at least one choice")

	choice := completion.Choices[0]
	if len(choice.Message.ToolCalls) > 0 {
		toolCall := choice.Message.ToolCalls[0]
		assert.Equal(s.T(), "get_weather", toolCall.Function.Name,
			"Tool call should be for get_weather")
		assert.NotEmpty(s.T(), toolCall.Function.Arguments,
			"Tool call should have arguments")

		var args map[string]interface{}
		err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args)
		assert.NoError(s.T(), err, "Tool call arguments should be valid JSON")
		assert.Contains(s.T(), args, "location", "Arguments should contain location")
	} else {
		assert.NotEmpty(s.T(), choice.Message.Content,
			"If no tool call, should have text content")
	}
}

// TestCallWithPromptStream verifies streaming response functionality
func (s *OpenAIAzureIntegrationTestSuite) TestCallWithPromptStream() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	stream, err := s.client.CallWithPromptStream(ctx, "Count from 1 to 3.")
	require.NoError(s.T(), err, "CallWithPromptStream should succeed")
	require.NotNil(s.T(), stream, "Stream should not be nil")

	var fullContent string
	chunkCount := 0

	for stream.Next() {
		chunk := stream.Current()
		chunkCount++
		if len(chunk.Choices) > 0 {
			fullContent += chunk.Choices[0].Delta.Content
		}
	}

	err = stream.Err()
	assert.NoError(s.T(), err, "Stream should complete without error")
	assert.Greater(s.T(), chunkCount, 0, "Should have received at least one chunk")
	assert.NotEmpty(s.T(), fullContent, "Accumulated content should not be empty")
}

// TestCallWithPromptAndVariables verifies template variable substitution
func (s *OpenAIAzureIntegrationTestSuite) TestCallWithPromptAndVariables() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	prompt := "You are a {{role}}. Reply with only: I am a {{role}}."
	variables := `{"role": "translator"}`

	response, err := s.client.CallWithPromptAndVariables(ctx, prompt, variables)
	require.NoError(s.T(), err, "CallWithPromptAndVariables should succeed")
	require.NotNil(s.T(), response, "Response should not be nil")

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	assert.NoError(s.T(), err, "Response should be valid JSON")
	assert.Contains(s.T(), result, "choices", "Response should contain choices")
}

// TestCallWithPromptAndVariables_InvalidJSON verifies error on bad variable JSON
func (s *OpenAIAzureIntegrationTestSuite) TestCallWithPromptAndVariables_InvalidJSON() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	prompt := "Hello {{name}}"
	variables := `{invalid json}`

	_, err := s.client.CallWithPromptAndVariables(ctx, prompt, variables)
	assert.Error(s.T(), err, "Should fail with invalid JSON variables")
	assert.Contains(s.T(), err.Error(), "variable substitution failed",
		"Error should indicate variable substitution failure")
}

// TestGetModel verifies the model getter returns the configured model
func (s *OpenAIAzureIntegrationTestSuite) TestGetModel() {
	expectedModel := os.Getenv("OPENAI_AZURE_MODEL")
	model := s.client.GetModel()
	assert.Equal(s.T(), expectedModel, model,
		"GetModel should return the model from OPENAI_AZURE_MODEL")
}

// TestCallWithPrompt_ContextCancellation verifies that cancelled contexts are handled
func (s *OpenAIAzureIntegrationTestSuite) TestCallWithPrompt_ContextCancellation() {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := s.client.CallWithPrompt(ctx, "This should not complete")
	assert.Error(s.T(), err, "Cancelled context should produce an error")
}

// TestCallWithPromptStream_ContextCancellation verifies streaming handles cancellation
func (s *OpenAIAzureIntegrationTestSuite) TestCallWithPromptStream_ContextCancellation() {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	stream, err := s.client.CallWithPromptStream(ctx, "This should not complete")
	if err != nil {
		assert.Error(s.T(), err, "Should error on cancelled context")
		return
	}
	for stream.Next() {
		// drain
	}
	assert.Error(s.T(), stream.Err(), "Stream should error on cancelled context")
}

// TestNewOpenAIAzureClient_NilConfig verifies nil config is rejected
func (s *OpenAIAzureIntegrationTestSuite) TestNewOpenAIAzureClient_NilConfig() {
	_, err := NewOpenAIAzureClient(nil)
	assert.Error(s.T(), err, "Nil config should produce an error")
	assert.Contains(s.T(), err.Error(), "configuration is required")
}

// TestNewOpenAIAzureClient_Defaults verifies env var defaults are applied when config fields are empty
func (s *OpenAIAzureIntegrationTestSuite) TestNewOpenAIAzureClient_Defaults() {
	config := &types.AIConfig{
		Provider: types.ProviderOpenAIAzure,
	}

	client, err := NewOpenAIAzureClient(config)
	require.NoError(s.T(), err, "Client creation with env var defaults should succeed")
	defer client.CloseIdleConnections()

	expectedModel := os.Getenv("OPENAI_AZURE_MODEL")
	assert.Equal(s.T(), expectedModel, client.GetModel(),
		"Model should default to OPENAI_AZURE_MODEL env var")
	assert.Equal(s.T(), 1000, client.maxTokens,
		"Default maxTokens should be 1000")
	assert.InDelta(s.T(), 0.7, client.temperature, 0.001,
		"Default temperature should be 0.7")
}

// TestNewOpenAIAzureClient_CustomConfig verifies custom config values override env var defaults
func (s *OpenAIAzureIntegrationTestSuite) TestNewOpenAIAzureClient_CustomConfig() {
	config := &types.AIConfig{
		Provider:    types.ProviderOpenAIAzure,
		BaseURL:     os.Getenv("OPENAI_AZURE_ENDPOINT"),
		Model:       os.Getenv("OPENAI_AZURE_MODEL"),
		MaxTokens:   2000,
		Temperature: 0.5,
	}

	client, err := NewOpenAIAzureClient(config)
	require.NoError(s.T(), err, "Client creation with custom config should succeed")
	defer client.CloseIdleConnections()

	assert.Equal(s.T(), os.Getenv("OPENAI_AZURE_MODEL"), client.GetModel())
	assert.Equal(s.T(), 2000, client.maxTokens)
	assert.InDelta(s.T(), 0.5, client.temperature, 0.001)
}

// TestCloseIdleConnections verifies resource cleanup does not panic
func (s *OpenAIAzureIntegrationTestSuite) TestCloseIdleConnections() {
	assert.NotPanics(s.T(), func() {
		s.client.CloseIdleConnections()
	}, "CloseIdleConnections should not panic")

	// Verify client still works after closing idle connections
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := s.client.CallWithPrompt(ctx, "Reply with 'ok'.")
	assert.NoError(s.T(), err,
		"Client should still work after closing idle connections")
}

// TestCallWithMessages_EmptyMessages verifies behavior with empty message slice
func (s *OpenAIAzureIntegrationTestSuite) TestCallWithMessages_EmptyMessages() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	messages := []openai.ChatCompletionMessageParamUnion{}

	_, err := s.client.CallWithMessages(ctx, messages)
	assert.Error(s.T(), err, "Empty messages should produce an error from the API")
}

// TestCallWithPrompt_UsageTracking verifies that usage information is returned
func (s *OpenAIAzureIntegrationTestSuite) TestCallWithPrompt_UsageTracking() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	response, err := s.client.CallWithPrompt(ctx, "Say 'hi'.")
	require.NoError(s.T(), err)

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	require.NoError(s.T(), err)

	usage, ok := result["usage"].(map[string]interface{})
	require.True(s.T(), ok, "Response should contain usage object")

	assert.Greater(s.T(), usage["prompt_tokens"], float64(0),
		"Prompt tokens should be greater than 0")
	assert.Greater(s.T(), usage["completion_tokens"], float64(0),
		"Completion tokens should be greater than 0")
	assert.Greater(s.T(), usage["total_tokens"], float64(0),
		"Total tokens should be greater than 0")
}

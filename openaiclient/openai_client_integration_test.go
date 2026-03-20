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

// OpenAIClientIntegrationTestSuite tests the OpenAI client against the real OpenAI API
type OpenAIClientIntegrationTestSuite struct {
	suite.Suite
	cleanupCwd func()
	client     *OpenAIClient
}

func TestOpenAIClientIntegrationTestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	suite.Run(t, new(OpenAIClientIntegrationTestSuite))
}

func (s *OpenAIClientIntegrationTestSuite) SetupSuite() {
	testutil.SetupEnvironment(s.T(), "../")
	s.cleanupCwd = testutil.SetupCurrentDirectory(s.T(), "../")

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		s.T().Skip("OPENAI_API_KEY not set, skipping OpenAI integration tests")
	}

	config := &types.AIConfig{
		Provider: "openai",
		APIKey:   apiKey,
		Model:    "gpt-5.4-mini",
	}

	client, err := NewOpenAIClient(config)
	require.NoError(s.T(), err, "Failed to create OpenAI client")
	s.client = client
}

func (s *OpenAIClientIntegrationTestSuite) TearDownSuite() {
	if s.client != nil {
		s.client.CloseIdleConnections()
	}
	if s.cleanupCwd != nil {
		s.cleanupCwd()
	}
}

// TestValidateCredentials verifies that valid API credentials pass validation
func (s *OpenAIClientIntegrationTestSuite) TestValidateCredentials() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := s.client.ValidateCredentials(ctx)
	assert.NoError(s.T(), err, "Valid credentials should pass validation")
}

// TestValidateCredentials_InvalidKey verifies that an invalid API key is rejected
func (s *OpenAIClientIntegrationTestSuite) TestValidateCredentials_InvalidKey() {
	config := &types.AIConfig{
		Provider: "openai",
		APIKey:   "sk-invalid-key-for-testing",
		Model:    "gpt-5.4-mini",
	}
	invalidClient, err := NewOpenAIClient(config)
	require.NoError(s.T(), err, "Client creation should succeed even with invalid key")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = invalidClient.ValidateCredentials(ctx)
	assert.Error(s.T(), err, "Invalid credentials should fail validation")
	assert.Contains(s.T(), err.Error(), "invalid API key",
		"Error should indicate invalid API key")
	invalidClient.CloseIdleConnections()
}

// TestCallWithPrompt verifies a basic prompt call returns a valid JSON response
func (s *OpenAIClientIntegrationTestSuite) TestCallWithPrompt() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	response, err := s.client.CallWithPrompt(ctx, "Reply with only the word 'hello'.")
	require.NoError(s.T(), err, "CallWithPrompt should succeed")
	require.NotNil(s.T(), response, "Response should not be nil")

	// Verify the response is valid JSON
	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	assert.NoError(s.T(), err, "Response should be valid JSON")

	// Verify expected top-level fields exist
	assert.Contains(s.T(), result, "choices", "Response should contain choices")
	assert.Contains(s.T(), result, "model", "Response should contain model")
	assert.Contains(s.T(), result, "usage", "Response should contain usage")
}

// TestCallWithMessages verifies multi-turn conversation support
func (s *OpenAIClientIntegrationTestSuite) TestCallWithMessages() {
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
func (s *OpenAIClientIntegrationTestSuite) TestCallWithMessages_MultiTurn() {
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
func (s *OpenAIClientIntegrationTestSuite) TestCallWithTools() {
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
	// The model should either call the tool or respond with text
	if len(choice.Message.ToolCalls) > 0 {
		toolCall := choice.Message.ToolCalls[0]
		assert.Equal(s.T(), "get_weather", toolCall.Function.Name,
			"Tool call should be for get_weather")
		assert.NotEmpty(s.T(), toolCall.Function.Arguments,
			"Tool call should have arguments")

		// Verify arguments contain location
		var args map[string]interface{}
		err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args)
		assert.NoError(s.T(), err, "Tool call arguments should be valid JSON")
		assert.Contains(s.T(), args, "location", "Arguments should contain location")
	} else {
		// Model chose to respond with text instead of calling the tool
		assert.NotEmpty(s.T(), choice.Message.Content,
			"If no tool call, should have text content")
	}
}

// TestCallWithPromptStream verifies streaming response functionality
func (s *OpenAIClientIntegrationTestSuite) TestCallWithPromptStream() {
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
func (s *OpenAIClientIntegrationTestSuite) TestCallWithPromptAndVariables() {
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
func (s *OpenAIClientIntegrationTestSuite) TestCallWithPromptAndVariables_InvalidJSON() {
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
func (s *OpenAIClientIntegrationTestSuite) TestGetModel() {
	model := s.client.GetModel()
	assert.Equal(s.T(), "gpt-5.4-mini", model,
		"GetModel should return the configured model")
}

// TestCallWithPrompt_ContextCancellation verifies that cancelled contexts are handled
func (s *OpenAIClientIntegrationTestSuite) TestCallWithPrompt_ContextCancellation() {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := s.client.CallWithPrompt(ctx, "This should not complete")
	assert.Error(s.T(), err, "Cancelled context should produce an error")
}

// TestCallWithPromptStream_ContextCancellation verifies streaming handles cancellation
func (s *OpenAIClientIntegrationTestSuite) TestCallWithPromptStream_ContextCancellation() {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	stream, err := s.client.CallWithPromptStream(ctx, "This should not complete")
	// Either the stream creation fails or the stream itself errors
	if err != nil {
		assert.Error(s.T(), err, "Should error on cancelled context")
		return
	}
	// If stream was returned, iterating should produce an error
	for stream.Next() {
		// drain
	}
	assert.Error(s.T(), stream.Err(), "Stream should error on cancelled context")
}

// TestNewOpenAIClient_Defaults verifies default values are applied correctly
func (s *OpenAIClientIntegrationTestSuite) TestNewOpenAIClient_Defaults() {
	apiKey := os.Getenv("OPENAI_API_KEY")

	config := &types.AIConfig{
		Provider: "openai",
		APIKey:   apiKey,
	}

	client, err := NewOpenAIClient(config)
	require.NoError(s.T(), err, "Client creation with defaults should succeed")
	defer client.CloseIdleConnections()

	assert.Equal(s.T(), string(openai.ChatModelGPT4oMini), client.GetModel(),
		"Default model should be gpt-5.4-mini")
	assert.Equal(s.T(), 1000, client.maxTokens,
		"Default maxTokens should be 1000")
	assert.InDelta(s.T(), 0.7, client.temperature, 0.001,
		"Default temperature should be 0.7")
}

// TestNewOpenAIClient_CustomConfig verifies custom config values are respected
func (s *OpenAIClientIntegrationTestSuite) TestNewOpenAIClient_CustomConfig() {
	apiKey := os.Getenv("OPENAI_API_KEY")

	config := &types.AIConfig{
		Provider:    "openai",
		APIKey:      apiKey,
		Model:       "gpt-5.4-mini",
		MaxTokens:   2000,
		Temperature: 0.5,
	}

	client, err := NewOpenAIClient(config)
	require.NoError(s.T(), err, "Client creation with custom config should succeed")
	defer client.CloseIdleConnections()

	assert.Equal(s.T(), "gpt-5.4-mini", client.GetModel())
	assert.Equal(s.T(), 2000, client.maxTokens)
	assert.InDelta(s.T(), 0.5, client.temperature, 0.001)
}

// TestNewOpenAIClient_NilConfig verifies nil config is rejected
func (s *OpenAIClientIntegrationTestSuite) TestNewOpenAIClient_NilConfig() {
	_, err := NewOpenAIClient(nil)
	assert.Error(s.T(), err, "Nil config should produce an error")
	assert.Contains(s.T(), err.Error(), "configuration is required")
}

// TestNewOpenAIClient_EmptyAPIKey verifies empty API key is rejected
func (s *OpenAIClientIntegrationTestSuite) TestNewOpenAIClient_EmptyAPIKey() {
	config := &types.AIConfig{
		Provider: "openai",
		APIKey:   "",
	}

	_, err := NewOpenAIClient(config)
	assert.Error(s.T(), err, "Empty API key should produce an error")
	assert.Contains(s.T(), err.Error(), "API key is required")
}

// TestNewOpenAIClient_WhitespaceAPIKey verifies whitespace-only API key is rejected
func (s *OpenAIClientIntegrationTestSuite) TestNewOpenAIClient_WhitespaceAPIKey() {
	config := &types.AIConfig{
		Provider: "openai",
		APIKey:   "   ",
	}

	_, err := NewOpenAIClient(config)
	assert.Error(s.T(), err, "Whitespace API key should produce an error")
	assert.Contains(s.T(), err.Error(), "API key is required")
}

// TestCloseIdleConnections verifies resource cleanup does not panic
func (s *OpenAIClientIntegrationTestSuite) TestCloseIdleConnections() {
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
func (s *OpenAIClientIntegrationTestSuite) TestCallWithMessages_EmptyMessages() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	messages := []openai.ChatCompletionMessageParamUnion{}

	_, err := s.client.CallWithMessages(ctx, messages)
	assert.Error(s.T(), err, "Empty messages should produce an error from the API")
}

// TestCallWithPrompt_UsageTracking verifies that usage information is returned
func (s *OpenAIClientIntegrationTestSuite) TestCallWithPrompt_UsageTracking() {
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

package openai

import (
	"context"
	"fmt"
	"log"

	"github.com/kengibson1111/go-aiprovider/types"
	"github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/shared"
)

// ExampleBasicUsage demonstrates basic OpenAI client usage with SDK types.
//
// This example shows how to create a client and make a simple completion request,
// highlighting the direct field access to SDK response types without JSON processing.
func ExampleBasicUsage() {
	// Create client configuration
	config := &types.AIConfig{
		APIKey: "your-api-key-here",
		Model:  "gpt-4o-mini", // Optional, defaults to gpt-4o-mini
	}

	// Create the client
	client, err := NewOpenAIClient(config)
	if err != nil {
		log.Fatal("Failed to create client:", err)
	}

	// Make a completion request
	ctx := context.Background()
	completion, err := client.callWithPrompt(ctx, "Hello, how are you?")
	if err != nil {
		log.Fatal("API call failed:", err)
	}

	// Access response directly without JSON unmarshaling
	response := completion.Choices[0].Message.Content
	tokensUsed := completion.Usage.TotalTokens
	model := completion.Model

	fmt.Printf("Response: %s\n", response)
	fmt.Printf("Tokens used: %d\n", tokensUsed)
	fmt.Printf("Model: %s\n", model)
}

// ExampleMultiTurnConversation demonstrates multi-turn conversation support.
//
// This example shows how to use CallWithMessages for complex conversations
// with system, user, and assistant messages using native SDK types.
func ExampleMultiTurnConversation() {
	config := &types.AIConfig{
		APIKey: "your-api-key-here",
	}

	client, err := NewOpenAIClient(config)
	if err != nil {
		log.Fatal(err)
	}

	// Build conversation with multiple message types
	messages := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage("You are a helpful assistant."),
		openai.UserMessage("What is the capital of France?"),
		openai.AssistantMessage("The capital of France is Paris."),
		openai.UserMessage("What about Germany?"),
	}

	ctx := context.Background()
	completion, err := client.CallWithMessages(ctx, messages)
	if err != nil {
		log.Fatal(err)
	}

	// Direct access to response content
	response := completion.Choices[0].Message.Content
	fmt.Printf("Assistant: %s\n", response)
}

// ExampleFunctionCalling demonstrates function calling capabilities.
//
// This example shows how to use CallWithTools for function calling,
// allowing the model to call predefined functions as part of its response.
func ExampleFunctionCalling() {
	config := &types.AIConfig{
		APIKey: "your-api-key-here",
	}

	client, err := NewOpenAIClient(config)
	if err != nil {
		log.Fatal(err)
	}

	// Define available tools/functions
	tools := []openai.ChatCompletionToolUnionParam{
		openai.ChatCompletionFunctionTool(shared.FunctionDefinitionParam{
			Name:        "get_weather",
			Description: openai.String("Get current weather for a location"),
			Parameters: map[string]interface{}{
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

	ctx := context.Background()
	completion, err := client.CallWithTools(ctx, "What's the weather in Paris?", tools)
	if err != nil {
		log.Fatal(err)
	}

	// Check if the model wants to call a function
	if len(completion.Choices[0].Message.ToolCalls) > 0 {
		toolCall := completion.Choices[0].Message.ToolCalls[0]
		fmt.Printf("Function called: %s\n", toolCall.Function.Name)
		fmt.Printf("Arguments: %s\n", toolCall.Function.Arguments)
	} else {
		// Regular text response
		fmt.Printf("Response: %s\n", completion.Choices[0].Message.Content)
	}
}

// ExampleStreamingResponse demonstrates streaming response processing.
//
// This example shows how to use CallWithPromptStream for real-time response
// processing, useful for applications that need to display partial responses.
func ExampleStreamingResponse() {
	config := &types.AIConfig{
		APIKey: "your-api-key-here",
	}

	client, err := NewOpenAIClient(config)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	stream, err := client.CallWithPromptStream(ctx, "Tell me a story")
	if err != nil {
		log.Fatal(err)
	}

	// Process streaming chunks as they arrive
	fmt.Print("Story: ")
	for stream.Next() {
		chunk := stream.Current()
		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
			fmt.Print(chunk.Choices[0].Delta.Content)
		}
	}
	fmt.Println()

	// Check for streaming errors
	if err := stream.Err(); err != nil {
		log.Fatal("Streaming error:", err)
	}
}

// ExampleTemplateProcessing demonstrates template processing with variable substitution.
//
// This example shows how to use CallWithPromptAndVariables for dynamic prompt
// generation with variable substitution while returning SDK types.
func ExampleTemplateProcessing() {
	config := &types.AIConfig{
		APIKey: "your-api-key-here",
	}

	client, err := NewOpenAIClient(config)
	if err != nil {
		log.Fatal(err)
	}

	// Template with variables
	prompt := "You are a {{role}} assistant. Help with {{task}} in {{language}}."
	variables := `{
		"role": "senior developer",
		"task": "code review",
		"language": "Go"
	}`

	ctx := context.Background()
	completion, err := client.callWithPromptAndVariables(ctx, prompt, variables)
	if err != nil {
		log.Fatal(err)
	}

	// Direct access to processed response
	response := completion.Choices[0].Message.Content
	fmt.Printf("Response: %s\n", response)
}

// ExampleAzureOpenAI demonstrates using the client with Azure OpenAI Service.
//
// This example shows how to configure the client for Azure OpenAI endpoints
// using the BaseURL configuration option.
func ExampleAzureOpenAI() {
	config := &types.AIConfig{
		APIKey:  "your-azure-api-key",
		BaseURL: "https://your-resource.openai.azure.com/",
		Model:   "gpt-4", // Azure deployment name
	}

	client, err := NewOpenAIClient(config)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	completion, err := client.callWithPrompt(ctx, "Hello from Azure!")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Azure Response: %s\n", completion.Choices[0].Message.Content)
}

// ExampleErrorHandling demonstrates comprehensive error handling patterns.
//
// This example shows how the client provides structured error handling with
// user-friendly messages for common API error scenarios.
func ExampleErrorHandling() {
	config := &types.AIConfig{
		APIKey: "invalid-key", // Intentionally invalid for demonstration
	}

	client, err := NewOpenAIClient(config)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	completion, err := client.callWithPrompt(ctx, "Test prompt")
	if err != nil {
		// The client provides user-friendly error messages
		fmt.Printf("Error occurred: %s\n", err.Error())
		// Example outputs:
		// - "invalid API key: please check your OpenAI API key configuration"
		// - "rate limit exceeded: too many requests, please wait before retrying"
		// - "quota exceeded: your OpenAI account has insufficient quota, please check your billing"
		return
	}

	fmt.Printf("Success: %s\n", completion.Choices[0].Message.Content)
}

// ExamplePerformanceBenefits demonstrates the performance benefits of SDK integration.
//
// This example shows how the SDK-based implementation eliminates JSON processing
// overhead and provides direct field access for better performance.
func ExamplePerformanceBenefits() {
	config := &types.AIConfig{
		APIKey: "your-api-key-here",
	}

	client, err := NewOpenAIClient(config)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	completion, err := client.callWithPrompt(ctx, "Explain performance benefits")
	if err != nil {
		log.Fatal(err)
	}

	// Performance benefits demonstrated:

	// 1. No JSON unmarshaling overhead - direct field access
	content := completion.Choices[0].Message.Content

	// 2. Type-safe access to all response fields
	finishReason := completion.Choices[0].FinishReason
	promptTokens := completion.Usage.PromptTokens
	completionTokens := completion.Usage.CompletionTokens
	totalTokens := completion.Usage.TotalTokens

	// 3. Access to SDK-specific metadata
	model := completion.Model

	fmt.Printf("Content: %s\n", content)
	fmt.Printf("Finish Reason: %s\n", finishReason)
	fmt.Printf("Token Usage - Prompt: %d, Completion: %d, Total: %d\n",
		promptTokens, completionTokens, totalTokens)
	fmt.Printf("Model: %s\n", model)

	// 4. No intermediate JSON byte arrays in memory
	// 5. Compile-time type checking for all field access
	// 6. Automatic handling of SDK updates and new fields
}

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/kengibson1111/go-aiprovider/client"
	"github.com/kengibson1111/go-aiprovider/types"
)

// BasicUsageExample demonstrates creating a client and making a simple prompt call.
func BasicUsageExample(factory *client.ClientFactory) {
	fmt.Println("=== Basic Usage Example ===")

	config := &types.AIConfig{
		Provider:    "openai",
		APIKey:      os.Getenv("OPENAI_API_KEY"),
		BaseURL:     os.Getenv("OPENAI_API_ENDPOINT"),
		Model:       "gpt-4o-mini",
		MaxTokens:   1000,
		Temperature: 0.7,
	}

	aiClient, err := factory.CreateClient(config)
	if err != nil {
		log.Printf("Failed to create client: %v", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	response, err := aiClient.CallWithPrompt(ctx, "Explain the concept of goroutines in one sentence.")
	if err != nil {
		log.Printf("API call failed: %v", err)
		return
	}

	var result map[string]any
	if err := json.Unmarshal(response, &result); err != nil {
		log.Printf("Failed to parse response: %v", err)
		return
	}

	fmt.Printf("Response: %s\n", string(response))
	fmt.Println("Basic usage example completed")
}

// TimeoutExample shows how to use context with timeout for request cancellation.
func TimeoutExample(factory *client.ClientFactory) {
	fmt.Println("\n=== Timeout Example ===")

	config := &types.AIConfig{
		Provider:    "openai",
		APIKey:      os.Getenv("OPENAI_API_KEY"),
		BaseURL:     os.Getenv("OPENAI_API_ENDPOINT"),
		Model:       "gpt-4o-mini",
		MaxTokens:   1000,
		Temperature: 0.7,
	}

	aiClient, err := factory.CreateClient(config)
	if err != nil {
		log.Printf("Failed to create client: %v", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	response, err := aiClient.CallWithPrompt(ctx, "Write a brief explanation of machine learning.")
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			log.Println("Request timed out")
			return
		}
		log.Printf("Request failed: %v", err)
		return
	}

	fmt.Printf("Response received: %s\n", string(response))
	fmt.Println("Timeout example completed")
}

// TemplateVariablesExample demonstrates using prompt templates with variable substitution.
func TemplateVariablesExample(factory *client.ClientFactory) {
	fmt.Println("\n=== Template Variables Example ===")

	config := &types.AIConfig{
		Provider:    "openai",
		APIKey:      os.Getenv("OPENAI_API_KEY"),
		BaseURL:     os.Getenv("OPENAI_API_ENDPOINT"),
		Model:       "gpt-4o-mini",
		MaxTokens:   1000,
		Temperature: 0.7,
	}

	aiClient, err := factory.CreateClient(config)
	if err != nil {
		log.Printf("Failed to create client: %v", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	prompt := "You are a {{role}}. Reply with only: I am a {{role}}."
	variables := `{"role": "translator"}`

	response, err := aiClient.CallWithPromptAndVariables(ctx, prompt, variables)
	if err != nil {
		log.Printf("Request failed: %v", err)
		return
	}

	fmt.Printf("Response: %s\n", string(response))
	fmt.Println("Template variables example completed")
}

// ErrorHandlingExample demonstrates proper error handling using types.ErrorResponse.
func ErrorHandlingExample(factory *client.ClientFactory) {
	fmt.Println("\n=== Error Handling Example ===")

	// Use an invalid API key to trigger an error
	config := &types.AIConfig{
		Provider:    "openai",
		APIKey:      "sk-invalid-key-for-testing",
		Model:       "gpt-4o-mini",
		MaxTokens:   100,
		Temperature: 0.7,
	}

	aiClient, err := factory.CreateClient(config)
	if err != nil {
		log.Printf("Failed to create client: %v", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err = aiClient.CallWithPrompt(ctx, "Test prompt for error handling")
	if err != nil {
		// All API errors from CallWithPrompt, CallWithPromptAndVariables, and
		// ValidateCredentials are returned as *types.ErrorResponse, which can be
		// inspected using errors.As for structured error handling.
		var apiErr *types.ErrorResponse
		if errors.As(err, &apiErr) {
			fmt.Printf("API error - Code: %s, Message: %s\n", apiErr.Code, apiErr.Message)
			if apiErr.Retry {
				fmt.Println("This error is retryable")
			}

			switch apiErr.Code {
			case "invalid_api_key":
				fmt.Println("Action: Check your API key configuration")
			case "rate_limit_exceeded":
				fmt.Println("Action: Wait before retrying")
			case "insufficient_quota":
				fmt.Println("Action: Check your billing")
			case "model_not_found":
				fmt.Println("Action: Verify the model name")
			default:
				fmt.Printf("Action: Review error details - %s\n", apiErr.Message)
			}
		} else if errors.Is(err, context.DeadlineExceeded) {
			fmt.Println("Request timed out")
		} else {
			fmt.Printf("Unexpected error: %v\n", err)
		}
	}

	fmt.Println("Error handling example completed")
}

// ValidateCredentialsExample demonstrates credential validation before making calls.
func ValidateCredentialsExample(factory *client.ClientFactory) {
	fmt.Println("\n=== Validate Credentials Example ===")

	config := &types.AIConfig{
		Provider:    "openai",
		APIKey:      os.Getenv("OPENAI_API_KEY"),
		BaseURL:     os.Getenv("OPENAI_API_ENDPOINT"),
		Model:       "gpt-4o-mini",
		MaxTokens:   1000,
		Temperature: 0.7,
	}

	aiClient, err := factory.CreateClient(config)
	if err != nil {
		log.Printf("Failed to create client: %v", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := aiClient.ValidateCredentials(ctx); err != nil {
		var apiErr *types.ErrorResponse
		if errors.As(err, &apiErr) {
			fmt.Printf("Credential validation failed - Code: %s, Message: %s\n", apiErr.Code, apiErr.Message)
		} else {
			fmt.Printf("Credential validation failed: %v\n", err)
		}
		return
	}

	fmt.Println("Credentials are valid")
	fmt.Println("Validate credentials example completed")
}

func main() {
	fmt.Println("OpenAI SDK - Basic Usage Examples")
	fmt.Println("=================================")

	// SetupEnvironment loads the .env file from the repo root so that environment
	// variables (API keys, endpoints, etc.) are available without manual export.
	// This example must be run from the repo's root directory
	// (e.g., go run examples/openai_sdk_examples/basic_usage/main.go).
	client.SetupEnvironment("../../")

	// SetupCurrentDirectory ensures the working directory is the repo root,
	// which is required for resolving any root-relative paths used by the client
	// libraries. The cleanup function restores the original directory on exit.
	// This example must be run from the repo's root directory.
	cleanup := client.SetupCurrentDirectory("../../")
	defer cleanup()

	factory := client.NewClientFactory()

	// Run examples
	BasicUsageExample(factory)
	TimeoutExample(factory)
	TemplateVariablesExample(factory)
	ErrorHandlingExample(factory)
	ValidateCredentialsExample(factory)

	fmt.Println("\nAll basic usage examples completed")
}

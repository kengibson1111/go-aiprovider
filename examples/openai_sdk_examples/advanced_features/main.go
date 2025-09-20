package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/openai/openai-go/v2"
)

// ConversationExample demonstrates multi-turn conversations
func ConversationExample() {
	fmt.Println("=== Multi-turn Conversation Example ===")

	ctx := context.Background()

	// Build a conversation with system, user, and assistant messages
	messages := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage("You are a helpful Go programming tutor. Keep responses concise and practical."),
		openai.UserMessage("What's the difference between a slice and an array in Go?"),
		openai.AssistantMessage("Arrays have fixed size set at compile time (e.g., [5]int), while slices are dynamic views over arrays (e.g., []int). Slices can grow/shrink and are more commonly used."),
		openai.UserMessage("Can you show me how to append to a slice?"),
	}

	fmt.Println("Conversation messages:")
	for i, msg := range messages {
		// Note: In actual implementation, you'd need to handle the union type properly
		fmt.Printf("%d. %T\n", i+1, msg)
	}

	// In actual implementation:
	// completion, err := client.CallWithMessages(ctx, messages)
	// if err != nil {
	//     log.Printf("Conversation failed: %v", err)
	//     return
	// }
	//
	// fmt.Printf("Tutor response: %s\n", completion.Choices[0].Message.Content)

	fmt.Println("✓ Conversation example completed")
}

// FunctionCallingExample demonstrates function calling capabilities
func FunctionCallingExample() {
	fmt.Println("\n=== Function Calling Example ===")

	ctx := context.Background()

	// Define a weather function tool
	weatherTool := openai.ChatCompletionToolParam{
		Type: openai.F(openai.ChatCompletionToolTypeFunction),
		Function: openai.F(openai.FunctionDefinitionParam{
			Name:        openai.String("get_weather"),
			Description: openai.String("Get current weather information for a location"),
			Parameters: openai.F(openai.FunctionParameters{
				"type": "object",
				"properties": map[string]interface{}{
					"location": map[string]interface{}{
						"type":        "string",
						"description": "The city and state, e.g. San Francisco, CA",
					},
					"unit": map[string]interface{}{
						"type": "string",
						"enum": []string{"celsius", "fahrenheit"},
					},
				},
				"required": []string{"location"},
			}),
		}),
	}

	// Define a calculator function tool
	calculatorTool := openai.ChatCompletionToolParam{
		Type: openai.F(openai.ChatCompletionToolTypeFunction),
		Function: openai.F(openai.FunctionDefinitionParam{
			Name:        openai.String("calculate"),
			Description: openai.String("Perform basic mathematical calculations"),
			Parameters: openai.F(openai.FunctionParameters{
				"type": "object",
				"properties": map[string]interface{}{
					"expression": map[string]interface{}{
						"type":        "string",
						"description": "Mathematical expression to evaluate, e.g. '2 + 3 * 4'",
					},
				},
				"required": []string{"expression"},
			}),
		}),
	}

	tools := []openai.ChatCompletionToolUnionParam{weatherTool, calculatorTool}

	fmt.Printf("Defined %d function tools:\n", len(tools))
	fmt.Println("1. get_weather - Get weather information")
	fmt.Println("2. calculate - Perform calculations")

	prompt := "What's the weather like in New York and what's 15 * 24?"

	// In actual implementation:
	// completion, err := client.CallWithTools(ctx, prompt, tools)
	// if err != nil {
	//     log.Printf("Function calling failed: %v", err)
	//     return
	// }
	//
	// choice := completion.Choices[0]
	// if len(choice.Message.ToolCalls) > 0 {
	//     fmt.Printf("Model wants to call %d function(s):\n", len(choice.Message.ToolCalls))
	//     for i, toolCall := range choice.Message.ToolCalls {
	//         fmt.Printf("%d. Function: %s\n", i+1, toolCall.Function.Name)
	//         fmt.Printf("   Arguments: %s\n", toolCall.Function.Arguments)
	//     }
	// } else {
	//     fmt.Printf("Direct response: %s\n", choice.Message.Content)
	// }

	fmt.Println("✓ Function calling example completed")
}

// StreamingExample demonstrates streaming responses
func StreamingExample() {
	fmt.Println("\n=== Streaming Response Example ===")

	ctx := context.Background()
	prompt := "Write a short story about a robot learning to paint, but stream the response"

	fmt.Printf("Starting streaming request for: %s\n", prompt)

	// In actual implementation:
	// accumulator, err := client.CallWithPromptStream(ctx, prompt)
	// if err != nil {
	//     log.Printf("Streaming failed: %v", err)
	//     return
	// }
	//
	// fmt.Print("Streaming response: ")
	//
	// // Process streaming chunks
	// for accumulator.HasNext() {
	//     chunk := accumulator.Next()
	//     if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
	//         fmt.Print(chunk.Choices[0].Delta.Content)
	//     }
	// }
	//
	// // Check for errors after streaming
	// if err := accumulator.Err(); err != nil {
	//     log.Printf("\nStreaming error: %v", err)
	//     return
	// }
	//
	// // Get final accumulated result
	// finalCompletion := accumulator.ChatCompletion()
	// fmt.Printf("\n\nStreaming completed. Total tokens: %d\n", finalCompletion.Usage.TotalTokens)

	// Simulate streaming output
	story := "Once upon a time, there was a robot named Artie who discovered the joy of painting..."
	fmt.Print("Streaming response: ")
	for _, char := range story {
		fmt.Print(string(char))
		time.Sleep(50 * time.Millisecond) // Simulate streaming delay
	}
	fmt.Println("\n\n✓ Streaming example completed")
}

// StreamingToWriterExample shows memory-efficient streaming
func StreamingToWriterExample() {
	fmt.Println("\n=== Memory-Efficient Streaming Example ===")

	ctx := context.Background()
	prompt := "Generate a long technical document about Go concurrency patterns"

	// Create a string builder to capture output
	var output strings.Builder

	fmt.Println("Streaming directly to writer (memory-efficient)...")

	// In actual implementation:
	// accumulator, err := client.CallWithPromptStream(ctx, prompt)
	// if err != nil {
	//     log.Printf("Streaming failed: %v", err)
	//     return
	// }
	//
	// // Stream directly to writer without accumulating in memory
	// for accumulator.HasNext() {
	//     chunk := accumulator.Next()
	//     if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
	//         if _, err := output.Write([]byte(chunk.Choices[0].Delta.Content)); err != nil {
	//             log.Printf("Write failed: %v", err)
	//             return
	//         }
	//     }
	// }
	//
	// if err := accumulator.Err(); err != nil {
	//     log.Printf("Streaming error: %v", err)
	//     return
	// }

	// Simulate the streaming to writer
	sampleContent := "Go concurrency is built around goroutines and channels..."
	output.WriteString(sampleContent)

	fmt.Printf("Streamed %d bytes to writer\n", output.Len())
	fmt.Printf("Content preview: %s...\n", output.String()[:min(50, output.Len())])

	fmt.Println("✓ Memory-efficient streaming example completed")
}

// ComplexConversationExample shows a realistic conversation flow
func ComplexConversationExample() {
	fmt.Println("\n=== Complex Conversation Flow Example ===")

	ctx := context.Background()

	// Simulate a code review conversation
	conversation := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage("You are a senior Go developer conducting a code review. Be constructive and specific."),
		openai.UserMessage("Please review this Go function:\n\nfunc processUsers(users []User) {\n    for i := 0; i < len(users); i++ {\n        if users[i].Active {\n            fmt.Println(users[i].Name)\n        }\n    }\n}"),
	}

	fmt.Println("Code review conversation:")
	fmt.Println("System: Senior Go developer role")
	fmt.Println("User: Submitted code for review")

	// In actual implementation, you would:
	// 1. Send initial conversation
	// completion1, err := client.CallWithMessages(ctx, conversation)
	// 2. Add the assistant's response to conversation
	// conversation = append(conversation, openai.AssistantMessage(completion1.Choices[0].Message.Content))
	// 3. Add user follow-up
	// conversation = append(conversation, openai.UserMessage("How would you refactor this?"))
	// 4. Continue the conversation
	// completion2, err := client.CallWithMessages(ctx, conversation)

	// Simulate the conversation flow
	fmt.Println("\nSimulated conversation flow:")
	fmt.Println("Assistant: I see a few improvements we could make...")
	fmt.Println("User: How would you refactor this?")
	fmt.Println("Assistant: Here's a more idiomatic version using range...")

	fmt.Println("✓ Complex conversation example completed")
}

// TemplateWithAdvancedFeaturesExample combines templates with advanced features
func TemplateWithAdvancedFeaturesExample() {
	fmt.Println("\n=== Template + Advanced Features Example ===")

	ctx := context.Background()

	// Template for code generation with streaming
	template := `
You are a {{role}} expert. Generate {{language}} code for:

Task: {{task}}
Requirements:
{{#each requirements}}
- {{this}}
{{/each}}

Please provide well-commented, production-ready code.
`

	variables := `{
		"role": "Go programming",
		"language": "Go",
		"task": "HTTP client with retry logic",
		"requirements": [
			"Exponential backoff",
			"Configurable max retries",
			"Context support",
			"Proper error handling"
		]
	}`

	fmt.Println("Using template with variables for code generation...")
	fmt.Printf("Template variables: %s\n", variables)

	// In actual implementation:
	// 1. Process template with variables
	// completion, err := client.CallWithPromptAndVariables(ctx, template, variables)
	// 2. Or use streaming for long code generation
	// accumulator, err := client.CallWithPromptStream(ctx, processedPrompt)

	fmt.Println("✓ Template + advanced features example completed")
}

// Helper function for min (Go 1.21+)
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// StreamToFile demonstrates streaming to a file
func StreamToFile() {
	fmt.Println("\n=== Stream to File Example ===")

	// Create a temporary file
	file, err := os.CreateTemp("", "openai_stream_*.txt")
	if err != nil {
		log.Printf("Failed to create temp file: %v", err)
		return
	}
	defer os.Remove(file.Name()) // Clean up
	defer file.Close()

	fmt.Printf("Streaming to file: %s\n", file.Name())

	// In actual implementation:
	// ctx := context.Background()
	// accumulator, err := client.CallWithPromptStream(ctx, "Write a long article about AI")
	// if err != nil {
	//     log.Printf("Streaming failed: %v", err)
	//     return
	// }
	//
	// for accumulator.HasNext() {
	//     chunk := accumulator.Next()
	//     if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
	//         if _, err := file.WriteString(chunk.Choices[0].Delta.Content); err != nil {
	//             log.Printf("Write to file failed: %v", err)
	//             return
	//         }
	//     }
	// }

	// Simulate writing to file
	sampleContent := "This is a simulated AI-generated article about artificial intelligence..."
	if _, err := file.WriteString(sampleContent); err != nil {
		log.Printf("Write failed: %v", err)
		return
	}

	// Get file info
	if info, err := file.Stat(); err == nil {
		fmt.Printf("Wrote %d bytes to file\n", info.Size())
	}

	fmt.Println("✓ Stream to file example completed")
}

func main() {
	fmt.Println("OpenAI SDK Migration - Advanced Features Examples")
	fmt.Println("===============================================")

	// Check for API key
	if os.Getenv("OPENAI_API_KEY") == "" {
		fmt.Println("⚠️ Warning: OPENAI_API_KEY environment variable not set")
	}

	// Run advanced examples
	ConversationExample()
	FunctionCallingExample()
	StreamingExample()
	StreamingToWriterExample()
	ComplexConversationExample()
	TemplateWithAdvancedFeaturesExample()
	StreamToFile()

	fmt.Println("\n🚀 All advanced features examples completed!")
	fmt.Println("\nKey benefits of the new SDK:")
	fmt.Println("• Native Go types - no JSON unmarshaling")
	fmt.Println("• Built-in streaming support")
	fmt.Println("• Function calling capabilities")
	fmt.Println("• Multi-turn conversations")
	fmt.Println("• Better error handling")
	fmt.Println("• Improved performance")
}

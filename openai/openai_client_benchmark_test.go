package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/kengibson1111/go-aiprovider/types"
	"github.com/kengibson1111/go-aiprovider/utils"
	"github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/packages/ssestream"
)

// MockOldImplementation simulates the old JSON-based implementation for benchmarking
type MockOldImplementation struct {
	model       string
	maxTokens   int
	temperature float64
	logger      *utils.Logger
}

// Old custom types that were used before SDK migration
type oldOpenAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type oldOpenAIRequest struct {
	Model       string             `json:"model"`
	Messages    []oldOpenAIMessage `json:"messages"`
	MaxTokens   int                `json:"max_tokens"`
	Temperature float64            `json:"temperature"`
}

type oldOpenAIResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// NewMockOldImplementation creates a mock of the old JSON-based implementation
func NewMockOldImplementation(config *types.AIConfig) *MockOldImplementation {
	model := config.Model
	if model == "" {
		model = "gpt-3.5-turbo" // Old default
	}

	maxTokens := config.MaxTokens
	if maxTokens == 0 {
		maxTokens = 1000
	}

	temperature := config.Temperature
	if temperature == 0.0 {
		temperature = 0.7
	}

	return &MockOldImplementation{
		model:       model,
		maxTokens:   maxTokens,
		temperature: temperature,
		logger:      utils.NewLogger("MockOldImplementation"),
	}
}

// CallWithPromptOld simulates the old JSON-based approach with marshaling/unmarshaling overhead
func (m *MockOldImplementation) CallWithPromptOld(ctx context.Context, prompt string) ([]byte, error) {
	// Simulate building the old request structure
	request := oldOpenAIRequest{
		Model: m.model,
		Messages: []oldOpenAIMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		MaxTokens:   m.maxTokens,
		Temperature: m.temperature,
	}

	// Simulate JSON marshaling overhead (old implementation)
	requestBytes, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Simulate processing the request (we'll create a mock response)
	mockResponse := m.createMockResponse(prompt)

	// Simulate JSON marshaling of response (old implementation overhead)
	responseBytes, err := json.Marshal(mockResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	// Simulate the overhead of processing the request bytes (even though we don't use them)
	_ = len(requestBytes)

	return responseBytes, nil
}

// GenerateCompletionOld simulates the old approach with JSON processing overhead
func (m *MockOldImplementation) GenerateCompletionOld(ctx context.Context, req types.CompletionRequest) (*types.CompletionResponse, error) {
	// Build prompt (same logic as current implementation)
	prompt := m.buildCompletionPromptOld(req)

	// Call with old JSON-based approach
	respBytes, err := m.CallWithPromptOld(ctx, prompt)
	if err != nil {
		return &types.CompletionResponse{
			Suggestions: []string{},
			Confidence:  0.0,
			Error:       fmt.Sprintf("ERROR: %v", err),
		}, nil
	}

	// Simulate JSON unmarshaling overhead (old implementation)
	var response oldOpenAIResponse
	if err := json.Unmarshal(respBytes, &response); err != nil {
		return &types.CompletionResponse{
			Suggestions: []string{},
			Confidence:  0.0,
			Error:       fmt.Sprintf("ERROR: failed to unmarshal response: %v", err),
		}, nil
	}

	// Extract suggestions with JSON processing overhead
	suggestions := m.extractCompletionSuggestionsOld(response)
	confidence := m.calculateConfidenceOld(response)

	return &types.CompletionResponse{
		Suggestions: suggestions,
		Confidence:  confidence,
	}, nil
}

// Helper methods for old implementation simulation

func (m *MockOldImplementation) createMockResponse(prompt string) oldOpenAIResponse {
	// Create a realistic mock response for benchmarking
	content := fmt.Sprintf("Mock completion for: %s", prompt[:min(len(prompt), 20)])

	return oldOpenAIResponse{
		ID:      "chatcmpl-mock123",
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   m.model,
		Choices: []struct {
			Index   int `json:"index"`
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		}{
			{
				Index: 0,
				Message: struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				}{
					Role:    "assistant",
					Content: content,
				},
				FinishReason: "stop",
			},
		},
		Usage: struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		}{
			PromptTokens:     len(strings.Fields(prompt)),
			CompletionTokens: len(strings.Fields(content)),
			TotalTokens:      len(strings.Fields(prompt)) + len(strings.Fields(content)),
		},
	}
}

func (m *MockOldImplementation) buildCompletionPromptOld(req types.CompletionRequest) string {
	var prompt strings.Builder

	prompt.WriteString(fmt.Sprintf("You are a code completion assistant for %s. ", req.Language))
	prompt.WriteString("Provide code completions that continue from the cursor position. ")
	prompt.WriteString("Return only the completion text without explanations or markdown formatting.\n\n")

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
	beforeCursor := req.Code[:req.Cursor]
	afterCursor := req.Code[req.Cursor:]

	prompt.WriteString(beforeCursor)
	prompt.WriteString("<CURSOR>")
	prompt.WriteString(afterCursor)
	prompt.WriteString("\n\nProvide the completion for <CURSOR> position:")

	return prompt.String()
}

func (m *MockOldImplementation) extractCompletionSuggestionsOld(response oldOpenAIResponse) []string {
	if len(response.Choices) == 0 {
		return []string{}
	}

	text := response.Choices[0].Message.Content
	if text == "" {
		return []string{}
	}

	text = strings.TrimSpace(text)
	lines := strings.Split(text, "\n")
	var suggestions []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			suggestions = append(suggestions, line)
		}
	}

	if len(suggestions) == 0 {
		return []string{text}
	}

	return suggestions
}

func (m *MockOldImplementation) calculateConfidenceOld(response oldOpenAIResponse) float64 {
	if len(response.Choices) == 0 {
		return 0.0
	}

	confidence := 0.7
	choice := response.Choices[0]

	switch choice.FinishReason {
	case "stop":
		confidence += 0.2
	case "length":
		confidence -= 0.1
	case "content_filter":
		confidence -= 0.3
	}

	if len(choice.Message.Content) > 50 {
		confidence += 0.1
	}

	if confidence > 1.0 {
		confidence = 1.0
	}
	if confidence < 0.0 {
		confidence = 0.0
	}

	return confidence
}

// Mock SDK client for new implementation benchmarking
type BenchmarkMockSDKClient struct{}

func (m *BenchmarkMockSDKClient) Chat() ChatServiceInterface {
	return &BenchmarkMockChatService{}
}

type BenchmarkMockChatService struct{}

func (m *BenchmarkMockChatService) Completions() CompletionsServiceInterface {
	return &BenchmarkMockCompletionsService{}
}

type BenchmarkMockCompletionsService struct{}

func (m *BenchmarkMockCompletionsService) New(ctx context.Context, params openai.ChatCompletionNewParams) (*openai.ChatCompletion, error) {
	// Create a mock SDK response without JSON processing
	prompt := ""
	if len(params.Messages) > 0 {
		// For benchmarking, we'll just use a simple prompt extraction
		prompt = "benchmark prompt"
	}

	content := fmt.Sprintf("Mock completion for: %s", prompt[:min(len(prompt), 20)])

	return &openai.ChatCompletion{
		ID:      "chatcmpl-mock123",
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   "gpt-4o-mini",
		Choices: []openai.ChatCompletionChoice{
			{
				Index: 0,
				Message: openai.ChatCompletionMessage{
					Role:    "assistant",
					Content: content,
				},
				FinishReason: "stop",
			},
		},
		Usage: openai.CompletionUsage{
			PromptTokens:     int64(len(strings.Fields(prompt))),
			CompletionTokens: int64(len(strings.Fields(content))),
			TotalTokens:      int64(len(strings.Fields(prompt)) + len(strings.Fields(content))),
		},
	}, nil
}

func (m *BenchmarkMockCompletionsService) NewStreaming(ctx context.Context, params openai.ChatCompletionNewParams) *ssestream.Stream[openai.ChatCompletionChunk] {
	// Not used in these benchmarks
	return nil
}

// Benchmark functions

// BenchmarkCallWithPrompt_OldImplementation benchmarks the old JSON-based approach
func BenchmarkCallWithPrompt_OldImplementation(b *testing.B) {
	config := &types.AIConfig{
		APIKey:      "test-key",
		Model:       "gpt-3.5-turbo",
		MaxTokens:   1000,
		Temperature: 0.7,
	}

	oldClient := NewMockOldImplementation(config)
	ctx := context.Background()
	prompt := "Write a function that calculates the factorial of a number in Go"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := oldClient.CallWithPromptOld(ctx, prompt)
		if err != nil {
			b.Fatalf("CallWithPromptOld failed: %v", err)
		}
	}
}

// BenchmarkCallWithPrompt_NewImplementation benchmarks the new SDK-based approach
func BenchmarkCallWithPrompt_NewImplementation(b *testing.B) {
	config := &types.AIConfig{
		APIKey:      "test-key",
		Model:       "gpt-4o-mini",
		MaxTokens:   1000,
		Temperature: 0.7,
	}

	// Create client with mock SDK
	client := &OpenAIClient{
		client:      &BenchmarkMockSDKClient{},
		model:       config.Model,
		maxTokens:   config.MaxTokens,
		temperature: config.Temperature,
		logger:      utils.NewLogger("BenchmarkClient"),
	}

	ctx := context.Background()
	prompt := "Write a function that calculates the factorial of a number in Go"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := client.CallWithPrompt(ctx, prompt)
		if err != nil {
			b.Fatalf("CallWithPrompt failed: %v", err)
		}
	}
}

// BenchmarkGenerateCompletion_OldImplementation benchmarks old completion generation
func BenchmarkGenerateCompletion_OldImplementation(b *testing.B) {
	config := &types.AIConfig{
		APIKey:      "test-key",
		Model:       "gpt-3.5-turbo",
		MaxTokens:   1000,
		Temperature: 0.7,
	}

	oldClient := NewMockOldImplementation(config)
	ctx := context.Background()

	testCode := "func factorial(n int) int {\n\tif n <= 1 {\n\t\treturn 1\n\t}\n\treturn n * factorial("
	req := types.CompletionRequest{
		Language: "go",
		Code:     testCode,
		Cursor:   len(testCode),
		Context: types.CodeContext{
			Imports:         []string{"fmt"},
			CurrentFunction: "factorial",
			ProjectType:     "library",
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := oldClient.GenerateCompletionOld(ctx, req)
		if err != nil {
			b.Fatalf("GenerateCompletionOld failed: %v", err)
		}
	}
}

// BenchmarkGenerateCompletion_NewImplementation benchmarks new completion generation
func BenchmarkGenerateCompletion_NewImplementation(b *testing.B) {
	config := &types.AIConfig{
		APIKey:      "test-key",
		Model:       "gpt-4o-mini",
		MaxTokens:   1000,
		Temperature: 0.7,
	}

	// Create client with mock SDK
	client := &OpenAIClient{
		client:      &BenchmarkMockSDKClient{},
		model:       config.Model,
		maxTokens:   config.MaxTokens,
		temperature: config.Temperature,
		logger:      utils.NewLogger("BenchmarkClient"),
	}

	ctx := context.Background()

	testCode := "func factorial(n int) int {\n\tif n <= 1 {\n\t\treturn 1\n\t}\n\treturn n * factorial("
	req := types.CompletionRequest{
		Language: "go",
		Code:     testCode,
		Cursor:   len(testCode),
		Context: types.CodeContext{
			Imports:         []string{"fmt"},
			CurrentFunction: "factorial",
			ProjectType:     "library",
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := client.GenerateCompletion(ctx, req)
		if err != nil {
			b.Fatalf("GenerateCompletion failed: %v", err)
		}
	}
}

// BenchmarkResponseProcessing_OldVsNew compares response processing overhead
func BenchmarkResponseProcessing_OldVsNew(b *testing.B) {
	// Create mock data for both approaches
	mockSDKResponse := &openai.ChatCompletion{
		ID:      "chatcmpl-test123",
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   "gpt-4o-mini",
		Choices: []openai.ChatCompletionChoice{
			{
				Index: 0,
				Message: openai.ChatCompletionMessage{
					Role:    "assistant",
					Content: "println(\"Hello, World!\")",
				},
				FinishReason: "stop",
			},
		},
		Usage: openai.CompletionUsage{
			PromptTokens:     10,
			CompletionTokens: 5,
			TotalTokens:      15,
		},
	}

	mockOldResponse := oldOpenAIResponse{
		ID:      "chatcmpl-test123",
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   "gpt-3.5-turbo",
		Choices: []struct {
			Index   int `json:"index"`
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		}{
			{
				Index: 0,
				Message: struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				}{
					Role:    "assistant",
					Content: "println(\"Hello, World!\")",
				},
				FinishReason: "stop",
			},
		},
		Usage: struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		}{
			PromptTokens:     10,
			CompletionTokens: 5,
			TotalTokens:      15,
		},
	}

	// Benchmark old approach with JSON marshaling/unmarshaling
	b.Run("OldJSONProcessing", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			// Simulate the old approach: marshal to JSON then unmarshal
			jsonBytes, err := json.Marshal(mockOldResponse)
			if err != nil {
				b.Fatalf("Marshal failed: %v", err)
			}

			var response oldOpenAIResponse
			err = json.Unmarshal(jsonBytes, &response)
			if err != nil {
				b.Fatalf("Unmarshal failed: %v", err)
			}

			// Extract content (old way)
			if len(response.Choices) > 0 {
				_ = response.Choices[0].Message.Content
			}
		}
	})

	// Benchmark new approach with direct field access
	b.Run("NewDirectAccess", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			// Direct field access (new way)
			if len(mockSDKResponse.Choices) > 0 {
				_ = mockSDKResponse.Choices[0].Message.Content
			}
		}
	})
}

// BenchmarkMemoryAllocation compares memory allocation patterns
func BenchmarkMemoryAllocation(b *testing.B) {
	prompt := "Write a function that calculates the factorial of a number"

	b.Run("OldImplementation", func(b *testing.B) {
		config := &types.AIConfig{
			APIKey:      "test-key",
			Model:       "gpt-3.5-turbo",
			MaxTokens:   1000,
			Temperature: 0.7,
		}

		oldClient := NewMockOldImplementation(config)
		ctx := context.Background()

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, err := oldClient.CallWithPromptOld(ctx, prompt)
			if err != nil {
				b.Fatalf("CallWithPromptOld failed: %v", err)
			}
		}
	})

	b.Run("NewImplementation", func(b *testing.B) {
		config := &types.AIConfig{
			APIKey:      "test-key",
			Model:       "gpt-4o-mini",
			MaxTokens:   1000,
			Temperature: 0.7,
		}

		client := &OpenAIClient{
			client:      &BenchmarkMockSDKClient{},
			model:       config.Model,
			maxTokens:   config.MaxTokens,
			temperature: config.Temperature,
			logger:      utils.NewLogger("BenchmarkClient"),
		}

		ctx := context.Background()

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, err := client.CallWithPrompt(ctx, prompt)
			if err != nil {
				b.Fatalf("CallWithPrompt failed: %v", err)
			}
		}
	})
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

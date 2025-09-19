package types

import (
	"encoding/json"
	"testing"
)

// TestErrorResponse tests the ErrorResponse struct
func TestErrorResponse(t *testing.T) {
	t.Run("JSON serialization", func(t *testing.T) {
		err := ErrorResponse{
			Code:    "INVALID_REQUEST",
			Message: "The request is invalid",
			Details: "Missing required field: apiKey",
			Retry:   false,
		}

		data, jsonErr := json.Marshal(err)
		if jsonErr != nil {
			t.Fatalf("Failed to marshal ErrorResponse: %v", jsonErr)
		}

		expected := `{"code":"INVALID_REQUEST","message":"The request is invalid","details":"Missing required field: apiKey","retry":false}`
		if string(data) != expected {
			t.Errorf("Expected %s, got %s", expected, string(data))
		}
	})

	t.Run("JSON deserialization", func(t *testing.T) {
		jsonData := `{"code":"RATE_LIMIT","message":"Rate limit exceeded","retry":true}`
		var err ErrorResponse

		jsonErr := json.Unmarshal([]byte(jsonData), &err)
		if jsonErr != nil {
			t.Fatalf("Failed to unmarshal ErrorResponse: %v", jsonErr)
		}

		if err.Code != "RATE_LIMIT" {
			t.Errorf("Expected code 'RATE_LIMIT', got '%s'", err.Code)
		}
		if err.Message != "Rate limit exceeded" {
			t.Errorf("Expected message 'Rate limit exceeded', got '%s'", err.Message)
		}
		if !err.Retry {
			t.Error("Expected retry to be true")
		}
		if err.Details != "" {
			t.Errorf("Expected empty details, got '%s'", err.Details)
		}
	})

	t.Run("JSON serialization without details", func(t *testing.T) {
		err := ErrorResponse{
			Code:    "SERVER_ERROR",
			Message: "Internal server error",
			Retry:   true,
		}

		data, jsonErr := json.Marshal(err)
		if jsonErr != nil {
			t.Fatalf("Failed to marshal ErrorResponse: %v", jsonErr)
		}

		// Details should be omitted when empty due to omitempty tag
		expected := `{"code":"SERVER_ERROR","message":"Internal server error","retry":true}`
		if string(data) != expected {
			t.Errorf("Expected %s, got %s", expected, string(data))
		}
	})
}

// TestAIConfig tests the AIConfig struct
func TestAIConfig(t *testing.T) {
	t.Run("JSON serialization", func(t *testing.T) {
		config := AIConfig{
			Provider:    "openai",
			APIKey:      "sk-test123",
			BaseURL:     "https://api.openai.com/v1",
			Model:       "gpt-4o-mini",
			MaxTokens:   1000,
			Temperature: 0.7,
		}

		data, err := json.Marshal(config)
		if err != nil {
			t.Fatalf("Failed to marshal AIConfig: %v", err)
		}

		expected := `{"provider":"openai","apiKey":"sk-test123","baseUrl":"https://api.openai.com/v1","model":"gpt-4o-mini","maxTokens":1000,"temperature":0.7}`
		if string(data) != expected {
			t.Errorf("Expected %s, got %s", expected, string(data))
		}
	})

	t.Run("JSON deserialization", func(t *testing.T) {
		jsonData := `{"provider":"claude","apiKey":"sk-ant-test","model":"claude-3-sonnet","maxTokens":2000,"temperature":0.5}`
		var config AIConfig

		err := json.Unmarshal([]byte(jsonData), &config)
		if err != nil {
			t.Fatalf("Failed to unmarshal AIConfig: %v", err)
		}

		if config.Provider != "claude" {
			t.Errorf("Expected provider 'claude', got '%s'", config.Provider)
		}
		if config.APIKey != "sk-ant-test" {
			t.Errorf("Expected apiKey 'sk-ant-test', got '%s'", config.APIKey)
		}
		if config.Model != "claude-3-sonnet" {
			t.Errorf("Expected model 'claude-3-sonnet', got '%s'", config.Model)
		}
		if config.MaxTokens != 2000 {
			t.Errorf("Expected maxTokens 2000, got %d", config.MaxTokens)
		}
		if config.Temperature != 0.5 {
			t.Errorf("Expected temperature 0.5, got %f", config.Temperature)
		}
		if config.BaseURL != "" {
			t.Errorf("Expected empty baseUrl, got '%s'", config.BaseURL)
		}
	})

	t.Run("JSON serialization without optional fields", func(t *testing.T) {
		config := AIConfig{
			Provider:    "claude",
			APIKey:      "sk-ant-test",
			Model:       "claude-3-haiku",
			MaxTokens:   500,
			Temperature: 0.3,
		}

		data, err := json.Marshal(config)
		if err != nil {
			t.Fatalf("Failed to marshal AIConfig: %v", err)
		}

		// BaseURL should be omitted when empty due to omitempty tag
		expected := `{"provider":"claude","apiKey":"sk-ant-test","model":"claude-3-haiku","maxTokens":500,"temperature":0.3}`
		if string(data) != expected {
			t.Errorf("Expected %s, got %s", expected, string(data))
		}
	})

	t.Run("Zero values", func(t *testing.T) {
		var config AIConfig

		if config.Provider != "" {
			t.Errorf("Expected empty provider, got '%s'", config.Provider)
		}
		if config.MaxTokens != 0 {
			t.Errorf("Expected maxTokens 0, got %d", config.MaxTokens)
		}
		if config.Temperature != 0.0 {
			t.Errorf("Expected temperature 0.0, got %f", config.Temperature)
		}
	})
}

// TestCompletionRequest tests the CompletionRequest struct
func TestCompletionRequest(t *testing.T) {
	t.Run("JSON serialization", func(t *testing.T) {
		context := CodeContext{
			CurrentFunction: "handleRequest",
			Imports:         []string{"import React from 'react'", "import axios from 'axios'"},
			ProjectType:     "React",
			RecentChanges:   []string{"Added error handling", "Updated API endpoint"},
		}

		request := CompletionRequest{
			Code:     "function handleRequest() {\n  // cursor here\n}",
			Cursor:   35,
			Language: "typescript",
			Context:  context,
		}

		data, err := json.Marshal(request)
		if err != nil {
			t.Fatalf("Failed to marshal CompletionRequest: %v", err)
		}

		// Verify the JSON contains expected fields
		var unmarshaled map[string]any
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Fatalf("Failed to unmarshal for verification: %v", err)
		}

		if unmarshaled["code"] != "function handleRequest() {\n  // cursor here\n}" {
			t.Errorf("Unexpected code field: %v", unmarshaled["code"])
		}
		if unmarshaled["cursor"] != float64(35) {
			t.Errorf("Unexpected cursor field: %v", unmarshaled["cursor"])
		}
		if unmarshaled["language"] != "typescript" {
			t.Errorf("Unexpected language field: %v", unmarshaled["language"])
		}
	})

	t.Run("JSON deserialization", func(t *testing.T) {
		jsonData := `{
			"code": "const x = 1;",
			"cursor": 10,
			"language": "javascript",
			"context": {
				"currentFunction": "main",
				"imports": ["import fs from 'fs'"],
				"projectType": "Node.js",
				"recentChanges": ["Fixed bug"]
			}
		}`

		var request CompletionRequest
		err := json.Unmarshal([]byte(jsonData), &request)
		if err != nil {
			t.Fatalf("Failed to unmarshal CompletionRequest: %v", err)
		}

		if request.Code != "const x = 1;" {
			t.Errorf("Expected code 'const x = 1;', got '%s'", request.Code)
		}
		if request.Cursor != 10 {
			t.Errorf("Expected cursor 10, got %d", request.Cursor)
		}
		if request.Language != "javascript" {
			t.Errorf("Expected language 'javascript', got '%s'", request.Language)
		}
		if request.Context.CurrentFunction != "main" {
			t.Errorf("Expected currentFunction 'main', got '%s'", request.Context.CurrentFunction)
		}
	})

	t.Run("Empty context", func(t *testing.T) {
		request := CompletionRequest{
			Code:     "console.log('hello');",
			Cursor:   0,
			Language: "javascript",
			Context:  CodeContext{},
		}

		data, err := json.Marshal(request)
		if err != nil {
			t.Fatalf("Failed to marshal CompletionRequest with empty context: %v", err)
		}

		var unmarshaled CompletionRequest
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Fatalf("Failed to unmarshal CompletionRequest: %v", err)
		}

		if unmarshaled.Context.CurrentFunction != "" {
			t.Errorf("Expected empty currentFunction, got '%s'", unmarshaled.Context.CurrentFunction)
		}
	})
}

// TestCompletionResponse tests the CompletionResponse struct
func TestCompletionResponse(t *testing.T) {
	t.Run("JSON serialization with suggestions", func(t *testing.T) {
		response := CompletionResponse{
			Suggestions: []string{"console.log('hello world');", "console.error('error');"},
			Confidence:  0.85,
		}

		data, err := json.Marshal(response)
		if err != nil {
			t.Fatalf("Failed to marshal CompletionResponse: %v", err)
		}

		expected := `{"suggestions":["console.log('hello world');","console.error('error');"],"confidence":0.85}`
		if string(data) != expected {
			t.Errorf("Expected %s, got %s", expected, string(data))
		}
	})

	t.Run("JSON serialization with error", func(t *testing.T) {
		response := CompletionResponse{
			Suggestions: []string{},
			Confidence:  0.0,
			Error:       "API rate limit exceeded",
		}

		data, err := json.Marshal(response)
		if err != nil {
			t.Fatalf("Failed to marshal CompletionResponse: %v", err)
		}

		expected := `{"suggestions":[],"confidence":0,"error":"API rate limit exceeded"}`
		if string(data) != expected {
			t.Errorf("Expected %s, got %s", expected, string(data))
		}
	})

	t.Run("JSON deserialization", func(t *testing.T) {
		jsonData := `{"suggestions":["if (condition) {","  return true;","}"],"confidence":0.92}`
		var response CompletionResponse

		err := json.Unmarshal([]byte(jsonData), &response)
		if err != nil {
			t.Fatalf("Failed to unmarshal CompletionResponse: %v", err)
		}

		expectedSuggestions := []string{"if (condition) {", "  return true;", "}"}
		if len(response.Suggestions) != len(expectedSuggestions) {
			t.Errorf("Expected %d suggestions, got %d", len(expectedSuggestions), len(response.Suggestions))
		}
		for i, expected := range expectedSuggestions {
			if i < len(response.Suggestions) && response.Suggestions[i] != expected {
				t.Errorf("Expected suggestion[%d] '%s', got '%s'", i, expected, response.Suggestions[i])
			}
		}
		if response.Confidence != 0.92 {
			t.Errorf("Expected confidence 0.92, got %f", response.Confidence)
		}
		if response.Error != "" {
			t.Errorf("Expected empty error, got '%s'", response.Error)
		}
	})

	t.Run("JSON serialization without error field", func(t *testing.T) {
		response := CompletionResponse{
			Suggestions: []string{"return value;"},
			Confidence:  0.75,
		}

		data, err := json.Marshal(response)
		if err != nil {
			t.Fatalf("Failed to marshal CompletionResponse: %v", err)
		}

		// Error should be omitted when empty due to omitempty tag
		expected := `{"suggestions":["return value;"],"confidence":0.75}`
		if string(data) != expected {
			t.Errorf("Expected %s, got %s", expected, string(data))
		}
	})

	t.Run("Empty suggestions", func(t *testing.T) {
		response := CompletionResponse{
			Suggestions: []string{},
			Confidence:  0.0,
		}

		data, err := json.Marshal(response)
		if err != nil {
			t.Fatalf("Failed to marshal CompletionResponse: %v", err)
		}

		expected := `{"suggestions":[],"confidence":0}`
		if string(data) != expected {
			t.Errorf("Expected %s, got %s", expected, string(data))
		}
	})

	t.Run("Nil suggestions", func(t *testing.T) {
		response := CompletionResponse{
			Suggestions: nil,
			Confidence:  0.0,
		}

		data, err := json.Marshal(response)
		if err != nil {
			t.Fatalf("Failed to marshal CompletionResponse: %v", err)
		}

		expected := `{"suggestions":null,"confidence":0}`
		if string(data) != expected {
			t.Errorf("Expected %s, got %s", expected, string(data))
		}
	})
}

// TestCodeGenerationRequest tests the CodeGenerationRequest struct
func TestCodeGenerationRequest(t *testing.T) {
	t.Run("JSON serialization", func(t *testing.T) {
		context := CodeContext{
			CurrentFunction: "processData",
			Imports:         []string{"import pandas as pd", "import numpy as np"},
			ProjectType:     "Python",
			RecentChanges:   []string{"Added data validation"},
		}

		request := CodeGenerationRequest{
			Prompt:   "Create a function that validates user input",
			Context:  context,
			Language: "python",
		}

		data, err := json.Marshal(request)
		if err != nil {
			t.Fatalf("Failed to marshal CodeGenerationRequest: %v", err)
		}

		// Verify the JSON contains expected fields
		var unmarshaled map[string]any
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Fatalf("Failed to unmarshal for verification: %v", err)
		}

		if unmarshaled["prompt"] != "Create a function that validates user input" {
			t.Errorf("Unexpected prompt field: %v", unmarshaled["prompt"])
		}
		if unmarshaled["language"] != "python" {
			t.Errorf("Unexpected language field: %v", unmarshaled["language"])
		}
	})

	t.Run("JSON deserialization", func(t *testing.T) {
		jsonData := `{
			"prompt": "Generate a REST API endpoint",
			"language": "go",
			"context": {
				"currentFunction": "handleAPI",
				"imports": ["net/http", "encoding/json"],
				"projectType": "Go HTTP",
				"recentChanges": ["Added middleware"]
			}
		}`

		var request CodeGenerationRequest
		err := json.Unmarshal([]byte(jsonData), &request)
		if err != nil {
			t.Fatalf("Failed to unmarshal CodeGenerationRequest: %v", err)
		}

		if request.Prompt != "Generate a REST API endpoint" {
			t.Errorf("Expected prompt 'Generate a REST API endpoint', got '%s'", request.Prompt)
		}
		if request.Language != "go" {
			t.Errorf("Expected language 'go', got '%s'", request.Language)
		}
		if request.Context.CurrentFunction != "handleAPI" {
			t.Errorf("Expected currentFunction 'handleAPI', got '%s'", request.Context.CurrentFunction)
		}
		if len(request.Context.Imports) != 2 {
			t.Errorf("Expected 2 imports, got %d", len(request.Context.Imports))
		}
	})

	t.Run("Empty prompt", func(t *testing.T) {
		request := CodeGenerationRequest{
			Prompt:   "",
			Context:  CodeContext{},
			Language: "javascript",
		}

		data, err := json.Marshal(request)
		if err != nil {
			t.Fatalf("Failed to marshal CodeGenerationRequest with empty prompt: %v", err)
		}

		var unmarshaled CodeGenerationRequest
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Fatalf("Failed to unmarshal CodeGenerationRequest: %v", err)
		}

		if unmarshaled.Prompt != "" {
			t.Errorf("Expected empty prompt, got '%s'", unmarshaled.Prompt)
		}
	})

	t.Run("Complex context", func(t *testing.T) {
		context := CodeContext{
			CurrentFunction: "complexFunction",
			Imports:         []string{"import React, { useState, useEffect } from 'react'", "import axios from 'axios'"},
			ProjectType:     "React",
			RecentChanges:   []string{"Added hooks", "Updated state management", "Fixed memory leak"},
		}

		request := CodeGenerationRequest{
			Prompt:   "Create a custom React hook for data fetching",
			Context:  context,
			Language: "typescript",
		}

		data, err := json.Marshal(request)
		if err != nil {
			t.Fatalf("Failed to marshal CodeGenerationRequest: %v", err)
		}

		var unmarshaled CodeGenerationRequest
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Fatalf("Failed to unmarshal CodeGenerationRequest: %v", err)
		}

		if len(unmarshaled.Context.RecentChanges) != 3 {
			t.Errorf("Expected 3 recent changes, got %d", len(unmarshaled.Context.RecentChanges))
		}
	})
}

// TestCodeGenerationResponse tests the CodeGenerationResponse struct
func TestCodeGenerationResponse(t *testing.T) {
	t.Run("JSON serialization with code", func(t *testing.T) {
		response := CodeGenerationResponse{
			Code: "function validateInput(input) {\n  return input && input.length > 0;\n}",
		}

		data, err := json.Marshal(response)
		if err != nil {
			t.Fatalf("Failed to marshal CodeGenerationResponse: %v", err)
		}

		// Verify by unmarshaling and checking the content
		var unmarshaled CodeGenerationResponse
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Fatalf("Failed to unmarshal CodeGenerationResponse: %v", err)
		}

		expectedCode := "function validateInput(input) {\n  return input && input.length > 0;\n}"
		if unmarshaled.Code != expectedCode {
			t.Errorf("Expected code '%s', got '%s'", expectedCode, unmarshaled.Code)
		}
	})

	t.Run("JSON serialization with error", func(t *testing.T) {
		response := CodeGenerationResponse{
			Code:  "",
			Error: "Unable to generate code: insufficient context",
		}

		data, err := json.Marshal(response)
		if err != nil {
			t.Fatalf("Failed to marshal CodeGenerationResponse: %v", err)
		}

		expected := `{"code":"","error":"Unable to generate code: insufficient context"}`
		if string(data) != expected {
			t.Errorf("Expected %s, got %s", expected, string(data))
		}
	})

	t.Run("JSON deserialization", func(t *testing.T) {
		jsonData := `{"code":"def validate_email(email):\n    return '@' in email and '.' in email"}`
		var response CodeGenerationResponse

		err := json.Unmarshal([]byte(jsonData), &response)
		if err != nil {
			t.Fatalf("Failed to unmarshal CodeGenerationResponse: %v", err)
		}

		expectedCode := "def validate_email(email):\n    return '@' in email and '.' in email"
		if response.Code != expectedCode {
			t.Errorf("Expected code '%s', got '%s'", expectedCode, response.Code)
		}
		if response.Error != "" {
			t.Errorf("Expected empty error, got '%s'", response.Error)
		}
	})

	t.Run("JSON serialization without error field", func(t *testing.T) {
		response := CodeGenerationResponse{
			Code: "const result = processData(input);",
		}

		data, err := json.Marshal(response)
		if err != nil {
			t.Fatalf("Failed to marshal CodeGenerationResponse: %v", err)
		}

		// Error should be omitted when empty due to omitempty tag
		expected := `{"code":"const result = processData(input);"}`
		if string(data) != expected {
			t.Errorf("Expected %s, got %s", expected, string(data))
		}
	})

	t.Run("Empty code and error", func(t *testing.T) {
		response := CodeGenerationResponse{
			Code:  "",
			Error: "",
		}

		data, err := json.Marshal(response)
		if err != nil {
			t.Fatalf("Failed to marshal CodeGenerationResponse: %v", err)
		}

		expected := `{"code":""}`
		if string(data) != expected {
			t.Errorf("Expected %s, got %s", expected, string(data))
		}
	})

	t.Run("Multiline code", func(t *testing.T) {
		multilineCode := `package main

import "fmt"

func main() {
    fmt.Println("Hello, World!")
}`

		response := CodeGenerationResponse{
			Code: multilineCode,
		}

		data, err := json.Marshal(response)
		if err != nil {
			t.Fatalf("Failed to marshal CodeGenerationResponse: %v", err)
		}

		var unmarshaled CodeGenerationResponse
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Fatalf("Failed to unmarshal CodeGenerationResponse: %v", err)
		}

		if unmarshaled.Code != multilineCode {
			t.Errorf("Multiline code not preserved correctly")
		}
	})
}

// TestStructValidation tests validation logic for configuration structs
func TestStructValidation(t *testing.T) {
	t.Run("AIConfig validation scenarios", func(t *testing.T) {
		// Test valid configuration
		validConfig := AIConfig{
			Provider:    "openai",
			APIKey:      "sk-test123",
			Model:       "gpt-4o-mini",
			MaxTokens:   1000,
			Temperature: 0.7,
		}

		// Ensure valid config can be marshaled/unmarshaled
		data, err := json.Marshal(validConfig)
		if err != nil {
			t.Fatalf("Valid config should marshal without error: %v", err)
		}

		var unmarshaled AIConfig
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Fatalf("Valid config should unmarshal without error: %v", err)
		}

		// Test edge case values
		edgeConfig := AIConfig{
			Provider:    "claude",
			APIKey:      "sk-ant-" + string(make([]byte, 100)), // Very long API key
			Model:       "claude-3-opus-20240229",
			MaxTokens:   0,   // Zero tokens
			Temperature: 2.0, // High temperature
		}

		data, err = json.Marshal(edgeConfig)
		if err != nil {
			t.Fatalf("Edge case config should marshal without error: %v", err)
		}

		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Fatalf("Edge case config should unmarshal without error: %v", err)
		}
	})

	t.Run("Error response formatting", func(t *testing.T) {
		// Test different error scenarios
		testCases := []struct {
			name     string
			response ErrorResponse
		}{
			{
				name: "Authentication error",
				response: ErrorResponse{
					Code:    "AUTH_FAILED",
					Message: "Invalid API key",
					Details: "The provided API key is not valid",
					Retry:   false,
				},
			},
			{
				name: "Rate limit error",
				response: ErrorResponse{
					Code:    "RATE_LIMIT",
					Message: "Too many requests",
					Retry:   true,
				},
			},
			{
				name: "Server error",
				response: ErrorResponse{
					Code:    "SERVER_ERROR",
					Message: "Internal server error",
					Details: "Temporary server issue",
					Retry:   true,
				},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				data, err := json.Marshal(tc.response)
				if err != nil {
					t.Fatalf("Failed to marshal error response: %v", err)
				}

				var unmarshaled ErrorResponse
				if err := json.Unmarshal(data, &unmarshaled); err != nil {
					t.Fatalf("Failed to unmarshal error response: %v", err)
				}

				if unmarshaled.Code != tc.response.Code {
					t.Errorf("Code mismatch: expected %s, got %s", tc.response.Code, unmarshaled.Code)
				}
				if unmarshaled.Message != tc.response.Message {
					t.Errorf("Message mismatch: expected %s, got %s", tc.response.Message, unmarshaled.Message)
				}
				if unmarshaled.Retry != tc.response.Retry {
					t.Errorf("Retry mismatch: expected %t, got %t", tc.response.Retry, unmarshaled.Retry)
				}
			})
		}
	})
}

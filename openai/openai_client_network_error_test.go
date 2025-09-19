package openai

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kengibson1111/go-aiprovider/types"
	"github.com/openai/openai-go/v2"
)

// TestOpenAIClient_NetworkFailureScenarios tests real network failure scenarios
// This covers requirement 8.4: Test network failure scenarios
func TestOpenAIClient_NetworkFailureScenarios(t *testing.T) {
	tests := []struct {
		name          string
		setupServer   func() *httptest.Server
		expectError   bool
		errorContains string
		timeout       time.Duration
	}{
		{
			name: "server connection refused",
			setupServer: func() *httptest.Server {
				// Create server but close it immediately to simulate connection refused
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
				server.Close()
				return server
			},
			expectError:   true,
			errorContains: "network error",
			timeout:       5 * time.Second,
		},
		{
			name: "server timeout - no response",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Never respond to simulate timeout
					time.Sleep(10 * time.Second)
				}))
			},
			expectError:   true,
			errorContains: "timeout",
			timeout:       1 * time.Second,
		},
		{
			name: "server returns non-JSON response",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "text/html")
					w.WriteHeader(http.StatusOK)
					w.Write([]byte("<html><body>Not JSON</body></html>"))
				}))
			},
			expectError:   true,
			errorContains: "request failed",
			timeout:       5 * time.Second,
		},
		{
			name: "server returns malformed JSON",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`{"incomplete": json`))
				}))
			},
			expectError:   true,
			errorContains: "request failed",
			timeout:       5 * time.Second,
		},
		{
			name: "server closes connection during request",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Close connection immediately
					hj, ok := w.(http.Hijacker)
					if !ok {
						http.Error(w, "webserver doesn't support hijacking", http.StatusInternalServerError)
						return
					}
					conn, _, err := hj.Hijack()
					if err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
					conn.Close()
				}))
			},
			expectError:   true,
			errorContains: "network error",
			timeout:       5 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupServer()
			defer server.Close()

			config := &types.AIConfig{
				Provider: "openai",
				APIKey:   "test-key",
				BaseURL:  server.URL,
				Model:    "gpt-4o-mini",
			}

			client, err := NewOpenAIClient(config)
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), tt.timeout)
			defer cancel()

			_, err = client.CallWithPrompt(ctx, "test prompt")

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.errorContains)) {
					t.Errorf("Expected error to contain '%s', got: %s", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

// TestOpenAIClient_RealNetworkConditions tests with various network conditions
func TestOpenAIClient_RealNetworkConditions(t *testing.T) {
	t.Run("invalid hostname", func(t *testing.T) {
		config := &types.AIConfig{
			Provider: "openai",
			APIKey:   "test-key",
			BaseURL:  "https://invalid-hostname-that-does-not-exist.com",
			Model:    "gpt-4o-mini",
		}

		client, err := NewOpenAIClient(config)
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err = client.CallWithPrompt(ctx, "test prompt")
		if err == nil {
			t.Errorf("Expected error for invalid hostname")
			return
		}

		if !strings.Contains(err.Error(), "network error") {
			t.Errorf("Expected network error, got: %v", err)
		}
	})

	t.Run("localhost connection refused", func(t *testing.T) {
		// Find an unused port
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("Failed to find unused port: %v", err)
		}
		port := listener.Addr().(*net.TCPAddr).Port
		listener.Close()

		config := &types.AIConfig{
			Provider: "openai",
			APIKey:   "test-key",
			BaseURL:  "http://127.0.0.1:" + string(rune(port)),
			Model:    "gpt-4o-mini",
		}

		client, err := NewOpenAIClient(config)
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		_, err = client.CallWithPrompt(ctx, "test prompt")
		if err == nil {
			t.Errorf("Expected error for connection refused")
			return
		}

		if !strings.Contains(err.Error(), "network error") {
			t.Errorf("Expected network error, got: %v", err)
		}
	})
}

// TestOpenAIClient_ErrorHandlingConsistency verifies error handling is consistent across methods
func TestOpenAIClient_ErrorHandlingConsistency(t *testing.T) {
	// Create a server that always returns rate limit error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error": {"message": "Rate limit exceeded", "type": "rate_limit_error", "code": "rate_limit_exceeded"}}`))
	}))
	defer server.Close()

	config := &types.AIConfig{
		Provider: "openai",
		APIKey:   "test-key",
		BaseURL:  server.URL,
		Model:    "gpt-4o-mini",
	}

	client, err := NewOpenAIClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	// Test that all methods handle the same error consistently
	methods := []struct {
		name string
		test func() error
	}{
		{
			name: "CallWithPrompt",
			test: func() error {
				_, err := client.CallWithPrompt(ctx, "test")
				return err
			},
		},
		{
			name: "CallWithMessages",
			test: func() error {
				messages := []openai.ChatCompletionMessageParamUnion{
					openai.UserMessage("test"),
				}
				_, err := client.CallWithMessages(ctx, messages)
				return err
			},
		},
		{
			name: "CallWithTools",
			test: func() error {
				tools := []openai.ChatCompletionToolUnionParam{}
				_, err := client.CallWithTools(ctx, "test", tools)
				return err
			},
		},
		{
			name: "ValidateCredentials",
			test: func() error {
				return client.ValidateCredentials(ctx)
			},
		},
	}

	for _, method := range methods {
		t.Run(method.name, func(t *testing.T) {
			err := method.test()
			if err == nil {
				t.Errorf("Expected error but got none")
				return
			}

			if !strings.Contains(err.Error(), "rate limit exceeded") {
				t.Errorf("Expected rate limit error, got: %v", err)
			}
		})
	}
}

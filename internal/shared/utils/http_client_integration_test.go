//go:build integration

package utils

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kengibson1111/go-aiprovider/internal/shared/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// HTTPClientIntegrationTestSuite tests BaseHTTPClient with real HTTP servers
type HTTPClientIntegrationTestSuite struct {
	suite.Suite
	cleanupCwd func()
}

func TestHTTPClientIntegrationTestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	suite.Run(t, new(HTTPClientIntegrationTestSuite))
}

func (s *HTTPClientIntegrationTestSuite) SetupSuite() {
	testutil.SetupEnvironment(s.T(), "../../../")
	s.cleanupCwd = testutil.SetupCurrentDirectory(s.T(), "../../../")
}

func (s *HTTPClientIntegrationTestSuite) TearDownSuite() {
	if s.cleanupCwd != nil {
		s.cleanupCwd()
	}
}

// TestDoRequest_SuccessfulGET verifies a successful GET request through the full HTTP stack
func (s *HTTPClientIntegrationTestSuite) TestDoRequest_SuccessfulGET() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal("GET", r.Method)
		s.Equal("/api/test", r.URL.Path)
		s.Equal("application/json", r.Header.Get("Content-Type"))
		s.Equal("Go-AIProvider/1.0", r.Header.Get("User-Agent"))

		w.Header().Set("X-Custom-Header", "test-value")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	client := NewBaseHTTPClient(server.URL, "test-api-key", 10*time.Second)

	resp, err := client.DoRequest(context.Background(), HTTPRequest{
		Method: "GET",
		Path:   "/api/test",
	})

	require.NoError(s.T(), err)
	require.NotNil(s.T(), resp)
	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)
	assert.Equal(s.T(), `{"status":"ok"}`, string(resp.Body))
	assert.Equal(s.T(), []string{"test-value"}, resp.Headers["X-Custom-Header"])
}

// TestDoRequest_SuccessfulPOST verifies a POST request with body and custom headers
func (s *HTTPClientIntegrationTestSuite) TestDoRequest_SuccessfulPOST() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal("POST", r.Method)
		s.Equal("/v1/messages", r.URL.Path)
		s.Equal("test-key-123", r.Header.Get("x-api-key"))
		s.Equal("2023-06-01", r.Header.Get("anthropic-version"))

		body, err := io.ReadAll(r.Body)
		s.NoError(err)
		s.Contains(string(body), `"model":"claude-3"`)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"msg_123","type":"message"}`))
	}))
	defer server.Close()

	client := NewBaseHTTPClient(server.URL, "test-key-123", 10*time.Second)

	resp, err := client.DoRequest(context.Background(), HTTPRequest{
		Method: "POST",
		Path:   "/v1/messages",
		Headers: map[string]string{
			"x-api-key":         "test-key-123",
			"anthropic-version": "2023-06-01",
		},
		Body: strings.NewReader(`{"model":"claude-3","messages":[{"role":"user","content":"Hello"}]}`),
	})

	require.NoError(s.T(), err)
	require.NotNil(s.T(), resp)
	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)
	assert.Contains(s.T(), string(resp.Body), "msg_123")
}

// TestDoRequest_ErrorStatusCodes verifies the client handles various HTTP error status codes
func (s *HTTPClientIntegrationTestSuite) TestDoRequest_ErrorStatusCodes() {
	testCases := []struct {
		name       string
		statusCode int
		body       string
	}{
		{
			name:       "bad_request_400",
			statusCode: http.StatusBadRequest,
			body:       `{"error":{"type":"invalid_request","message":"bad request"}}`,
		},
		{
			name:       "unauthorized_401",
			statusCode: http.StatusUnauthorized,
			body:       `{"error":{"type":"authentication_error","message":"invalid api key"}}`,
		},
		{
			name:       "forbidden_403",
			statusCode: http.StatusForbidden,
			body:       `{"error":{"type":"permission_error","message":"not allowed"}}`,
		},
		{
			name:       "not_found_404",
			statusCode: http.StatusNotFound,
			body:       `{"error":{"type":"not_found","message":"resource not found"}}`,
		},
		{
			name:       "rate_limited_429",
			statusCode: http.StatusTooManyRequests,
			body:       `{"error":{"type":"rate_limit","message":"too many requests"}}`,
		},
		{
			name:       "server_error_500",
			statusCode: http.StatusInternalServerError,
			body:       `{"error":{"type":"server_error","message":"internal error"}}`,
		},
		{
			name:       "service_unavailable_503",
			statusCode: http.StatusServiceUnavailable,
			body:       `{"error":{"type":"overloaded","message":"service unavailable"}}`,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
				w.Write([]byte(tc.body))
			}))
			defer server.Close()

			client := NewBaseHTTPClient(server.URL, "test-key", 10*time.Second)

			resp, err := client.DoRequest(context.Background(), HTTPRequest{
				Method: "GET",
				Path:   "/api/test",
			})

			// DoRequest returns the response even for error status codes (no transport error)
			require.NoError(s.T(), err)
			require.NotNil(s.T(), resp)
			assert.Equal(s.T(), tc.statusCode, resp.StatusCode)
			assert.Equal(s.T(), tc.body, string(resp.Body))
		})
	}
}

// TestDoRequest_RetryOnTransientFailure verifies retry logic when the server fails then recovers
func (s *HTTPClientIntegrationTestSuite) TestDoRequest_RetryOnTransientFailure() {
	var requestCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&requestCount, 1)
		if count <= 2 {
			// Force a connection close to simulate a network error on first 2 attempts
			hj, ok := w.(http.Hijacker)
			if ok {
				conn, _, _ := hj.Hijack()
				conn.Close()
				return
			}
		}
		// Third attempt succeeds
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"recovered":true}`))
	}))
	defer server.Close()

	client := NewBaseHTTPClient(server.URL, "test-key", 10*time.Second)
	// Use a short backoff for testing by overriding the HTTP client timeout
	client.HttpClient.Timeout = 5 * time.Second

	resp, err := client.DoRequest(context.Background(), HTTPRequest{
		Method: "GET",
		Path:   "/api/retry-test",
	})

	require.NoError(s.T(), err)
	require.NotNil(s.T(), resp)
	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)
	assert.Contains(s.T(), string(resp.Body), "recovered")
	assert.GreaterOrEqual(s.T(), atomic.LoadInt32(&requestCount), int32(3), "Should have retried at least twice before succeeding")
}

// TestDoRequest_ContextCancellation verifies the client respects context cancellation
func (s *HTTPClientIntegrationTestSuite) TestDoRequest_ContextCancellation() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate a slow server
		time.Sleep(5 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewBaseHTTPClient(server.URL, "test-key", 30*time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := client.DoRequest(ctx, HTTPRequest{
		Method: "GET",
		Path:   "/api/slow",
	})
	elapsed := time.Since(start)

	require.Error(s.T(), err)
	// Should fail quickly due to context cancellation, not wait for the full 5s
	assert.Less(s.T(), elapsed, 3*time.Second, "Request should be cancelled quickly by context timeout")
}

// TestDoRequest_BaseURLTrailingSlash verifies trailing slash normalization on base URL
func (s *HTTPClientIntegrationTestSuite) TestDoRequest_BaseURLTrailingSlash() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal("/api/endpoint", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	// Create client with trailing slash on base URL
	client := NewBaseHTTPClient(server.URL+"/", "test-key", 10*time.Second)

	resp, err := client.DoRequest(context.Background(), HTTPRequest{
		Method: "GET",
		Path:   "/api/endpoint",
	})

	require.NoError(s.T(), err)
	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)
}

// TestDoRequest_CustomHeadersOverrideDefaults verifies custom headers override defaults
func (s *HTTPClientIntegrationTestSuite) TestDoRequest_CustomHeadersOverrideDefaults() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Custom Content-Type should override the default application/json
		s.Equal("text/plain", r.Header.Get("Content-Type"))
		s.Equal("CustomAgent/2.0", r.Header.Get("User-Agent"))
		s.Equal("bearer test-token", r.Header.Get("Authorization"))

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	client := NewBaseHTTPClient(server.URL, "test-key", 10*time.Second)

	resp, err := client.DoRequest(context.Background(), HTTPRequest{
		Method: "POST",
		Path:   "/api/custom",
		Headers: map[string]string{
			"Content-Type":  "text/plain",
			"User-Agent":    "CustomAgent/2.0",
			"Authorization": "bearer test-token",
		},
		Body: strings.NewReader("plain text body"),
	})

	require.NoError(s.T(), err)
	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)
}

// TestDoRequest_LargeResponseBody verifies the client handles large response bodies
func (s *HTTPClientIntegrationTestSuite) TestDoRequest_LargeResponseBody() {
	// Generate a ~100KB response
	largeBody := strings.Repeat(`{"data":"`+strings.Repeat("x", 990)+`"},`, 100)
	largeBody = "[" + largeBody[:len(largeBody)-1] + "]"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(largeBody))
	}))
	defer server.Close()

	client := NewBaseHTTPClient(server.URL, "test-key", 10*time.Second)

	resp, err := client.DoRequest(context.Background(), HTTPRequest{
		Method: "GET",
		Path:   "/api/large",
	})

	require.NoError(s.T(), err)
	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)
	assert.Equal(s.T(), len(largeBody), len(resp.Body), "Full response body should be read")
}

// TestValidateResponse_WithRealServerResponses verifies ValidateResponse against actual HTTP responses
func (s *HTTPClientIntegrationTestSuite) TestValidateResponse_WithRealServerResponses() {
	testCases := []struct {
		name        string
		statusCode  int
		body        string
		expectError bool
		errContains string
	}{
		{
			name:        "success_200",
			statusCode:  http.StatusOK,
			body:        `{"result":"success"}`,
			expectError: false,
		},
		{
			name:        "created_201",
			statusCode:  http.StatusCreated,
			body:        `{"id":"new-resource"}`,
			expectError: false,
		},
		{
			name:        "no_content_204",
			statusCode:  http.StatusNoContent,
			body:        "",
			expectError: false,
		},
		{
			name:        "bad_request_400",
			statusCode:  http.StatusBadRequest,
			body:        `{"error":"invalid input"}`,
			expectError: true,
			errContains: "HTTP error: 400",
		},
		{
			name:        "rate_limited_429",
			statusCode:  http.StatusTooManyRequests,
			body:        `{"error":"rate limited"}`,
			expectError: true,
			errContains: "retryable error",
		},
		{
			name:        "server_error_500",
			statusCode:  http.StatusInternalServerError,
			body:        `{"error":"internal"}`,
			expectError: true,
			errContains: "retryable error",
		},
		{
			name:        "bad_gateway_502",
			statusCode:  http.StatusBadGateway,
			body:        "Bad Gateway",
			expectError: true,
			errContains: "retryable error",
		},
		{
			name:        "service_unavailable_503",
			statusCode:  http.StatusServiceUnavailable,
			body:        "Service Unavailable",
			expectError: true,
			errContains: "retryable error",
		},
		{
			name:        "gateway_timeout_504",
			statusCode:  http.StatusGatewayTimeout,
			body:        "Gateway Timeout",
			expectError: true,
			errContains: "retryable error",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
				w.Write([]byte(tc.body))
			}))
			defer server.Close()

			client := NewBaseHTTPClient(server.URL, "test-key", 10*time.Second)

			resp, err := client.DoRequest(context.Background(), HTTPRequest{
				Method: "GET",
				Path:   "/api/validate",
			})
			require.NoError(s.T(), err)

			validationErr := client.ValidateResponse(resp)
			if tc.expectError {
				require.Error(s.T(), validationErr)
				assert.Contains(s.T(), validationErr.Error(), tc.errContains)
			} else {
				assert.NoError(s.T(), validationErr)
			}
		})
	}
}

// TestIsRetryableError_WithRealServerResponses verifies retryable error detection from real responses
func (s *HTTPClientIntegrationTestSuite) TestIsRetryableError_WithRealServerResponses() {
	testCases := []struct {
		name        string
		statusCode  int
		isRetryable bool
	}{
		{"ok_200", http.StatusOK, false},
		{"bad_request_400", http.StatusBadRequest, false},
		{"unauthorized_401", http.StatusUnauthorized, false},
		{"forbidden_403", http.StatusForbidden, false},
		{"not_found_404", http.StatusNotFound, false},
		{"rate_limited_429", http.StatusTooManyRequests, true},
		{"server_error_500", http.StatusInternalServerError, true},
		{"bad_gateway_502", http.StatusBadGateway, true},
		{"service_unavailable_503", http.StatusServiceUnavailable, true},
		{"gateway_timeout_504", http.StatusGatewayTimeout, true},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
				w.Write([]byte(fmt.Sprintf(`{"status":%d}`, tc.statusCode)))
			}))
			defer server.Close()

			client := NewBaseHTTPClient(server.URL, "test-key", 10*time.Second)

			resp, err := client.DoRequest(context.Background(), HTTPRequest{
				Method: "GET",
				Path:   "/api/check",
			})
			require.NoError(s.T(), err)

			assert.Equal(s.T(), tc.isRetryable, client.IsRetryableError(resp.StatusCode),
				"Status %d retryable expectation mismatch", tc.statusCode)
		})
	}
}

// TestDoRequest_ConnectionRefused verifies behavior when the server is unreachable
func (s *HTTPClientIntegrationTestSuite) TestDoRequest_ConnectionRefused() {
	// Use a server that's immediately closed to simulate connection refused
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	serverURL := server.URL
	server.Close() // Close immediately so connections are refused

	// Use a very short timeout to avoid long waits during retry backoff
	client := NewBaseHTTPClient(serverURL, "test-key", 1*time.Second)
	client.HttpClient.Timeout = 1 * time.Second

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	_, err := client.DoRequest(ctx, HTTPRequest{
		Method: "GET",
		Path:   "/api/unreachable",
	})

	require.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "request failed after")
}

// TestDoRequest_ConcurrentRequests verifies the client handles concurrent requests correctly
func (s *HTTPClientIntegrationTestSuite) TestDoRequest_ConcurrentRequests() {
	var requestCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&requestCount, 1)
		// Small delay to simulate real work
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf(`{"request":%d}`, count)))
	}))
	defer server.Close()

	client := NewBaseHTTPClient(server.URL, "test-key", 10*time.Second)
	concurrency := 10

	type result struct {
		resp *HTTPResponse
		err  error
	}

	results := make(chan result, concurrency)

	for i := 0; i < concurrency; i++ {
		go func(idx int) {
			resp, err := client.DoRequest(context.Background(), HTTPRequest{
				Method: "GET",
				Path:   fmt.Sprintf("/api/concurrent/%d", idx),
			})
			results <- result{resp: resp, err: err}
		}(i)
	}

	for i := 0; i < concurrency; i++ {
		r := <-results
		require.NoError(s.T(), r.err, "Concurrent request should not fail")
		require.NotNil(s.T(), r.resp)
		assert.Equal(s.T(), http.StatusOK, r.resp.StatusCode)
	}

	assert.Equal(s.T(), int32(concurrency), atomic.LoadInt32(&requestCount),
		"All concurrent requests should reach the server")
}

// TestDoRequest_EmptyResponseBody verifies handling of empty response bodies
func (s *HTTPClientIntegrationTestSuite) TestDoRequest_EmptyResponseBody() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
		// No body written
	}))
	defer server.Close()

	client := NewBaseHTTPClient(server.URL, "test-key", 10*time.Second)

	resp, err := client.DoRequest(context.Background(), HTTPRequest{
		Method: "DELETE",
		Path:   "/api/resource/123",
	})

	require.NoError(s.T(), err)
	require.NotNil(s.T(), resp)
	assert.Equal(s.T(), http.StatusNoContent, resp.StatusCode)
	assert.Empty(s.T(), resp.Body)
}

// TestNewBaseHTTPClient_Configuration verifies client construction and configuration
func (s *HTTPClientIntegrationTestSuite) TestNewBaseHTTPClient_Configuration() {
	testCases := []struct {
		name            string
		baseURL         string
		apiKey          string
		timeout         time.Duration
		expectedBaseURL string
	}{
		{
			name:            "standard_url",
			baseURL:         "https://api.example.com",
			apiKey:          "key-123",
			timeout:         30 * time.Second,
			expectedBaseURL: "https://api.example.com",
		},
		{
			name:            "trailing_slash_removed",
			baseURL:         "https://api.example.com/",
			apiKey:          "key-456",
			timeout:         15 * time.Second,
			expectedBaseURL: "https://api.example.com",
		},
		{
			name:            "with_path_prefix",
			baseURL:         "https://api.example.com/v1/",
			apiKey:          "key-789",
			timeout:         60 * time.Second,
			expectedBaseURL: "https://api.example.com/v1",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			client := NewBaseHTTPClient(tc.baseURL, tc.apiKey, tc.timeout)

			require.NotNil(s.T(), client)
			assert.Equal(s.T(), tc.apiKey, client.ApiKey)
			assert.NotNil(s.T(), client.HttpClient)
			assert.Equal(s.T(), tc.timeout, client.HttpClient.Timeout)
		})
	}
}

// TestDoRequest_ResponseHeadersPropagated verifies all response headers are captured
func (s *HTTPClientIntegrationTestSuite) TestDoRequest_ResponseHeadersPropagated() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Request-Id", "req-abc-123")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.Header().Set("X-RateLimit-Reset", "1609459200")
		w.Header().Add("X-Multi-Value", "value1")
		w.Header().Add("X-Multi-Value", "value2")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client := NewBaseHTTPClient(server.URL, "test-key", 10*time.Second)

	resp, err := client.DoRequest(context.Background(), HTTPRequest{
		Method: "GET",
		Path:   "/api/headers",
	})

	require.NoError(s.T(), err)
	require.NotNil(s.T(), resp.Headers)
	assert.Equal(s.T(), []string{"req-abc-123"}, resp.Headers["X-Request-Id"])
	assert.Equal(s.T(), []string{"99"}, resp.Headers["X-Ratelimit-Remaining"])
	assert.Equal(s.T(), []string{"value1", "value2"}, resp.Headers["X-Multi-Value"])
}

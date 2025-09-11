package utils

import (
	"bytes"
	"log"
	"os"
	"strings"
	"testing"
)

func TestNewLogger(t *testing.T) {
	tests := []struct {
		name   string
		prefix string
	}{
		{
			name:   "creates logger with simple prefix",
			prefix: "TEST",
		},
		{
			name:   "creates logger with empty prefix",
			prefix: "",
		},
		{
			name:   "creates logger with complex prefix",
			prefix: "AI-CLIENT-MODULE",
		},
		{
			name:   "creates logger with special characters",
			prefix: "test-123_module",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewLogger(tt.prefix)

			if logger == nil {
				t.Fatal("NewLogger returned nil")
			}

			if logger.prefix != tt.prefix {
				t.Errorf("Expected prefix %q, got %q", tt.prefix, logger.prefix)
			}
		})
	}
}

func TestLogger_Info(t *testing.T) {
	tests := []struct {
		name           string
		prefix         string
		message        string
		args           []any
		expectedPrefix string
		expectedLevel  string
		expectedMsg    string
	}{
		{
			name:           "logs simple info message",
			prefix:         "TEST",
			message:        "simple message",
			args:           nil,
			expectedPrefix: "[TEST]",
			expectedLevel:  "INFO:",
			expectedMsg:    "simple message",
		},
		{
			name:           "logs info message with format args",
			prefix:         "CLIENT",
			message:        "user %s has %d items",
			args:           []any{"john", 5},
			expectedPrefix: "[CLIENT]",
			expectedLevel:  "INFO:",
			expectedMsg:    "user john has 5 items",
		},
		{
			name:           "logs info message with empty prefix",
			prefix:         "",
			message:        "test message",
			args:           nil,
			expectedPrefix: "[]",
			expectedLevel:  "INFO:",
			expectedMsg:    "test message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture log output
			var buf bytes.Buffer
			log.SetOutput(&buf)
			defer log.SetOutput(os.Stderr) // Reset to default

			logger := NewLogger(tt.prefix)
			logger.Info(tt.message, tt.args...)

			output := buf.String()

			// Verify the log contains expected components
			if !strings.Contains(output, tt.expectedPrefix) {
				t.Errorf("Expected log to contain prefix %q, got: %s", tt.expectedPrefix, output)
			}

			if !strings.Contains(output, tt.expectedLevel) {
				t.Errorf("Expected log to contain level %q, got: %s", tt.expectedLevel, output)
			}

			if !strings.Contains(output, tt.expectedMsg) {
				t.Errorf("Expected log to contain message %q, got: %s", tt.expectedMsg, output)
			}
		})
	}
}

func TestLogger_Error(t *testing.T) {
	tests := []struct {
		name           string
		prefix         string
		message        string
		args           []any
		expectedPrefix string
		expectedLevel  string
		expectedMsg    string
	}{
		{
			name:           "logs simple error message",
			prefix:         "ERROR_TEST",
			message:        "something went wrong",
			args:           nil,
			expectedPrefix: "[ERROR_TEST]",
			expectedLevel:  "ERROR:",
			expectedMsg:    "something went wrong",
		},
		{
			name:           "logs error message with format args",
			prefix:         "API",
			message:        "failed to connect to %s with status %d",
			args:           []any{"api.example.com", 500},
			expectedPrefix: "[API]",
			expectedLevel:  "ERROR:",
			expectedMsg:    "failed to connect to api.example.com with status 500",
		},
		{
			name:           "logs error message with multiple format types",
			prefix:         "DB",
			message:        "query failed: %s, retries: %d, success: %t",
			args:           []any{"timeout", 3, false},
			expectedPrefix: "[DB]",
			expectedLevel:  "ERROR:",
			expectedMsg:    "query failed: timeout, retries: 3, success: false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture log output
			var buf bytes.Buffer
			log.SetOutput(&buf)
			defer log.SetOutput(os.Stderr) // Reset to default

			logger := NewLogger(tt.prefix)
			logger.Error(tt.message, tt.args...)

			output := buf.String()

			// Verify the log contains expected components
			if !strings.Contains(output, tt.expectedPrefix) {
				t.Errorf("Expected log to contain prefix %q, got: %s", tt.expectedPrefix, output)
			}

			if !strings.Contains(output, tt.expectedLevel) {
				t.Errorf("Expected log to contain level %q, got: %s", tt.expectedLevel, output)
			}

			if !strings.Contains(output, tt.expectedMsg) {
				t.Errorf("Expected log to contain message %q, got: %s", tt.expectedMsg, output)
			}
		})
	}
}

func TestLogger_Warn(t *testing.T) {
	tests := []struct {
		name           string
		prefix         string
		message        string
		args           []any
		expectedPrefix string
		expectedLevel  string
		expectedMsg    string
	}{
		{
			name:           "logs simple warning message",
			prefix:         "WARN_TEST",
			message:        "this is a warning",
			args:           nil,
			expectedPrefix: "[WARN_TEST]",
			expectedLevel:  "WARN:",
			expectedMsg:    "this is a warning",
		},
		{
			name:           "logs warning message with format args",
			prefix:         "CONFIG",
			message:        "deprecated option %s will be removed in version %s",
			args:           []any{"old_flag", "2.0"},
			expectedPrefix: "[CONFIG]",
			expectedLevel:  "WARN:",
			expectedMsg:    "deprecated option old_flag will be removed in version 2.0",
		},
		{
			name:           "logs warning with numeric formatting",
			prefix:         "PERF",
			message:        "slow operation took %.2f seconds",
			args:           []any{1.2345},
			expectedPrefix: "[PERF]",
			expectedLevel:  "WARN:",
			expectedMsg:    "slow operation took 1.23 seconds",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture log output
			var buf bytes.Buffer
			log.SetOutput(&buf)
			defer log.SetOutput(os.Stderr) // Reset to default

			logger := NewLogger(tt.prefix)
			logger.Warn(tt.message, tt.args...)

			output := buf.String()

			// Verify the log contains expected components
			if !strings.Contains(output, tt.expectedPrefix) {
				t.Errorf("Expected log to contain prefix %q, got: %s", tt.expectedPrefix, output)
			}

			if !strings.Contains(output, tt.expectedLevel) {
				t.Errorf("Expected log to contain level %q, got: %s", tt.expectedLevel, output)
			}

			if !strings.Contains(output, tt.expectedMsg) {
				t.Errorf("Expected log to contain message %q, got: %s", tt.expectedMsg, output)
			}
		})
	}
}

func TestLogger_MessageStructure(t *testing.T) {
	tests := []struct {
		name    string
		prefix  string
		logFunc func(*Logger, string, ...any)
		message string
		args    []any
		level   string
	}{
		{
			name:    "info message structure",
			prefix:  "STRUCT_TEST",
			logFunc: (*Logger).Info,
			message: "test message",
			args:    nil,
			level:   "INFO",
		},
		{
			name:    "error message structure",
			prefix:  "STRUCT_TEST",
			logFunc: (*Logger).Error,
			message: "error message",
			args:    nil,
			level:   "ERROR",
		},
		{
			name:    "warn message structure",
			prefix:  "STRUCT_TEST",
			logFunc: (*Logger).Warn,
			message: "warning message",
			args:    nil,
			level:   "WARN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture log output
			var buf bytes.Buffer
			log.SetOutput(&buf)
			defer log.SetOutput(os.Stderr) // Reset to default

			logger := NewLogger(tt.prefix)
			tt.logFunc(logger, tt.message, tt.args...)

			output := strings.TrimSpace(buf.String())

			// Verify message structure: timestamp [PREFIX] LEVEL: message
			// The exact timestamp format may vary, so we check the structure
			expectedPattern := "[" + tt.prefix + "] " + tt.level + ": " + tt.message

			if !strings.Contains(output, expectedPattern) {
				t.Errorf("Expected log output to contain pattern %q, got: %s", expectedPattern, output)
			}

			// Verify the log starts with a timestamp (contains date/time info)
			// Go's default log format includes date and time
			if len(output) < len(expectedPattern) {
				t.Errorf("Log output seems too short, expected at least timestamp + pattern, got: %s", output)
			}
		})
	}
}

func TestLogger_EdgeCases(t *testing.T) {
	t.Run("handles nil pointer args", func(t *testing.T) {
		var buf bytes.Buffer
		log.SetOutput(&buf)
		defer log.SetOutput(os.Stderr)

		logger := NewLogger("TEST")
		var nilStr *string
		logger.Info("message with %v", nilStr)

		output := buf.String()
		if !strings.Contains(output, "INFO:") {
			t.Error("Expected log to contain INFO level")
		}
		if !strings.Contains(output, "<nil>") {
			t.Error("Expected log to contain <nil> for nil pointer")
		}
	})

	t.Run("handles empty message", func(t *testing.T) {
		var buf bytes.Buffer
		log.SetOutput(&buf)
		defer log.SetOutput(os.Stderr)

		logger := NewLogger("TEST")
		logger.Info("")

		output := buf.String()
		if !strings.Contains(output, "[TEST] INFO:") {
			t.Error("Expected log to contain prefix and level even with empty message")
		}
	})

	t.Run("handles message without format specifiers", func(t *testing.T) {
		var buf bytes.Buffer
		log.SetOutput(&buf)
		defer log.SetOutput(os.Stderr)

		logger := NewLogger("TEST")
		logger.Info("simple message without formatting")

		output := buf.String()
		// Should contain the message as-is since no formatting
		if !strings.Contains(output, "simple message without formatting") {
			t.Errorf("Expected log to contain original message, got: %s", output)
		}
	})

	t.Run("handles multiple format specifiers", func(t *testing.T) {
		var buf bytes.Buffer
		log.SetOutput(&buf)
		defer log.SetOutput(os.Stderr)

		logger := NewLogger("TEST")
		logger.Info("message %s %s %s", "arg1", "arg2", "arg3")

		output := buf.String()
		// Should handle gracefully - Go's fmt.Sprintf handles this
		if !strings.Contains(output, "message arg1 arg2 arg3") {
			t.Errorf("Expected log to contain formatted message, got: %s", output)
		}
	})
}

func TestLogger_ConcurrentAccess(t *testing.T) {
	// Test that multiple goroutines can use the same logger safely
	logger := NewLogger("CONCURRENT")

	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	done := make(chan bool, 3)

	// Start multiple goroutines logging simultaneously
	go func() {
		for i := 0; i < 10; i++ {
			logger.Info("goroutine 1 message %d", i)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 10; i++ {
			logger.Error("goroutine 2 error %d", i)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 10; i++ {
			logger.Warn("goroutine 3 warning %d", i)
		}
		done <- true
	}()

	// Wait for all goroutines to complete
	for i := 0; i < 3; i++ {
		<-done
	}

	output := buf.String()

	// Verify we got messages from all levels
	if !strings.Contains(output, "INFO:") {
		t.Error("Expected to find INFO messages in concurrent test")
	}
	if !strings.Contains(output, "ERROR:") {
		t.Error("Expected to find ERROR messages in concurrent test")
	}
	if !strings.Contains(output, "WARN:") {
		t.Error("Expected to find WARN messages in concurrent test")
	}

	// Count total messages (should be 30)
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 30 {
		t.Errorf("Expected 30 log lines, got %d", len(lines))
	}
}

func TestLogger_PrefixVariations(t *testing.T) {
	tests := []struct {
		name   string
		prefix string
	}{
		{"single character", "A"},
		{"numeric prefix", "123"},
		{"mixed alphanumeric", "Test123"},
		{"with hyphens", "test-module"},
		{"with underscores", "test_module"},
		{"with dots", "test.module"},
		{"long prefix", "very-long-module-name-for-testing"},
		{"unicode characters", "测试"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			log.SetOutput(&buf)
			defer log.SetOutput(os.Stderr)

			logger := NewLogger(tt.prefix)
			logger.Info("test message")

			output := buf.String()
			expectedPrefix := "[" + tt.prefix + "]"

			if !strings.Contains(output, expectedPrefix) {
				t.Errorf("Expected log to contain prefix %q, got: %s", expectedPrefix, output)
			}
		})
	}
}

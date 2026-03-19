package logging

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestDefaultLogger_Debug(t *testing.T) {
	tests := []struct {
		name    string
		verbose bool
		message string
		args    []any
		expect  bool
	}{
		{
			name:    "verbose enabled - should log debug",
			verbose: true,
			message: "debug message: %s",
			args:    []any{"test"},
			expect:  true,
		},
		{
			name:    "verbose disabled - should not log debug",
			verbose: false,
			message: "debug message: %s",
			args:    []any{"test"},
			expect:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			logger := NewDefaultLogger()
			logger.SetVerbose(tt.verbose)
			logger.Debug(tt.message, tt.args...)

			w.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			buf.ReadFrom(r)
			output := buf.String()

			if tt.expect {
				if !strings.Contains(output, "[DEBUG]") {
					t.Errorf("Expected debug log output, got: %s", output)
				}
				if !strings.Contains(output, "test") {
					t.Errorf("Expected 'test' in output, got: %s", output)
				}
			} else {
				if output != "" {
					t.Errorf("Expected no debug output when verbose=false, got: %s", output)
				}
			}
		})
	}
}

func TestDefaultLogger_Info(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	logger := NewDefaultLogger()
	logger.SetVerbose(false)
	logger.Info("test info message: %s", "value")

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "[INFO]") {
		t.Errorf("Expected [INFO] in output, got: %s", output)
	}
	if !strings.Contains(output, "test info message: value") {
		t.Errorf("Expected info message in output, got: %s", output)
	}
}

func TestDefaultLogger_Warn(t *testing.T) {
	// Capture stdout (logger outputs to stdout, not stderr)
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	logger := NewDefaultLogger()
	logger.SetVerbose(false)
	logger.Warn("test warning message: %s", "value")

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "[WARN]") {
		t.Errorf("Expected [WARN] in output, got: %s", output)
	}
	if !strings.Contains(output, "test warning message: value") {
		t.Errorf("Expected warning message in output, got: %s", output)
	}
}

func TestDefaultLogger_Error(t *testing.T) {
	// Capture stdout (logger outputs to stdout, not stderr)
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	logger := NewDefaultLogger()
	logger.SetVerbose(false)
	logger.Error("test error message: %s", "value")

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "[ERROR]") {
		t.Errorf("Expected [ERROR] in output, got: %s", output)
	}
	if !strings.Contains(output, "test error message: value") {
		t.Errorf("Expected error message in output, got: %s", output)
	}
}

func TestDefaultLogger_SetVerbose(t *testing.T) {
	logger := NewDefaultLogger()
	logger.SetVerbose(false)

	// Test 1: Initially verbose is false, should not log debug
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	logger.Debug("should not appear")

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if output != "" {
		t.Errorf("Expected no output when verbose=false, got: %s", output)
	}

	// Test 2: Enable verbose mode, should log debug
	oldStdout = os.Stdout
	r, w, _ = os.Pipe()
	os.Stdout = w

	logger.SetVerbose(true)
	logger.Debug("should appear")

	w.Close()
	os.Stdout = oldStdout

	buf.Reset()
	buf.ReadFrom(r)
	output = buf.String()

	if !strings.Contains(output, "[DEBUG]") {
		t.Errorf("Expected [DEBUG] in output after SetVerbose(true), got: %s", output)
	}
	if !strings.Contains(output, "should appear") {
		t.Errorf("Expected debug output after SetVerbose(true), got: %s", output)
	}

	// Test 3: Disable verbose mode again, should not log debug
	oldStdout = os.Stdout
	r, w, _ = os.Pipe()
	os.Stdout = w

	logger.SetVerbose(false)
	logger.Debug("should not appear again")

	w.Close()
	os.Stdout = oldStdout

	buf.Reset()
	buf.ReadFrom(r)
	output = buf.String()

	if output != "" {
		t.Errorf("Expected no output after SetVerbose(false), got: %s", output)
	}
}

func TestDefaultLogger_Progress(t *testing.T) {
	tests := []struct {
		name    string
		verbose bool
		expect  bool
	}{
		{
			name:    "verbose enabled - should log progress",
			verbose: true,
			expect:  true,
		},
		{
			name:    "verbose disabled - should not log progress",
			verbose: false,
			expect:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			logger := NewDefaultLogger()
			logger.SetVerbose(tt.verbose)
			logger.Progress("processing item %d", 42)

			w.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			buf.ReadFrom(r)
			output := buf.String()

			if tt.expect {
				if !strings.Contains(output, "[PROGRESS]") {
					t.Errorf("Expected [PROGRESS] in output, got: %s", output)
				}
				if !strings.Contains(output, "processing item 42") {
					t.Errorf("Expected progress message in output, got: %s", output)
				}
			} else {
				if output != "" {
					t.Errorf("Expected no progress output when verbose=false, got: %s", output)
				}
			}
		})
	}
}

func TestDefaultLogger_Status(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	logger := NewDefaultLogger()
	logger.SetVerbose(false) // Status should always show regardless of verbose setting
	logger.Status("operation completed successfully")

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "[INFO]") {
		t.Errorf("Expected [INFO] in output, got: %s", output)
	}
	if !strings.Contains(output, "operation completed successfully") {
		t.Errorf("Expected status message in output, got: %s", output)
	}
}

func TestDefaultLogger_ShouldLog(t *testing.T) {
	tests := []struct {
		name        string
		loggerLevel LogLevel
		testLevel   LogLevel
		expected    bool
	}{
		{
			name:        "debug level logger should log debug",
			loggerLevel: LogLevelDebug,
			testLevel:   LogLevelDebug,
			expected:    true,
		},
		{
			name:        "debug level logger should log info",
			loggerLevel: LogLevelDebug,
			testLevel:   LogLevelInfo,
			expected:    true,
		},
		{
			name:        "info level logger should not log debug",
			loggerLevel: LogLevelInfo,
			testLevel:   LogLevelDebug,
			expected:    false,
		},
		{
			name:        "info level logger should log info",
			loggerLevel: LogLevelInfo,
			testLevel:   LogLevelInfo,
			expected:    true,
		},
		{
			name:        "error level logger should not log warn",
			loggerLevel: LogLevelError,
			testLevel:   LogLevelWarn,
			expected:    false,
		},
		{
			name:        "error level logger should log error",
			loggerLevel: LogLevelError,
			testLevel:   LogLevelError,
			expected:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := &DefaultLogger{Level: tt.loggerLevel}
			result := logger.ShouldLog(tt.testLevel)
			if result != tt.expected {
				t.Errorf("ShouldLog() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestDefaultLogger_IsVerbose(t *testing.T) {
	// Save original environment variables
	originalLogLevel := os.Getenv("LOG_LEVEL")
	originalVerbose := os.Getenv("VERBOSE")

	// Restore environment after test
	defer func() {
		if originalLogLevel != "" {
			os.Setenv("LOG_LEVEL", originalLogLevel)
		} else {
			os.Unsetenv("LOG_LEVEL")
		}
		if originalVerbose != "" {
			os.Setenv("VERBOSE", originalVerbose)
		} else {
			os.Unsetenv("VERBOSE")
		}
	}()

	// Clear environment variables to test default behavior
	os.Unsetenv("LOG_LEVEL")
	os.Unsetenv("VERBOSE")

	logger := NewDefaultLogger()

	// Test default state
	if logger.IsVerbose() {
		t.Errorf("Expected IsVerbose() to be false by default")
	}

	// Test after setting verbose
	logger.SetVerbose(true)
	if !logger.IsVerbose() {
		t.Errorf("Expected IsVerbose() to be true after SetVerbose(true)")
	}

	// Test after disabling verbose
	logger.SetVerbose(false)
	if logger.IsVerbose() {
		t.Errorf("Expected IsVerbose() to be false after SetVerbose(false)")
	}
}

func TestLogLevel_String(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected string
	}{
		{LogLevelDebug, "DEBUG"},
		{LogLevelInfo, "INFO"},
		{LogLevelWarn, "WARN"},
		{LogLevelError, "ERROR"},
		{LogLevel(999), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.level.String()
			if result != tt.expected {
				t.Errorf("String() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestNewDefaultLogger_EnvironmentVariables(t *testing.T) {
	tests := []struct {
		name            string
		logLevel        string
		verbose         string
		expectedLevel   LogLevel
		expectedVerbose bool
	}{
		{
			name:            "default values",
			logLevel:        "",
			verbose:         "",
			expectedLevel:   LogLevelInfo,
			expectedVerbose: false,
		},
		{
			name:            "debug log level",
			logLevel:        "debug",
			verbose:         "",
			expectedLevel:   LogLevelDebug,
			expectedVerbose: false,
		},
		{
			name:            "error log level with verbose",
			logLevel:        "error",
			verbose:         "true",
			expectedLevel:   LogLevelError,
			expectedVerbose: true,
		},
		{
			name:            "warn log level with verbose false",
			logLevel:        "warn",
			verbose:         "false",
			expectedLevel:   LogLevelWarn,
			expectedVerbose: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original environment
			originalLogLevel := os.Getenv("LOG_LEVEL")
			originalVerbose := os.Getenv("VERBOSE")

			// Restore environment after subtest
			defer func() {
				if originalLogLevel != "" {
					os.Setenv("LOG_LEVEL", originalLogLevel)
				} else {
					os.Unsetenv("LOG_LEVEL")
				}
				if originalVerbose != "" {
					os.Setenv("VERBOSE", originalVerbose)
				} else {
					os.Unsetenv("VERBOSE")
				}
			}()

			// Set environment variables
			os.Setenv("LOG_LEVEL", tt.logLevel)
			os.Setenv("VERBOSE", tt.verbose)

			logger := NewDefaultLogger()

			if logger.Level != tt.expectedLevel {
				t.Errorf("Expected Level = %v, got %v", tt.expectedLevel, logger.Level)
			}
			if logger.IsVerbose() != tt.expectedVerbose {
				t.Errorf("Expected verbose = %v, got %v", tt.expectedVerbose, logger.IsVerbose())
			}
		})
	}
}

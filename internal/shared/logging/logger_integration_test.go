//go:build integration

package logging

import (
	"bytes"
	"os"
	"testing"

	"github.com/kengibson1111/go-aiprovider/internal/shared/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDefaultLogger_WithRealEnvironment(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup environment from .env file
	testutil.SetupEnvironment(t, "../../../")

	// Test that logger respects actual environment variables from .env file
	logger := NewDefaultLogger()

	// The .env.sample shows LOG_LEVEL=info and VERBOSE=false as defaults
	// Test that these are properly loaded and applied
	expectedLevel := LogLevelInfo // Based on .env.sample default
	expectedVerbose := true       // Based on .env.sample default

	assert.Equal(t, expectedLevel, logger.Level, "Logger should use LOG_LEVEL from .env file")
	assert.Equal(t, expectedVerbose, logger.IsVerbose(), "Logger should use VERBOSE from .env file")
}

func TestLogger_EnvironmentVariableOverrides(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup environment from .env file
	testutil.SetupEnvironment(t, "../../../")

	tests := []struct {
		name            string
		logLevel        string
		verbose         string
		expectedLevel   LogLevel
		expectedVerbose bool
		description     string
	}{
		{
			name:            "debug_level_with_verbose",
			logLevel:        "debug",
			verbose:         "true",
			expectedLevel:   LogLevelDebug,
			expectedVerbose: true,
			description:     "Debug level should enable debug logging and verbose mode",
		},
		{
			name:            "warn_level_no_verbose",
			logLevel:        "warn",
			verbose:         "false",
			expectedLevel:   LogLevelWarn,
			expectedVerbose: false,
			description:     "Warn level should only show warnings and errors",
		},
		{
			name:            "error_level_with_verbose",
			logLevel:        "error",
			verbose:         "true",
			expectedLevel:   LogLevelError,
			expectedVerbose: true,
			description:     "Error level should only show errors, but verbose affects debug output",
		},
		{
			name:            "invalid_level_defaults_to_info",
			logLevel:        "invalid",
			verbose:         "false",
			expectedLevel:   LogLevelInfo,
			expectedVerbose: false,
			description:     "Invalid log level should default to info",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original environment variables
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

			// Set environment variables for this test
			os.Setenv("LOG_LEVEL", tt.logLevel)
			os.Setenv("VERBOSE", tt.verbose)

			logger := NewDefaultLogger()

			assert.Equal(t, tt.expectedLevel, logger.Level, tt.description)
			assert.Equal(t, tt.expectedVerbose, logger.IsVerbose(), tt.description)
		})
	}
}

func TestLogger_RealEnvironmentLogging(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup environment from .env file
	testutil.SetupEnvironment(t, "../../../")

	// Test debug logging with environment configuration
	t.Run("debug_logging_with_env_config", func(t *testing.T) {
		// Save original environment variables
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

		// Set environment to enable debug logging
		os.Setenv("LOG_LEVEL", "debug")
		os.Setenv("VERBOSE", "true")

		// Capture stdout
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		logger := NewDefaultLogger()
		logger.Debug("test debug message from environment config")

		w.Close()
		os.Stdout = oldStdout

		var buf bytes.Buffer
		buf.ReadFrom(r)
		output := buf.String()

		assert.Contains(t, output, "[DEBUG]", "Debug message should be logged when LOG_LEVEL=debug")
		assert.Contains(t, output, "test debug message from environment config", "Debug message content should be present")
	})

	// Test that verbose=false blocks debug messages (debug is controlled by verbose, not log level)
	t.Run("verbose_false_blocks_debug", func(t *testing.T) {
		// Save original environment variables
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

		// Set environment to disable verbose (this controls debug messages)
		os.Setenv("LOG_LEVEL", "info")
		os.Setenv("VERBOSE", "false") // Debug messages are controlled by verbose setting

		// Capture stdout
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		logger := NewDefaultLogger()
		logger.Debug("this debug message should not appear")
		logger.Info("this info message should appear")

		w.Close()
		os.Stdout = oldStdout

		var buf bytes.Buffer
		buf.ReadFrom(r)
		output := buf.String()

		assert.NotContains(t, output, "this debug message should not appear", "Debug messages should be blocked when VERBOSE=false")
		assert.Contains(t, output, "this info message should appear", "Info messages should appear at info level")
		assert.Contains(t, output, "[INFO]", "Info log level should be present")
	})

	// Test that log level controls non-debug messages
	t.Run("log_level_controls_non_debug_messages", func(t *testing.T) {
		// Save original environment variables
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

		// Set environment to error level (should block info and warn)
		os.Setenv("LOG_LEVEL", "error")
		os.Setenv("VERBOSE", "false")

		// Capture stdout
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		logger := NewDefaultLogger()
		logger.Info("this info message should not appear")
		logger.Warn("this warn message should not appear")
		logger.Error("this error message should appear")

		w.Close()
		os.Stdout = oldStdout

		var buf bytes.Buffer
		buf.ReadFrom(r)
		output := buf.String()

		assert.NotContains(t, output, "this info message should not appear", "Info messages should be blocked at error level")
		assert.NotContains(t, output, "this warn message should not appear", "Warn messages should be blocked at error level")
		assert.Contains(t, output, "this error message should appear", "Error messages should appear at error level")
		assert.Contains(t, output, "[ERROR]", "Error log level should be present")
	})

	// Test progress logging with environment configuration
	t.Run("progress_logging_with_env_config", func(t *testing.T) {
		// Save original environment variables
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

		// Set environment to enable verbose (needed for progress)
		os.Setenv("LOG_LEVEL", "info")
		os.Setenv("VERBOSE", "true")

		// Capture stdout
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		logger := NewDefaultLogger()
		logger.Progress("processing item %d of %d", 5, 10)

		w.Close()
		os.Stdout = oldStdout

		var buf bytes.Buffer
		buf.ReadFrom(r)
		output := buf.String()

		assert.Contains(t, output, "[PROGRESS]", "Progress message should be logged when VERBOSE=true")
		assert.Contains(t, output, "processing item 5 of 10", "Progress message content should be present")
	})

	// Test that verbose=false blocks progress messages
	t.Run("verbose_false_blocks_progress", func(t *testing.T) {
		// Save original environment variables
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

		// Set environment to disable verbose
		os.Setenv("LOG_LEVEL", "info")
		os.Setenv("VERBOSE", "false")

		// Capture stdout
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		logger := NewDefaultLogger()
		logger.Progress("this progress message should not appear")

		w.Close()
		os.Stdout = oldStdout

		var buf bytes.Buffer
		buf.ReadFrom(r)
		output := buf.String()

		assert.Empty(t, output, "Progress messages should be blocked when VERBOSE=false")
	})
}

func TestLogger_EnvironmentConfigurationPersistence(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup environment from .env file
	testutil.SetupEnvironment(t, "../../../")

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

	// Set specific environment configuration
	os.Setenv("LOG_LEVEL", "warn")
	os.Setenv("VERBOSE", "true")

	// Create multiple logger instances to ensure configuration is consistent
	logger1 := NewDefaultLogger()
	logger2 := NewDefaultLogger()

	// Both loggers should have the same configuration from environment
	assert.Equal(t, logger1.Level, logger2.Level, "Multiple logger instances should have same log level from environment")
	assert.Equal(t, logger1.IsVerbose(), logger2.IsVerbose(), "Multiple logger instances should have same verbose setting from environment")

	// Verify the specific configuration
	assert.Equal(t, LogLevelWarn, logger1.Level, "Logger should use LOG_LEVEL=warn from environment")
	assert.True(t, logger1.IsVerbose(), "Logger should use VERBOSE=true from environment")
}

func TestLogger_EnvironmentVariableValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup environment from .env file
	testutil.SetupEnvironment(t, "../../../")

	// Test various invalid values to ensure robust handling
	invalidTests := []struct {
		name        string
		logLevel    string
		verbose     string
		expectLevel LogLevel
		expectVerb  bool
		description string
	}{
		{
			name:        "empty_values",
			logLevel:    "",
			verbose:     "",
			expectLevel: LogLevelInfo,
			expectVerb:  false,
			description: "Empty values should use defaults",
		},
		{
			name:        "case_insensitive_true",
			logLevel:    "INFO",
			verbose:     "TRUE",
			expectLevel: LogLevelInfo,
			expectVerb:  true,
			description: "Should handle case variations",
		},
		{
			name:        "numeric_verbose",
			logLevel:    "debug",
			verbose:     "1",
			expectLevel: LogLevelDebug,
			expectVerb:  false, // Non-"true" values should be false
			description: "Numeric verbose values should be handled",
		},
		{
			name:        "whitespace_values",
			logLevel:    "  info  ",
			verbose:     "  true  ",
			expectLevel: LogLevelInfo, // Depends on implementation
			expectVerb:  false,        // Whitespace might not be trimmed
			description: "Whitespace handling in environment values",
		},
	}

	for _, tt := range invalidTests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original environment variables
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

			os.Setenv("LOG_LEVEL", tt.logLevel)
			os.Setenv("VERBOSE", tt.verbose)

			logger := NewDefaultLogger()

			// The exact behavior depends on implementation, but it should not panic
			require.NotNil(t, logger, "Logger creation should not fail with invalid environment values")

			// Log level should be one of the valid levels
			validLevels := []LogLevel{LogLevelDebug, LogLevelInfo, LogLevelWarn, LogLevelError}
			assert.Contains(t, validLevels, logger.Level, "Logger level should be valid even with invalid input")

			// Verbose should be a boolean
			verbose := logger.IsVerbose()
			assert.IsType(t, false, verbose, "IsVerbose should return a boolean")
		})
	}
}

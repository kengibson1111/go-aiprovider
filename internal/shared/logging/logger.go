package logging

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// LogLevel represents the logging level
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
)

// String returns the string representation of the log level
func (l LogLevel) String() string {
	switch l {
	case LogLevelDebug:
		return "DEBUG"
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarn:
		return "WARN"
	case LogLevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// Logger interface defines the common logging contract used across the application
type Logger interface {
	Debug(format string, args ...any)
	Info(format string, args ...any)
	Warn(format string, args ...any)
	Error(format string, args ...any)
}

// DefaultLogger implements the Logger interface with configurable verbosity
type DefaultLogger struct {
	Level   LogLevel
	verbose bool
}

// NewDefaultLogger creates a new logger with configuration from environment variables
func NewDefaultLogger() *DefaultLogger {
	logger := &DefaultLogger{
		Level:   LogLevelInfo, // Default level
		verbose: false,        // Default verbose setting
	}

	// Read LOG_LEVEL environment variable
	if logLevelStr := os.Getenv("LOG_LEVEL"); logLevelStr != "" {
		switch strings.ToLower(logLevelStr) {
		case "debug":
			logger.Level = LogLevelDebug
		case "info":
			logger.Level = LogLevelInfo
		case "warn", "warning":
			logger.Level = LogLevelWarn
		case "error":
			logger.Level = LogLevelError
		}
	}

	// Read VERBOSE environment variable
	if verboseStr := os.Getenv("VERBOSE"); verboseStr != "" {
		switch strings.ToLower(verboseStr) {
		case "true", "1", "yes", "on":
			logger.verbose = true
		case "false", "0", "no", "off":
			logger.verbose = false
		}
	}

	return logger
}

// SetVerbose overrides the verbose setting (for CLI flag integration)
func (l *DefaultLogger) SetVerbose(verbose bool) {
	l.verbose = verbose
}

// ShouldLog determines if a message at the given level should be logged
func (l *DefaultLogger) ShouldLog(level LogLevel) bool {
	return level >= l.Level
}

// IsVerbose returns true if verbose logging is enabled
func (l *DefaultLogger) IsVerbose() bool {
	return l.verbose
}

// Debug logs a debug message (only when verbose is enabled)
func (l *DefaultLogger) Debug(format string, args ...any) {
	if l.verbose {
		l.log(LogLevelDebug, format, args...)
	}
}

// Info logs an info message
func (l *DefaultLogger) Info(format string, args ...any) {
	if l.ShouldLog(LogLevelInfo) {
		l.log(LogLevelInfo, format, args...)
	}
}

// Warn logs a warning message
func (l *DefaultLogger) Warn(format string, args ...any) {
	if l.ShouldLog(LogLevelWarn) {
		l.log(LogLevelWarn, format, args...)
	}
}

// Error logs an error message
func (l *DefaultLogger) Error(format string, args ...any) {
	if l.ShouldLog(LogLevelError) {
		l.log(LogLevelError, format, args...)
	}
}

// Progress logs progress information (respects verbose setting)
func (l *DefaultLogger) Progress(format string, args ...any) {
	if l.verbose {
		l.log(LogLevelInfo, "[PROGRESS] "+format, args...)
	}
}

// Status logs status information (always shown)
func (l *DefaultLogger) Status(format string, args ...any) {
	l.log(LogLevelInfo, format, args...)
}

// log is the internal logging method
func (l *DefaultLogger) log(level LogLevel, format string, args ...any) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	prefix := fmt.Sprintf("[%s] [%s] ", timestamp, level.String())

	message := fmt.Sprintf(format, args...)
	fmt.Printf("%s%s\n", prefix, message)
}

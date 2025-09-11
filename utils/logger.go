package utils

import (
	"fmt"
	"log"
)

// Logger provides structured logging for WASM module
type Logger struct {
	prefix string
}

// NewLogger creates a new logger instance
func NewLogger(prefix string) *Logger {
	return &Logger{prefix: prefix}
}

// Info logs an info message
func (l *Logger) Info(msg string, args ...interface{}) {
	log.Printf("[%s] INFO: %s", l.prefix, fmt.Sprintf(msg, args...))
}

// Error logs an error message
func (l *Logger) Error(msg string, args ...interface{}) {
	log.Printf("[%s] ERROR: %s", l.prefix, fmt.Sprintf(msg, args...))
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, args ...interface{}) {
	log.Printf("[%s] WARN: %s", l.prefix, fmt.Sprintf(msg, args...))
}

// Debug logs a debug message
func (l *Logger) Debug(msg string, args ...interface{}) {
	log.Printf("[%s] DEBUG: %s", l.prefix, fmt.Sprintf(msg, args...))
}

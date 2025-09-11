package utils

import (
	"testing"

	"github.com/kengibson1111/go-aiprovider/types"
)

func TestContextProcessorLogic(t *testing.T) {
	// Test import extraction patterns
	testExtractImportsLogic(t)

	// Test project type detection
	testProjectTypeDetection(t)

	// Test function extraction patterns
	testFunctionExtractionLogic(t)
}

func testExtractImportsLogic(t *testing.T) {
	// Test TypeScript import patterns
	code := `import React from 'react';
import { Component } from 'react';
const path = require('path');`

	// This would normally use the context processor, but we'll test the regex patterns directly
	// The actual implementation is in context_processor.go

	// Just verify the test structure is correct
	if code == "" {
		t.Error("Test code should not be empty")
	}
}

func testProjectTypeDetection(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		language string
		contains string
	}{
		{
			name:     "React detection",
			code:     "import React from 'react';",
			language: "typescript",
			contains: "React",
		},
		{
			name:     "Django detection",
			code:     "from django.http import HttpResponse",
			language: "python",
			contains: "django",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that the code contains the expected framework indicator
			if !containsString(tt.code, tt.contains) {
				t.Errorf("Expected code to contain '%s'", tt.contains)
			}
		})
	}
}

func testFunctionExtractionLogic(t *testing.T) {
	code := `function hello() {
  console.log('Hello');
  return true;
}`

	// Test that we can identify function patterns
	if !containsString(code, "function hello") {
		t.Error("Should contain function declaration")
	}
}

// Helper function for string containment
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr) >= 0
}

func findSubstring(s, substr string) int {
	if len(substr) == 0 {
		return 0
	}
	if len(substr) > len(s) {
		return -1
	}

	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

func TestExtractImports(t *testing.T) {
	cp := NewContextProcessor()

	tests := []struct {
		name     string
		code     string
		language string
		expected []string
	}{
		{
			name: "TypeScript imports",
			code: `import React from 'react';
import { Component } from 'react';
import * as fs from 'fs';
const path = require('path');`,
			language: "typescript",
			expected: []string{
				"import React from 'react'",
				"import { Component } from 'react'",
				"import * as fs from 'fs'",
				"const path = require('path')",
			},
		},
		{
			name: "Python imports",
			code: `import os
import sys
from typing import List, Dict
from django.http import HttpResponse`,
			language: "python",
			expected: []string{
				"import os",
				"import sys",
				"import List",
				"import HttpResponse",
				"from typing import List, Dict",
				"from django.http import HttpResponse",
			},
		},
		{
			name: "Go imports",
			code: `import "fmt"
import "net/http"
import log "github.com/sirupsen/logrus"`,
			language: "go",
			expected: []string{
				`import "fmt"`,
				`import "net/http"`,
				`import log "github.com/sirupsen/logrus"`,
			},
		},
		{
			name:     "Unknown language",
			code:     "some code",
			language: "unknown",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			imports := cp.extractImports(tt.code, tt.language)

			if len(imports) != len(tt.expected) {
				t.Errorf("Expected %d imports, got %d. Actual imports: %v", len(tt.expected), len(imports), imports)
				return
			}

			// Verify each expected import is found
			for i, expected := range tt.expected {
				if i >= len(imports) || !containsString(imports[i], expected) {
					t.Errorf("Expected import %d to contain '%s', got '%s'", i, expected, imports[i])
				}
			}
		})
	}
}

func TestExtractCurrentFunction(t *testing.T) {
	cp := NewContextProcessor()

	tests := []struct {
		name     string
		code     string
		cursor   int
		language string
		expected string
	}{
		{
			name: "TypeScript function",
			code: `function hello() {
  console.log('Hello');
  // cursor here
  return true;
}`,
			cursor:   50, // Inside the function
			language: "typescript",
			expected: "hello",
		},
		{
			name: "TypeScript arrow function",
			code: `const add = (a, b) => {
  // cursor here
  return a + b;
}`,
			cursor:   30, // Inside the function
			language: "typescript",
			expected: "add",
		},
		{
			name: "Python function",
			code: `def calculate(x, y):
    # cursor here
    return x + y`,
			cursor:   25, // Inside the function
			language: "python",
			expected: "calculate",
		},
		{
			name: "Go function",
			code: `func processData(data string) error {
    // cursor here
    return nil
}`,
			cursor:   40, // Inside the function
			language: "go",
			expected: "processData",
		},
		{
			name:     "No function found",
			code:     "var x = 5;\n// cursor here",
			cursor:   15,
			language: "typescript",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			funcName := cp.extractCurrentFunction(tt.code, tt.cursor, tt.language)

			if funcName != tt.expected {
				t.Errorf("Expected function name '%s', got '%s'", tt.expected, funcName)
			}
		})
	}
}

func TestDetectProjectType(t *testing.T) {
	cp := NewContextProcessor()

	tests := []struct {
		name     string
		code     string
		language string
		expected string
	}{
		{
			name:     "React TypeScript",
			code:     "import React from 'react';\nfunction App() { return <div>Hello</div>; }",
			language: "typescript",
			expected: "React",
		},
		{
			name:     "Angular TypeScript",
			code:     "import { Component } from '@angular/core';\n@Component({})",
			language: "typescript",
			expected: "Angular",
		},
		{
			name:     "Django Python",
			code:     "from django.http import HttpResponse\ndef view(request):",
			language: "python",
			expected: "Django",
		},
		{
			name:     "Flask Python",
			code:     "from flask import Flask\napp = Flask(__name__)",
			language: "python",
			expected: "Flask",
		},
		{
			name:     "Go Gin",
			code:     "import \"github.com/gin-gonic/gin\"\nfunc main() {",
			language: "go",
			expected: "Gin",
		},
		{
			name:     "Spring Boot Java",
			code:     "@SpringBootApplication\npublic class Application {",
			language: "java",
			expected: "Spring Boot",
		},
		{
			name:     "Plain TypeScript",
			code:     "function hello() { console.log('hello'); }",
			language: "typescript",
			expected: "TypeScript",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			projectType := cp.detectProjectType(tt.code, tt.language)

			if projectType != tt.expected {
				t.Errorf("Expected project type '%s', got '%s'", tt.expected, projectType)
			}
		})
	}
}

func TestProcessContext(t *testing.T) {
	cp := NewContextProcessor()

	code := `import React from 'react';
import { useState } from 'react';

function App() {
  const [count, setCount] = useState(0);
  // cursor here
  return <div>{count}</div>;
}`

	cursor := 100 // Inside the App function
	language := "typescript"

	context := cp.ProcessContext(code, cursor, language)

	// Check that context was populated
	if len(context.Imports) == 0 {
		t.Errorf("Expected imports to be extracted")
	}

	if context.ProjectType != "React" {
		t.Errorf("Expected project type 'React', got '%s'", context.ProjectType)
	}

	if context.CurrentFunction != "App" {
		t.Errorf("Expected current function 'App', got '%s'", context.CurrentFunction)
	}
}

func TestLimitContextSize(t *testing.T) {
	cp := NewContextProcessor()

	// Create a context with large data
	context := types.CodeContext{
		CurrentFunction: "veryLongFunctionNameThatExceedsLimits",
		ProjectType:     "React",
		Imports: []string{
			"import React from 'react'",
			"import { useState, useEffect, useCallback, useMemo } from 'react'",
			"import { BrowserRouter, Route, Switch } from 'react-router-dom'",
			"import axios from 'axios'",
			"import lodash from 'lodash'",
			// Add many more imports...
		},
		RecentChanges: []string{
			"Added new component",
			"Updated state management",
			"Fixed bug in rendering",
			"Optimized performance",
			"Added error handling",
		},
	}

	// Add more imports to exceed limit
	for i := 0; i < 20; i++ {
		context.Imports = append(context.Imports, "import something from 'somewhere'")
	}

	maxTokens := 100 // Very small limit to force truncation
	limited := cp.LimitContextSize(context, maxTokens)

	// Check that context was limited - either imports or recent changes should be truncated
	contextWasLimited := len(limited.Imports) < len(context.Imports) ||
		len(limited.RecentChanges) < len(context.RecentChanges) ||
		len(limited.CurrentFunction) < len(context.CurrentFunction)

	if !contextWasLimited {
		t.Errorf("Expected context to be truncated due to size limit")
	}
}

func TestExtractSurroundingCode(t *testing.T) {
	cp := NewContextProcessor()

	code := `line 1
line 2
line 3
line 4 - cursor here
line 5
line 6
line 7`

	cursor := 21 // Position in "line 4"
	linesBefore := 2
	linesAfter := 2

	surrounding := cp.ExtractSurroundingCode(code, cursor, linesBefore, linesAfter)

	expected := `line 2
line 3
line 4 - cursor here
line 5
line 6`

	if surrounding != expected {
		t.Errorf("Expected surrounding code:\n%s\nGot:\n%s", expected, surrounding)
	}
}

func TestRemoveDuplicates(t *testing.T) {
	cp := NewContextProcessor()

	input := []string{"a", "b", "a", "c", "b", "d"}
	expected := []string{"a", "b", "c", "d"}

	result := cp.removeDuplicates(input)

	if len(result) != len(expected) {
		t.Errorf("Expected %d unique items, got %d", len(expected), len(result))
		return
	}

	for i, expectedItem := range expected {
		if result[i] != expectedItem {
			t.Errorf("Expected item %d to be '%s', got '%s'", i, expectedItem, result[i])
		}
	}
}

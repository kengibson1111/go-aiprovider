package utils

import (
	"regexp"
	"strings"

	"github.com/kengibson1111/go-aiprovider/types"
)

// ContextProcessor handles code context extraction and analysis
type ContextProcessor struct {
	logger *Logger
}

// NewContextProcessor creates a new context processor
func NewContextProcessor() *ContextProcessor {
	return &ContextProcessor{
		logger: NewLogger("ContextProcessor"),
	}
}

// ProcessContext extracts and processes code context for AI requests
func (cp *ContextProcessor) ProcessContext(code string, cursor int, language string) types.CodeContext {
	cp.logger.Info("Processing context for language: %s, cursor: %d", language, cursor)

	context := types.CodeContext{
		Imports:       cp.extractImports(code, language),
		ProjectType:   cp.detectProjectType(code, language),
		RecentChanges: []string{}, // Will be populated by extension
	}

	// Extract current function context
	if currentFunc := cp.extractCurrentFunction(code, cursor, language); currentFunc != "" {
		context.CurrentFunction = currentFunc
	}

	cp.logger.Info("Context processed: %d imports, project type: %s", len(context.Imports), context.ProjectType)
	return context
}

// ProcessContextWithStyle processes context and applies style-aware enhancements
func (cp *ContextProcessor) ProcessContextWithStyle(context types.CodeContext, code string, language string) types.CodeContext {
	cp.logger.Info("Processing context with style analysis for language: %s", language)

	// If style analysis is provided, use it to enhance context processing
	if context.StyleAnalysis != nil {
		// Apply style-aware import formatting
		context.Imports = cp.formatImportsWithStyle(context.Imports, context.StyleAnalysis, language)

		// Apply style-aware function context formatting
		if context.CurrentFunction != "" {
			context.CurrentFunction = cp.formatFunctionWithStyle(context.CurrentFunction, context.StyleAnalysis, language)
		}

		// Enhance project type detection with style information
		context.ProjectType = cp.enhanceProjectTypeWithStyle(context.ProjectType, context.StyleAnalysis, language)
	}

	cp.logger.Info("Style-aware context processing completed")
	return context
}

// formatImportsWithStyle formats import statements according to detected style preferences
func (cp *ContextProcessor) formatImportsWithStyle(imports []string, style *types.StyleAnalysis, language string) []string {
	if style == nil || len(imports) == 0 {
		return imports
	}

	formattedImports := make([]string, len(imports))

	for i, imp := range imports {
		formatted := imp

		// Apply indentation style
		if style.Indentation.Type == "tabs" {
			// Convert spaces to tabs if needed
			formatted = cp.convertSpacesToTabs(formatted, style.Indentation.Size)
		} else if style.Indentation.Type == "spaces" {
			// Ensure consistent space indentation
			formatted = cp.normalizeSpaceIndentation(formatted, style.Indentation.Size)
		}

		// Apply linting preferences
		if style.Linting.HasPrettier {
			formatted = cp.applyPrettierStyle(formatted, style.Linting.PrettierConfig, language)
		}

		formattedImports[i] = formatted
	}

	return formattedImports
}

// formatFunctionWithStyle formats function context according to style preferences
func (cp *ContextProcessor) formatFunctionWithStyle(function string, style *types.StyleAnalysis, language string) string {
	if style == nil || function == "" {
		return function
	}

	formatted := function

	// Apply naming conventions for function names
	if style.Naming.Functions != "mixed" {
		formatted = cp.applyNamingConvention(formatted, style.Naming.Functions, language)
	}

	// Apply TypeScript-specific formatting
	if style.TypeScript.IsTypeScriptProject && style.TypeScript.UsesTypeAnnotations {
		formatted = cp.enhanceWithTypeAnnotations(formatted, language)
	}

	return formatted
}

// enhanceProjectTypeWithStyle enhances project type detection with style information
func (cp *ContextProcessor) enhanceProjectTypeWithStyle(projectType string, style *types.StyleAnalysis, language string) string {
	if style == nil {
		return projectType
	}

	enhanced := projectType

	// Add style-specific project information
	if style.TypeScript.IsTypeScriptProject {
		if style.TypeScript.HasStrictMode {
			enhanced += " (Strict TypeScript)"
		} else {
			enhanced += " (TypeScript)"
		}
	}

	if style.Linting.HasESLint && style.Linting.HasPrettier {
		enhanced += " with ESLint+Prettier"
	} else if style.Linting.HasESLint {
		enhanced += " with ESLint"
	} else if style.Linting.HasPrettier {
		enhanced += " with Prettier"
	}

	return enhanced
}

// convertSpacesToTabs converts space indentation to tab indentation
func (cp *ContextProcessor) convertSpacesToTabs(text string, spaceSize int) string {
	if spaceSize <= 0 {
		spaceSize = 4 // default
	}

	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if len(line) > 0 && line[0] == ' ' {
			// Count leading spaces
			spaces := 0
			for j, char := range line {
				if char == ' ' {
					spaces++
				} else {
					// Convert spaces to tabs
					tabs := spaces / spaceSize
					remainder := spaces % spaceSize
					lines[i] = strings.Repeat("\t", tabs) + strings.Repeat(" ", remainder) + line[j:]
					break
				}
			}
		}
	}

	return strings.Join(lines, "\n")
}

// normalizeSpaceIndentation ensures consistent space indentation
func (cp *ContextProcessor) normalizeSpaceIndentation(text string, spaceSize int) string {
	if spaceSize <= 0 {
		spaceSize = 4 // default
	}

	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if len(line) > 0 && (line[0] == ' ' || line[0] == '\t') {
			// Count indentation level
			indent := 0
			j := 0
			for j < len(line) {
				if line[j] == ' ' {
					indent++
				} else if line[j] == '\t' {
					indent += spaceSize // Convert tab to spaces
				} else {
					break
				}
				j++
			}

			// Normalize to space indentation
			if j > 0 {
				normalizedIndent := (indent / spaceSize) * spaceSize
				lines[i] = strings.Repeat(" ", normalizedIndent) + line[j:]
			}
		}
	}

	return strings.Join(lines, "\n")
}

// applyPrettierStyle applies Prettier formatting preferences
func (cp *ContextProcessor) applyPrettierStyle(text string, prettierConfig map[string]any, language string) string {
	if prettierConfig == nil {
		return text
	}

	formatted := text

	// Apply semicolon preference
	if semi, ok := prettierConfig["semi"].(bool); ok {
		if strings.Contains(language, "javascript") || strings.Contains(language, "typescript") {
			if semi {
				// Ensure semicolons are present
				if !strings.HasSuffix(strings.TrimSpace(formatted), ";") &&
					!strings.HasSuffix(strings.TrimSpace(formatted), "}") {
					formatted = strings.TrimSpace(formatted) + ";"
				}
			} else {
				// Remove unnecessary semicolons
				formatted = strings.TrimSuffix(strings.TrimSpace(formatted), ";")
			}
		}
	}

	// Apply quote preference
	if singleQuote, ok := prettierConfig["singleQuote"].(bool); ok {
		if singleQuote {
			// Convert double quotes to single quotes (simple implementation)
			formatted = strings.ReplaceAll(formatted, `"`, `'`)
		} else {
			// Convert single quotes to double quotes (simple implementation)
			formatted = strings.ReplaceAll(formatted, `'`, `"`)
		}
	}

	return formatted
}

// applyNamingConvention applies naming convention to identifiers in text
func (cp *ContextProcessor) applyNamingConvention(text string, convention string, language string) string {
	// This is a simplified implementation - in practice, you'd want more sophisticated parsing
	// For now, we'll just return the text as-is since proper identifier renaming requires AST parsing
	return text
}

// enhanceWithTypeAnnotations adds TypeScript type annotations where appropriate
func (cp *ContextProcessor) enhanceWithTypeAnnotations(text string, language string) string {
	if !strings.Contains(strings.ToLower(language), "typescript") {
		return text
	}

	// This is a simplified implementation - in practice, you'd want more sophisticated type inference
	// For now, we'll just return the text as-is since proper type annotation requires AST analysis
	return text
}

// extractImports extracts import statements from code
func (cp *ContextProcessor) extractImports(code, language string) []string {
	var imports []string
	var patterns []string

	switch strings.ToLower(language) {
	case "typescript", "javascript", "tsx", "jsx":
		patterns = []string{
			`import\s+.*?\s+from\s+['"][^'"]+['"]`,
			`import\s+['"][^'"]+['"]`,
			`const\s+.*?\s*=\s*require\(['"][^'"]+['"]\)`,
		}
	case "python":
		patterns = []string{
			`import\s+[\w\.]+`,
			`from\s+[\w\.]+\s+import\s+.*`,
		}
	case "go":
		patterns = []string{
			`import\s+['"][^'"]+['"]`,
			`import\s+\w+\s+['"][^'"]+['"]`,
		}
	case "java":
		patterns = []string{
			`import\s+[\w\.]+\*?;`,
		}
	case "csharp", "c#":
		patterns = []string{
			`using\s+[\w\.]+;`,
		}
	default:
		cp.logger.Warn("Unknown language for import extraction: %s", language)
		return imports
	}

	for _, pattern := range patterns {
		regex := regexp.MustCompile(pattern)
		matches := regex.FindAllString(code, -1)
		for _, match := range matches {
			// Clean up the import statement
			cleaned := strings.TrimSpace(match)
			if cleaned != "" {
				imports = append(imports, cleaned)
			}
		}
	}

	// Remove duplicates
	imports = cp.removeDuplicates(imports)

	// Limit to prevent context overflow
	maxImports := 20
	if len(imports) > maxImports {
		cp.logger.Warn("Truncating imports from %d to %d", len(imports), maxImports)
		imports = imports[:maxImports]
	}

	return imports
}

// extractCurrentFunction finds the function containing the cursor position
func (cp *ContextProcessor) extractCurrentFunction(code string, cursor int, language string) string {
	if cursor < 0 || cursor > len(code) {
		return ""
	}

	lines := strings.Split(code, "\n")
	currentLine := 0
	currentPos := 0

	// Find which line the cursor is on
	for i, line := range lines {
		if currentPos+len(line)+1 > cursor { // +1 for newline
			currentLine = i
			break
		}
		currentPos += len(line) + 1
	}

	// Look backwards from current line to find function declaration
	var patterns []string
	switch strings.ToLower(language) {
	case "typescript", "javascript", "tsx", "jsx":
		patterns = []string{
			`function\s+(\w+)\s*\([^)]*\)`,
			`(\w+)\s*:\s*\([^)]*\)\s*=>\s*{`,
			`(\w+)\s*=\s*\([^)]*\)\s*=>\s*{`,
			`(\w+)\s*\([^)]*\)\s*{`,
			`async\s+function\s+(\w+)\s*\([^)]*\)`,
		}
	case "python":
		patterns = []string{
			`def\s+(\w+)\s*\([^)]*\):`,
			`async\s+def\s+(\w+)\s*\([^)]*\):`,
		}
	case "go":
		patterns = []string{
			`func\s+(\w+)\s*\([^)]*\)`,
			`func\s+\(\w+\s+\*?\w+\)\s+(\w+)\s*\([^)]*\)`, // method
		}
	case "java":
		patterns = []string{
			`(?:public|private|protected)?\s*(?:static)?\s*\w+\s+(\w+)\s*\([^)]*\)`,
		}
	case "csharp", "c#":
		patterns = []string{
			`(?:public|private|protected|internal)?\s*(?:static)?\s*(?:async)?\s*\w+\s+(\w+)\s*\([^)]*\)`,
		}
	default:
		return ""
	}

	// Search backwards from current line
	for i := currentLine; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])

		for _, pattern := range patterns {
			regex := regexp.MustCompile(pattern)
			matches := regex.FindStringSubmatch(line)
			if len(matches) > 1 {
				funcName := matches[1]
				cp.logger.Info("Found current function: %s", funcName)
				return funcName
			}
		}

		// Stop searching if we hit a class/interface declaration or another function
		if cp.isBlockBoundary(line, language) {
			break
		}
	}

	return ""
}

// isBlockBoundary checks if a line represents a block boundary (class, interface, etc.)
func (cp *ContextProcessor) isBlockBoundary(line, language string) bool {
	line = strings.TrimSpace(line)

	switch strings.ToLower(language) {
	case "typescript", "javascript", "tsx", "jsx":
		patterns := []string{
			`^class\s+\w+`,
			`^interface\s+\w+`,
			`^export\s+class\s+\w+`,
			`^export\s+interface\s+\w+`,
		}
		for _, pattern := range patterns {
			if matched, _ := regexp.MatchString(pattern, line); matched {
				return true
			}
		}
	case "python":
		patterns := []string{
			`^class\s+\w+`,
		}
		for _, pattern := range patterns {
			if matched, _ := regexp.MatchString(pattern, line); matched {
				return true
			}
		}
	case "go":
		patterns := []string{
			`^type\s+\w+\s+struct`,
			`^type\s+\w+\s+interface`,
		}
		for _, pattern := range patterns {
			if matched, _ := regexp.MatchString(pattern, line); matched {
				return true
			}
		}
	case "java":
		patterns := []string{
			`^(?:public|private|protected)?\s*class\s+\w+`,
			`^(?:public|private|protected)?\s*interface\s+\w+`,
		}
		for _, pattern := range patterns {
			if matched, _ := regexp.MatchString(pattern, line); matched {
				return true
			}
		}
	}

	return false
}

// detectProjectType analyzes code to determine project type
func (cp *ContextProcessor) detectProjectType(code, language string) string {
	switch strings.ToLower(language) {
	case "typescript", "tsx":
		if strings.Contains(code, "import React") || strings.Contains(code, "from 'react'") {
			return "React"
		}
		if strings.Contains(code, "@angular") || strings.Contains(code, "import { Component }") {
			return "Angular"
		}
		if strings.Contains(code, "import Vue") || strings.Contains(code, "from 'vue'") {
			return "Vue"
		}
		if strings.Contains(code, "import express") || strings.Contains(code, "from 'express'") {
			return "Node.js/Express"
		}
		return "TypeScript"

	case "javascript", "jsx":
		if strings.Contains(code, "import React") || strings.Contains(code, "require('react')") {
			return "React"
		}
		if strings.Contains(code, "require('express')") || strings.Contains(code, "import express") {
			return "Node.js/Express"
		}
		return "JavaScript"

	case "python":
		if strings.Contains(code, "from django") || strings.Contains(code, "import django") {
			return "Django"
		}
		if strings.Contains(code, "from flask") || strings.Contains(code, "import flask") {
			return "Flask"
		}
		if strings.Contains(code, "import fastapi") || strings.Contains(code, "from fastapi") {
			return "FastAPI"
		}
		return "Python"

	case "go":
		if strings.Contains(code, "github.com/gin-gonic/gin") {
			return "Gin"
		}
		if strings.Contains(code, "net/http") {
			return "Go HTTP"
		}
		return "Go"

	case "java":
		if strings.Contains(code, "@SpringBootApplication") || strings.Contains(code, "org.springframework") {
			return "Spring Boot"
		}
		if strings.Contains(code, "javax.servlet") {
			return "Java Servlet"
		}
		return "Java"

	case "csharp", "c#":
		if strings.Contains(code, "using Microsoft.AspNetCore") {
			return "ASP.NET Core"
		}
		if strings.Contains(code, "using System.Web") {
			return "ASP.NET"
		}
		return "C#"

	default:
		return language
	}
}

// LimitContextSize ensures context doesn't exceed API token limits
func (cp *ContextProcessor) LimitContextSize(context types.CodeContext, maxTokens int) types.CodeContext {
	// Rough estimation: 1 token ≈ 4 characters
	maxChars := maxTokens * 4

	limited := context
	currentSize := cp.estimateContextSize(context)

	if currentSize <= maxChars {
		return limited
	}

	cp.logger.Warn("Context size %d exceeds limit %d, truncating", currentSize, maxChars)

	// Prioritize: CurrentFunction > ProjectType > Imports > RecentChanges

	// Truncate imports if needed
	if currentSize > maxChars && len(limited.Imports) > 0 {
		maxImports := 10
		if len(limited.Imports) > maxImports {
			limited.Imports = limited.Imports[:maxImports]
			currentSize = cp.estimateContextSize(limited)
		}
	}

	// Truncate recent changes if still too large
	if currentSize > maxChars && len(limited.RecentChanges) > 0 {
		maxChanges := 5
		if len(limited.RecentChanges) > maxChanges {
			limited.RecentChanges = limited.RecentChanges[:maxChanges]
			currentSize = cp.estimateContextSize(limited)
		}
	}

	// If still too large, truncate current function
	if currentSize > maxChars && len(limited.CurrentFunction) > 100 {
		limited.CurrentFunction = limited.CurrentFunction[:100] + "..."
	}

	return limited
}

// estimateContextSize estimates the size of context in characters
func (cp *ContextProcessor) estimateContextSize(context types.CodeContext) int {
	size := len(context.CurrentFunction) + len(context.ProjectType)

	for _, imp := range context.Imports {
		size += len(imp)
	}

	for _, change := range context.RecentChanges {
		size += len(change)
	}

	return size
}

// removeDuplicates removes duplicate strings from a slice
func (cp *ContextProcessor) removeDuplicates(items []string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, item := range items {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	return result
}

// ExtractSurroundingCode extracts code around the cursor position for context
func (cp *ContextProcessor) ExtractSurroundingCode(code string, cursor int, linesBefore, linesAfter int) string {
	if cursor < 0 || cursor > len(code) {
		return ""
	}

	lines := strings.Split(code, "\n")
	currentLine := 0
	currentPos := 0

	// Find which line the cursor is on
	for i, line := range lines {
		if currentPos+len(line)+1 > cursor { // +1 for newline
			currentLine = i
			break
		}
		currentPos += len(line) + 1
	}

	// Calculate range
	startLine := currentLine - linesBefore
	if startLine < 0 {
		startLine = 0
	}

	endLine := currentLine + linesAfter
	if endLine >= len(lines) {
		endLine = len(lines) - 1
	}

	// Extract surrounding lines
	surroundingLines := lines[startLine : endLine+1]
	return strings.Join(surroundingLines, "\n")
}

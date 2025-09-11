package types

// ErrorResponse represents a structured error response
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
	Retry   bool   `json:"retry"`
}

// AIConfig represents the AI service configuration
type AIConfig struct {
	Provider    string  `json:"provider"`
	APIKey      string  `json:"apiKey"`
	BaseURL     string  `json:"baseUrl,omitempty"`
	Model       string  `json:"model"`
	MaxTokens   int     `json:"maxTokens"`
	Temperature float64 `json:"temperature"`
}

// CompletionRequest represents a code completion request
type CompletionRequest struct {
	Code     string      `json:"code"`
	Cursor   int         `json:"cursor"`
	Language string      `json:"language"`
	Context  CodeContext `json:"context"`
}

// CompletionResponse represents a code completion response
type CompletionResponse struct {
	Suggestions []string `json:"suggestions"`
	Confidence  float64  `json:"confidence"`
	Error       string   `json:"error,omitempty"`
}

// CodeGenerationRequest represents a manual code generation request
type CodeGenerationRequest struct {
	Prompt   string      `json:"prompt"`
	Context  CodeContext `json:"context"`
	Language string      `json:"language"`
}

// CodeGenerationResponse represents a code generation response
type CodeGenerationResponse struct {
	Code  string `json:"code"`
	Error string `json:"error,omitempty"`
}

// CodeContext represents the code context for AI requests
type CodeContext struct {
	CurrentFunction string         `json:"currentFunction"`
	Imports         []string       `json:"imports"`
	ProjectType     string         `json:"projectType"`
	RecentChanges   []string       `json:"recentChanges"`
	StyleAnalysis   *StyleAnalysis `json:"styleAnalysis,omitempty"`
}

// StyleAnalysis represents code style preferences
type StyleAnalysis struct {
	Indentation IndentationStyle  `json:"indentation"`
	Naming      NamingConventions `json:"naming"`
	Linting     LintingConfig     `json:"linting"`
	TypeScript  TypeScriptInfo    `json:"typescript"`
}

// IndentationStyle represents indentation preferences
type IndentationStyle struct {
	Type       string  `json:"type"` // "spaces", "tabs", "mixed"
	Size       int     `json:"size"`
	Confidence float64 `json:"confidence"`
}

// NamingConventions represents naming convention preferences
type NamingConventions struct {
	Variables  string  `json:"variables"` // "camelCase", "snake_case", "PascalCase", "mixed"
	Functions  string  `json:"functions"`
	Classes    string  `json:"classes"`
	Constants  string  `json:"constants"`  // "UPPER_CASE", "camelCase", "PascalCase", "mixed"
	Interfaces string  `json:"interfaces"` // "PascalCase", "IPascalCase", "mixed"
	Types      string  `json:"types"`
	Confidence float64 `json:"confidence"`
}

// LintingConfig represents linting configuration
type LintingConfig struct {
	HasESLint      bool           `json:"hasESLint"`
	HasPrettier    bool           `json:"hasPrettier"`
	ESLintRules    map[string]any `json:"eslintRules,omitempty"`
	PrettierConfig map[string]any `json:"prettierConfig,omitempty"`
	ConfigFiles    []string       `json:"configFiles"`
}

// TypeScriptInfo represents TypeScript project information
type TypeScriptInfo struct {
	IsTypeScriptProject bool           `json:"isTypeScriptProject"`
	HasStrictMode       bool           `json:"hasStrictMode"`
	UsesTypeAnnotations bool           `json:"usesTypeAnnotations"`
	CompilerOptions     map[string]any `json:"compilerOptions,omitempty"`
	ConfigFile          string         `json:"configFile,omitempty"`
}

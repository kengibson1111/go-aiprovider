// Package openaiclient provides Azure OpenAI connectivity using the official OpenAI Go SDK v2
// with the azure sub-package for endpoint and authentication configuration.
//
// This file provides NewOpenAIAzureClient which creates an OpenAIClient configured for
// Azure OpenAI Service using Microsoft Entra ID (DefaultAzureCredential) for authentication.
// All completion methods (CallWithPrompt, CallWithMessages, CallWithTools, streaming, etc.)
// are inherited from the shared OpenAIClient struct in openai_client.go.
//
// # Authentication
//
// Azure Entra ID authentication is configured via environment variables following the
// OPENAI_AZURE_ naming convention. The client maps these to the standard AZURE_ variables
// expected by the azidentity.DefaultAzureCredential:
//
//   - OPENAI_AZURE_SP_TENANT_ID   → AZURE_TENANT_ID
//   - OPENAI_AZURE_SP_CLIENT_ID   → AZURE_CLIENT_ID
//   - OPENAI_AZURE_SP_CLIENT_SECRET → AZURE_CLIENT_SECRET
//
// # Required Configuration
//
//   - AIConfig.BaseURL: Azure OpenAI endpoint (e.g., https://your-resource.openai.azure.com)
//   - AIConfig.Model: Deployment name in Azure (e.g., gpt-4o-mini)
//   - OPENAI_AZURE_SP_TENANT_ID: Microsoft Entra tenant (directory) ID
//   - OPENAI_AZURE_SP_CLIENT_ID: App registration client (application) ID
//   - OPENAI_AZURE_SP_CLIENT_SECRET: App registration client secret
//
// # Required Environment Variables
//
//   - OPENAI_AZURE_API_VERSION: Azure OpenAI API version (e.g., 2024-12-01-preview)
package openaiclient

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/kengibson1111/go-aiprovider/internal/shared/logging"
	"github.com/kengibson1111/go-aiprovider/types"
	"github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/azure"
	"github.com/openai/openai-go/v2/option"
)

// requiredAzureEnvVars lists the OPENAI_AZURE_ environment variables that must be set
// for model access.
var requiredAzureEnvVars = map[string]string{
	"OPENAI_AZURE_ENDPOINT":    "OPENAI_AZURE_ENDPOINT",
	"OPENAI_AZURE_API_VERSION": "OPENAI_AZURE_API_VERSION",
	"OPENAI_AZURE_MODEL":       "OPENAI_AZURE_MODEL",
}

// requiredAzureIdentityEnvVars lists the OPENAI_AZURE_ environment variables that must be set
// for Entra ID authentication, along with the standard AZURE_ variables they map to.
var requiredAzureIdentityEnvVars = map[string]string{
	"OPENAI_AZURE_SP_TENANT_ID":     "AZURE_TENANT_ID",
	"OPENAI_AZURE_SP_CLIENT_ID":     "AZURE_CLIENT_ID",
	"OPENAI_AZURE_SP_CLIENT_SECRET": "AZURE_CLIENT_SECRET",
}

// setAzureEnvFromConfig validates that all required OPENAI_AZURE_ environment variables
// are set, then maps them to the standard AZURE_ variables that
// azidentity.DefaultAzureCredential expects. Only sets a standard variable if it is
// not already set, so explicit AZURE_ vars take precedence.
func setAzureEnvFromConfig() error {
	var missing []string
	for src := range requiredAzureEnvVars {
		if strings.TrimSpace(os.Getenv(src)) == "" {
			missing = append(missing, src)
		}
	}

	for src := range requiredAzureIdentityEnvVars {
		if strings.TrimSpace(os.Getenv(src)) == "" {
			missing = append(missing, src)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("required Azure environment variables not set: %s", strings.Join(missing, ", "))
	}

	for src, dst := range requiredAzureIdentityEnvVars {
		if os.Getenv(dst) == "" {
			os.Setenv(dst, os.Getenv(src))
		}
	}
	return nil
}

// NewOpenAIAzureClient creates an OpenAIClient configured for Azure OpenAI Service.
//
// It uses Microsoft Entra ID authentication via DefaultAzureCredential, which supports
// environment variables, managed identity, Azure CLI, and other credential sources.
// The OPENAI_AZURE_ environment variables are mapped to standard AZURE_ variables
// before credential creation.
//
// The returned *OpenAIClient is the same type used by NewOpenAIClient, so all existing
// methods (CallWithPrompt, CallWithMessages, CallWithTools, streaming, etc.) work
// identically against the Azure endpoint.
//
// Parameters:
//   - config: AIConfig with BaseURL set to the Azure endpoint and Model set to the deployment name
//
// Returns:
//   - *OpenAIClient: Configured client ready for Azure OpenAI API calls
//   - error: Configuration validation, credential, or SDK initialization error
func NewOpenAIAzureClient(config *types.AIConfig) (*OpenAIClient, error) {
	if config == nil {
		return nil, fmt.Errorf("configuration is required")
	}

	// Validate and map OPENAI_AZURE_ env vars to standard AZURE_ env vars for DefaultAzureCredential
	if err := setAzureEnvFromConfig(); err != nil {
		return nil, err
	}

	if strings.TrimSpace(config.BaseURL) == "" {
		// setAzureEnvFromConfig() validates OPENAI_AZURE_ENDPOINT
		config.BaseURL = strings.TrimSpace(os.Getenv("OPENAI_AZURE_ENDPOINT"))
	}

	// setAzureEnvFromConfig() validates OPENAI_AZURE_API_VERSION
	apiVersion := strings.TrimSpace(os.Getenv("OPENAI_AZURE_API_VERSION"))

	// Create Azure identity credential
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure credential: %w", err)
	}

	// Create optimized HTTP client (reuses the same function from openai_client.go)
	httpClient := createOptimizedHTTPClient()

	// Build SDK options with Azure endpoint and Entra ID token credential
	opts := []option.RequestOption{
		azure.WithEndpoint(config.BaseURL, apiVersion),
		azure.WithTokenCredential(cred),
		option.WithHTTPClient(httpClient),
		option.WithMaxRetries(3),
		option.WithRequestTimeout(25 * time.Second),
	}

	sdkClient := openai.NewClient(opts...)

	// Model defaults to gpt-4o-mini for Azure if not specified
	model := config.Model
	if model == "" {
		// setAzureEnvFromConfig() validates OPENAI_AZURE_MODEL
		model = strings.TrimSpace(os.Getenv("OPENAI_AZURE_MODEL"))
	}

	maxTokens := config.MaxTokens
	if maxTokens == 0 {
		maxTokens = 1000
	}

	temperature := config.Temperature
	if temperature == 0.0 {
		temperature = 0.7
	}

	logger := logging.NewDefaultLogger()

	client := &OpenAIClient{
		client:      &OpenAISDKClientWrapper{client: &sdkClient},
		httpClient:  httpClient,
		model:       model,
		maxTokens:   maxTokens,
		temperature: temperature,
		logger:      logger,
	}

	logger.Info("Azure OpenAI client created with model: %s, endpoint: %s, api-version: %s", model, config.BaseURL, apiVersion)

	return client, nil
}

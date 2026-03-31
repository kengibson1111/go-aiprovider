// Package openaiclient provides Azure OpenAI connectivity using the official OpenAI Go SDK v2
// with the azure sub-package for endpoint and authentication configuration.
//
// This file provides NewOpenAIAzureUPClient which creates an OpenAIClient configured for
// Azure OpenAI Service using UsernamePasswordCredential for authentication.
// All completion methods (CallWithPrompt, CallWithMessages, CallWithTools, streaming, etc.)
// are inherited from the shared OpenAIClient struct in openai_client.go.
//
// # Authentication
//
// UsernamePasswordCredential authentication is configured via environment variables following
// the OPENAI_AZURE_UP_ naming convention:
//
//   - OPENAI_AZURE_CLIENT_ID: App registration client (application) ID
//   - OPENAI_AZURE_UP_USERNAME:  Azure AD username (typically an email address)
//   - OPENAI_AZURE_UP_PASSWORD:  Azure AD password
//
// The tenant ID is shared with the service-principal configuration:
//
//   - OPENAI_AZURE_TENANT_ID: Microsoft Entra tenant (directory) ID
//
// # Required Configuration
//
//   - AIConfig.BaseURL: Azure OpenAI endpoint (e.g., https://your-resource.openai.azure.com)
//   - AIConfig.Model: Deployment name in Azure (e.g., gpt-4o-mini)
//   - OPENAI_AZURE_TENANT_ID: Microsoft Entra tenant (directory) ID
//   - OPENAI_AZURE_CLIENT_ID: App registration client (application) ID
//   - OPENAI_AZURE_UP_USERNAME: Azure AD username
//   - OPENAI_AZURE_UP_PASSWORD: Azure AD password
//
// # Required Environment Variables
//
//   - OPENAI_AZURE_API_VERSION: Azure OpenAI API version (e.g., 2024-12-01-preview)
//
// # Deprecation Notice
//
// UsernamePasswordCredential is deprecated in the azidentity package. It is provided here
// to support environments where service principal credentials are not available and
// username/password authentication is the only option.
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

// requiredAzureUPIdentityEnvVars lists the OPENAI_AZURE_UP_ environment variables that must
// be set for UsernamePasswordCredential authentication.
var requiredAzureUPIdentityEnvVars = []string{
	"OPENAI_AZURE_TENANT_ID",
	"OPENAI_AZURE_CLIENT_ID",
	"OPENAI_AZURE_UP_USERNAME",
	"OPENAI_AZURE_UP_PASSWORD",
}

// validateAzureUPEnv validates that all required environment variables are set for
// UsernamePasswordCredential authentication against Azure OpenAI. It checks both the
// shared Azure model-access variables (requiredAzureEnvVars) and the UP-specific
// identity variables.
func validateAzureUPEnv() error {
	var missing []string

	// Validate shared Azure model-access env vars (endpoint, api-version, model)
	for src := range requiredAzureEnvVars {
		if strings.TrimSpace(os.Getenv(src)) == "" {
			missing = append(missing, src)
		}
	}

	// Validate UP-specific identity env vars
	for _, src := range requiredAzureUPIdentityEnvVars {
		if strings.TrimSpace(os.Getenv(src)) == "" {
			missing = append(missing, src)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("required Azure environment variables not set: %s", strings.Join(missing, ", "))
	}

	return nil
}

// NewOpenAIAzureUPClient creates an OpenAIClient configured for Azure OpenAI Service
// using UsernamePasswordCredential for authentication.
//
// This constructor uses the deprecated azidentity.NewUsernamePasswordCredential to
// authenticate with Azure AD using a username and password. It is intended for
// environments where service principal credentials are not available.
//
// The returned *OpenAIClient is the same type used by NewOpenAIClient and
// NewOpenAIAzureClient, so all existing methods (CallWithPrompt, CallWithMessages,
// CallWithTools, streaming, etc.) work identically against the Azure endpoint.
//
// Parameters:
//   - config: AIConfig with BaseURL set to the Azure endpoint and Model set to the deployment name
//
// Returns:
//   - *OpenAIClient: Configured client ready for Azure OpenAI API calls
//   - error: Configuration validation, credential, or SDK initialization error
func NewOpenAIAzureUPClient(config *types.AIConfig) (*OpenAIClient, error) {
	if config == nil {
		return nil, fmt.Errorf("configuration is required")
	}

	// Validate all required environment variables
	if err := validateAzureUPEnv(); err != nil {
		return nil, err
	}

	if strings.TrimSpace(config.BaseURL) == "" {
		config.BaseURL = strings.TrimSpace(os.Getenv("OPENAI_AZURE_ENDPOINT"))
	}

	apiVersion := strings.TrimSpace(os.Getenv("OPENAI_AZURE_API_VERSION"))

	// Read UP-specific identity values
	tenantID := strings.TrimSpace(os.Getenv("OPENAI_AZURE_TENANT_ID"))
	clientID := strings.TrimSpace(os.Getenv("OPENAI_AZURE_CLIENT_ID"))
	username := strings.TrimSpace(os.Getenv("OPENAI_AZURE_UP_USERNAME"))
	password := strings.TrimSpace(os.Getenv("OPENAI_AZURE_UP_PASSWORD"))

	// Create UsernamePasswordCredential
	cred, err := azidentity.NewUsernamePasswordCredential(tenantID, clientID, username, password, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create UsernamePasswordCredential: %w", err)
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

	model := config.Model
	if model == "" {
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

	logger.Info("Azure OpenAI client (UsernamePassword) created with model: %s, endpoint: %s, api-version: %s", model, config.BaseURL, apiVersion)

	return client, nil
}

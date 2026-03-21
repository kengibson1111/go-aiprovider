# Azure OpenAI Service Setup Guide

This guide walks through setting up Azure OpenAI Service resources for use with the `openai-azure` provider in go-aiprovider. All commands use PowerShell and the Azure CLI.

## Prerequisites

- An Azure subscription (Pay-As-You-Go recommended for model deployment access)
- Azure CLI installed (`winget install Microsoft.AzureCLI`)

## Step 1: Authenticate

```powershell
az login
```

## Step 2: Create a Resource Group

Skip this if you already have a resource group.

```powershell
az group create --name my-rg --location eastus
```

Verify it exists:

```powershell
az group list --output table
```

## Step 3: Register the Cognitive Services Provider

This is a one-time step per subscription.

```powershell
az provider register --namespace Microsoft.CognitiveServices
```

Check registration status (wait for `Registered`):

```powershell
az provider show --namespace Microsoft.CognitiveServices --query "registrationState" --output tsv
```

## Step 4: Create the Azure OpenAI Resource

The `--custom-domain` flag is required for Entra ID (token-based) authentication. Without it, Azure assigns a regional endpoint that only supports API key auth.

```powershell
az cognitiveservices account create `
    --name my-openai-resource `
    --resource-group my-rg `
    --kind OpenAI `
    --sku S0 `
    --location eastus `
    --custom-domain my-openai-resource
```

This produces an endpoint like `https://my-openai-resource.openai.azure.com`.

## Step 5: Deploy a Model

The deployment name should match the model name for simplicity.

```powershell
az cognitiveservices account deployment create `
    --name my-openai-resource `
    --resource-group my-rg `
    --deployment-name gpt-4o-mini `
    --model-name gpt-4o-mini `
    --model-version "2024-07-18" `
    --model-format OpenAI `
    --sku-capacity 1 `
    --sku-name Standard
```

## Step 6: Create a Service Principal

This creates an app registration with a client ID and secret for Entra ID authentication.

```powershell
$resourceId = az cognitiveservices account show `
    --name my-openai-resource `
    --resource-group my-rg `
    --query id -o tsv

az ad sp create-for-rbac `
    --name go-aiprovider-dev `
    --role "Cognitive Services OpenAI User" `
    --scopes $resourceId
```

The output contains three values you need for your `.env` file:

| Output field | Environment variable           |
| ------------ | ------------------------------ |
| `appId`      | `OPENAI_AZURE_CLIENT_ID`       |
| `password`   | `OPENAI_AZURE_CLIENT_SECRET`   |
| `tenant`     | `OPENAI_AZURE_TENANT_ID`       |

## Step 7: Assign RBAC Role to the Service Principal

The `create-for-rbac` command above already scopes the role, but if you need to re-assign it (e.g., after recreating the resource):

```powershell
az role assignment create `
    --assignee <appId-from-step-6> `
    --role "Cognitive Services OpenAI User" `
    --scope $resourceId
```

Role assignments can take up to 5 minutes to propagate.

### Optional: Assign Role to Your User Account

This allows your personal Azure CLI login to access the resource directly (useful for ad-hoc testing outside the service principal).

```powershell
$userId = az ad signed-in-user show --query id -o tsv

az role assignment create `
    --assignee $userId `
    --role "Cognitive Services OpenAI User" `
    --scope $resourceId
```

## Step 8: Retrieve Configuration Values

Get the endpoint:

```powershell
az cognitiveservices account show `
    --name my-openai-resource `
    --resource-group my-rg `
    --query "properties.endpoint" --output tsv
```

Get the tenant ID (if you didn't save it from Step 6):

```powershell
az account show --query "tenantId" --output tsv
```

## Step 9: Configure Environment Variables

Add the following to your `.env` file (see `.env.sample` for the full template):

```env
OPENAI_AZURE_ENDPOINT=https://my-openai-resource.openai.azure.com
OPENAI_AZURE_API_VERSION=2024-12-01-preview
OPENAI_AZURE_MODEL=gpt-4o-mini
OPENAI_AZURE_TENANT_ID=<tenant from step 6>
OPENAI_AZURE_CLIENT_ID=<appId from step 6>
OPENAI_AZURE_CLIENT_SECRET=<password from step 6>
```

## Step 10: Run Integration Tests

```powershell
go test ./openaiclient/... -v -tags=integration -run TestOpenAIAzureIntegrationTestSuite -timeout 5m
```

## Cleanup / Recreating Resources

If you need to recreate a resource (e.g., to add a custom domain), delete and purge first. Azure uses soft-delete for Cognitive Services, so both steps are required before the name can be reused.

```powershell
az cognitiveservices account delete `
    --name my-openai-resource `
    --resource-group my-rg

az cognitiveservices account purge `
    --name my-openai-resource `
    --resource-group my-rg `
    --location eastus
```

After purging, the custom domain name may take a few minutes to become available again. If you get a `CustomDomainInUse` error, either wait or pick a different name.

After cleanup, repeat Steps 4, 5, and 7 to recreate the resource, redeploy the model, and re-assign the RBAC role to the existing service principal. The service principal from Step 6 survives resource deletion and does not need to be recreated. If the endpoint name changed, update `OPENAI_AZURE_ENDPOINT` in your `.env` file (Step 9).

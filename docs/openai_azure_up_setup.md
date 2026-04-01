# Azure OpenAI Service Setup Guide (UsernamePassword)

This guide walks through the additional Entra ID configuration needed to use the `openai-azure-up` provider in go-aiprovider. It assumes you have already completed the base setup in [openai_azure_setup.md](openai_azure_setup.md) (Steps 1–5) and have a working Azure OpenAI resource with a deployed model.

The `openai-azure-up` provider authenticates using `UsernamePasswordCredential` (the Resource Owner Password Credentials / ROPC flow). This is deprecated by Microsoft but useful in environments where interactive login or service principal secrets are not an option.

## Prerequisites

- Completed Steps 1–5 from [openai_azure_setup.md](openai_azure_setup.md)
- Azure CLI authenticated (`az login`)
- The app registration client ID from Step 6 of the base guide (or a new one created below)

## Step 1: Create an Entra ID User (or Use an Existing One)

If you already have a user account you want to use, skip to Step 2.

Create a new user in your Entra ID tenant:

```powershell
az ad user create `
    --display-name "Go AIProvider Dev" `
    --user-principal-name goaiprovider@yourtenant.onmicrosoft.com `
    --password "YourTemporaryPassword123!" `
    --force-change-password-next-sign-in false
```

Replace `yourtenant.onmicrosoft.com` with your actual tenant domain. You can find it with:

```powershell
az rest --method GET --url "https://graph.microsoft.com/v1.0/domains" `
    --query "value[?isDefault].id" -o tsv
```

Verify the user was created:

```powershell
az ad user show --id goaiprovider@yourtenant.onmicrosoft.com --query id -o tsv
```

## Step 2: Enable the ROPC Flow on the App Registration

The ROPC (Resource Owner Password Credentials) flow must be explicitly enabled on the app registration. This is the `Allow public client flows` setting.

```powershell
az ad app update `
    --id <your-app-registration-client-id> `
    --is-fallback-public-client true
```

You can verify the setting:

```powershell
az ad app show --id <your-app-registration-client-id> `
    --query "isFallbackPublicClient" -o tsv
```

This should return `true`.

## Step 3: Assign the RBAC Role to the User

The user needs the `Cognitive Services OpenAI User` role on the Azure OpenAI resource, just like the service principal does.

```powershell
$resourceId = az cognitiveservices account show `
    --name my-openai-resource `
    --resource-group my-rg `
    --query id -o tsv

$userId = az ad user show `
    --id goaiprovider@yourtenant.onmicrosoft.com `
    --query id -o tsv

az role assignment create `
    --assignee $userId `
    --role "Cognitive Services OpenAI User" `
    --scope $resourceId
```

Role assignments can take up to 5 minutes to propagate.

## Step 4: Verify There Are No Conditional Access Policies Blocking ROPC

Conditional Access policies (e.g., requiring MFA) will block the ROPC flow. If your tenant enforces MFA for all users, you may need to create an exclusion for this user or use a test tenant without MFA.

Check for policies that might apply:

```powershell
az rest --method GET `
    --url "https://graph.microsoft.com/v1.0/identity/conditionalAccess/policies" `
    --query "value[].{name:displayName, state:state}" -o table
```

If MFA is enforced tenant-wide via Security Defaults, you can disable it in the Azure Portal under **Entra ID > Properties > Manage security defaults** (not recommended for production tenants).

## Step 5: Grant Delegated API Permission (Cognitive Services)

The UsernamePassword (ROPC) flow is a delegated flow — the app acts on behalf of a user. Entra ID requires a consent record granting the app permission to call the Cognitive Services API as the user. Without this, authentication fails with `AADSTS65001: consent_required`.

This is different from the service principal flow used by `openai-azure`, which authenticates as the app itself and doesn't need delegated consent.

### Find the Cognitive Services service principal in your tenant

```powershell
az ad sp list `
    --filter "displayName eq 'Microsoft Cognitive Services'" `
    --query "[].{appId:appId, displayName:displayName}" -o table
```

Note the `AppId` value — you'll use it in the next commands.

### Look up the `user_impersonation` permission ID

```powershell
az ad sp show `
    --id <cognitive-services-app-id> `
    --query "oauth2PermissionScopes[?value=='user_impersonation'].{id:id, value:value}" -o json
```

Note the `id` (a GUID) — this is the permission ID.

### Declare the permission on the app registration

```powershell
az ad app permission add `
    --id <your-app-registration-client-id> `
    --api <cognitive-services-app-id> `
    --api-permissions <permission-id>=Scope
```

### Grant admin consent

This creates the OAuth2 consent record so Entra ID allows the app to acquire tokens on behalf of users:

```powershell
az ad app permission grant `
    --id <your-app-registration-client-id> `
    --api <cognitive-services-app-id> `
    --scope "user_impersonation"
```

### Verify in the portal

In **App registrations** → your app → **API permissions**, you should see `user_impersonation` under Microsoft Cognitive Services with Status showing "Granted for \<your tenant\>".

## Step 6: Configure Environment Variables

Add the following to your `.env` file. The endpoint, model, API version, tenant ID, and client ID are shared with the `openai-azure` provider:

`OPENAI_AZURE_TENANT_ID` accepts either a tenant GUID (e.g., `xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx`) or a tenant domain (e.g., `yourtenant.onmicrosoft.com`). The `azidentity` package resolves either form to the correct Entra ID authority URL (`https://login.microsoftonline.com/<tenant>`). This applies to both the `openai-azure` and `openai-azure-up` providers.

```env
# Shared Azure OpenAI variables (from base setup)
OPENAI_AZURE_ENDPOINT=https://my-openai-resource.openai.azure.com
OPENAI_AZURE_API_VERSION=2024-12-01-preview
OPENAI_AZURE_MODEL=gpt-4o-mini
OPENAI_AZURE_TENANT_ID=<tenant-id-or-domain>
OPENAI_AZURE_CLIENT_ID=<app-registration-client-id>

# UsernamePassword-specific variables
OPENAI_AZURE_UP_USERNAME=goaiprovider@yourtenant.onmicrosoft.com
OPENAI_AZURE_UP_PASSWORD=YourTemporaryPassword123!
```

## Step 7: Run Integration Tests

```powershell
go test ./openaiclient/... -v -tags=integration -run TestOpenAIAzureUPIntegrationTestSuite -timeout 5m
```

## Troubleshooting

### AADSTS50126: Invalid username or password

Double-check the username (must be the full UPN, e.g., `user@tenant.onmicrosoft.com`) and password. If the user was created with `--force-change-password-next-sign-in true`, sign in interactively once to set a permanent password before using ROPC.

### AADSTS65001: The user or administrator has not consented

The app registration is missing the delegated `user_impersonation` permission for Cognitive Services, or admin consent hasn't been granted. Complete Step 5 above. You can verify the consent record exists:

```powershell
az ad app permission list --id <your-app-registration-client-id> -o table
```

If the permission is listed but consent isn't granted, re-run the `az ad app permission grant` command from Step 5.

### AADSTS7000218: The request body must contain the following parameter: 'client_assertion' or 'client_secret'

This means the ROPC public client flow is not enabled. Re-run Step 2.

### AADSTS50076: MFA required

A Conditional Access policy or Security Defaults is enforcing MFA. See Step 4.

### Role assignment not taking effect

RBAC propagation can take up to 5 minutes. Verify the assignment:

```powershell
az role assignment list `
    --assignee $userId `
    --scope $resourceId `
    --output table
```

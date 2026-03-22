# Claude on Amazon Bedrock Setup Guide

This guide walks through setting up Amazon Bedrock for use with the `claude-bedrock` provider in go-aiprovider. All commands use PowerShell and the AWS CLI.

## Prerequisites

- An AWS account
- AWS CLI installed (`winget install Amazon.AWSCLI` or [installer](https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html))

## Step 1: Enable Model Access in Bedrock

1. Go to AWS Console → Amazon Bedrock → Model Catalog
2. Select the Claude model you want to use (e.g., Claude Sonnet 4)
3. Some models are available immediately; others require use-case approval
4. If a use-case approval popup appears, fill out the form and submit
5. Wait for the approval email before proceeding (typically minutes, sometimes up to a day)

## Step 2: Create an IAM User

For local development and integration tests, create an IAM user with programmatic access.

1. Go to IAM → Users → Create user
2. Create a custom policy with the following permissions and attach it to the user:

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "bedrock:InvokeModel",
                "bedrock:InvokeModelWithResponseStream"
            ],
            "Resource": [
                "arn:aws:bedrock:us-east-1::foundation-model/anthropic.*",
                "arn:aws:bedrock:us-east-2::foundation-model/anthropic.*",
                "arn:aws:bedrock:us-west-2::foundation-model/anthropic.*",
                "arn:aws:bedrock:us-east-1:<ACCOUNT_ID>:inference-profile/us.anthropic.*"
            ]
        },
        {
            "Effect": "Allow",
            "Action": [
                "aws-marketplace:ViewSubscriptions",
                "aws-marketplace:Subscribe"
            ],
            "Resource": "*"
        }
    ]
}
```

Replace `<ACCOUNT_ID>` with your AWS account ID.

### Policy Notes

- The `foundation-model` resources cover direct model invocation across regions
- The `inference-profile` resource is required because Bedrock now routes on-demand requests through inference profiles (model IDs prefixed with `us.`)
- The `aws-marketplace` actions are required for Anthropic models, which use a Marketplace subscription flow; these actions do not support resource-level restrictions, so `"Resource": "*"` is necessary

3. Create access keys for the user: IAM → Users → select user → Security credentials → Create access key

## Step 3: Configure Local Credentials

Run `aws configure` and enter the access key ID and secret from Step 2:

```powershell
aws configure
```

This writes credentials to `~/.aws/credentials`, which the AWS SDK default credential chain reads automatically. This also supports future SSO configuration via `aws configure sso`.

Verify your credentials:

```powershell
aws sts get-caller-identity
```

## Step 4: Configure Environment Variables

Add the following to your `.env` file (see `.env.sample` for the full template):

```env
CLAUDE_BEDROCK_REGION=us-east-1
CLAUDE_BEDROCK_MODEL=us.anthropic.claude-sonnet-4-20250514-v1:0
```

### Model ID Format

Bedrock requires an inference profile ID, not a raw model ID. System-defined inference profiles use a region prefix:

| Region group | Inference profile ID example                        |
| ------------ | --------------------------------------------------- |
| US           | `us.anthropic.claude-sonnet-4-20250514-v1:0`        |
| EU           | `eu.anthropic.claude-sonnet-4-20250514-v1:0`        |

### Optional: Custom Endpoint

If you need to override the default regional Bedrock endpoint (e.g., for a VPC endpoint), set:

```env
CLAUDE_BEDROCK_ENDPOINT=https://bedrock-runtime.us-east-1.amazonaws.com
```

If omitted, the AWS SDK derives the endpoint automatically from `CLAUDE_BEDROCK_REGION`.

## Step 5: Verify Model Access

Quick smoke test using the AWS CLI:

```powershell
aws bedrock-runtime invoke-model `
    --model-id "us.anthropic.claude-sonnet-4-20250514-v1:0" `
    --region us-east-1 `
    --content-type "application/json" `
    --accept "application/json" `
    --body '{"anthropic_version":"bedrock-2023-05-31","max_tokens":10,"messages":[{"role":"user","content":"Hi"}]}' `
    response.json; Get-Content response.json
```

If this returns a JSON response, your IAM user, Marketplace subscription, and model access are all working.

## Step 6: Run Integration Tests

```powershell
go test ./claudeclient -v -timeout 120s -tags=integration -run TestClaudeBedrockIntegrationTestSuite
```

## Authentication Flow

The `claude-bedrock` provider uses the AWS SDK v2 default credential chain, which checks (in order):

1. Environment variables (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_SESSION_TOKEN`)
2. Shared credentials file (`~/.aws/credentials`)
3. SSO / AWS Identity Center (via `~/.aws/config`)
4. IAM instance role (EC2, ECS, Lambda)

No API key is needed — AWS credentials handle all authentication and request signing (SigV4) automatically.

## Troubleshooting

| Error | Cause | Fix |
| ----- | ----- | --- |
| `ValidationException: Invocation of model ID ... with on-demand throughput isn't supported` | Using raw model ID instead of inference profile | Use `us.anthropic.claude-...` prefix in `CLAUDE_BEDROCK_MODEL` |
| `AccessDeniedException: ... not authorized to perform: bedrock:InvokeModel` | IAM policy missing inference profile resource | Add `inference-profile/us.anthropic.*` to policy Resource array |
| `AccessDeniedException: ... aws-marketplace:ViewSubscriptions` | Missing Marketplace permissions or pending use-case approval | Add Marketplace actions to policy; check approval status in Bedrock console |
| `No credential providers found` | AWS credentials not configured | Run `aws configure` or set `AWS_ACCESS_KEY_ID`/`AWS_SECRET_ACCESS_KEY` env vars |

# AWS AgentCore Deployment Guide

This guide covers deploying the Stats Agent Team to AWS using [Amazon Bedrock AgentCore](https://docs.aws.amazon.com/bedrock/latest/userguide/agentcore.html) with the [agentkit-aws-cdk](https://github.com/agentplexus/agentkit-aws-cdk) toolkit.

## Overview

AgentCore provides a managed runtime for containerized AI agents with built-in:

- Container orchestration and scaling
- VPC networking with private subnets
- API Gateway integration
- Secrets management
- CloudWatch logging

### Architecture

```
                         External Access
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    AWS Bedrock AgentCore                         │
│                                                                 │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │                      Gateway                              │  │
│  │              stats-agent-gateway                          │  │
│  │                 (GatewayUrl)                              │  │
│  └─────────────────────────┬─────────────────────────────────┘  │
│                            │                                    │
│                            ▼                                    │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │              Orchestration (isDefault: true)              │  │
│  │                     Entry Point                           │  │
│  │           ghcr.io/.../stats-agent-orchestration-eino      │  │
│  └────────────┬──────────────┬──────────────┬────────────────┘  │
│               │              │              │                   │
│               ▼              ▼              ▼                   │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐   │
│  │   Research   │  │  Synthesis   │  │    Verification      │   │
│  │   (internal) │  │  (internal)  │  │     (internal)       │   │
│  └──────────────┘  └──────────────┘  └──────────────────────┘   │
│                                                                 │
│                     Private VPC (10.0.0.0/16)                   │
└─────────────────────────────────────────────────────────────────┘
```

**Key points:**

- Only the **Orchestration agent** is externally accessible via the Gateway
- **Research, Synthesis, Verification** agents run internally in private subnets
- All inter-agent communication happens over private VPC networking

## Prerequisites

- AWS CLI configured with appropriate credentials
- Node.js 18+ and npm (for CDK)
- Go 1.23+ (for building the CDK app)
- AWS CDK CLI: `npm install -g aws-cdk`

## Container Images

Pre-built container images are published to GitHub Container Registry on each release:

| Agent | Image |
|-------|-------|
| Research | `ghcr.io/agentplexus/stats-agent-research:latest` |
| Synthesis | `ghcr.io/agentplexus/stats-agent-synthesis:latest` |
| Verification | `ghcr.io/agentplexus/stats-agent-verification:latest` |
| Orchestration | `ghcr.io/agentplexus/stats-agent-orchestration-eino:latest` |
| Direct | `ghcr.io/agentplexus/stats-agent-direct:latest` |

## Quick Start

### One-Command Deployment

Use the `deploy` tool from agentkit-aws-cdk for automated deployment:

```bash
# Install the deploy tool
go install github.com/agentplexus/agentkit-aws-cdk/cmd/deploy@latest

# Preview deployment (recommended first)
cd cdk
deploy --env ../.env --dry-run

# Full deployment
deploy --env ../.env --region us-east-1
```

This single command:

1. Pushes secrets from `.env` to AWS Secrets Manager
2. Bootstraps CDK (if needed)
3. Deploys the stack

### Manual Step-by-Step

If you prefer manual control:

#### 1. Create AWS Secrets

```bash
# Install push-secrets tool
go install github.com/agentplexus/agentkit-aws-cdk/cmd/push-secrets@latest

# Preview what will be created
push-secrets --dry-run .env

# Push to AWS Secrets Manager
push-secrets --region us-east-1 .env
```

This creates three secrets:

- `stats-agent/llm` - LLM provider API keys
- `stats-agent/search` - Search provider API keys
- `stats-agent/config` - Configuration and observability settings

#### 2. Bootstrap CDK

```bash
export AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
cdk bootstrap aws://${AWS_ACCOUNT_ID}/us-east-1
```

#### 3. Deploy

```bash
cd cdk
cp config.example.json config.json  # Create your config
go mod tidy
cdk deploy
```

### Get Gateway URL

After deployment, get the outputs:

```bash
aws cloudformation describe-stacks \
  --stack-name stats-agent-team \
  --query 'Stacks[0].Outputs' \
  --no-cli-pager
```

Output:
```
stats-agent-team.GatewayUrl = https://xxx.bedrock-agentcore.us-east-1.amazonaws.com/...
stats-agent-team.GatewayArn = arn:aws:bedrock:us-east-1:123456789:gateway/...
```

### Test the Deployment

```bash
curl -X POST https://<gateway-url>/orchestrate \
  -H "Content-Type: application/json" \
  -d '{
    "topic": "renewable energy adoption rates",
    "min_verified_stats": 3,
    "max_candidates": 10,
    "reputable_only": true
  }'
```

## Configuration Reference

For templates, see:

- **JSON**: [agentkit-aws-cdk/examples/2-cdk-json/config.json](https://github.com/agentplexus/agentkit-aws-cdk/blob/main/examples/2-cdk-json/config.json)
- **YAML**: [agentkit-aws-cdk/examples/2-cdk-json/config.yaml](https://github.com/agentplexus/agentkit-aws-cdk/blob/main/examples/2-cdk-json/config.yaml)

### config.json Structure

```json
{
  "stackName": "stats-agent-team",
  "description": "Statistics research and verification multi-agent system",
  "agents": [...],
  "gateway": {
    "enabled": true,
    "name": "stats-agent-gateway",
    "description": "External entry point"
  },
  "vpc": {
    "createVPC": true,
    "vpcCidr": "10.0.0.0/16",
    "maxAZs": 2,
    "enableVPCEndpoints": true
  },
  "observability": {
    "enableCloudWatchLogs": true,
    "logRetentionDays": 30
  },
  "iam": {
    "enableBedrockAccess": true
  },
  "tags": {...},
  "removalPolicy": "destroy"
}
```

### Agent Configuration

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Agent identifier |
| `containerImage` | string | Yes | GHCR or ECR image URL |
| `memoryMB` | int | No | Memory allocation (default: 512) |
| `timeoutSeconds` | int | No | Request timeout (default: 30) |
| `protocol` | string | No | HTTP, MCP, or A2A |
| `isDefault` | bool | No | Mark as gateway entry point |
| `environment` | object | No | Environment variables |
| `secretsARNs` | array | No | AWS Secrets Manager ARNs |

### Gateway Configuration

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `enabled` | bool | Yes | Enable external gateway |
| `name` | string | No | Gateway name |
| `description` | string | No | Gateway description |

## AWS Secrets

### Required Secrets

| Secret Path | Keys | Purpose |
|-------------|------|---------|
| `stats-agent/llm` | `GOOGLE_API_KEY` or `OPENAI_API_KEY` or `ANTHROPIC_API_KEY` | LLM provider authentication |
| `stats-agent/search` | `SERPER_API_KEY` or `SERPAPI_API_KEY` | Web search API |

### Optional Secrets

| Secret Path | Keys | Purpose |
|-------------|------|---------|
| `stats-agent/observability` | `OPIK_API_KEY` | LLM observability (Opik) |

### Creating Secrets

**For Gemini (recommended):**

```bash
aws secretsmanager create-secret \
  --name stats-agent/llm \
  --secret-string '{"GOOGLE_API_KEY":"your-key"}'
```

**For OpenAI:**

```bash
aws secretsmanager create-secret \
  --name stats-agent/llm \
  --secret-string '{"OPENAI_API_KEY":"your-key"}'
```

**For Claude:**

```bash
aws secretsmanager create-secret \
  --name stats-agent/llm \
  --secret-string '{"ANTHROPIC_API_KEY":"your-key"}'
```

## Environment Variables

### LLM Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `LLM_PROVIDER` | Provider: gemini, openai, claude | gemini |
| `LLM_MODEL` | Model override | Provider default |

### Search Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `SEARCH_PROVIDER` | Provider: serper, serpapi | serper |

### Agent URLs (Orchestration only)

| Variable | Description | Default |
|----------|-------------|---------|
| `RESEARCH_AGENT_URL` | Research agent endpoint | http://research:8001 |
| `SYNTHESIS_AGENT_URL` | Synthesis agent endpoint | http://synthesis:8004 |
| `VERIFICATION_AGENT_URL` | Verification agent endpoint | http://verification:8002 |

## Testing with MCP Server

You can use the local MCP server to test the AWS-deployed agents:

### Configure MCP for Remote Agents

In your Claude Code MCP configuration:

```json
{
  "mcpServers": {
    "stats-agent": {
      "command": "/path/to/stats-agent-team/bin/mcp-server",
      "env": {
        "RESEARCH_AGENT_URL": "https://<gateway-url>/research",
        "SYNTHESIS_AGENT_URL": "https://<gateway-url>/synthesis",
        "VERIFICATION_AGENT_URL": "https://<gateway-url>/verification",
        "GOOGLE_API_KEY": "your-key"
      }
    }
  }
}
```

!!! note
    The MCP server embeds the Eino orchestration logic locally. This means the orchestration runs on your machine while the Research, Synthesis, and Verification agents run on AWS.

### Alternative: Call AWS Orchestration Directly

For pure remote testing, call the AWS orchestration endpoint directly:

```bash
curl -X POST https://<gateway-url>/orchestrate \
  -H "Content-Type: application/json" \
  -d '{"topic": "AI adoption in healthcare", "min_verified_stats": 5}'
```

## Monitoring

### CloudWatch Logs

Logs are automatically sent to CloudWatch with 30-day retention:

```bash
# View logs for all agents
aws logs tail /aws/agentcore/stats-agent-team --follow

# View logs for specific agent
aws logs tail /aws/agentcore/stats-agent-team/orchestration --follow
```

### Stack Outputs

View deployment outputs:

```bash
aws cloudformation describe-stacks \
  --stack-name stats-agent-team \
  --query 'Stacks[0].Outputs'
```

## Updating the Deployment

### Update Configuration

1. Edit `cdk/config.json` (your local copy)
2. Run `cdk diff` to preview changes
3. Run `cdk deploy` to apply

### Update Container Images

Images are tagged with `latest` by default. To use a specific version:

```json
{
  "containerImage": "ghcr.io/agentplexus/stats-agent-orchestration-eino:v1.0.0"
}
```

Then redeploy:

```bash
cdk deploy
```

## Cleanup

To remove all deployed resources:

```bash
cd cdk
cdk destroy
```

!!! warning
    This will delete all AgentCore resources. Secrets in AWS Secrets Manager are retained unless manually deleted.

## Troubleshooting

### Deployment Fails

**Check CDK bootstrap:**

```bash
cdk bootstrap aws://${AWS_ACCOUNT_ID}/${AWS_REGION}
```

**Check IAM permissions:**
Ensure your AWS credentials have permissions for:

- Bedrock AgentCore
- VPC creation
- Secrets Manager read
- CloudWatch Logs

### Agent Not Responding

**Check agent logs:**

```bash
aws logs tail /aws/agentcore/stats-agent-team/orchestration --since 1h
```

**Verify secrets exist:**

```bash
aws secretsmanager describe-secret --secret-id stats-agent/llm
aws secretsmanager describe-secret --secret-id stats-agent/search
```

### Internal Agent Communication Fails

The orchestration agent calls internal agents via URLs like `http://research:8001`. If these fail:

1. Check VPC endpoints are enabled
2. Verify security groups allow internal traffic
3. Check agent logs for connection errors

## Cost Considerations

AgentCore pricing includes:

- **Runtime hours**: Based on memory allocation and execution time
- **Gateway requests**: Per-request pricing
- **Data transfer**: Standard AWS data transfer rates
- **CloudWatch Logs**: Log storage and ingestion

For cost optimization:

- Use appropriate memory allocations (512MB default is usually sufficient)
- Set reasonable timeouts to avoid runaway requests
- Consider reserved capacity for production workloads

## Next Steps

- [Kubernetes Deployment](kubernetes.md) - Alternative deployment on EKS
- [MCP Server Integration](mcp-server.md) - Claude Code integration
- [Security Guide](../operations/security.md) - Security best practices
- [Scaling Guide](../operations/scaling.md) - Production scaling

## References

- [agentkit-aws-cdk Repository](https://github.com/agentplexus/agentkit-aws-cdk)
- [Amazon Bedrock AgentCore Documentation](https://docs.aws.amazon.com/bedrock/latest/userguide/agentcore.html)
- [AWS CDK Documentation](https://docs.aws.amazon.com/cdk/latest/guide/)

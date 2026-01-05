# CDK Deployment

This directory contains the AWS CDK configuration for deploying stats-agent-team to AWS Bedrock AgentCore.

## Quick Start

```bash
# Install deploy tool
go install github.com/agentplexus/agentkit-aws-cdk/cmd/deploy@latest

# Create your config from the example
cp config.example.json config.json

# Edit config.json with your values (optional - example works as-is)
# vim config.json

# Preview deployment
deploy --dry-run

# Deploy
deploy --region us-east-1
```

## Configuration

### config.json

The `config.json` file defines your agent stack. For your own project, copy and customize:

```bash
# Copy to your project
cp config.json ~/myproject/cdk/config.json
```

**Required changes for your project:**

| Field | Description | Example |
|-------|-------------|---------|
| `stackName` | Unique stack identifier | `my-agent-team` |
| `description` | Stack description | `My custom agent system` |
| `agents[].name` | Agent identifiers | `researcher`, `analyzer` |
| `agents[].containerImage` | Your container images | `ghcr.io/myorg/my-agent:latest` |
| `gateway.name` | Gateway identifier | `my-agent-gateway` |

**Optional customizations:**

| Field | Default | Description |
|-------|---------|-------------|
| `agents[].memoryMB` | 512 | Memory allocation per agent |
| `agents[].timeoutSeconds` | 30 | Request timeout |
| `agents[].isDefault` | false | Mark as gateway entry point (one required) |
| `vpc.vpcCidr` | 10.0.0.0/16 | VPC CIDR block |
| `vpc.maxAZs` | 2 | Number of availability zones |
| `observability.logRetentionDays` | 30 | CloudWatch log retention |

### Credentials in ~/.agentplexus

Store credentials separately from your project in `~/.agentplexus/`:

```bash
# Create project directory for credentials
mkdir -p ~/.agentplexus/projects/my-agent-team

# Add credentials (these are pushed to AWS Secrets Manager)
cat > ~/.agentplexus/projects/my-agent-team/.env << 'EOF'
GOOGLE_API_KEY=your-key
SERPER_API_KEY=your-key
LLM_PROVIDER=gemini
EOF

chmod 600 ~/.agentplexus/projects/my-agent-team/.env
```

The deploy tool auto-detects your project from `config.json` stackName and loads credentials from the matching `~/.agentplexus/projects/{stackName}/.env`.

**Directory structure:**

```
~/.agentplexus/
├── .env                              # Global fallback credentials
└── projects/
    └── stats-agent-team/
        └── .env                      # Project-specific credentials

~/myproject/
└── cdk/
    ├── config.json                   # CDK configuration (stays here)
    ├── main.go
    ├── go.mod
    └── cdk.json
```

**Note:** The `config.json` must remain in your CDK directory because CDK runs from there. Only credentials (`.env`) should be stored in `~/.agentplexus/`.

## Files

| File | Purpose |
|------|---------|
| `config.example.json` | Example configuration (copy to `config.json`) |
| `config.json` | Your configuration (git-ignored) |
| `main.go` | CDK app entry point |
| `go.mod` | Go module dependencies |
| `cdk.json` | CDK CLI configuration |

## Example config.json

```json
{
  "stackName": "my-agent-team",
  "description": "My custom agent system",
  "agents": [
    {
      "name": "worker",
      "description": "Worker agent",
      "containerImage": "ghcr.io/myorg/my-worker:latest",
      "memoryMB": 512,
      "timeoutSeconds": 30,
      "protocol": "HTTP",
      "environment": {
        "LOG_LEVEL": "info"
      }
    },
    {
      "name": "orchestrator",
      "description": "Entry point orchestrator",
      "containerImage": "ghcr.io/myorg/my-orchestrator:latest",
      "memoryMB": 512,
      "timeoutSeconds": 300,
      "protocol": "HTTP",
      "isDefault": true,
      "environment": {
        "WORKER_URL": "http://worker:8001"
      }
    }
  ],
  "gateway": {
    "enabled": true,
    "name": "my-agent-gateway",
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
  "tags": {
    "Project": "my-agent-team",
    "Environment": "production"
  },
  "removalPolicy": "destroy"
}
```

## Agent Configuration Reference

Each agent in the `agents` array supports:

```json
{
  "name": "string",           // Required: unique identifier
  "description": "string",    // Optional: agent description
  "containerImage": "string", // Required: container image URL
  "memoryMB": 512,            // Optional: memory in MB (default: 512)
  "timeoutSeconds": 30,       // Optional: timeout (default: 30)
  "protocol": "HTTP",         // Optional: HTTP, MCP, or A2A
  "isDefault": false,         // Optional: gateway entry point
  "environment": {},          // Optional: environment variables
  "secretsARNs": []           // Optional: AWS Secrets Manager ARNs
}
```

## Secrets

Secrets are automatically pushed to AWS Secrets Manager by the deploy tool:

| Secret | Keys |
|--------|------|
| `{prefix}/llm` | `GOOGLE_API_KEY`, `ANTHROPIC_API_KEY`, `OPENAI_API_KEY` |
| `{prefix}/search` | `SERPER_API_KEY`, `SERPAPI_API_KEY` |
| `{prefix}/config` | `LLM_PROVIDER`, `LLM_MODEL`, observability settings |

Reference them in your agent config:

```json
{
  "secretsARNs": [
    "arn:aws:secretsmanager:us-east-1:123456789:secret:stats-agent/llm",
    "arn:aws:secretsmanager:us-east-1:123456789:secret:stats-agent/search"
  ]
}
```

## Related Documentation

- [AWS AgentCore Deployment Guide](../docsrc/guides/aws-agentcore.md)
- [agentkit-aws-cdk Repository](https://github.com/agentplexus/agentkit-aws-cdk)
- [Credentials Configuration](https://agentplexus.github.io/agentkit/configuration/credentials/)

# Kubernetes Deployment Guide

This guide covers deploying the Stats Agent Team to Kubernetes using Helm, with support for local development (Minikube) and production (AWS EKS).

## Architecture

In Kubernetes, each agent runs as a separate deployment with its own pod(s):

```
┌────────────────────────────────────────────────────────────────┐
│                        Kubernetes Cluster                      │
│                                                                │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐  │
│  │   Research   │  │  Synthesis   │  │    Verification      │  │
│  │    Agent     │  │    Agent     │  │       Agent          │  │
│  │   :8001      │  │    :8004     │  │       :8002          │  │
│  └──────────────┘  └──────────────┘  └──────────────────────┘  │
│          │                │                     │              │
│          └────────────────┼─────────────────────┘              │
│                           │                                    │
│                    ┌──────┴──────┐                             │
│                    │ Orchestrator│                             │
│                    │   (Eino)    │                             │
│                    │    :8000    │                             │
│                    └─────────────┘                             │
│                           │                                    │
│                    ┌──────┴──────┐                             │
│                    │   Ingress   │                             │
│                    └─────────────┘                             │
└────────────────────────────────────────────────────────────────┘
```

## Prerequisites

- Docker
- kubectl
- Helm 3.x
- For Minikube: minikube
- For EKS: AWS CLI, eksctl (optional)

## Container Images

Pre-built container images are published to GitHub Container Registry (ghcr.io) on each release:

```bash
# Available images
docker pull ghcr.io/agentplexus/stats-agent-research:latest
docker pull ghcr.io/agentplexus/stats-agent-synthesis:latest
docker pull ghcr.io/agentplexus/stats-agent-verification:latest
docker pull ghcr.io/agentplexus/stats-agent-orchestration-eino:latest
docker pull ghcr.io/agentplexus/stats-agent-direct:latest

# Or use a specific version
docker pull ghcr.io/agentplexus/stats-agent-research:v1.0.0
```

## Quick Start

### Using Published Images (Recommended)

```bash
# Deploy directly using published images
helm upgrade --install stats-agent ./helm/stats-agent-team \
  --namespace stats-agent \
  --create-namespace \
  --set secrets.geminiApiKey=YOUR_GEMINI_KEY \
  --set secrets.serperApiKey=YOUR_SERPER_KEY

# Access the orchestration agent
kubectl port-forward -n stats-agent svc/stats-agent-stats-agent-team-orchestration 8000:8000

# Test the deployment
curl http://localhost:8000/health
```

### Minikube (Local Development with Local Builds)

```bash
# 1. Setup Minikube
make k8s-minikube-setup

# 2. Build images in Minikube's Docker daemon
make k8s-minikube-build

# 3. Deploy with Helm (uses local images)
make k8s-minikube-deploy \
  --set secrets.geminiApiKey=YOUR_GEMINI_KEY \
  --set secrets.serperApiKey=YOUR_SERPER_KEY

# 4. Access the orchestration agent
kubectl port-forward -n stats-agent svc/stats-agent-stats-agent-team-orchestration 8000:8000

# 5. Test the deployment
curl http://localhost:8000/health
```

### AWS EKS (Production)

```bash
# Option 1: Use published images from ghcr.io (recommended)
helm upgrade --install stats-agent ./helm/stats-agent-team \
  -f ./helm/stats-agent-team/values-eks.yaml \
  --namespace stats-agent \
  --create-namespace \
  --set ingress.host=stats-agent.example.com

# Option 2: Use your own ECR registry
# 1. Build images locally
make k8s-build-images

# 2. Push to ECR
make k8s-eks-push REGISTRY=123456789012.dkr.ecr.us-west-2.amazonaws.com

# 3. Deploy to EKS
make k8s-eks-deploy REGISTRY=123456789012.dkr.ecr.us-west-2.amazonaws.com

# 4. Get the ALB address
kubectl get ingress -n stats-agent
```

## Detailed Setup

### Building Container Images

Each agent is built as a separate container using `Dockerfile.agent`:

```bash
# Build all agents
make k8s-build-images

# Or build individually
docker build --build-arg AGENT=research -t stats-agent-research:latest -f Dockerfile.agent .
docker build --build-arg AGENT=synthesis -t stats-agent-synthesis:latest -f Dockerfile.agent .
docker build --build-arg AGENT=verification -t stats-agent-verification:latest -f Dockerfile.agent .
docker build --build-arg AGENT=orchestration-eino -t stats-agent-orchestration-eino:latest -f Dockerfile.agent .
docker build --build-arg AGENT=direct -t stats-agent-direct:latest -f Dockerfile.agent .
```

### Helm Chart Structure

```
helm/stats-agent-team/
├── Chart.yaml              # Chart metadata
├── values.yaml             # Default values
├── values-minikube.yaml    # Minikube-specific values
├── values-eks.yaml         # EKS production values
└── templates/
    ├── _helpers.tpl        # Template helpers
    ├── namespace.yaml      # Namespace
    ├── configmap.yaml      # Configuration
    ├── secret.yaml         # API keys
    ├── serviceaccount.yaml # Service account
    ├── *-deployment.yaml   # Agent deployments
    ├── *-service.yaml      # Agent services
    └── ingress.yaml        # Ingress (EKS)
```

### Configuration

#### API Keys

**Option 1: Helm values (development only)**
```bash
helm upgrade --install stats-agent ./helm/stats-agent-team \
  --set secrets.geminiApiKey=YOUR_KEY \
  --set secrets.serperApiKey=YOUR_KEY
```

**Option 2: External secrets (recommended for production)**

Disable secret creation in values:
```yaml
secrets:
  create: false
```

Then create secrets manually or use External Secrets Operator with AWS Secrets Manager.

#### LLM Provider

Configure in values.yaml or via `--set`:

```yaml
llm:
  provider: gemini  # gemini, claude, openai, ollama
  geminiModel: "gemini-2.0-flash-exp"
```

#### Resource Limits

Adjust per-agent resources in values:

```yaml
synthesis:
  resources:
    requests:
      cpu: 500m
      memory: 512Mi
    limits:
      cpu: 2000m
      memory: 1Gi
```

## Minikube Deployment

### Setup

```bash
# Start Minikube with adequate resources
make k8s-minikube-setup

# This runs:
# minikube start --cpus=4 --memory=8192
# minikube addons enable ingress
# minikube addons enable metrics-server
```

### Build and Deploy

```bash
# Build images in Minikube's Docker daemon
make k8s-minikube-build

# Deploy
make k8s-minikube-deploy
```

### Accessing Services

```bash
# Port forward to orchestration agent
kubectl port-forward -n stats-agent svc/stats-agent-stats-agent-team-orchestration 8000:8000

# Or use minikube service
minikube service stats-agent-stats-agent-team-orchestration -n stats-agent

# View all pods
kubectl get pods -n stats-agent

# View logs
kubectl logs -n stats-agent -l app.kubernetes.io/component=orchestration -f
```

### Cleanup

```bash
make k8s-minikube-delete
minikube stop  # or minikube delete
```

## AWS EKS Deployment

### Prerequisites

1. EKS cluster with AWS Load Balancer Controller installed
2. ECR repositories for each agent image
3. IAM role for IRSA (if using AWS Secrets Manager)

### Create ECR Repositories

```bash
REGION=us-west-2
ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)

for agent in research synthesis verification orchestration-eino direct; do
  aws ecr create-repository --repository-name stats-agent-$agent --region $REGION
done
```

### Build and Push Images

```bash
REGISTRY=$ACCOUNT_ID.dkr.ecr.$REGION.amazonaws.com

# Login to ECR
aws ecr get-login-password --region $REGION | docker login --username AWS --password-stdin $REGISTRY

# Build and push
make k8s-build-images
make k8s-eks-push REGISTRY=$REGISTRY
```

### Deploy

```bash
# Basic deployment
make k8s-eks-deploy REGISTRY=$REGISTRY

# With custom domain
helm upgrade --install stats-agent ./helm/stats-agent-team \
  -f ./helm/stats-agent-team/values-eks.yaml \
  --namespace stats-agent \
  --create-namespace \
  --set global.image.registry=$REGISTRY \
  --set ingress.host=stats-agent.example.com
```

### Secrets Management (AWS Secrets Manager)

1. Install External Secrets Operator:
```bash
helm repo add external-secrets https://charts.external-secrets.io
helm install external-secrets external-secrets/external-secrets -n external-secrets --create-namespace
```

2. Create a ClusterSecretStore:
```yaml
apiVersion: external-secrets.io/v1beta1
kind: ClusterSecretStore
metadata:
  name: aws-secrets-manager
spec:
  provider:
    aws:
      service: SecretsManager
      region: us-west-2
      auth:
        jwt:
          serviceAccountRef:
            name: external-secrets
            namespace: external-secrets
```

3. Create an ExternalSecret:
```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: stats-agent-secrets
  namespace: stats-agent
spec:
  refreshInterval: 1h
  secretStoreRef:
    kind: ClusterSecretStore
    name: aws-secrets-manager
  target:
    name: stats-agent-stats-agent-team-secrets
  data:
    - secretKey: GEMINI_API_KEY
      remoteRef:
        key: stats-agent/api-keys
        property: gemini
```

### HTTPS with ACM

```bash
helm upgrade --install stats-agent ./helm/stats-agent-team \
  -f ./helm/stats-agent-team/values-eks.yaml \
  --set ingress.annotations."alb\.ingress\.kubernetes\.io/certificate-arn"=arn:aws:acm:REGION:ACCOUNT:certificate/CERT_ID \
  --set ingress.annotations."alb\.ingress\.kubernetes\.io/listen-ports"='[{"HTTPS":443}]'
```

### Cleanup

```bash
make k8s-eks-delete
```

## Monitoring and Debugging

### View Logs

```bash
# All agents
kubectl logs -n stats-agent -l app.kubernetes.io/instance=stats-agent -f

# Specific agent
kubectl logs -n stats-agent -l app.kubernetes.io/component=synthesis -f
```

### Check Health

```bash
# Get pod status
kubectl get pods -n stats-agent

# Describe a pod
kubectl describe pod -n stats-agent -l app.kubernetes.io/component=orchestration

# Check endpoints
kubectl get endpoints -n stats-agent
```

### Scaling

```bash
# Scale an agent
kubectl scale deployment -n stats-agent stats-agent-stats-agent-team-synthesis --replicas=3

# Or via Helm
helm upgrade stats-agent ./helm/stats-agent-team --set synthesis.replicaCount=3
```

## Helm Commands Reference

```bash
# Lint chart
make helm-lint

# Render templates locally
make helm-template

# View release status
helm status stats-agent -n stats-agent

# View release history
helm history stats-agent -n stats-agent

# Rollback
helm rollback stats-agent 1 -n stats-agent
```

## Troubleshooting

### Pods not starting

```bash
# Check events
kubectl get events -n stats-agent --sort-by='.lastTimestamp'

# Check pod logs
kubectl logs -n stats-agent <pod-name> --previous
```

### Service discovery issues

Agents communicate via Kubernetes DNS. Verify:
```bash
# From any pod
kubectl exec -it -n stats-agent <pod-name> -- wget -qO- http://stats-agent-stats-agent-team-research:8001/health
```

### Image pull errors

For Minikube:
```bash
# Ensure images are built in Minikube's Docker
eval $(minikube docker-env)
docker images | grep stats-agent
```

For EKS:
```bash
# Verify ECR login
aws ecr get-login-password | docker login --username AWS --password-stdin $REGISTRY

# Check image pull secrets
kubectl get secret ecr-creds -n stats-agent -o yaml
```

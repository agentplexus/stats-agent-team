# Helm Chart Testing Guide

This guide covers testing the Stats Agent Team Helm chart for correctness, security, and Kubernetes compatibility.

## Testing Layers

The chart testing strategy follows industry best practices with multiple validation layers:

| Layer | Tool | Purpose | CI Stage |
|-------|------|---------|----------|
| **Secrets** | `gitleaks` | Hardcoded secret detection | Always |
| **Secrets** | `trivy` (secret scan) | Secret pattern detection | Always |
| **Config** | `trivy` (config scan) | IaC misconfiguration | Always |
| **Vulnerabilities** | `trivy` (image scan) | Container CVE detection | Always |
| **Syntax** | `helm lint` | YAML/template validation | Always |
| **Unit Tests** | `helm-unittest` | Template logic verification | Always |
| **Schema** | `kubeconform` | K8s API compatibility | Always |
| **Best Practices** | `Polaris` | K8s security compliance | Always |
| **Go Validation** | `go test` | Values struct validation | Always |
| **Integration** | `Kind` + `helm install` | Real cluster deployment | On merge |

## Quick Start

```bash
# Run all chart tests locally
make helm-test-all

# Quick test (lint + unittest only)
make helm-test

# Individual tests
make helm-lint
make helm-unittest
make helm-kubeconform
make helm-polaris

# Security scanning (requires gitleaks and trivy installed)
gitleaks detect --config .gitleaks.toml --source . -v
trivy config ./helm/stats-agent-team
trivy fs --scanners secret .
```

## Tool Installation

### Required
```bash
# Helm (required)
brew install helm

# helm-unittest plugin (auto-installed by Makefile)
helm plugin install https://github.com/helm-unittest/helm-unittest.git
```

### Optional (for full test suite)
```bash
# kubeconform - K8s schema validation
brew install kubeconform

# Polaris - security best practices
brew install FairwindsOps/tap/polaris

# chart-testing - official Helm testing tool
brew install chart-testing
```

## Test Types

### 1. Security Scanning

#### Gitleaks (Secret Detection)

Scans git history and current files for hardcoded secrets:

```bash
# Install gitleaks
brew install gitleaks

# Run locally
gitleaks detect --config .gitleaks.toml --source . -v

# Scan git history
gitleaks detect --config .gitleaks.toml --source . --log-opts="--all"
```

Configuration: `.gitleaks.toml`

#### Trivy (Multi-Scanner)

Comprehensive security scanner for configs, secrets, and container vulnerabilities:

```bash
# Install trivy
brew install trivy

# Scan Helm chart for misconfigurations
trivy config ./helm/stats-agent-team

# Scan for secrets in filesystem
trivy fs --scanners secret .

# Scan container image for vulnerabilities
docker build --build-arg AGENT=research -t stats-agent-research:scan -f Dockerfile.agent .
trivy image stats-agent-research:scan

# Generate SBOM (Software Bill of Materials)
trivy image --format cyclonedx -o sbom.json stats-agent-research:scan
```

Configuration: `.trivyignore` (for ignoring specific CVEs or rules)

### 3. Helm Lint

Basic syntax and structure validation:

```bash
make helm-lint

# Or directly:
helm lint ./helm/stats-agent-team
helm lint ./helm/stats-agent-team -f ./helm/stats-agent-team/values-minikube.yaml
helm lint ./helm/stats-agent-team -f ./helm/stats-agent-team/values-eks.yaml
```

### 4. Unit Tests (helm-unittest)

Unit tests verify template rendering logic. Tests are located in `helm/stats-agent-team/tests/`.

```bash
make helm-unittest

# Or directly:
helm unittest ./helm/stats-agent-team
```

#### Writing Unit Tests

Test files use YAML format with BDD-style assertions:

```yaml
# tests/deployment_test.yaml
suite: deployment tests
templates:
  - research-deployment.yaml

tests:
  - it: should create deployment when enabled
    set:
      research.enabled: true
    asserts:
      - isKind:
          of: Deployment
      - equal:
          path: spec.replicas
          value: 1

  - it: should not create deployment when disabled
    set:
      research.enabled: false
    asserts:
      - hasDocuments:
          count: 0
```

#### Available Assertions

| Assertion | Description |
|-----------|-------------|
| `equal` | Exact value match |
| `notEqual` | Value does not match |
| `matchRegex` | Regex pattern match |
| `contains` | Array contains value |
| `isKind` | Resource kind check |
| `isAPIVersion` | API version check |
| `hasDocuments` | Document count |
| `exists` | Path exists |
| `notExists` | Path does not exist |
| `isNull` | Value is null |
| `isNotNull` | Value is not null |
| `isEmpty` | Value is empty |
| `isNotEmpty` | Value is not empty |
| `lengthEqual` | Array/string length |

### 5. Schema Validation (kubeconform)

Validates rendered templates against Kubernetes API schemas:

```bash
make helm-kubeconform

# Or directly:
helm template stats-agent ./helm/stats-agent-team | \
  kubeconform -strict -summary -kubernetes-version 1.29.0

# With specific values:
helm template stats-agent ./helm/stats-agent-team \
  --set research.autoscaling.enabled=true \
  --set research.pdb.enabled=true | \
  kubeconform -strict -summary -kubernetes-version 1.29.0
```

#### Testing Multiple K8s Versions

```bash
for version in 1.27.0 1.28.0 1.29.0; do
  echo "Testing Kubernetes $version"
  helm template stats-agent ./helm/stats-agent-team | \
    kubeconform -strict -kubernetes-version $version
done
```

### 6. Security Best Practices (Polaris)

Checks for security issues and Kubernetes best practices:

```bash
make helm-polaris

# Or directly:
helm template stats-agent ./helm/stats-agent-team | \
  polaris audit --audit-path - --format pretty

# With score threshold (fails if below 70):
helm template stats-agent ./helm/stats-agent-team | \
  polaris audit --audit-path - --set-exit-code-below-score 70
```

#### Common Polaris Checks

- Resource requests/limits configured
- Security context set (non-root, read-only filesystem)
- Liveness/readiness probes defined
- No privileged containers
- No host network/PID access

### 7. Go Struct Validation

Validates values.yaml against Go struct definitions:

```bash
go test -v ./pkg/helm/...
```

This catches:
- Invalid LLM/search provider values
- Port conflicts between agents
- Missing required fields
- Invalid resource quantity formats

### 8. Integration Tests

Deploy to a real cluster (Kind) and verify functionality:

```bash
# Create Kind cluster
kind create cluster --name chart-testing

# Build and load images
make k8s-build-images
kind load docker-image stats-agent-research:latest --name chart-testing
kind load docker-image stats-agent-synthesis:latest --name chart-testing
kind load docker-image stats-agent-verification:latest --name chart-testing
kind load docker-image stats-agent-orchestration-eino:latest --name chart-testing

# Install chart
helm install stats-agent ./helm/stats-agent-team \
  --namespace stats-agent \
  --create-namespace \
  --set global.image.pullPolicy=Never \
  --wait --timeout 5m

# Verify pods are running
kubectl get pods -n stats-agent

# Cleanup
kind delete cluster --name chart-testing
```

## CI/CD Integration

### GitHub Actions Workflow

The `.github/workflows/helm.yaml` workflow runs automatically on:
- Push to `main` affecting `helm/**`
- Pull requests affecting `helm/**`

#### Workflow Jobs

**Security Scanning:**
1. **gitleaks** - Hardcoded secret detection in code and git history
2. **trivy-config** - Helm chart misconfiguration and secret pattern detection
3. **trivy-image** - Container vulnerability scanning with SBOM generation

**Chart Validation:**
4. **lint** - `helm lint` with all values files
5. **unittest** - `helm-unittest` plugin tests
6. **kubeconform** - Schema validation against K8s 1.29
7. **polaris** - Security best practices audit
8. **go-validation** - Go struct validation tests
9. **integration** - Kind cluster deployment test (requires security scans to pass)
10. **chart-testing** - Official `ct lint` tool

### Local Pre-commit Testing

Before pushing changes, run:

```bash
# Quick validation
make helm-test

# Full test suite
make helm-test-all

# Go validation
go test ./pkg/helm/...
```

## Test Directory Structure

```
helm/stats-agent-team/
├── tests/
│   ├── deployment_test.yaml    # Deployment template tests
│   ├── service_test.yaml       # Service template tests
│   ├── hpa_test.yaml           # HPA template tests
│   ├── pdb_test.yaml           # PDB template tests
│   └── ingress_test.yaml       # Ingress template tests
├── templates/
│   └── ...
├── values.yaml
├── values-minikube.yaml
└── values-eks.yaml
```

## Debugging Test Failures

### View Rendered Templates

```bash
# Render with specific values
helm template stats-agent ./helm/stats-agent-team \
  --set research.enabled=true \
  --set research.autoscaling.enabled=true \
  --debug

# Render specific template
helm template stats-agent ./helm/stats-agent-team \
  --show-only templates/hpa.yaml
```

### Verbose Unit Test Output

```bash
helm unittest ./helm/stats-agent-team --color --output-type detailed
```

### Check Template Syntax

```bash
# Validate YAML syntax
helm template stats-agent ./helm/stats-agent-team | yq '.'

# Check specific path
helm template stats-agent ./helm/stats-agent-team | \
  yq '.spec.template.spec.containers[0].resources'
```

## Adding New Tests

When adding new templates:

1. Create test file in `tests/<template>_test.yaml`
2. Add tests for:
   - Enabled/disabled states
   - Default values
   - Custom values
   - Edge cases
3. Run tests locally: `make helm-unittest`
4. Update this documentation if needed

### Test Template

```yaml
suite: <component> tests
templates:
  - <template-file>.yaml

tests:
  - it: should create <resource> when enabled
    set:
      <component>.enabled: true
    asserts:
      - isKind:
          of: <ResourceKind>
      - equal:
          path: metadata.name
          value: RELEASE-NAME-stats-agent-team-<component>

  - it: should not create <resource> when disabled
    set:
      <component>.enabled: false
    asserts:
      - hasDocuments:
          count: 0

  - it: should set custom values correctly
    set:
      <component>.enabled: true
      <component>.<field>: <value>
    asserts:
      - equal:
          path: <yaml.path>
          value: <expected>
```

## References

- [helm-unittest Documentation](https://github.com/helm-unittest/helm-unittest)
- [kubeconform Documentation](https://github.com/yannh/kubeconform)
- [Polaris Documentation](https://polaris.docs.fairwinds.com/)
- [chart-testing Documentation](https://github.com/helm/chart-testing)
- [Helm Testing Best Practices](https://helm.sh/docs/topics/chart_tests/)

# Scaling Guide

This guide covers horizontal and vertical scaling for the Stats Agent Team, following patterns established by [Apache Superset](https://github.com/apache/superset/tree/master/helm/superset) and [ArgoCD](https://github.com/argoproj/argo-helm/tree/main/charts/argo-cd).

## Overview

Each agent in the Stats Agent Team supports:

- **Horizontal Pod Autoscaling (HPA)** - Automatically scale replicas based on CPU/memory utilization
- **Pod Disruption Budgets (PDB)** - Ensure availability during voluntary disruptions
- **Resource Limits** - Control CPU and memory allocation per pod

All scaling features are **disabled by default** and can be enabled per-agent via Helm values.

## Quick Start

### Enable Autoscaling for Production

```yaml
# values-production.yaml
orchestration:
  autoscaling:
    enabled: true
    minReplicas: 2
    maxReplicas: 10
    targetCPUUtilizationPercentage: 70
  pdb:
    enabled: true
    minAvailable: 1

synthesis:
  autoscaling:
    enabled: true
    minReplicas: 2
    maxReplicas: 20
    targetCPUUtilizationPercentage: 80
  pdb:
    enabled: true
    minAvailable: 1
```

Deploy with:
```bash
helm upgrade --install stats-agent ./helm/stats-agent-team \
  -f values-production.yaml \
  --namespace stats-agent
```

## Horizontal Pod Autoscaler (HPA)

### Configuration Options

Each agent supports these autoscaling settings:

```yaml
<agent>:
  autoscaling:
    # Enable HPA for this agent
    enabled: false

    # Minimum number of replicas
    minReplicas: 1

    # Maximum number of replicas
    maxReplicas: 10

    # Scale up when average CPU exceeds this percentage
    targetCPUUtilizationPercentage: 80

    # Scale up when average memory exceeds this percentage (optional)
    # targetMemoryUtilizationPercentage: 80

    # Advanced scaling behavior (optional)
    # behavior:
    #   scaleDown:
    #     stabilizationWindowSeconds: 300
    #     policies:
    #       - type: Percent
    #         value: 10
    #         periodSeconds: 60
    #   scaleUp:
    #     stabilizationWindowSeconds: 0
    #     policies:
    #       - type: Percent
    #         value: 100
    #         periodSeconds: 15
```

### Scaling Recommendations by Agent

| Agent | Scaling Behavior | Recommended Settings |
|-------|------------------|---------------------|
| **Research** | I/O bound (web requests) | CPU-based, moderate scaling |
| **Synthesis** | CPU intensive (LLM calls) | CPU-based, aggressive scaling |
| **Verification** | Mixed (fetch + LLM) | CPU-based, moderate scaling |
| **Orchestration** | Stateless coordinator | Conservative scaling, ensure HA |
| **Direct** | CPU intensive (LLM) | CPU-based, aggressive scaling |

### Example: High-Traffic Configuration

```yaml
# For high-traffic production deployments
research:
  autoscaling:
    enabled: true
    minReplicas: 3
    maxReplicas: 15
    targetCPUUtilizationPercentage: 70

synthesis:
  autoscaling:
    enabled: true
    minReplicas: 5
    maxReplicas: 50
    targetCPUUtilizationPercentage: 60
    targetMemoryUtilizationPercentage: 70

verification:
  autoscaling:
    enabled: true
    minReplicas: 3
    maxReplicas: 20
    targetCPUUtilizationPercentage: 70

orchestration:
  autoscaling:
    enabled: true
    minReplicas: 2
    maxReplicas: 5
    targetCPUUtilizationPercentage: 80
```

### Scaling Behavior Tuning

For gradual scale-down to avoid flapping:

```yaml
synthesis:
  autoscaling:
    enabled: true
    minReplicas: 2
    maxReplicas: 20
    targetCPUUtilizationPercentage: 70
    behavior:
      scaleDown:
        stabilizationWindowSeconds: 300  # Wait 5 min before scaling down
        policies:
          - type: Percent
            value: 10                     # Scale down max 10% at a time
            periodSeconds: 60
      scaleUp:
        stabilizationWindowSeconds: 0    # Scale up immediately
        policies:
          - type: Percent
            value: 100                    # Double capacity if needed
            periodSeconds: 15
          - type: Pods
            value: 4                      # Or add 4 pods
            periodSeconds: 15
        selectPolicy: Max                 # Use whichever adds more pods
```

## Pod Disruption Budgets (PDB)

PDBs ensure a minimum number of pods remain available during voluntary disruptions like:
- Node drains
- Cluster upgrades
- Deployment rollouts

### Configuration Options

```yaml
<agent>:
  pdb:
    # Enable PDB for this agent
    enabled: false

    # Minimum pods that must remain available (use one or the other)
    minAvailable: 1          # Can be integer or percentage: "50%"

    # Maximum pods that can be unavailable
    # maxUnavailable: 1      # Can be integer or percentage: "25%"
```

**Note:** `minAvailable` and `maxUnavailable` are mutually exclusive. If neither is specified when PDB is enabled, `maxUnavailable: 1` is used as the default.

### Example: High Availability Configuration

```yaml
# Ensure at least 2 pods of each critical agent remain available
research:
  replicaCount: 3
  pdb:
    enabled: true
    minAvailable: 2

synthesis:
  replicaCount: 4
  pdb:
    enabled: true
    minAvailable: 2

verification:
  replicaCount: 3
  pdb:
    enabled: true
    minAvailable: 2

orchestration:
  replicaCount: 3
  pdb:
    enabled: true
    minAvailable: 2
```

### Example: Percentage-Based PDB

```yaml
# Allow up to 25% of pods to be unavailable during disruptions
synthesis:
  replicaCount: 8
  pdb:
    enabled: true
    maxUnavailable: "25%"
```

## Vertical Scaling (Resources)

Each agent's resource requests and limits can be configured:

```yaml
<agent>:
  resources:
    requests:
      cpu: 200m
      memory: 256Mi
    limits:
      cpu: 1000m
      memory: 512Mi
```

### Recommended Resource Profiles

#### Development / Minikube

```yaml
research:
  resources:
    requests:
      cpu: 50m
      memory: 64Mi
    limits:
      cpu: 200m
      memory: 128Mi

synthesis:
  resources:
    requests:
      cpu: 100m
      memory: 128Mi
    limits:
      cpu: 500m
      memory: 256Mi
```

#### Production

```yaml
research:
  resources:
    requests:
      cpu: 200m
      memory: 256Mi
    limits:
      cpu: 1000m
      memory: 512Mi

synthesis:
  resources:
    requests:
      cpu: 500m
      memory: 512Mi
    limits:
      cpu: 2000m
      memory: 1Gi

verification:
  resources:
    requests:
      cpu: 500m
      memory: 512Mi
    limits:
      cpu: 2000m
      memory: 1Gi

orchestration:
  resources:
    requests:
      cpu: 200m
      memory: 256Mi
    limits:
      cpu: 1000m
      memory: 512Mi
```

## Monitoring Scaling

### View HPA Status

```bash
# List all HPAs
kubectl get hpa -n stats-agent

# Watch HPA in real-time
kubectl get hpa -n stats-agent -w

# Describe specific HPA
kubectl describe hpa stats-agent-stats-agent-team-synthesis -n stats-agent
```

### View PDB Status

```bash
# List all PDBs
kubectl get pdb -n stats-agent

# Check disruption status
kubectl describe pdb stats-agent-stats-agent-team-orchestration -n stats-agent
```

### View Pod Resource Usage

```bash
# Current resource usage (requires metrics-server)
kubectl top pods -n stats-agent

# Resource requests/limits
kubectl describe pods -n stats-agent | grep -A 5 "Requests\|Limits"
```

## Scaling Patterns

### Pattern 1: Start Small, Scale Up

Begin with conservative settings and increase based on observed load:

```yaml
# Initial deployment
synthesis:
  replicaCount: 2
  autoscaling:
    enabled: true
    minReplicas: 2
    maxReplicas: 5
    targetCPUUtilizationPercentage: 80
```

Monitor and adjust:
```bash
# Watch scaling behavior
kubectl get hpa -n stats-agent -w

# If constantly at max, increase maxReplicas
helm upgrade stats-agent ./helm/stats-agent-team \
  --set synthesis.autoscaling.maxReplicas=10
```

### Pattern 2: Predictable Load Scaling

For predictable traffic patterns, use scheduled scaling with KEDA or CronJobs:

```yaml
# Scale up during business hours (example with KEDA)
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: synthesis-scaler
spec:
  scaleTargetRef:
    name: stats-agent-stats-agent-team-synthesis
  minReplicaCount: 2
  maxReplicaCount: 20
  triggers:
    - type: cron
      metadata:
        timezone: America/Los_Angeles
        start: 0 8 * * 1-5    # 8 AM weekdays
        end: 0 18 * * 1-5     # 6 PM weekdays
        desiredReplicas: "10"
```

### Pattern 3: Queue-Based Scaling

For batch processing workloads, scale based on queue depth (requires KEDA):

```yaml
# Scale based on pending work
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: synthesis-queue-scaler
spec:
  scaleTargetRef:
    name: stats-agent-stats-agent-team-synthesis
  minReplicaCount: 1
  maxReplicaCount: 50
  triggers:
    - type: redis
      metadata:
        address: redis:6379
        listName: synthesis-queue
        listLength: "10"  # 1 pod per 10 queue items
```

## Troubleshooting

### HPA Not Scaling

1. **Check metrics-server is running:**
   ```bash
   kubectl get pods -n kube-system | grep metrics-server
   ```

2. **Verify HPA can read metrics:**
   ```bash
   kubectl describe hpa <hpa-name> -n stats-agent
   ```

3. **Check resource requests are set:**
   HPA requires resource requests to calculate utilization percentages.

### Pods Evicted During Upgrade

1. **Enable PDB:**
   ```yaml
   orchestration:
     pdb:
       enabled: true
       minAvailable: 1
   ```

2. **Check PDB is working:**
   ```bash
   kubectl get pdb -n stats-agent
   ```

### Scaling Too Aggressively

Add stabilization windows:
```yaml
autoscaling:
  behavior:
    scaleDown:
      stabilizationWindowSeconds: 300
```

## References

- [Kubernetes HPA Documentation](https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/)
- [Kubernetes PDB Documentation](https://kubernetes.io/docs/tasks/run-application/configure-pdb/)
- [Apache Superset Helm Chart](https://github.com/apache/superset/tree/master/helm/superset)
- [ArgoCD Helm Chart](https://github.com/argoproj/argo-helm/tree/main/charts/argo-cd)
- [KEDA - Kubernetes Event-driven Autoscaling](https://keda.sh/)

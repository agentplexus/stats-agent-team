# ClaimsReport Integration

The Statistics Agent Team supports exporting verified statistics as [structured-evaluation](https://github.com/plexusone/structured-evaluation) ClaimsReport format. This enables standardized validation workflows and integration with claim verification systems.

## Overview

ClaimsReport provides a standardized format for representing verifiable claims with:

- **Claim categorization** - Statistical, factual, or opinion-based claims
- **Source validation** - External source URLs with reliability classification
- **Verdict tracking** - Verified, unverified, or rejected status
- **Audit trail** - Full rationale for each verification decision

## HTTP API

### Format Parameter

Request ClaimsReport JSON output by adding `?format=claims` to any endpoint:

```bash
# Orchestration endpoint
curl -X POST "http://localhost:8000/orchestrate?format=claims" \
  -H "Content-Type: application/json" \
  -d '{
    "topic": "climate change statistics",
    "min_verified_stats": 5
  }'

# Verification endpoint (include topic for report title)
curl -X POST "http://localhost:8002/verify?format=claims&topic=climate" \
  -H "Content-Type: application/json" \
  -d '{"statistics": [...]}'
```

### Response Format

The ClaimsReport JSON includes:

```json
{
  "metadata": {
    "document_title": "Statistics: climate change statistics",
    "generated_at": "2026-06-01T10:30:00Z",
    "source_type": "statistics-research"
  },
  "claims": [
    {
      "id": "stat-1",
      "text": "Global temperature increase: 1.10 °C",
      "category": "statistical",
      "location": {
        "section": "verified-statistics"
      },
      "validation": {
        "external": {
          "url": "https://climate.nasa.gov/vital-signs/global-temperature/",
          "source_type": "reputable_vendor",
          "quoted_text": "Global temperature has increased by 1.1°C...",
          "verified_match": true,
          "reliability": "high"
        }
      },
      "verdict": "verified",
      "rationale": "Excerpt verified in source: NASA"
    }
  ],
  "summary": {
    "total_claims": 5,
    "verified": 4,
    "rejected": 1
  }
}
```

## Go API

### OrchestrationResponse

Convert orchestration results to ClaimsReport:

```go
import "github.com/plexusone/agent-team-stats/pkg/models"

// After orchestration
resp, err := orchestrator.Orchestrate(ctx, req)
if err != nil {
    return err
}

// Convert to ClaimsReport (verified statistics only)
report := resp.ToClaimsReport()

// Or include verification failures for full audit trail
report := resp.ToClaimsReportWithFailures(failures)
```

### VerificationResponse

Convert verification results directly:

```go
// After verification
verifyResp, err := verifier.Verify(ctx, statistics)
if err != nil {
    return err
}

// Convert to ClaimsReport
report := verifyResp.ToClaimsReport("climate change")
```

## Source Classification

Sources are automatically classified based on their reputation:

| Source | Classification | Reliability |
|--------|----------------|-------------|
| WHO, CDC, NIH, FDA, EPA | `reputable_vendor` | `high` |
| Census Bureau, BLS, Federal Reserve | `reputable_vendor` | `high` |
| NASA, NOAA | `reputable_vendor` | `high` |
| Pew Research Center, Gallup | `reputable_vendor` | `high` |
| Other sources | `community` | Default |

Verified statistics receive `ReliabilityHigh`, while failed verifications receive `ReliabilityLow`.

## Claim Verdicts

Each claim receives a verdict based on verification status:

| Verdict | Description |
|---------|-------------|
| `verified` | Excerpt found in source, values match |
| `unverified` | Could not verify against source |
| `rejected` | Verification failed (excerpt not found, values mismatch) |

## Integration Example

Full workflow with ClaimsReport output:

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "os"

    "github.com/plexusone/agent-team-stats/pkg/models"
)

func main() {
    ctx := context.Background()

    // Run orchestration
    req := &models.OrchestrationRequest{
        Topic:            "renewable energy statistics",
        MinVerifiedStats: 5,
    }

    resp, err := orchestrator.Orchestrate(ctx, req)
    if err != nil {
        panic(err)
    }

    // Convert to ClaimsReport
    report := resp.ToClaimsReport()

    // Output as JSON
    data, _ := json.MarshalIndent(report, "", "  ")
    fmt.Println(string(data))

    // Or save to file
    os.WriteFile("statistics.claims.json", data, 0600)
}
```

## Use Cases

### Fact-Checking Pipelines

Integrate with automated fact-checking systems:

```bash
# Get claims report
curl -s "http://localhost:8000/orchestrate?format=claims" \
  -d '{"topic": "economic indicators"}' | \
  jq '.claims[] | select(.verdict == "verified")'
```

### Audit Logging

Maintain audit trail of all verification attempts:

```go
// Include failures for complete audit
report := resp.ToClaimsReportWithFailures(failures)

// Log all claims with rationale
for _, claim := range report.Claims {
    log.Info("claim processed",
        "id", claim.ID,
        "verdict", claim.Verdict,
        "rationale", claim.Rationale,
    )
}
```

### Quality Metrics

Calculate verification success rates:

```go
report := resp.ToClaimsReport()
summary := report.Summary

successRate := float64(summary.Verified) / float64(summary.TotalClaims) * 100
fmt.Printf("Verification rate: %.1f%%\n", successRate)
```

## Related Documentation

- [structured-evaluation](https://github.com/plexusone/structured-evaluation) - Claims validation framework
- [Output Format](../getting-started/quickstart.md#output-format) - Default JSON output
- [API Usage](../getting-started/quickstart.md#api-usage) - HTTP API reference

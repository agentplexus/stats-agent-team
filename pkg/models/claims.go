package models

import (
	"fmt"

	"github.com/plexusone/structured-evaluation/claims"
)

// ToClaimsReport converts OrchestrationResponse to a ClaimsReport.
// This provides a standardized output format compatible with structured-evaluation.
func (r *OrchestrationResponse) ToClaimsReport() *claims.ClaimsReport {
	report := claims.NewClaimsReport("statistics-research")
	report.Metadata.DocumentTitle = fmt.Sprintf("Statistics: %s", r.Topic)
	report.Metadata.GeneratedAt = r.Timestamp

	for i, stat := range r.Statistics {
		claimText := formatStatisticClaim(stat)

		claim := claims.NewClaim(
			fmt.Sprintf("stat-%d", i+1),
			claimText,
			claims.ClaimStatistical,
			claims.Location{Section: "verified-statistics"},
		)

		// Set external validation from URL source
		validation := claims.NewExternalValidation(
			stat.SourceURL,
			classifySourceType(stat.Source),
		)
		validation.External.QuotedText = stat.Excerpt
		validation.External.VerifiedMatch = stat.Verified
		validation.External.Reliability = claims.ReliabilityHigh // Verified sources
		claim.SetValidation(validation)

		if stat.Verified {
			claim.Verdict = claims.VerdictVerified
			claim.Rationale = fmt.Sprintf("Excerpt verified in source: %s", stat.Source)
		} else {
			claim.Verdict = claims.VerdictUnverified
			claim.Rationale = "Statistic not verified against source"
		}

		report.AddClaim(*claim)
	}

	report.Finalize()
	return report
}

// ToClaimsReportWithFailures includes both verified and failed verification results.
// This provides a complete audit trail of all statistics that were checked.
func (r *OrchestrationResponse) ToClaimsReportWithFailures(
	failures []VerificationResult,
) *claims.ClaimsReport {
	report := r.ToClaimsReport()

	for i, fail := range failures {
		if fail.Statistic == nil {
			continue
		}
		stat := fail.Statistic
		claimText := formatStatisticClaim(*stat)

		claim := claims.NewClaim(
			fmt.Sprintf("fail-%d", i+1),
			claimText,
			claims.ClaimStatistical,
			claims.Location{Section: "unverified-statistics"},
		)

		validation := claims.NewExternalValidation(
			stat.SourceURL,
			classifySourceType(stat.Source),
		)
		validation.External.QuotedText = stat.Excerpt
		validation.External.VerifiedMatch = false
		validation.External.Reliability = claims.ReliabilityLow // Failed verification
		claim.SetValidation(validation)

		claim.Verdict = claims.VerdictRejected
		claim.Rationale = fail.Reason

		report.AddClaim(*claim)
	}

	report.Finalize()
	return report
}

// ToClaimsReport converts a VerificationResponse to a ClaimsReport.
// This is useful when you want to report on verification results directly.
func (r *VerificationResponse) ToClaimsReport(topic string) *claims.ClaimsReport {
	report := claims.NewClaimsReport("statistics-verification")
	report.Metadata.DocumentTitle = fmt.Sprintf("Verification: %s", topic)
	report.Metadata.GeneratedAt = r.Timestamp

	for i, result := range r.Results {
		if result.Statistic == nil {
			continue
		}
		stat := result.Statistic
		claimText := formatStatisticClaim(*stat)

		claim := claims.NewClaim(
			fmt.Sprintf("verify-%d", i+1),
			claimText,
			claims.ClaimStatistical,
			claims.Location{Section: "verification-results"},
		)

		validation := claims.NewExternalValidation(
			stat.SourceURL,
			classifySourceType(stat.Source),
		)
		validation.External.QuotedText = stat.Excerpt
		validation.External.VerifiedMatch = result.Verified
		claim.SetValidation(validation)

		if result.Verified {
			claim.Verdict = claims.VerdictVerified
			validation.External.Reliability = claims.ReliabilityHigh
			claim.Rationale = fmt.Sprintf("Excerpt found in source: %s", stat.Source)
		} else {
			claim.Verdict = claims.VerdictRejected
			validation.External.Reliability = claims.ReliabilityLow
			claim.Rationale = result.Reason
		}

		report.AddClaim(*claim)
	}

	report.Finalize()
	return report
}

// formatStatisticClaim formats a statistic into a claim text string.
func formatStatisticClaim(stat Statistic) string {
	if stat.Unit != "" {
		return fmt.Sprintf("%s: %.2f %s", stat.Name, stat.Value, stat.Unit)
	}
	return fmt.Sprintf("%s: %.2f", stat.Name, stat.Value)
}

// classifySourceType maps source names to claims.ExternalSourceType.
// Returns ExternalReputableVendor for known authoritative sources,
// or ExternalCommunity for general sources.
func classifySourceType(source string) claims.ExternalSourceType {
	// Common authoritative government/research sources
	switch source {
	case "WHO", "World Health Organization":
		return claims.ExternalReputableVendor
	case "CDC", "Centers for Disease Control":
		return claims.ExternalReputableVendor
	case "NIH", "National Institutes of Health":
		return claims.ExternalReputableVendor
	case "FDA", "Food and Drug Administration":
		return claims.ExternalReputableVendor
	case "EPA", "Environmental Protection Agency":
		return claims.ExternalReputableVendor
	case "Census Bureau", "US Census":
		return claims.ExternalReputableVendor
	case "Bureau of Labor Statistics", "BLS":
		return claims.ExternalReputableVendor
	case "Federal Reserve", "The Fed":
		return claims.ExternalReputableVendor
	case "NASA":
		return claims.ExternalReputableVendor
	case "NOAA":
		return claims.ExternalReputableVendor
	case "Pew Research Center", "Pew":
		return claims.ExternalReputableVendor
	case "Gallup":
		return claims.ExternalReputableVendor
	default:
		// General/unknown sources
		return claims.ExternalCommunity
	}
}

package models

import (
	"testing"
	"time"

	"github.com/plexusone/structured-evaluation/claims"
)

func TestOrchestrationResponse_ToClaimsReport(t *testing.T) {
	now := time.Now()
	resp := &OrchestrationResponse{
		Topic: "climate change statistics",
		Statistics: []Statistic{
			{
				Name:      "Global temperature increase",
				Value:     1.1,
				Unit:      "°C",
				Source:    "NASA",
				SourceURL: "https://climate.nasa.gov/vital-signs/global-temperature/",
				Excerpt:   "Global temperature has increased by 1.1°C since pre-industrial times",
				Verified:  true,
				DateFound: now,
			},
			{
				Name:      "Sea level rise",
				Value:     3.3,
				Unit:      "mm/year",
				Source:    "NOAA",
				SourceURL: "https://www.noaa.gov/sea-level",
				Excerpt:   "Sea level is rising at 3.3mm per year",
				Verified:  true,
				DateFound: now,
			},
		},
		TotalCandidates: 5,
		VerifiedCount:   2,
		FailedCount:     3,
		Timestamp:       now,
	}

	report := resp.ToClaimsReport()

	// Verify report metadata
	if report.Metadata.DocumentTitle != "Statistics: climate change statistics" {
		t.Errorf("expected title 'Statistics: climate change statistics', got %q", report.Metadata.DocumentTitle)
	}

	// Verify claims count
	if len(report.Claims) != 2 {
		t.Fatalf("expected 2 claims, got %d", len(report.Claims))
	}

	// Verify first claim
	claim1 := report.Claims[0]
	if claim1.ID != "stat-1" {
		t.Errorf("expected claim ID 'stat-1', got %q", claim1.ID)
	}
	if claim1.Category != claims.ClaimStatistical {
		t.Errorf("expected category ClaimStatistical, got %v", claim1.Category)
	}
	if claim1.Verdict != claims.VerdictVerified {
		t.Errorf("expected verdict VerdictVerified, got %v", claim1.Verdict)
	}
	if claim1.Validation == nil || claim1.Validation.External == nil {
		t.Fatal("expected external validation to be set")
	}
	if claim1.Validation.External.URL != "https://climate.nasa.gov/vital-signs/global-temperature/" {
		t.Errorf("expected URL 'https://climate.nasa.gov/vital-signs/global-temperature/', got %q", claim1.Validation.External.URL)
	}
	if !claim1.Validation.External.VerifiedMatch {
		t.Error("expected VerifiedMatch to be true")
	}
	if claim1.Validation.External.Reliability != claims.ReliabilityHigh {
		t.Errorf("expected reliability ReliabilityHigh, got %v", claim1.Validation.External.Reliability)
	}
}

func TestOrchestrationResponse_ToClaimsReportWithFailures(t *testing.T) {
	now := time.Now()
	resp := &OrchestrationResponse{
		Topic: "test topic",
		Statistics: []Statistic{
			{
				Name:      "Verified stat",
				Value:     100,
				Unit:      "%",
				Source:    "Test Source",
				SourceURL: "https://example.com/verified",
				Excerpt:   "This is 100% verified",
				Verified:  true,
				DateFound: now,
			},
		},
		TotalCandidates: 2,
		VerifiedCount:   1,
		FailedCount:     1,
		Timestamp:       now,
	}

	failures := []VerificationResult{
		{
			Statistic: &Statistic{
				Name:      "Failed stat",
				Value:     50,
				Unit:      "%",
				Source:    "Bad Source",
				SourceURL: "https://example.com/failed",
				Excerpt:   "This excerpt was not found",
				Verified:  false,
				DateFound: now,
			},
			Verified: false,
			Reason:   "Excerpt not found in source content",
		},
	}

	report := resp.ToClaimsReportWithFailures(failures)

	// Should have 2 claims (1 verified + 1 failed)
	if len(report.Claims) != 2 {
		t.Fatalf("expected 2 claims, got %d", len(report.Claims))
	}

	// Find the failed claim
	var failedClaim *claims.Claim
	for i := range report.Claims {
		if report.Claims[i].ID == "fail-1" {
			failedClaim = &report.Claims[i]
			break
		}
	}

	if failedClaim == nil {
		t.Fatal("expected to find failed claim with ID 'fail-1'")
	}

	if failedClaim.Verdict != claims.VerdictRejected {
		t.Errorf("expected verdict VerdictRejected, got %v", failedClaim.Verdict)
	}
	if failedClaim.Rationale != "Excerpt not found in source content" {
		t.Errorf("expected rationale 'Excerpt not found in source content', got %q", failedClaim.Rationale)
	}
}

func TestVerificationResponse_ToClaimsReport(t *testing.T) {
	now := time.Now()
	resp := &VerificationResponse{
		Results: []VerificationResult{
			{
				Statistic: &Statistic{
					Name:      "Test stat",
					Value:     42,
					Unit:      "units",
					Source:    "Test Source",
					SourceURL: "https://example.com/test",
					Excerpt:   "The value is 42 units",
					Verified:  true,
					DateFound: now,
				},
				Verified: true,
				Reason:   "",
			},
			{
				Statistic: &Statistic{
					Name:      "Bad stat",
					Value:     99,
					Unit:      "",
					Source:    "Bad Source",
					SourceURL: "https://example.com/bad",
					Excerpt:   "Not found",
					Verified:  false,
					DateFound: now,
				},
				Verified: false,
				Reason:   "Excerpt not found",
			},
		},
		Verified:  1,
		Failed:    1,
		Timestamp: now,
	}

	report := resp.ToClaimsReport("test verification")

	if report.Metadata.DocumentTitle != "Verification: test verification" {
		t.Errorf("expected title 'Verification: test verification', got %q", report.Metadata.DocumentTitle)
	}

	if len(report.Claims) != 2 {
		t.Fatalf("expected 2 claims, got %d", len(report.Claims))
	}

	// Check verified claim
	if report.Claims[0].Verdict != claims.VerdictVerified {
		t.Errorf("expected first claim to be verified, got %v", report.Claims[0].Verdict)
	}

	// Check rejected claim
	if report.Claims[1].Verdict != claims.VerdictRejected {
		t.Errorf("expected second claim to be rejected, got %v", report.Claims[1].Verdict)
	}
}

func TestFormatStatisticClaim(t *testing.T) {
	tests := []struct {
		name     string
		stat     Statistic
		expected string
	}{
		{
			name: "with unit",
			stat: Statistic{
				Name:  "Temperature",
				Value: 25.5,
				Unit:  "°C",
			},
			expected: "Temperature: 25.50 °C",
		},
		{
			name: "without unit",
			stat: Statistic{
				Name:  "Count",
				Value: 100,
				Unit:  "",
			},
			expected: "Count: 100.00",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatStatisticClaim(tt.stat)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestClassifySourceType(t *testing.T) {
	tests := []struct {
		source   string
		expected claims.ExternalSourceType
	}{
		{"WHO", claims.ExternalReputableVendor},
		{"World Health Organization", claims.ExternalReputableVendor},
		{"CDC", claims.ExternalReputableVendor},
		{"NIH", claims.ExternalReputableVendor},
		{"Random Blog", claims.ExternalCommunity},
		{"Pew Research Center", claims.ExternalReputableVendor},
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			result := classifySourceType(tt.source)
			if result != tt.expected {
				t.Errorf("classifySourceType(%q) = %v, expected %v", tt.source, result, tt.expected)
			}
		})
	}
}

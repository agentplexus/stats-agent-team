package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/grokify/stats-agent/pkg/config"
	"github.com/grokify/stats-agent/pkg/models"
	"github.com/trpc-group/trpc-a2a-go/agent"
	"github.com/trpc-group/trpc-a2a-go/server"
	agentgo "github.com/trpc-group/trpc-agent-go"
)

// VerificationAgent is responsible for validating statistics in their sources
type VerificationAgent struct {
	cfg    *config.Config
	client *http.Client
	agent  *agentgo.Agent
}

// NewVerificationAgent creates a new verification agent
func NewVerificationAgent(cfg *config.Config) *VerificationAgent {
	// Initialize the trpc-agent
	agentInstance := agentgo.NewAgent(
		agentgo.WithName("statistics-verification-agent"),
		agentgo.WithDescription("Verifies that statistics actually exist in their claimed sources"),
		agentgo.WithSystemPrompt(`You are a statistics verification agent. Your job is to:
1. Fetch the content from the provided source URL
2. Search for the verbatim excerpt in the source content
3. Verify the numerical value matches exactly
4. Check if the source is reputable
5. Flag any discrepancies, hallucinations, or mismatches

Verification criteria:
- The exact excerpt must be present in the source
- The numerical value must match (allowing for reasonable formatting differences)
- The source must be accessible and legitimate
- The context must support the claimed statistic

Return a JSON object with:
{
  "verified": true/false,
  "reason": "explanation if verification failed"
}`),
	)

	return &VerificationAgent{
		cfg:    cfg,
		client: &http.Client{Timeout: 30 * time.Second},
		agent:  agentInstance,
	}
}

// VerifyStatistic verifies a single candidate statistic
func (va *VerificationAgent) VerifyStatistic(ctx context.Context, candidate models.CandidateStatistic) models.VerificationResult {
	log.Printf("Verification Agent: Verifying statistic from %s", candidate.SourceURL)

	// Fetch the source content
	sourceContent, err := va.fetchSourceContent(ctx, candidate.SourceURL)
	if err != nil {
		log.Printf("Failed to fetch source: %v", err)
		return models.VerificationResult{
			Statistic: &models.Statistic{
				Name:      candidate.Name,
				Value:     candidate.Value,
				Source:    candidate.Source,
				SourceURL: candidate.SourceURL,
				Excerpt:   candidate.Excerpt,
				Verified:  false,
				DateFound: time.Now(),
			},
			Verified: false,
			Reason:   fmt.Sprintf("Failed to fetch source: %v", err),
		}
	}

	// Use LLM to verify the statistic in the content
	verified, reason := va.verifyWithLLM(ctx, candidate, sourceContent)

	stat := &models.Statistic{
		Name:      candidate.Name,
		Value:     candidate.Value,
		Source:    candidate.Source,
		SourceURL: candidate.SourceURL,
		Excerpt:   candidate.Excerpt,
		Verified:  verified,
		DateFound: time.Now(),
	}

	return models.VerificationResult{
		Statistic: stat,
		Verified:  verified,
		Reason:    reason,
	}
}

// fetchSourceContent fetches the content from a URL
func (va *VerificationAgent) fetchSourceContent(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "StatisticsVerificationAgent/1.0")

	resp, err := va.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// Limit response size to prevent abuse
	limitedReader := io.LimitReader(resp.Body, 10*1024*1024) // 10MB limit
	body, err := io.ReadAll(limitedReader)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	return string(body), nil
}

// verifyWithLLM uses LLM to verify the statistic in the source content
func (va *VerificationAgent) verifyWithLLM(ctx context.Context, candidate models.CandidateStatistic, sourceContent string) (bool, string) {
	// For demonstration, implement a simple text-based verification
	// In production, this would use the LLM agent for more sophisticated verification

	// Simple check: does the excerpt appear in the source?
	if strings.Contains(sourceContent, candidate.Excerpt) {
		// Check if the value appears near the excerpt
		excerptIndex := strings.Index(sourceContent, candidate.Excerpt)
		contextWindow := 500 // characters before and after
		start := max(0, excerptIndex-contextWindow)
		end := min(len(sourceContent), excerptIndex+len(candidate.Excerpt)+contextWindow)
		context := sourceContent[start:end]

		if strings.Contains(context, candidate.Value) {
			return true, ""
		}
		return false, "Value not found in excerpt context"
	}

	// Fallback: use LLM for fuzzy matching
	prompt := fmt.Sprintf(`Verify if this statistic appears in the source content:

Statistic: %s
Value: %s
Claimed Excerpt: "%s"

Source Content (first 5000 chars):
%s

Return JSON: {"verified": true/false, "reason": "explanation"}`,
		candidate.Name,
		candidate.Value,
		candidate.Excerpt,
		truncate(sourceContent, 5000),
	)

	result, err := va.agent.Run(ctx, prompt)
	if err != nil {
		log.Printf("LLM verification failed: %v", err)
		return false, fmt.Sprintf("LLM verification failed: %v", err)
	}

	// Parse LLM response
	var llmResult struct {
		Verified bool   `json:"verified"`
		Reason   string `json:"reason"`
	}
	if err := json.Unmarshal([]byte(result), &llmResult); err != nil {
		// Fallback to simple text search
		return false, "Excerpt not found in source content"
	}

	return llmResult.Verified, llmResult.Reason
}

// Verify processes a verification request
func (va *VerificationAgent) Verify(ctx context.Context, req *models.VerificationRequest) (*models.VerificationResponse, error) {
	log.Printf("Verification Agent: Verifying %d candidates", len(req.Candidates))

	results := make([]models.VerificationResult, 0, len(req.Candidates))
	verifiedCount := 0
	failedCount := 0

	for _, candidate := range req.Candidates {
		result := va.VerifyStatistic(ctx, candidate)
		results = append(results, result)

		if result.Verified {
			verifiedCount++
		} else {
			failedCount++
		}
	}

	response := &models.VerificationResponse{
		Results:   results,
		Verified:  verifiedCount,
		Failed:    failedCount,
		Timestamp: time.Now(),
	}

	log.Printf("Verification Agent: %d verified, %d failed", verifiedCount, failedCount)
	return response, nil
}

// HandleVerificationRequest is the HTTP handler for verification requests
func (va *VerificationAgent) HandleVerificationRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.VerificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	resp, err := va.Verify(r.Context(), &req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Verification failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// StartA2AServer starts the A2A protocol server
func (va *VerificationAgent) StartA2AServer(port int) error {
	// Create A2A agent card
	card := &agent.AgentCard{
		Name:        "statistics-verification-agent",
		Description: "Verifies that statistics actually exist in their claimed sources",
		Skills: []agent.Skill{
			{
				Name:        "verify-statistics",
				Description: "Verify candidate statistics by checking their sources",
				InputMode:   "application/json",
				OutputMode:  "application/json",
			},
		},
	}

	// Create A2A server
	srv := server.NewServer(
		server.WithAgentCard(card),
		server.WithMessageHandler(va),
	)

	addr := fmt.Sprintf(":%d", port)
	log.Printf("Verification Agent starting A2A server on %s", addr)
	return http.ListenAndServe(addr, srv)
}

// ProcessMessage implements the A2A MessageHandler interface
func (va *VerificationAgent) ProcessMessage(ctx context.Context, msg *agent.Message) (*agent.Message, error) {
	var req models.VerificationRequest
	if err := json.Unmarshal([]byte(msg.Content), &req); err != nil {
		return nil, fmt.Errorf("invalid message content: %w", err)
	}

	resp, err := va.Verify(ctx, &req)
	if err != nil {
		return nil, err
	}

	respData, err := json.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	return &agent.Message{
		Content: string(respData),
		Role:    "assistant",
	}, nil
}

func main() {
	cfg := config.LoadConfig()
	verificationAgent := NewVerificationAgent(cfg)

	// Start HTTP server for non-A2A requests
	go func() {
		http.HandleFunc("/verify", verificationAgent.HandleVerificationRequest)
		http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})
		log.Println("Verification Agent HTTP server starting on :8002")
		if err := http.ListenAndServe(":8002", nil); err != nil {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	// Start A2A server if enabled
	if cfg.A2AEnabled {
		if err := verificationAgent.StartA2AServer(9002); err != nil {
			log.Fatalf("A2A server failed: %v", err)
		}
	} else {
		// Keep the program running
		select {}
	}
}

// Helper functions
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

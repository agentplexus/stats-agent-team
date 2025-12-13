package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/grokify/stats-agent/pkg/config"
	"github.com/grokify/stats-agent/pkg/models"
	"github.com/trpc-group/trpc-a2a-go/agent"
	"github.com/trpc-group/trpc-a2a-go/client"
	"github.com/trpc-group/trpc-a2a-go/server"
	agentgo "github.com/trpc-group/trpc-agent-go"
)

// OrchestrationAgent coordinates the research and verification agents
type OrchestrationAgent struct {
	cfg              *config.Config
	client           *http.Client
	agent            *agentgo.Agent
	researchClient   *client.Client
	verificationClient *client.Client
}

// NewOrchestrationAgent creates a new orchestration agent
func NewOrchestrationAgent(cfg *config.Config) *OrchestrationAgent {
	// Initialize the trpc-agent
	agentInstance := agentgo.NewAgent(
		agentgo.WithName("statistics-orchestration-agent"),
		agentgo.WithDescription("Coordinates research and verification agents to find verified statistics"),
		agentgo.WithSystemPrompt(`You are an orchestration agent that coordinates multiple agents to find verified statistics.

Your workflow:
1. Receive a request for statistics on a topic
2. Send request to research agent to find candidate statistics
3. Send candidates to verification agent for validation
4. If not enough verified statistics, request more from research agent
5. Return the final set of verified statistics

Decision-making criteria:
- Ensure minimum number of verified statistics is met
- Stop when sufficient verified statistics are obtained
- Handle failures gracefully with retries
- Prioritize quality over quantity`),
	)

	oa := &OrchestrationAgent{
		cfg:    cfg,
		client: &http.Client{Timeout: 60 * time.Second},
		agent:  agentInstance,
	}

	// Initialize A2A clients if enabled
	if cfg.A2AEnabled {
		oa.researchClient = client.NewClient(
			client.WithAgentURL(cfg.ResearchAgentURL),
		)
		oa.verificationClient = client.NewClient(
			client.WithAgentURL(cfg.VerificationAgentURL),
		)
	}

	return oa
}

// Orchestrate coordinates the workflow to find verified statistics
func (oa *OrchestrationAgent) Orchestrate(ctx context.Context, req *models.OrchestrationRequest) (*models.OrchestrationResponse, error) {
	log.Printf("Orchestration Agent: Starting orchestration for topic: %s", req.Topic)
	log.Printf("Target: %d verified statistics, max %d candidates", req.MinVerifiedStats, req.MaxCandidates)

	var allCandidates []models.CandidateStatistic
	var verifiedStatistics []models.Statistic
	totalVerified := 0
	totalFailed := 0
	maxRetries := 3
	retry := 0

	for retry < maxRetries && totalVerified < req.MinVerifiedStats {
		// Calculate how many more candidates we need
		candidatesNeeded := req.MinVerifiedStats - totalVerified
		if candidatesNeeded < 5 {
			candidatesNeeded = 5 // Always request at least 5 for buffer
		}

		// Don't exceed max candidates
		candidatesLeft := req.MaxCandidates - len(allCandidates)
		if candidatesLeft <= 0 {
			log.Printf("Reached maximum candidates limit (%d)", req.MaxCandidates)
			break
		}
		if candidatesNeeded > candidatesLeft {
			candidatesNeeded = candidatesLeft
		}

		// Step 1: Request statistics from research agent
		researchReq := &models.ResearchRequest{
			Topic:         req.Topic,
			MinStatistics: candidatesNeeded,
			MaxStatistics: candidatesNeeded + 5,
			ReputableOnly: req.ReputableOnly,
		}

		log.Printf("Orchestration: Requesting %d candidates from research agent (attempt %d/%d)",
			candidatesNeeded, retry+1, maxRetries)

		researchResp, err := oa.callResearchAgent(ctx, researchReq)
		if err != nil {
			log.Printf("Research agent failed: %v", err)
			retry++
			continue
		}

		log.Printf("Orchestration: Received %d candidates from research agent", len(researchResp.Candidates))
		allCandidates = append(allCandidates, researchResp.Candidates...)

		// Step 2: Send candidates to verification agent
		verifyReq := &models.VerificationRequest{
			Candidates: researchResp.Candidates,
		}

		log.Printf("Orchestration: Sending %d candidates to verification agent", len(verifyReq.Candidates))

		verifyResp, err := oa.callVerificationAgent(ctx, verifyReq)
		if err != nil {
			log.Printf("Verification agent failed: %v", err)
			retry++
			continue
		}

		log.Printf("Orchestration: Verification complete - %d verified, %d failed",
			verifyResp.Verified, verifyResp.Failed)

		// Step 3: Collect verified statistics
		for _, result := range verifyResp.Results {
			if result.Verified {
				verifiedStatistics = append(verifiedStatistics, *result.Statistic)
				totalVerified++
			} else {
				totalFailed++
				log.Printf("Statistic failed verification: %s - %s", result.Statistic.Name, result.Reason)
			}
		}

		log.Printf("Orchestration: Current progress - %d/%d verified statistics",
			totalVerified, req.MinVerifiedStats)

		// Check if we have enough verified statistics
		if totalVerified >= req.MinVerifiedStats {
			log.Printf("Orchestration: Target reached with %d verified statistics", totalVerified)
			break
		}

		retry++
	}

	// Build final response
	response := &models.OrchestrationResponse{
		Topic:           req.Topic,
		Statistics:      verifiedStatistics,
		TotalCandidates: len(allCandidates),
		VerifiedCount:   totalVerified,
		FailedCount:     totalFailed,
		Timestamp:       time.Now(),
	}

	if totalVerified < req.MinVerifiedStats {
		log.Printf("Warning: Only found %d verified statistics (target: %d)",
			totalVerified, req.MinVerifiedStats)
	} else {
		log.Printf("Orchestration: Successfully completed with %d verified statistics", totalVerified)
	}

	return response, nil
}

// callResearchAgent calls the research agent (via HTTP or A2A)
func (oa *OrchestrationAgent) callResearchAgent(ctx context.Context, req *models.ResearchRequest) (*models.ResearchResponse, error) {
	if oa.cfg.A2AEnabled && oa.researchClient != nil {
		return oa.callResearchAgentA2A(ctx, req)
	}
	return oa.callResearchAgentHTTP(ctx, req)
}

// callResearchAgentHTTP calls the research agent via HTTP
func (oa *OrchestrationAgent) callResearchAgentHTTP(ctx context.Context, req *models.ResearchRequest) (*models.ResearchResponse, error) {
	reqData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		fmt.Sprintf("%s/research", oa.cfg.ResearchAgentURL),
		bytes.NewReader(reqData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := oa.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	var researchResp models.ResearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&researchResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &researchResp, nil
}

// callResearchAgentA2A calls the research agent via A2A protocol
func (oa *OrchestrationAgent) callResearchAgentA2A(ctx context.Context, req *models.ResearchRequest) (*models.ResearchResponse, error) {
	reqData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	msg := &agent.Message{
		Content: string(reqData),
		Role:    "user",
	}

	respMsg, err := oa.researchClient.Send(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("A2A request failed: %w", err)
	}

	var researchResp models.ResearchResponse
	if err := json.Unmarshal([]byte(respMsg.Content), &researchResp); err != nil {
		return nil, fmt.Errorf("failed to decode A2A response: %w", err)
	}

	return &researchResp, nil
}

// callVerificationAgent calls the verification agent (via HTTP or A2A)
func (oa *OrchestrationAgent) callVerificationAgent(ctx context.Context, req *models.VerificationRequest) (*models.VerificationResponse, error) {
	if oa.cfg.A2AEnabled && oa.verificationClient != nil {
		return oa.callVerificationAgentA2A(ctx, req)
	}
	return oa.callVerificationAgentHTTP(ctx, req)
}

// callVerificationAgentHTTP calls the verification agent via HTTP
func (oa *OrchestrationAgent) callVerificationAgentHTTP(ctx context.Context, req *models.VerificationRequest) (*models.VerificationResponse, error) {
	reqData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		fmt.Sprintf("%s/verify", oa.cfg.VerificationAgentURL),
		bytes.NewReader(reqData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := oa.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	var verifyResp models.VerificationResponse
	if err := json.NewDecoder(resp.Body).Decode(&verifyResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &verifyResp, nil
}

// callVerificationAgentA2A calls the verification agent via A2A protocol
func (oa *OrchestrationAgent) callVerificationAgentA2A(ctx context.Context, req *models.VerificationRequest) (*models.VerificationResponse, error) {
	reqData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	msg := &agent.Message{
		Content: string(reqData),
		Role:    "user",
	}

	respMsg, err := oa.verificationClient.Send(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("A2A request failed: %w", err)
	}

	var verifyResp models.VerificationResponse
	if err := json.Unmarshal([]byte(respMsg.Content), &verifyResp); err != nil {
		return nil, fmt.Errorf("failed to decode A2A response: %w", err)
	}

	return &verifyResp, nil
}

// HandleOrchestrationRequest is the HTTP handler for orchestration requests
func (oa *OrchestrationAgent) HandleOrchestrationRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.OrchestrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	// Set defaults
	if req.MinVerifiedStats == 0 {
		req.MinVerifiedStats = 10
	}
	if req.MaxCandidates == 0 {
		req.MaxCandidates = 30
	}

	resp, err := oa.Orchestrate(r.Context(), &req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Orchestration failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// StartA2AServer starts the A2A protocol server
func (oa *OrchestrationAgent) StartA2AServer(port int) error {
	// Create A2A agent card
	card := &agent.AgentCard{
		Name:        "statistics-orchestration-agent",
		Description: "Coordinates research and verification agents to find verified statistics",
		Skills: []agent.Skill{
			{
				Name:        "orchestrate-statistics-search",
				Description: "Coordinate multi-agent workflow to find and verify statistics",
				InputMode:   "application/json",
				OutputMode:  "application/json",
			},
		},
	}

	// Create A2A server
	srv := server.NewServer(
		server.WithAgentCard(card),
		server.WithMessageHandler(oa),
	)

	addr := fmt.Sprintf(":%d", port)
	log.Printf("Orchestration Agent starting A2A server on %s", addr)
	return http.ListenAndServe(addr, srv)
}

// ProcessMessage implements the A2A MessageHandler interface
func (oa *OrchestrationAgent) ProcessMessage(ctx context.Context, msg *agent.Message) (*agent.Message, error) {
	var req models.OrchestrationRequest
	if err := json.Unmarshal([]byte(msg.Content), &req); err != nil {
		return nil, fmt.Errorf("invalid message content: %w", err)
	}

	resp, err := oa.Orchestrate(ctx, &req)
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
	orchestrationAgent := NewOrchestrationAgent(cfg)

	// Start HTTP server for non-A2A requests
	go func() {
		http.HandleFunc("/orchestrate", orchestrationAgent.HandleOrchestrationRequest)
		http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})
		log.Println("Orchestration Agent HTTP server starting on :8000")
		if err := http.ListenAndServe(":8000", nil); err != nil {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	// Start A2A server if enabled
	if cfg.A2AEnabled {
		if err := orchestrationAgent.StartA2AServer(9000); err != nil {
			log.Fatalf("A2A server failed: %v", err)
		}
	} else {
		// Keep the program running
		select {}
	}
}

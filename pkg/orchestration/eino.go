package orchestration

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/cloudwego/eino/compose"

	"github.com/grokify/stats-agent-team/pkg/config"
	"github.com/grokify/stats-agent-team/pkg/httpclient"
	"github.com/grokify/stats-agent-team/pkg/models"
)

// EinoOrchestrationAgent uses Eino framework for deterministic orchestration
type EinoOrchestrationAgent struct {
	cfg    *config.Config
	client *http.Client
	graph  *compose.Graph[*models.OrchestrationRequest, *models.OrchestrationResponse]
}

// NewEinoOrchestrationAgent creates a new Eino-based orchestration agent
func NewEinoOrchestrationAgent(cfg *config.Config) *EinoOrchestrationAgent {
	oa := &EinoOrchestrationAgent{
		cfg:    cfg,
		client: &http.Client{Timeout: 60 * time.Second},
	}

	// Build the deterministic workflow graph
	oa.graph = oa.buildWorkflowGraph()

	return oa
}

// buildWorkflowGraph creates a deterministic Eino graph for the workflow
func (oa *EinoOrchestrationAgent) buildWorkflowGraph() *compose.Graph[*models.OrchestrationRequest, *models.OrchestrationResponse] {
	// Create a new graph with typed input/output
	g := compose.NewGraph[*models.OrchestrationRequest, *models.OrchestrationResponse]()

	// Node names
	const (
		nodeValidateInput  = "validate_input"
		nodeResearch       = "research"
		nodeVerification   = "verification"
		nodeCheckQuality   = "check_quality"
		nodeRetryResearch  = "retry_research"
		nodeFormatResponse = "format_response"
	)

	// Add Lambda nodes for each step in the workflow

	// 1. Validate Input Node
	validateInputLambda := compose.InvokableLambda(func(ctx context.Context, req *models.OrchestrationRequest) (*models.OrchestrationRequest, error) {
		log.Printf("[Eino] Validating input for topic: %s", req.Topic)

		// Set defaults
		if req.MinVerifiedStats == 0 {
			req.MinVerifiedStats = 10
		}
		if req.MaxCandidates == 0 {
			req.MaxCandidates = 30
		}

		return req, nil
	})
	if err := g.AddLambdaNode(nodeValidateInput, validateInputLambda); err != nil {
		log.Printf("[Eino] Warning: failed to add validate input node: %v", err)
	}

	// 2. Research Node - calls research agent
	researchLambda := compose.InvokableLambda(func(ctx context.Context, req *models.OrchestrationRequest) (*ResearchState, error) {
		log.Printf("[Eino] Executing research for topic: %s", req.Topic)

		researchReq := &models.ResearchRequest{
			Topic:         req.Topic,
			MinStatistics: req.MinVerifiedStats,
			MaxStatistics: req.MaxCandidates,
			ReputableOnly: req.ReputableOnly,
		}

		resp, err := oa.callResearchAgent(ctx, researchReq)
		if err != nil {
			return nil, fmt.Errorf("research failed: %w", err)
		}

		return &ResearchState{
			Request:    req,
			Candidates: resp.Candidates,
		}, nil
	})
	if err := g.AddLambdaNode(nodeResearch, researchLambda); err != nil {
		log.Printf("[Eino] Warning: failed to add research node: %v", err)
	}

	// 3. Verification Node - calls verification agent
	verificationLambda := compose.InvokableLambda(func(ctx context.Context, state *ResearchState) (*VerificationState, error) {
		log.Printf("[Eino] Verifying %d candidates", len(state.Candidates))

		verifyReq := &models.VerificationRequest{
			Candidates: state.Candidates,
		}

		resp, err := oa.callVerificationAgent(ctx, verifyReq)
		if err != nil {
			return nil, fmt.Errorf("verification failed: %w", err)
		}

		// Extract verified statistics
		var verifiedStats []models.Statistic
		for _, result := range resp.Results {
			if result.Verified {
				verifiedStats = append(verifiedStats, *result.Statistic)
			}
		}

		return &VerificationState{
			Request:       state.Request,
			AllCandidates: state.Candidates,
			Verified:      verifiedStats,
			Failed:        resp.Failed,
		}, nil
	})
	if err := g.AddLambdaNode(nodeVerification, verificationLambda); err != nil {
		log.Printf("[Eino] Warning: failed to add verification node: %v", err)
	}

	// 4. Quality Check Node - deterministic decision
	qualityCheckLambda := compose.InvokableLambda(func(ctx context.Context, state *VerificationState) (*QualityDecision, error) {
		verified := len(state.Verified)
		target := state.Request.MinVerifiedStats

		log.Printf("[Eino] Quality check: %d verified (target: %d)", verified, target)

		decision := &QualityDecision{
			State:     state,
			NeedMore:  verified < target,
			Shortfall: target - verified,
		}

		if decision.NeedMore {
			log.Printf("[Eino] Need %d more verified statistics", decision.Shortfall)
		} else {
			log.Printf("[Eino] Quality target met")
		}

		return decision, nil
	})
	if err := g.AddLambdaNode(nodeCheckQuality, qualityCheckLambda); err != nil {
		log.Printf("[Eino] Warning: failed to add quality check node: %v", err)
	}

	// 5. Retry Research Node (if needed)
	retryResearchLambda := compose.InvokableLambda(func(ctx context.Context, decision *QualityDecision) (*ResearchState, error) {
		if !decision.NeedMore {
			// No retry needed, return existing state
			return &ResearchState{
				Request:    decision.State.Request,
				Candidates: decision.State.AllCandidates,
			}, nil
		}

		log.Printf("[Eino] Retrying research for %d more candidates", decision.Shortfall)

		// Request more candidates
		researchReq := &models.ResearchRequest{
			Topic:         decision.State.Request.Topic,
			MinStatistics: decision.Shortfall + 5, // buffer
			MaxStatistics: decision.Shortfall + 10,
			ReputableOnly: decision.State.Request.ReputableOnly,
		}

		resp, err := oa.callResearchAgent(ctx, researchReq)
		if err != nil {
			log.Printf("[Eino] Retry research failed: %v", err)
			// Return existing state on failure
			return &ResearchState{
				Request:    decision.State.Request,
				Candidates: decision.State.AllCandidates,
			}, nil
		}

		// Combine with existing candidates
		allCandidates := append(decision.State.AllCandidates, resp.Candidates...)

		return &ResearchState{
			Request:    decision.State.Request,
			Candidates: allCandidates,
		}, nil
	})
	if err := g.AddLambdaNode(nodeRetryResearch, retryResearchLambda); err != nil {
		log.Printf("[Eino] Warning: failed to add retry research node: %v", err)
	}

	// 6. Format Response Node
	formatResponseLambda := compose.InvokableLambda(func(ctx context.Context, state *VerificationState) (*models.OrchestrationResponse, error) {
		log.Printf("[Eino] Formatting response with %d verified statistics", len(state.Verified))

		return &models.OrchestrationResponse{
			Topic:           state.Request.Topic,
			Statistics:      state.Verified,
			TotalCandidates: len(state.AllCandidates),
			VerifiedCount:   len(state.Verified),
			FailedCount:     state.Failed,
			Timestamp:       time.Now(),
		}, nil
	})
	if err := g.AddLambdaNode(nodeFormatResponse, formatResponseLambda); err != nil {
		log.Printf("[Eino] Warning: failed to add format response node: %v", err)
	}

	// Add edges to define the workflow
	_ = g.AddEdge(compose.START, nodeValidateInput)
	_ = g.AddEdge(nodeValidateInput, nodeResearch)
	_ = g.AddEdge(nodeResearch, nodeVerification)
	_ = g.AddEdge(nodeVerification, nodeCheckQuality)

	// Conditional branching based on quality check
	_ = g.AddEdge(nodeCheckQuality, nodeRetryResearch)
	_ = g.AddEdge(nodeRetryResearch, nodeFormatResponse)
	_ = g.AddEdge(nodeFormatResponse, compose.END)

	return g
}

// Orchestrate executes the deterministic Eino workflow
func (oa *EinoOrchestrationAgent) Orchestrate(ctx context.Context, req *models.OrchestrationRequest) (*models.OrchestrationResponse, error) {
	log.Printf("[Eino Orchestrator] Starting deterministic workflow for topic: %s", req.Topic)

	// Compile the graph
	compiledGraph, err := oa.graph.Compile(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to compile graph: %w", err)
	}

	// Execute the graph
	result, err := compiledGraph.Invoke(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("workflow execution failed: %w", err)
	}

	log.Printf("[Eino Orchestrator] Workflow completed successfully")
	return result, nil
}

// Helper methods to call research and verification agents

func (oa *EinoOrchestrationAgent) callResearchAgent(ctx context.Context, req *models.ResearchRequest) (*models.ResearchResponse, error) {
	var resp models.ResearchResponse
	url := fmt.Sprintf("%s/research", oa.cfg.ResearchAgentURL)
	if err := httpclient.PostJSON(ctx, oa.client, url, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (oa *EinoOrchestrationAgent) callVerificationAgent(ctx context.Context, req *models.VerificationRequest) (*models.VerificationResponse, error) {
	var resp models.VerificationResponse
	url := fmt.Sprintf("%s/verify", oa.cfg.VerificationAgentURL)
	if err := httpclient.PostJSON(ctx, oa.client, url, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// HTTP Handler
func (oa *EinoOrchestrationAgent) HandleOrchestrationRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.OrchestrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	resp, err := oa.Orchestrate(r.Context(), &req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Orchestration failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("Failed to encode response: %v", err)
	}
}

// State types for the workflow
type ResearchState struct {
	Request    *models.OrchestrationRequest
	Candidates []models.CandidateStatistic
}

type VerificationState struct {
	Request       *models.OrchestrationRequest
	AllCandidates []models.CandidateStatistic
	Verified      []models.Statistic
	Failed        int
}

type QualityDecision struct {
	State     *VerificationState
	NeedMore  bool
	Shortfall int
}

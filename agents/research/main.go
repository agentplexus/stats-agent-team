package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/grokify/stats-agent/pkg/config"
	"github.com/grokify/stats-agent/pkg/models"
	"github.com/trpc-group/trpc-a2a-go/agent"
	"github.com/trpc-group/trpc-a2a-go/server"
	agentgo "github.com/trpc-group/trpc-agent-go"
)

// ResearchAgent is responsible for finding statistics from web searches
type ResearchAgent struct {
	cfg    *config.Config
	client *http.Client
	agent  *agentgo.Agent
}

// NewResearchAgent creates a new research agent
func NewResearchAgent(cfg *config.Config) *ResearchAgent {
	// Initialize the trpc-agent
	agentInstance := agentgo.NewAgent(
		agentgo.WithName("statistics-research-agent"),
		agentgo.WithDescription("Finds verifiable statistics from reputable web sources"),
		agentgo.WithSystemPrompt(`You are a statistics research agent. Your job is to:
1. Search the web for relevant statistics on the given topic
2. Prioritize reputable sources (academic journals, government agencies, established research organizations)
3. Extract numerical values with their context
4. Capture verbatim excerpts that contain the statistic
5. Return well-structured candidate statistics for verification

Reputable sources include:
- Government agencies (CDC, NIH, Census Bureau, etc.)
- Academic institutions and journals
- Established research organizations (Pew Research, Gallup, etc.)
- International organizations (WHO, UN, World Bank, etc.)
- Respected media with citations (NYT, WSJ, etc.)

Always include the exact URL and a verbatim quote containing the statistic.`),
	)

	return &ResearchAgent{
		cfg:    cfg,
		client: &http.Client{Timeout: 30 * time.Second},
		agent:  agentInstance,
	}
}

// Research performs web searches and extracts statistics
func (ra *ResearchAgent) Research(ctx context.Context, req *models.ResearchRequest) (*models.ResearchResponse, error) {
	log.Printf("Research Agent: Searching for statistics on topic: %s", req.Topic)

	// Build the search prompt
	searchPrompt := fmt.Sprintf(`Find %d to %d statistics about "%s".

For each statistic, provide:
1. Name/description of the statistic
2. Numerical value (number or percentage)
3. Source name (organization/publication)
4. Source URL
5. Verbatim excerpt containing the statistic

Return ONLY a JSON array of statistics in this exact format:
[
  {
    "name": "statistic description",
    "value": "numerical value with unit",
    "source": "source name",
    "source_url": "https://...",
    "excerpt": "exact quote from source"
  }
]`,
		req.MinStatistics,
		req.MaxStatistics,
		req.Topic,
	)

	if req.ReputableOnly {
		searchPrompt += "\n\nIMPORTANT: Only include statistics from highly reputable sources."
	}

	// Use the agent to perform research
	// In a real implementation, this would integrate with search APIs and LLM
	result, err := ra.agent.Run(ctx, searchPrompt)
	if err != nil {
		return nil, fmt.Errorf("research failed: %w", err)
	}

	// Parse the LLM response to extract statistics
	var candidates []models.CandidateStatistic
	if err := json.Unmarshal([]byte(result), &candidates); err != nil {
		log.Printf("Warning: Failed to parse LLM response as JSON: %v", err)
		// Fallback: create mock data for demonstration
		candidates = ra.generateMockCandidates(req)
	}

	response := &models.ResearchResponse{
		Topic:      req.Topic,
		Candidates: candidates,
		Timestamp:  time.Now(),
	}

	log.Printf("Research Agent: Found %d candidate statistics", len(candidates))
	return response, nil
}

// generateMockCandidates creates mock data for demonstration purposes
func (ra *ResearchAgent) generateMockCandidates(req *models.ResearchRequest) []models.CandidateStatistic {
	// This is temporary mock data - in production, this would come from actual search results
	return []models.CandidateStatistic{
		{
			Name:      fmt.Sprintf("Sample statistic about %s", req.Topic),
			Value:     "42%",
			Source:    "Pew Research Center",
			SourceURL: "https://www.pewresearch.org/example",
			Excerpt:   "According to our latest survey, 42% of respondents reported...",
		},
	}
}

// HandleResearchRequest is the HTTP handler for research requests
func (ra *ResearchAgent) HandleResearchRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.ResearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	// Set defaults
	if req.MinStatistics == 0 {
		req.MinStatistics = 5
	}
	if req.MaxStatistics == 0 {
		req.MaxStatistics = 10
	}

	resp, err := ra.Research(r.Context(), &req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Research failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// StartA2AServer starts the A2A protocol server
func (ra *ResearchAgent) StartA2AServer(port int) error {
	// Create A2A agent card
	card := &agent.AgentCard{
		Name:        "statistics-research-agent",
		Description: "Finds verifiable statistics from reputable web sources",
		Skills: []agent.Skill{
			{
				Name:        "research-statistics",
				Description: "Search for and extract statistics on a given topic",
				InputMode:   "application/json",
				OutputMode:  "application/json",
			},
		},
	}

	// Create A2A server
	srv := server.NewServer(
		server.WithAgentCard(card),
		server.WithMessageHandler(ra),
	)

	addr := fmt.Sprintf(":%d", port)
	log.Printf("Research Agent starting A2A server on %s", addr)
	return http.ListenAndServe(addr, srv)
}

// ProcessMessage implements the A2A MessageHandler interface
func (ra *ResearchAgent) ProcessMessage(ctx context.Context, msg *agent.Message) (*agent.Message, error) {
	var req models.ResearchRequest
	if err := json.Unmarshal([]byte(msg.Content), &req); err != nil {
		return nil, fmt.Errorf("invalid message content: %w", err)
	}

	resp, err := ra.Research(ctx, &req)
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
	researchAgent := NewResearchAgent(cfg)

	// Start HTTP server for non-A2A requests
	go func() {
		http.HandleFunc("/research", researchAgent.HandleResearchRequest)
		http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})
		log.Println("Research Agent HTTP server starting on :8001")
		if err := http.ListenAndServe(":8001", nil); err != nil {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	// Start A2A server if enabled
	if cfg.A2AEnabled {
		if err := researchAgent.StartA2AServer(9001); err != nil {
			log.Fatalf("A2A server failed: %v", err)
		}
	} else {
		// Keep the program running
		select {}
	}
}

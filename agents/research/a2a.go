package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"

	"github.com/a2aproject/a2a-go/a2a"
	"github.com/a2aproject/a2a-go/a2asrv"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model/gemini"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/server/adka2a"
	"google.golang.org/adk/session"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
	"google.golang.org/genai"

	"github.com/grokify/stats-agent-team/pkg/models"
)

// A2AServer represents the A2A protocol server for the Research Agent.
// Note: Research Agent is Tool-based (no LLM reasoning needed), but we wrap it
// in an ADK agent for A2A protocol compatibility. The LLM is minimal - just for
// tool invocation, not for reasoning about results.
type A2AServer struct {
	agent    *ResearchAgent
	adkAgent agent.Agent
	listener net.Listener
	baseURL  *url.URL
}

// NewA2AServer creates a new A2A server for the research agent
func NewA2AServer(ra *ResearchAgent, port string) (*A2AServer, error) {
	addr := "0.0.0.0:" + port
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	baseURL := &url.URL{Scheme: "http", Host: listener.Addr().String()}

	// Create the research tool that wraps the actual search functionality
	researchTool, err := functiontool.New(functiontool.Config{
		Name:        "web_search",
		Description: "Searches the web for sources related to a topic. Returns URLs and snippets from search results.",
	}, func(ctx tool.Context, input ResearchInput) (ResearchOutput, error) {
		results, err := ra.findSources(ctx, input.Topic, input.NumResults, input.ReputableOnly)
		if err != nil {
			return ResearchOutput{}, err
		}
		return ResearchOutput{SearchResults: results}, nil
	})
	if err != nil {
		listener.Close()
		return nil, err
	}

	// Create a minimal LLM model for tool invocation
	// Note: Research agent doesn't need LLM reasoning, but A2A requires an ADK agent
	ctx := context.Background()
	model, err := gemini.NewModel(ctx, "gemini-2.0-flash", &genai.ClientConfig{
		APIKey: os.Getenv("GOOGLE_API_KEY"),
	})
	if err != nil {
		listener.Close()
		return nil, err
	}

	// Create ADK agent wrapping the search tool
	adkAgent, err := llmagent.New(llmagent.Config{
		Name:        "research_agent",
		Model:       model,
		Description: "Finds relevant web sources for statistics research via search APIs",
		Instruction: `You are a research agent that finds web sources. When asked to find sources on a topic:
1. Use the web_search tool with the topic
2. Return the search results directly
Do not analyze or summarize - just return the raw search results.`,
		Tools: []tool.Tool{researchTool},
	})
	if err != nil {
		listener.Close()
		return nil, err
	}

	return &A2AServer{
		agent:    ra,
		adkAgent: adkAgent,
		listener: listener,
		baseURL:  baseURL,
	}, nil
}

// Start starts the A2A server
func (s *A2AServer) Start(ctx context.Context) error {
	agentPath := "/invoke"

	// Build agent card
	agentCard := &a2a.AgentCard{
		Name:               s.adkAgent.Name(),
		Description:        "Finds relevant web sources for statistics research (Tool-based, minimal LLM)",
		Skills:             adka2a.BuildAgentSkills(s.adkAgent),
		PreferredTransport: a2a.TransportProtocolJSONRPC,
		URL:                s.baseURL.JoinPath(agentPath).String(),
		Capabilities:       a2a.AgentCapabilities{Streaming: true},
	}

	mux := http.NewServeMux()

	// Register agent card endpoint
	mux.Handle(a2asrv.WellKnownAgentCardPath, a2asrv.NewStaticAgentCardHandler(agentCard))

	// Create executor
	executor := adka2a.NewExecutor(adka2a.ExecutorConfig{
		RunnerConfig: runner.Config{
			AppName:        s.adkAgent.Name(),
			Agent:          s.adkAgent,
			SessionService: session.InMemoryService(),
		},
	})

	// Create handlers
	requestHandler := a2asrv.NewHandler(executor)
	mux.Handle(agentPath, a2asrv.NewJSONRPCHandler(requestHandler))

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	log.Printf("Research Agent A2A server starting on %s", s.baseURL.String())
	log.Printf("  Agent Card: %s%s", s.baseURL.String(), a2asrv.WellKnownAgentCardPath)
	log.Printf("  Invoke: %s%s", s.baseURL.String(), agentPath)
	log.Printf("  Note: Tool-based agent (search API), LLM only for A2A protocol compatibility")

	return http.Serve(s.listener, mux)
}

// URL returns the base URL
func (s *A2AServer) URL() string {
	return s.baseURL.String()
}

// Close closes the server
func (s *A2AServer) Close() error {
	return s.listener.Close()
}

// Helper to convert search results to models format
func searchResultsToModels(results []models.SearchResult) []models.SearchResult {
	return results
}

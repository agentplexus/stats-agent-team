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
	"github.com/grokify/stats-agent-team/pkg/orchestration"
)

// A2AServer represents the A2A protocol server for the Eino Orchestration Agent.
// Note: Eino uses graph-based orchestration, but we wrap it in an ADK agent
// for A2A protocol compatibility. The LLM is minimal - just for tool invocation.
type A2AServer struct {
	einoAgent *orchestration.EinoOrchestrationAgent
	adkAgent  agent.Agent
	listener  net.Listener
	baseURL   *url.URL
}

// OrchestrationInput defines input for the orchestration tool
type OrchestrationInput struct {
	Topic            string `json:"topic" jsonschema:"description=The topic to find statistics for"`
	MinVerifiedStats int    `json:"min_verified_stats" jsonschema:"description=Minimum verified statistics to return"`
	MaxCandidates    int    `json:"max_candidates" jsonschema:"description=Maximum candidates to consider"`
	ReputableOnly    bool   `json:"reputable_only" jsonschema:"description=Only use reputable sources"`
}

// NewA2AServer creates a new A2A server for the Eino orchestration agent
func NewA2AServer(einoAgent *orchestration.EinoOrchestrationAgent, port string) (*A2AServer, error) {
	addr := "0.0.0.0:" + port
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	baseURL := &url.URL{Scheme: "http", Host: listener.Addr().String()}

	// Create the orchestration tool that wraps the Eino graph
	orchestrateTool, err := functiontool.New(functiontool.Config{
		Name:        "orchestrate_statistics_workflow",
		Description: "Orchestrates a deterministic workflow using Eino graph to find and verify statistics on a topic",
	}, func(ctx tool.Context, input OrchestrationInput) (*models.OrchestrationResponse, error) {
		req := &models.OrchestrationRequest{
			Topic:            input.Topic,
			MinVerifiedStats: input.MinVerifiedStats,
			MaxCandidates:    input.MaxCandidates,
			ReputableOnly:    input.ReputableOnly,
		}
		return einoAgent.Orchestrate(ctx, req)
	})
	if err != nil {
		listener.Close()
		return nil, err
	}

	// Create a minimal LLM model for A2A protocol
	ctx := context.Background()
	model, err := gemini.NewModel(ctx, "gemini-2.0-flash", &genai.ClientConfig{
		APIKey: os.Getenv("GOOGLE_API_KEY"),
	})
	if err != nil {
		listener.Close()
		return nil, err
	}

	// Create ADK agent wrapping the Eino orchestration
	adkAgent, err := llmagent.New(llmagent.Config{
		Name:        "eino_orchestration_agent",
		Model:       model,
		Description: "Orchestrates multi-agent workflow using Eino graph-based orchestration (deterministic)",
		Instruction: `You are an orchestration agent that coordinates a statistics research workflow.
When asked to find statistics on a topic:
1. Use the orchestrate_statistics_workflow tool with the topic
2. Return the verified statistics from the response
The workflow is deterministic (graph-based, not LLM-driven).`,
		Tools: []tool.Tool{orchestrateTool},
	})
	if err != nil {
		listener.Close()
		return nil, err
	}

	return &A2AServer{
		einoAgent: einoAgent,
		adkAgent:  adkAgent,
		listener:  listener,
		baseURL:   baseURL,
	}, nil
}

// Start starts the A2A server
func (s *A2AServer) Start(ctx context.Context) error {
	agentPath := "/invoke"

	// Build agent card
	agentCard := &a2a.AgentCard{
		Name:               s.adkAgent.Name(),
		Description:        "Eino graph-based orchestration for verified statistics (deterministic workflow)",
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

	log.Printf("Eino Orchestration Agent A2A server starting on %s", s.baseURL.String())
	log.Printf("  Agent Card: %s%s", s.baseURL.String(), a2asrv.WellKnownAgentCardPath)
	log.Printf("  Invoke: %s%s", s.baseURL.String(), agentPath)
	log.Printf("  Note: Uses Eino graph-based orchestration (deterministic, not LLM-driven)")

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

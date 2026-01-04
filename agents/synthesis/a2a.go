package main

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/a2aproject/a2a-go/a2a"
	"github.com/a2aproject/a2a-go/a2asrv"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/server/adka2a"
	"google.golang.org/adk/session"
)

// A2AServer represents the A2A protocol server for the Synthesis Agent
type A2AServer struct {
	agent    *SynthesisAgent
	listener net.Listener
	baseURL  *url.URL
	logger   *slog.Logger
}

// NewA2AServer creates a new A2A server for the synthesis agent
func NewA2AServer(agent *SynthesisAgent, port string, logger *slog.Logger) (*A2AServer, error) {
	addr := "0.0.0.0:" + port
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	baseURL := &url.URL{Scheme: "http", Host: listener.Addr().String()}

	return &A2AServer{
		agent:    agent,
		listener: listener,
		baseURL:  baseURL,
		logger:   logger,
	}, nil
}

// Start starts the A2A server
func (s *A2AServer) Start(context.Context) error {
	agentPath := "/invoke"

	// Build agent card with skills extracted from the ADK agent
	agentCard := &a2a.AgentCard{
		Name:               s.agent.adkAgent.Name(),
		Description:        "Extracts statistics from web content using LLM analysis",
		Skills:             adka2a.BuildAgentSkills(s.agent.adkAgent),
		PreferredTransport: a2a.TransportProtocolJSONRPC,
		URL:                s.baseURL.JoinPath(agentPath).String(),
		Capabilities:       a2a.AgentCapabilities{Streaming: true},
	}

	mux := http.NewServeMux()

	// Register agent card endpoint for discovery
	mux.Handle(a2asrv.WellKnownAgentCardPath, a2asrv.NewStaticAgentCardHandler(agentCard))

	// Create executor for A2A requests
	executor := adka2a.NewExecutor(adka2a.ExecutorConfig{
		RunnerConfig: runner.Config{
			AppName:        s.agent.adkAgent.Name(),
			Agent:          s.agent.adkAgent,
			SessionService: session.InMemoryService(),
		},
	})

	// Create request handler and JSON-RPC wrapper
	requestHandler := a2asrv.NewHandler(executor)
	mux.Handle(agentPath, a2asrv.NewJSONRPCHandler(requestHandler))

	// Add health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	s.logger.Info("A2A server starting",
		"url", s.baseURL.String(),
		"agent_card", s.baseURL.String()+a2asrv.WellKnownAgentCardPath,
		"invoke", s.baseURL.String()+agentPath)

	server := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	return server.Serve(s.listener)
}

// URL returns the base URL of the A2A server
func (s *A2AServer) URL() string {
	return s.baseURL.String()
}

// Close closes the A2A server
func (s *A2AServer) Close() error {
	return s.listener.Close()
}

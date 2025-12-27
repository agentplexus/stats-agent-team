package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/agentplexus/stats-agent-team/pkg/config"
	"github.com/agentplexus/stats-agent-team/pkg/orchestration"
)

func main() {
	cfg := config.LoadConfig()
	einoAgent := orchestration.NewEinoOrchestrationAgent(cfg)

	// Start A2A server if enabled (standard protocol for agent interoperability)
	// Note: Eino uses graph-based orchestration, wrapped in ADK for A2A compatibility
	if cfg.A2AEnabled {
		a2aServer, err := NewA2AServer(einoAgent, "9000")
		if err != nil {
			log.Printf("Failed to create A2A server: %v", err)
		} else {
			go func() {
				if err := a2aServer.Start(context.Background()); err != nil {
					log.Printf("A2A server error: %v", err)
				}
			}()
			log.Println("[Eino Orchestrator] A2A server started on :9000")
		}
	}

	// Start HTTP server with timeout (for custom security: SPIFFE, KYA, XAA, and observability)
	server := &http.Server{
		Addr:         ":8000",
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	http.HandleFunc("/orchestrate", einoAgent.HandleOrchestrationRequest)
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("OK")); err != nil {
			log.Printf("Failed to write health response: %v", err)
		}
	})

	log.Println("[Eino Orchestrator] HTTP server starting on :8000")
	log.Println("(Dual mode: HTTP for security/observability, A2A for interoperability)")
	log.Println("Note: Uses Eino graph-based deterministic orchestration")
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("HTTP server failed: %v", err)
	}
}

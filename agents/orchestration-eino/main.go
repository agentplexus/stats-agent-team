package main

import (
	"log"
	"net/http"

	"github.com/grokify/stats-agent-team/pkg/config"
	"github.com/grokify/stats-agent-team/pkg/orchestration"
)

func main() {
	cfg := config.LoadConfig()
	einoAgent := orchestration.NewEinoOrchestrationAgent(cfg)

	// Start HTTP server
	http.HandleFunc("/orchestrate", einoAgent.HandleOrchestrationRequest)
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	log.Println("[Eino Orchestrator] HTTP server starting on :8003")
	if err := http.ListenAndServe(":8003", nil); err != nil {
		log.Fatalf("HTTP server failed: %v", err)
	}
}

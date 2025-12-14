package main

import (
	"log"
	"net/http"
	"time"

	"github.com/grokify/stats-agent-team/pkg/config"
	"github.com/grokify/stats-agent-team/pkg/orchestration"
)

func main() {
	cfg := config.LoadConfig()
	einoAgent := orchestration.NewEinoOrchestrationAgent(cfg)

	// Start HTTP server with timeout
	server := &http.Server{
		Addr:         ":8003",
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

	log.Println("[Eino Orchestrator] HTTP server starting on :8003")
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("HTTP server failed: %v", err)
	}
}

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/jessevdk/go-flags"

	"github.com/grokify/stats-agent-team/pkg/config"
	"github.com/grokify/stats-agent-team/pkg/models"
)

// Options defines the CLI options structure
type Options struct {
	// Global options
	Verbose bool `short:"v" long:"verbose" description:"Show verbose debug information"`
	Version bool `long:"version" description:"Show version information"`

	// Commands
	Search SearchCommand `command:"search" description:"Search for verified statistics on a topic"`
}

// SearchCommand defines options for the search command
type SearchCommand struct {
	// Positional arguments
	Args struct {
		Topic string `positional-arg-name:"topic" description:"Topic to search for statistics"`
	} `positional-args:"yes" required:"yes"`

	// Search options
	MinStats      int    `short:"m" long:"min-stats" default:"10" description:"Minimum number of verified statistics required"`
	MaxCandidates int    `short:"c" long:"max-candidates" default:"30" description:"Maximum number of candidate statistics to gather"`
	ReputableOnly bool   `short:"r" long:"reputable-only" description:"Only use reputable sources"`
	Output        string `short:"o" long:"output" default:"both" choice:"json" choice:"text" choice:"both" description:"Output format"`

	// Orchestrator options
	OrchestratorURL string `long:"orchestrator-url" description:"Orchestrator URL (overrides env var)" env:"ORCHESTRATOR_URL"`
}

// Execute runs the search command
func (cmd *SearchCommand) Execute(args []string) error {
	topic := cmd.Args.Topic

	cfg := config.LoadConfig()

	// Override orchestrator URL if provided
	if cmd.OrchestratorURL != "" {
		cfg.OrchestratorURL = cmd.OrchestratorURL
	}

	// Create orchestration request
	req := &models.OrchestrationRequest{
		Topic:            topic,
		MinVerifiedStats: cmd.MinStats,
		MaxCandidates:    cmd.MaxCandidates,
		ReputableOnly:    cmd.ReputableOnly,
	}

	fmt.Printf("Searching for statistics about: %s\n", topic)
	fmt.Printf("Target: %d verified statistics from reputable sources\n\n", cmd.MinStats)

	// Call orchestration agent
	resp, err := callOrchestrator(cfg, req)
	if err != nil {
		return fmt.Errorf("orchestration failed: %w", err)
	}

	// Print results based on output format
	printResults(resp, cmd.Output)

	return nil
}

func main() {
	var opts Options

	parser := flags.NewParser(&opts, flags.Default)
	parser.LongDescription = `Statistics Agent - Multi-Agent System for Finding Verified Statistics

ARCHITECTURE:
This system uses a 4-agent architecture:

1. Research Agent (port 8001)
   - Searches for statistics from web sources via Serper/SerpAPI
   - Returns URLs with metadata

2. Synthesis Agent (port 8004)
   - Fetches webpage content from URLs
   - Uses LLM to extract statistics intelligently

3. Verification Agent (port 8002)
   - Re-fetches source URLs
   - Validates statistics in their sources
   - Checks for exact excerpts and values

4. Orchestration Agent (port 8000 ADK / 8003 Eino)
   - Coordinates the 4-agent workflow
   - Manages retry logic
   - Ensures quality standards

ENVIRONMENT VARIABLES:
LLM_PROVIDER          LLM provider (gemini, claude, openai, ollama)
GEMINI_API_KEY        API key for Gemini
CLAUDE_API_KEY        API key for Claude
OPENAI_API_KEY        API key for OpenAI
SEARCH_PROVIDER       Search provider (serper, serpapi)
SERPER_API_KEY        API key for Serper
SERPAPI_API_KEY       API key for SerpAPI
ORCHESTRATOR_URL      Orchestrator URL (default: http://localhost:8000)

EXAMPLES:
stats-agent search "climate change"
stats-agent search "AI adoption rates" --min-stats 15
stats-agent search "cybersecurity 2024" --output json
stats-agent search "renewable energy" --reputable-only
`

	// Parse arguments
	_, err := parser.Parse()
	if err != nil {
		if flagsErr, ok := err.(*flags.Error); ok {
			if flagsErr.Type == flags.ErrHelp {
				os.Exit(0)
			}
		}
		os.Exit(1)
	}

	// Handle version flag
	if opts.Version {
		fmt.Println("stats-agent version 1.0.0")
		fmt.Println("Multi-LLM support: Gemini, Claude, OpenAI, Ollama")
		os.Exit(0)
	}
}

func callOrchestrator(cfg *config.Config, req *models.OrchestrationRequest) (*models.OrchestrationResponse, error) {
	reqData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/orchestrate", cfg.OrchestratorURL)

	httpReq, err := http.NewRequestWithContext(context.Background(), "POST", url, bytes.NewReader(reqData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	httpResp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", httpResp.StatusCode, httpResp.Status)
	}

	var resp models.OrchestrationResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&resp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &resp, nil
}

func printResults(resp *models.OrchestrationResponse, outputFormat string) {
	if outputFormat == "json" {
		// JSON only
		jsonData, err := json.MarshalIndent(resp.Statistics, "", "  ")
		if err != nil {
			log.Printf("Error marshaling JSON: %v", err)
			return
		}
		fmt.Println(string(jsonData))
		return
	}

	// Text format (header + stats)
	fmt.Printf("=== Statistics Search Results ===\n\n")
	fmt.Printf("Topic: %s\n", resp.Topic)
	fmt.Printf("Found: %d verified statistics (from %d candidates)\n", resp.VerifiedCount, resp.TotalCandidates)
	fmt.Printf("Failed verification: %d\n", resp.FailedCount)
	fmt.Printf("Timestamp: %s\n\n", resp.Timestamp.Format("2006-01-02 15:04:05"))

	if len(resp.Statistics) == 0 {
		fmt.Println("No verified statistics found.")
		return
	}

	if outputFormat == "both" {
		// Print JSON
		fmt.Println("=== Verified Statistics (JSON) ===")
		fmt.Println()
		jsonData, err := json.MarshalIndent(resp.Statistics, "", "  ")
		if err != nil {
			log.Printf("Error marshaling JSON: %v", err)
			return
		}
		fmt.Println(string(jsonData))
		fmt.Println()
	}

	// Human-readable format
	fmt.Println("=== Human-Readable Format ===")
	fmt.Println()
	for i, stat := range resp.Statistics {
		fmt.Printf("%d. %s\n", i+1, stat.Name)
		fmt.Printf("   Value: %v %s\n", stat.Value, stat.Unit)
		fmt.Printf("   Source: %s\n", stat.Source)
		fmt.Printf("   URL: %s\n", stat.SourceURL)
		fmt.Printf("   Excerpt: \"%s\"\n", stat.Excerpt)
		fmt.Printf("   Verified: ✓\n")
		fmt.Printf("   Date Found: %s\n\n", stat.DateFound.Format("2006-01-02"))
	}
}

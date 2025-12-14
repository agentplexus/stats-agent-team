package agent

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/model"

	"github.com/grokify/stats-agent-team/pkg/config"
	"github.com/grokify/stats-agent-team/pkg/llm"
)

// BaseAgent provides common functionality for all agents
type BaseAgent struct {
	Cfg          *config.Config
	Client       *http.Client
	Model        model.LLM
	ModelFactory *llm.ModelFactory
}

// NewBaseAgent creates a new base agent with LLM initialization
func NewBaseAgent(cfg *config.Config, timeoutSec int) (*BaseAgent, error) {
	ctx := context.Background()

	// Create model using factory
	modelFactory := llm.NewModelFactory(cfg)
	llmModel, err := modelFactory.CreateModel(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create model: %w", err)
	}

	return &BaseAgent{
		Cfg:          cfg,
		Client:       &http.Client{Timeout: time.Duration(timeoutSec) * time.Second},
		Model:        llmModel,
		ModelFactory: modelFactory,
	}, nil
}

// GetProviderInfo returns information about the LLM provider
func (ba *BaseAgent) GetProviderInfo() string {
	return ba.ModelFactory.GetProviderInfo()
}

// FetchURL fetches content from a URL with proper error handling
func (ba *BaseAgent) FetchURL(ctx context.Context, url string, maxSizeMB int) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "StatsAgentTeam/1.0")

	resp, err := ba.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// Limit response size
	maxBytes := int64(maxSizeMB * 1024 * 1024)
	limitedReader := io.LimitReader(resp.Body, maxBytes)
	body, err := io.ReadAll(limitedReader)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	return string(body), nil
}

// LogInfo logs an informational message with agent context
func (ba *BaseAgent) LogInfo(agentName, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	log.Printf("[%s] %s", agentName, msg)
}

// LogError logs an error message with agent context
func (ba *BaseAgent) LogError(agentName, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	log.Printf("[%s] ERROR: %s", agentName, msg)
}

// AgentWrapper wraps common agent initialization patterns
type AgentWrapper struct {
	*BaseAgent
	ADKAgent agent.Agent
}

// NewAgentWrapper creates a wrapper with both base functionality and ADK agent
func NewAgentWrapper(base *BaseAgent, adkAgent agent.Agent) *AgentWrapper {
	return &AgentWrapper{
		BaseAgent: base,
		ADKAgent:  adkAgent,
	}
}

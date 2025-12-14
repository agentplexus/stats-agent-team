package llm

import (
	"context"
	"fmt"

	"google.golang.org/adk/model"
	"google.golang.org/adk/model/gemini"
	"google.golang.org/genai"

	"github.com/grokify/stats-agent-team/pkg/config"
	"github.com/grokify/stats-agent-team/pkg/llm/adapters"
)

// ModelFactory creates LLM models based on configuration
type ModelFactory struct {
	cfg *config.Config
}

// NewModelFactory creates a new model factory
func NewModelFactory(cfg *config.Config) *ModelFactory {
	return &ModelFactory{cfg: cfg}
}

// CreateModel creates an LLM model based on the configured provider
func (mf *ModelFactory) CreateModel(ctx context.Context) (model.LLM, error) {
	switch mf.cfg.LLMProvider {
	case "gemini", "":
		return mf.createGeminiModel(ctx)
	case "claude":
		return mf.createClaudeModel()
	case "openai":
		return mf.createOpenAIModel()
	case "xai":
		return mf.createXAIModel()
	case "ollama":
		return mf.createOllamaModel()
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s (supported: gemini, claude, openai, xai, ollama)", mf.cfg.LLMProvider)
	}
}

// createGeminiModel creates a Gemini model
func (mf *ModelFactory) createGeminiModel(ctx context.Context) (model.LLM, error) {
	apiKey := mf.cfg.GeminiAPIKey
	if apiKey == "" {
		apiKey = mf.cfg.LLMAPIKey
	}

	if apiKey == "" {
		return nil, fmt.Errorf("gemini API key not set - please set GOOGLE_API_KEY or GEMINI_API_KEY")
	}

	modelName := mf.cfg.LLMModel
	if modelName == "" {
		modelName = "gemini-2.0-flash-exp"
	}

	return gemini.NewModel(ctx, modelName, &genai.ClientConfig{
		APIKey: apiKey,
	})
}

// createClaudeModel creates a Claude model using gollm
func (mf *ModelFactory) createClaudeModel() (model.LLM, error) {
	apiKey := mf.cfg.ClaudeAPIKey
	if apiKey == "" {
		apiKey = mf.cfg.LLMAPIKey
	}

	if apiKey == "" {
		return nil, fmt.Errorf("claude API key not set - please set CLAUDE_API_KEY or ANTHROPIC_API_KEY")
	}

	modelName := mf.cfg.LLMModel
	if modelName == "" {
		modelName = "claude-3-5-sonnet-latest"
	}

	return adapters.NewGollmAdapter("anthropic", apiKey, modelName)
}

// createOpenAIModel creates an OpenAI model using gollm
func (mf *ModelFactory) createOpenAIModel() (model.LLM, error) {
	apiKey := mf.cfg.OpenAIAPIKey
	if apiKey == "" {
		apiKey = mf.cfg.LLMAPIKey
	}

	if apiKey == "" {
		return nil, fmt.Errorf("openai API key not set - please set OPENAI_API_KEY")
	}

	modelName := mf.cfg.LLMModel
	if modelName == "" {
		modelName = "gpt-4o-mini" // Use mini for cost efficiency
	}

	return adapters.NewGollmAdapter("openai", apiKey, modelName)
}

// createXAIModel creates an xAI Grok model using gollm
func (mf *ModelFactory) createXAIModel() (model.LLM, error) {
	apiKey := mf.cfg.XAIAPIKey
	if apiKey == "" {
		apiKey = mf.cfg.LLMAPIKey
	}

	if apiKey == "" {
		return nil, fmt.Errorf("xAI API key not set - please set XAI_API_KEY")
	}

	modelName := mf.cfg.LLMModel
	if modelName == "" {
		modelName = "grok-3"
	}

	return adapters.NewGollmAdapter("xai", apiKey, modelName)
}

// createOllamaModel creates an Ollama model using gollm
func (mf *ModelFactory) createOllamaModel() (model.LLM, error) {
	modelName := mf.cfg.LLMModel
	if modelName == "" {
		modelName = "llama3.2"
	}

	// Ollama doesn't need an API key for local instances
	// gollm will use the base URL from environment or default to localhost
	return adapters.NewGollmAdapter("ollama", "", modelName)
}

// GetProviderInfo returns information about the current provider
func (mf *ModelFactory) GetProviderInfo() string {
	return fmt.Sprintf("Provider: %s, Model: %s", mf.cfg.LLMProvider, mf.cfg.LLMModel)
}

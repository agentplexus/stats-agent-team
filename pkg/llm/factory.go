package llm

import (
	"context"
	"fmt"

	"google.golang.org/adk/model"
	"google.golang.org/adk/model/gemini"
	"google.golang.org/genai"

	"github.com/grokify/stats-agent-team/pkg/config"
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
	case "gemini":
		return mf.createGeminiModel(ctx)
	case "claude":
		return nil, fmt.Errorf("claude support via ADK is not yet implemented - use gemini for now")
	case "openai":
		return nil, fmt.Errorf("openai support via ADK is not yet implemented - use gemini for now")
	case "ollama":
		return nil, fmt.Errorf("ollama support via ADK is not yet implemented - use gemini for now")
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s (supported: gemini)", mf.cfg.LLMProvider)
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

// GetProviderInfo returns information about the current provider
func (mf *ModelFactory) GetProviderInfo() string {
	return fmt.Sprintf("Provider: %s, Model: %s", mf.cfg.LLMProvider, mf.cfg.LLMModel)
}

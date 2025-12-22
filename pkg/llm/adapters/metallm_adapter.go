package adapters

import (
	"context"
	"fmt"
	"iter"

	"github.com/grokify/metallm"
	"github.com/grokify/metallm/provider"
	"google.golang.org/adk/model"
	"google.golang.org/genai"
)

// MetaLLMAdapterConfig holds configuration for creating a MetaLLM adapter
type MetaLLMAdapterConfig struct {
	ProviderName      string
	APIKey            string
	ModelName         string
	ObservabilityHook metallm.ObservabilityHook
}

// MetaLLMAdapter adapts MetaLLM ChatClient to ADK's LLM interface
type MetaLLMAdapter struct {
	client *metallm.ChatClient
	model  string
}

// NewMetaLLMAdapter creates a new MetaLLM adapter
func NewMetaLLMAdapter(providerName, apiKey, modelName string) (*MetaLLMAdapter, error) {
	return NewMetaLLMAdapterWithConfig(MetaLLMAdapterConfig{
		ProviderName: providerName,
		APIKey:       apiKey,
		ModelName:    modelName,
	})
}

// NewMetaLLMAdapterWithConfig creates a new MetaLLM adapter with full configuration
func NewMetaLLMAdapterWithConfig(cfg MetaLLMAdapterConfig) (*MetaLLMAdapter, error) {
	// For ollama, API key is optional
	if cfg.ProviderName != "ollama" && cfg.APIKey == "" {
		return nil, fmt.Errorf("%s API key is required", cfg.ProviderName)
	}

	// Create MetaLLM config
	config := metallm.ClientConfig{
		Provider:          metallm.ProviderName(cfg.ProviderName),
		APIKey:            cfg.APIKey,
		ObservabilityHook: cfg.ObservabilityHook,
	}

	client, err := metallm.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create MetaLLM client: %w", err)
	}

	return &MetaLLMAdapter{
		client: client,
		model:  cfg.ModelName,
	}, nil
}

// Name returns the model name
func (m *MetaLLMAdapter) Name() string {
	return m.model
}

// GenerateContent implements the LLM interface
func (m *MetaLLMAdapter) GenerateContent(ctx context.Context, req *model.LLMRequest, stream bool) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		// Convert ADK request to MetaLLM request
		messages := make([]provider.Message, 0)

		for _, content := range req.Contents {
			var text string
			for _, part := range content.Parts {
				text += part.Text
			}

			role := provider.RoleUser
			if content.Role == "model" || content.Role == "assistant" {
				role = provider.RoleAssistant
			} else if content.Role == "system" {
				role = provider.RoleSystem
			}

			messages = append(messages, provider.Message{
				Role:    role,
				Content: text,
			})
		}

		// Create MetaLLM request
		metalReq := &provider.ChatCompletionRequest{
			Model:    m.model,
			Messages: messages,
		}

		// Call MetaLLM API
		resp, err := m.client.CreateChatCompletion(ctx, metalReq)
		if err != nil {
			yield(nil, fmt.Errorf("MetaLLM API error: %w", err))
			return
		}

		// Convert MetaLLM response to ADK response
		if len(resp.Choices) > 0 {
			adkResp := &model.LLMResponse{
				Content: &genai.Content{
					Parts: []*genai.Part{
						{Text: resp.Choices[0].Message.Content},
					},
				},
			}
			yield(adkResp, nil)
		}
	}
}

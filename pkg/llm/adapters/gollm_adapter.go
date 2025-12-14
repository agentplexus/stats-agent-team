package adapters

import (
	"context"
	"fmt"
	"iter"

	"github.com/grokify/gollm"
	"github.com/grokify/gollm/provider"
	"google.golang.org/adk/model"
	"google.golang.org/genai"
)

// GollmAdapter adapts gollm ChatClient to ADK's LLM interface
type GollmAdapter struct {
	client *gollm.ChatClient
	model  string
}

// NewGollmAdapter creates a new gollm adapter
func NewGollmAdapter(provider, apiKey, modelName string) (*GollmAdapter, error) {
	// For ollama, API key is optional
	if provider != "ollama" && apiKey == "" {
		return nil, fmt.Errorf("%s API key is required", provider)
	}

	// Create gollm config
	config := gollm.ClientConfig{
		Provider: gollm.ProviderName(provider),
		APIKey:   apiKey,
	}

	client, err := gollm.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create gollm client: %w", err)
	}

	return &GollmAdapter{
		client: client,
		model:  modelName,
	}, nil
}

// Name returns the model name
func (g *GollmAdapter) Name() string {
	return g.model
}

// GenerateContent implements the LLM interface
func (g *GollmAdapter) GenerateContent(ctx context.Context, req *model.LLMRequest, stream bool) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		// Convert ADK request to gollm request
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

		// Create gollm request
		gollmReq := &provider.ChatCompletionRequest{
			Model:    g.model,
			Messages: messages,
		}

		// Call gollm API
		resp, err := g.client.CreateChatCompletion(ctx, gollmReq)
		if err != nil {
			yield(nil, fmt.Errorf("gollm API error: %w", err))
			return
		}

		// Convert gollm response to ADK response
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

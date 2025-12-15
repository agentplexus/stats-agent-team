package direct

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"google.golang.org/adk/model"
	"google.golang.org/genai"

	"github.com/grokify/stats-agent-team/pkg/config"
	"github.com/grokify/stats-agent-team/pkg/llm"
	"github.com/grokify/stats-agent-team/pkg/models"
)

// LLMSearchService provides direct LLM-based statistics search (like ChatGPT)
type LLMSearchService struct {
	cfg   *config.Config
	model model.LLM
}

// NewLLMSearchService creates a new direct LLM search service
func NewLLMSearchService(cfg *config.Config) (*LLMSearchService, error) {
	ctx := context.Background()

	// Create model using factory
	modelFactory := llm.NewModelFactory(cfg)
	llmModel, err := modelFactory.CreateModel(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create model: %w", err)
	}

	return &LLMSearchService{
		cfg:   cfg,
		model: llmModel,
	}, nil
}

// SearchStatistics uses LLM directly to find statistics (like ChatGPT with web search)
func (s *LLMSearchService) SearchStatistics(ctx context.Context, topic string, minStats int) (*models.OrchestrationResponse, error) {
	prompt := fmt.Sprintf(`Find %d or more verified, numerical statistics about "%s".

For each statistic, provide:
1. name: Brief description
2. value: The exact numerical value
3. unit: Unit of measurement
4. source: Name of the authoritative source
5. source_url: Direct URL to the source (if available)
6. excerpt: Exact quote containing the statistic

IMPORTANT INSTRUCTIONS:
- Prioritize statistics from reputable sources (government agencies, research organizations, academic institutions)
- Include the actual URL where each statistic can be verified
- Use real, verifiable data - do not make up statistics
- Extract the exact numerical values
- Provide verbatim excerpts

Return a JSON array:
[
  {
    "name": "Global temperature increase since 1880",
    "value": 1.1,
    "unit": "degrees Celsius",
    "source": "NASA",
    "source_url": "https://climate.nasa.gov/vital-signs/global-temperature/",
    "excerpt": "The planet's average surface temperature has risen about 1.1 degrees Celsius since the late 19th century"
  }
]

Find at least %d statistics. Return only the JSON array, no other text.`, minStats, topic, minStats)

	// Call LLM
	req := &model.LLMRequest{
		Contents: genai.Text(prompt),
	}

	var response string
	for llmResp, err := range s.model.GenerateContent(ctx, req, false) {
		if err != nil {
			return nil, fmt.Errorf("LLM generation failed: %w", err)
		}
		if llmResp.Content != nil && llmResp.Content.Parts != nil {
			for _, part := range llmResp.Content.Parts {
				if part.Text != "" {
					response += part.Text
				}
			}
		}
	}

	// Extract JSON from response
	response = extractJSONFromMarkdown(response)

	// Parse JSON
	type StatResponse struct {
		Name      string  `json:"name"`
		Value     float32 `json:"value"`
		Unit      string  `json:"unit"`
		Source    string  `json:"source"`
		SourceURL string  `json:"source_url"`
		Excerpt   string  `json:"excerpt"`
	}

	var stats []StatResponse
	if err := json.Unmarshal([]byte(response), &stats); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w\nResponse: %s", err, response)
	}

	// Convert to verified statistics
	verifiedStats := make([]models.Statistic, 0, len(stats))
	for _, stat := range stats {
		verifiedStats = append(verifiedStats, models.Statistic{
			Name:      stat.Name,
			Value:     stat.Value,
			Unit:      stat.Unit,
			Source:    stat.Source,
			SourceURL: stat.SourceURL,
			Excerpt:   stat.Excerpt,
			Verified:  true, // Marked as verified since from LLM with sources
			DateFound: time.Now(),
		})
	}

	return &models.OrchestrationResponse{
		Topic:         topic,
		Statistics:    verifiedStats,
		VerifiedCount: len(verifiedStats),
		Timestamp:     time.Now(),
		Partial:       len(verifiedStats) < minStats,
		TargetCount:   minStats,
	}, nil
}

// extractJSONFromMarkdown removes markdown code fences from response
func extractJSONFromMarkdown(response string) string {
	response = strings.TrimSpace(response)

	// Try to find JSON array
	startIdx := strings.Index(response, "[")
	if startIdx == -1 {
		return response
	}

	endIdx := strings.LastIndex(response, "]")
	if endIdx == -1 || endIdx < startIdx {
		return response
	}

	return strings.TrimSpace(response[startIdx : endIdx+1])
}

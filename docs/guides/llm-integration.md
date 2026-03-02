# LLM Integration & Multi-Provider Support

## Overview

The Statistics Agent Team now features **full LLM-based extraction** with support for multiple LLM providers through a unified interface. The refactored codebase eliminates duplication and provides a clean, maintainable architecture.

## Supported LLM Providers

All agents support the following LLM providers:

| Provider | Model | Configuration | Integration |
|----------|-------|---------------|-------------|
| **Gemini** (Default) | `gemini-2.0-flash-exp` | `GEMINI_API_KEY` or `GOOGLE_API_KEY` | Google ADK (native) ✅ |
| **Claude** | `claude-3-5-sonnet-20241022` | `CLAUDE_API_KEY` or `ANTHROPIC_API_KEY` | MetaLLM adapter ✅ |
| **OpenAI** | `gpt-4o-mini` | `OPENAI_API_KEY` | MetaLLM adapter ✅ |
| **Ollama** | `llama3.2` | `OLLAMA_URL` (local) | MetaLLM adapter ✅ |

## Architecture

### Multi-Provider Support via MetaLLM

The system uses **two integration paths**:

1. **Gemini**: Direct via Google ADK (native)
2. **Claude, OpenAI, Ollama**: Via `metallm` adapter (`pkg/llm/adapters/metallm_adapter.go`)

The `metallm` adapter implements the ADK `model.LLM` interface, allowing seamless multi-provider support.

**Design Note:** The `pkg/llm/adapters/` directory is self-contained and can be moved to the `metallm` repository as `pkg/adk/` for broader reuse.

### Shared Base Agent (`pkg/agent/base.go`)

The new `BaseAgent` struct provides common functionality for all LLM-powered agents:

```go
type BaseAgent struct {
    Cfg          *config.Config
    Client       *http.Client
    Model        model.LLM
    ModelFactory *llm.ModelFactory
}
```

**Features:**
- ✅ Unified LLM initialization across all agents
- ✅ Shared HTTP client with configurable timeouts
- ✅ Common URL fetching with size limits
- ✅ Centralized logging helpers
- ✅ Zero code duplication

### Synthesis Agent - LLM-Based Extraction

The Synthesis Agent now uses **intelligent LLM analysis** instead of regex patterns:

**Before (Regex-based):**
```go
// Simple pattern matching for statistics
patterns := []string{
    `(\d+\.?\d*)\s*%`,                    // Percentages
    `(\d+\.?\d*)\s*(million|billion)`,    // Large numbers
}
```

**After (LLM-based):**
```go
// Use LLM to extract statistics with structured prompt
prompt := fmt.Sprintf(`Analyze the following webpage content and extract numerical statistics related to "%s".

For each statistic found, provide:
1. name: A brief descriptive name
2. value: The numerical value (as a number, not string)
3. unit: The unit of measurement
4. excerpt: The verbatim excerpt from the text

Return valid JSON array...`, topic)

response := sa.Model.GenerateContent(ctx, llmReq, false)
```

**Benefits:**
- ✅ Understands context and semantics
- ✅ Extracts complex statistics, not just simple patterns
- ✅ Handles various formats and units intelligently
- ✅ Returns structured JSON output
- ✅ Includes verbatim excerpts for verification

### Verification Agent - Refactored

The Verification Agent now uses the shared base:

**Before:**
```go
type VerificationAgent struct {
    cfg      *config.Config
    client   *http.Client
    adkAgent agent.Agent
}

func (va *VerificationAgent) fetchSourceContent(...) {
    // Custom HTTP fetching code
}
```

**After:**
```go
type VerificationAgent struct {
    *agentbase.BaseAgent
    adkAgent agent.Agent
}

// Use shared method
sourceContent, err := va.FetchURL(ctx, candidate.SourceURL, 1)
```

## Configuration

### Environment Variables

```bash
# LLM Provider Selection
LLM_PROVIDER=gemini  # Options: gemini, claude, openai, ollama

# API Keys (provide based on chosen provider)
GEMINI_API_KEY=your_gemini_key_here
CLAUDE_API_KEY=your_claude_key_here
OPENAI_API_KEY=your_openai_key_here

# For Ollama (local LLM)
OLLAMA_URL=http://localhost:11434
OLLAMA_MODEL=llama3.2

# Optional: Override default models
LLM_MODEL=gemini-2.0-flash-exp  # Or claude-3-5-sonnet-20241022, gpt-4, etc.
```

### Provider-Specific Setup

#### Gemini (Default - Recommended)
```bash
export LLM_PROVIDER=gemini
export GEMINI_API_KEY=your_api_key_here
```
- ✅ Fast and cost-effective
- ✅ `gemini-2.0-flash-exp` optimized for speed
- ✅ Good JSON output reliability

#### Claude
```bash
export LLM_PROVIDER=claude
export CLAUDE_API_KEY=your_api_key_here
```
- ✅ Excellent reasoning capabilities
- ✅ Good for complex extraction tasks

#### OpenAI
```bash
export LLM_PROVIDER=openai
export OPENAI_API_KEY=your_api_key_here
```
- ✅ GPT-4 for highest quality
- ⚠️ Higher cost

#### Ollama (Local)
```bash
export LLM_PROVIDER=ollama
export OLLAMA_URL=http://localhost:11434
export OLLAMA_MODEL=llama3.2
```
- ✅ Free, runs locally
- ✅ No API key required
- ⚠️ Slower, requires GPU

## Code Organization

### Before Refactor
```
agents/synthesis/main.go    - 373 lines (duplicated LLM init, HTTP client, fetching)
agents/verification/main.go - 220 lines (duplicated LLM init, HTTP client, fetching)
```

### After Refactor
```
pkg/agent/base.go           - 95 lines (shared functionality)
agents/synthesis/main.go    - 360 lines (focused on synthesis logic)
agents/verification/main.go - 185 lines (focused on verification logic)
```

**Improvements:**
- 🎯 Single source of truth for LLM initialization
- 🎯 Consistent HTTP client configuration
- 🎯 Shared URL fetching with proper error handling
- 🎯 Easier to add new agents
- 🎯 Easier to maintain and update

## Testing

### Build All Agents
```bash
make build
```

### Run Individual Agents
```bash
# Synthesis Agent (LLM-based extraction)
make run-synthesis

# Verification Agent
make run-verification

# Full workflow
make run-all-eino
```

### Test LLM Extraction

```bash
# Start synthesis agent
PORT=8004 make run-synthesis

# Test extraction
curl -X POST http://localhost:8004/synthesize \
  -H "Content-Type: application/json" \
  -d '{
    "topic": "renewable energy",
    "search_results": [
      {
        "url": "https://www.iea.org/reports/renewables-2023",
        "title": "Renewables 2023",
        "snippet": "Renewable capacity additions reach record levels",
        "domain": "iea.org"
      }
    ],
    "min_statistics": 3,
    "max_statistics": 10
  }'
```

## Performance Considerations

### LLM Provider Speed Comparison

| Provider | Avg Latency | Cost (1M tokens) | Quality |
|----------|-------------|------------------|---------|
| Gemini 2.0 Flash | ~500ms | $0.075 | ⭐⭐⭐⭐ |
| Claude 3.5 Sonnet | ~1.5s | $3.00 | ⭐⭐⭐⭐⭐ |
| GPT-4 | ~2.0s | $10.00 | ⭐⭐⭐⭐⭐ |
| Ollama (local) | ~3-5s | Free | ⭐⭐⭐ |

**Recommendation:** Use Gemini 2.0 Flash for production (best speed/cost/quality balance)

### Token Usage

**Synthesis Agent:**
- Input: ~2000 tokens per webpage (8000 char limit)
- Output: ~200 tokens (JSON array of statistics)
- Total per page: ~2200 tokens

**Cost Example (Gemini):**
- 10 pages analyzed = 22,000 tokens
- Cost: ~$0.00165 per request

## Migration Guide

### For Existing Agents

To add LLM support to a new agent:

```go
import agentbase "github.com/agentplexus/stats-agent-team/pkg/agent"

type MyAgent struct {
    *agentbase.BaseAgent
    // ... other fields
}

func NewMyAgent(cfg *config.Config) (*MyAgent, error) {
    base, err := agentbase.NewBaseAgent(cfg, 30) // 30 second timeout
    if err != nil {
        return nil, err
    }

    return &MyAgent{
        BaseAgent: base,
    }, nil
}

// Use base.Model for LLM calls
// Use base.FetchURL() for HTTP requests
// Use base.LogInfo() for logging
```

## Troubleshooting

### LLM Errors

**"failed to create model"**
- Check API key is set: `echo $GEMINI_API_KEY`
- Verify provider is correct: `echo $LLM_PROVIDER`

**"LLM generation failed"**
- Check API key has sufficient quota
- Verify network connectivity
- Try a different provider

### JSON Parsing Errors

The synthesis agent handles malformed JSON by:
1. Attempting direct JSON parsing
2. Removing markdown code fences (```json)
3. Extracting JSON from LLM response

If still failing, check LLM output format.

## Future Enhancements

- [ ] Support for Anthropic native API (in addition to ADK)
- [ ] Streaming responses for faster synthesis
- [ ] Caching of LLM responses to reduce costs
- [ ] Fine-tuned models for statistics extraction
- [ ] Batch processing for multiple URLs

## References

- [Google ADK Documentation](https://github.com/google/adk-go)
- [Gemini API](https://ai.google.dev/docs)
- [Claude API](https://docs.anthropic.com/)
- [OpenAI API](https://platform.openai.com/docs)
- [Ollama](https://ollama.ai/)

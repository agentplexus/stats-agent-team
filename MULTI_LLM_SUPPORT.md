# Multi-LLM Provider Support Implementation

## Summary

The Statistics Agent Team now supports **4 LLM providers** through a unified interface, allowing you to choose the best model for your use case.

## Supported Providers ✅

| Provider | Status | Default Model | Integration Method |
|----------|--------|---------------|-------------------|
| **Gemini** | ✅ Working | `gemini-2.0-flash-exp` | Google ADK (native) |
| **Claude** | ✅ Working | `claude-3-5-sonnet-20241022` | MetaLLM adapter |
| **OpenAI** | ✅ Working | `gpt-4o-mini` | MetaLLM adapter |
| **Ollama** | ✅ Working | `llama3.2` | MetaLLM adapter |

## Quick Start

### Using OpenAI (Your Current Setup)
```bash
export LLM_PROVIDER=openai
export OPENAI_API_KEY=your_key_here
export SEARCH_PROVIDER=serper
export SERPER_API_KEY=your_key_here

make run-all-eino
```

### Using Claude
```bash
export LLM_PROVIDER=claude
export CLAUDE_API_KEY=your_key_here
# or
export ANTHROPIC_API_KEY=your_key_here

make run-all-eino
```

### Using Gemini (Recommended for cost/speed)
```bash
export LLM_PROVIDER=gemini
export GEMINI_API_KEY=your_key_here
# or
export GOOGLE_API_KEY=your_key_here

make run-all-eino
```

### Using Ollama (Free, Local)
```bash
# Start Ollama first: ollama serve
export LLM_PROVIDER=ollama
export OLLAMA_URL=http://localhost:11434
export LLM_MODEL=llama3.2

make run-all-eino
```

## Implementation Details

### Architecture

The system uses **two integration paths**:

1. **Gemini** → Direct via Google ADK
   - Uses `google.golang.org/adk/model/gemini`
   - Native ADK support, most efficient

2. **Claude, OpenAI, Ollama** → Via MetaLLM adapter
   - Uses `github.com/grokify/metallm` v0.8.0
   - Adapter: `pkg/llm/adapters/metallm_adapter.go`
   - Implements ADK's `model.LLM` interface

### Code Organization

```
pkg/llm/
├── factory.go              # LLM factory with multi-provider support
└── adapters/
    └── metallm_adapter.go  # ADK interface adapter for MetaLLM
                            # (Self-contained, can move to MetaLLM repo)
```

### How It Works

```go
// Factory creates appropriate LLM based on provider
func (mf *ModelFactory) CreateModel(ctx context.Context) (model.LLM, error) {
    switch mf.cfg.LLMProvider {
    case "gemini":
        return mf.createGeminiModel(ctx)  // Native ADK
    case "claude":
        return adapters.NewMetaLLMAdapter("anthropic", apiKey, model)
    case "openai":
        return adapters.NewMetaLLMAdapter("openai", apiKey, model)
    case "ollama":
        return adapters.NewMetaLLMAdapter("ollama", "", model)
    }
}
```

### MetaLLM Adapter

The adapter (`pkg/llm/adapters/metallm_adapter.go`) is **self-contained** and portable:

```go
type MetaLLMAdapter struct {
    client *metallm.ChatClient
    model  string
}

// Implements google.golang.org/adk/model.LLM interface
func (m *MetaLLMAdapter) GenerateContent(ctx context.Context,
    req *model.LLMRequest, stream bool) iter.Seq2[*model.LLMResponse, error]
```

**Design Intent:** This entire `adapters/` directory can be moved to `metallm` as `pkg/adk/` for broader ecosystem use.

## Provider Comparison

### Performance

| Provider | Avg Latency | Cost (1M input tokens) | Best For |
|----------|-------------|------------------------|----------|
| Gemini 2.0 Flash | ~500ms | $0.075 | **Production** (best balance) |
| Claude 3.5 Sonnet | ~1.5s | $3.00 | Complex reasoning |
| GPT-4o-mini | ~1.0s | $0.15 | Good balance |
| Ollama (local) | ~3-5s | Free | Privacy, no API costs |

### Quality for Statistics Extraction

| Provider | JSON Output | Context Understanding | Accuracy |
|----------|-------------|----------------------|----------|
| Claude | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | Excellent |
| GPT-4o-mini | ⭐⭐⭐⭐ | ⭐⭐⭐⭐ | Very Good |
| Gemini Flash | ⭐⭐⭐⭐ | ⭐⭐⭐⭐ | Very Good |
| Ollama (llama3.2) | ⭐⭐⭐ | ⭐⭐⭐ | Good |

## Configuration Options

### Environment Variables

```bash
# Provider selection (required)
LLM_PROVIDER=openai  # Options: gemini, claude, openai, ollama

# API Keys (provider-specific)
GEMINI_API_KEY=sk-...       # For Gemini
CLAUDE_API_KEY=sk-ant-...   # For Claude
OPENAI_API_KEY=sk-...       # For OpenAI

# Optional: Override default models
LLM_MODEL=gpt-4o            # Use GPT-4o instead of mini
LLM_MODEL=claude-3-opus-20240229  # Use Opus instead of Sonnet

# Ollama specific
OLLAMA_URL=http://localhost:11434
LLM_MODEL=llama3.2
```

### Model Recommendations

**For Production (Speed + Cost):**
```bash
LLM_PROVIDER=gemini
LLM_MODEL=gemini-2.0-flash-exp
```

**For Best Quality:**
```bash
LLM_PROVIDER=claude
LLM_MODEL=claude-3-5-sonnet-20241022
```

**For Good Balance:**
```bash
LLM_PROVIDER=openai
LLM_MODEL=gpt-4o-mini
```

**For Privacy/Free:**
```bash
LLM_PROVIDER=ollama
LLM_MODEL=llama3.2
```

## Testing Different Providers

Test each provider with a simple request:

```bash
# Test OpenAI
export LLM_PROVIDER=openai
export OPENAI_API_KEY=your_key
make run-synthesis

# In another terminal
curl -X POST http://localhost:8004/synthesize \
  -H "Content-Type: application/json" \
  -d '{
    "topic": "renewable energy",
    "search_results": [
      {"url": "https://www.iea.org/reports/renewables-2023",
       "title": "Renewables 2023",
       "domain": "iea.org"}
    ],
    "min_statistics": 3
  }'
```

## Migration from OpenAI to Gemini

If you want to switch from OpenAI to Gemini (for cost savings):

```bash
# 1. Get Gemini API key from https://aistudio.google.com/apikey

# 2. Update environment
export LLM_PROVIDER=gemini
export GEMINI_API_KEY=your_gemini_key
# Remove or unset OPENAI_API_KEY

# 3. Restart agents
make run-all-eino
```

**Cost Savings:**
- OpenAI GPT-4o-mini: $0.15/1M input tokens
- Gemini 2.0 Flash: $0.075/1M input tokens
- **50% cost reduction**

## Troubleshooting

### "openai support via ADK is not yet implemented"

**Issue:** You have `LLM_PROVIDER=openai` but the old code is running.

**Fix:** Rebuild the agents:
```bash
make build
make run-all-eino
```

### "API key not set"

**Issue:** Missing API key for selected provider.

**Fix:** Check your environment:
```bash
# For OpenAI
echo $OPENAI_API_KEY

# For Claude
echo $CLAUDE_API_KEY

# For Gemini
echo $GEMINI_API_KEY
```

Set the appropriate key for your provider.

### Ollama Connection Error

**Issue:** Can't connect to Ollama.

**Fix:**
```bash
# Start Ollama
ollama serve

# Pull the model
ollama pull llama3.2

# Set environment
export OLLAMA_URL=http://localhost:11434
export LLM_MODEL=llama3.2
```

## Future Enhancements

### Planned
- [ ] Move `pkg/llm/adapters/` to `metallm` as `pkg/adk/`
- [ ] Add streaming support for faster responses
- [ ] Add response caching to reduce API costs
- [ ] Support for additional metallm providers (AWS Bedrock, Azure, etc.)

### Possible
- [ ] Automatic failover between providers
- [ ] Cost tracking and budgets
- [ ] A/B testing between providers
- [ ] Provider-specific optimizations

## Related Documentation

- **[LLM_INTEGRATION.md](LLM_INTEGRATION.md)** - Complete LLM integration guide
- **[4_AGENT_ARCHITECTURE.md](4_AGENT_ARCHITECTURE.md)** - 4-agent architecture details
- **[MetaLLM repository](https://github.com/grokify/metallm)** - Multi-provider LLM library

## Credits

- **Google ADK**: Native Gemini support
- **MetaLLM**: Multi-provider abstraction layer
- Integration design: Unified adapter pattern for portability

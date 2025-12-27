# Release Notes v0.2.0

**Release Date:** December 14, 2025

This is a major release that introduces the complete 4-agent architecture, Docker deployment, multi-LLM provider support, and real web search integration.

## Highlights

- **4-Agent Architecture** - Complete multi-agent pipeline with Research, Synthesis, Verification, and Orchestration agents
- **Docker Deployment** - Full containerization with docker-compose support
- **Multi-LLM Support** - Unified interface for Gemini, Claude, OpenAI, xAI Grok, and Ollama
- **Real Web Search** - Serper/SerpAPI integration for live search results

## New Features

### Synthesis Agent (`agents/synthesis/`)
New LLM-powered agent for extracting statistics from web content:

- Fetches webpage content from URLs provided by Research agent
- Uses LLM analysis to extract numerical statistics
- Finds verbatim excerpts containing statistics
- Creates structured `CandidateStatistic` objects with metadata
- Runs on port 8004

### 4-Agent Architecture Documentation
Added comprehensive architecture documentation (`4_AGENT_ARCHITECTURE.md`) describing:

- Agent responsibilities and communication patterns
- Research → Synthesis → Verification pipeline
- Orchestration strategies (ADK and Eino)
- Port assignments and API contracts

### Docker Support
Complete containerization for easy deployment:

- `Dockerfile` - Multi-stage build for optimized images
- `docker-compose.yml` - Full stack orchestration
- `docker-entrypoint.sh` - Flexible agent startup script
- `DOCKER.md` - Comprehensive Docker deployment guide
- `.dockerignore` - Optimized build context

### Serper/SerpAPI Integration
Real web search capabilities via the `metasearch` library:

- `pkg/search/service.go` - Unified search service
- Support for Serper and SerpAPI providers
- Configurable result limits and filtering
- `SEARCH_INTEGRATION.md` - Search provider documentation

### Multi-LLM Provider Support
Unified LLM interface supporting multiple providers:

- **Gemini** (Google) - Default provider
- **Claude** (Anthropic) - Via gollm adapter
- **OpenAI** - GPT-4 and GPT-3.5 support
- **xAI Grok** - xAI's Grok models
- **Ollama** - Local LLM deployment

New files:
- `pkg/llm/adapters/gollm_adapter.go` - Multi-provider adapter
- `pkg/llm/factory.go` - Enhanced model factory
- `MULTI_LLM_SUPPORT.md` - Provider configuration guide
- `LLM_INTEGRATION.md` - LLM integration documentation

### Direct LLM Search Mode
Alternative search mode using LLM memory (for quick results without web verification):

- `pkg/direct/llm_search.go` - Direct search implementation
- `--direct` CLI flag for direct mode
- Optional web verification of LLM-generated statistics

### Base Agent Package
New shared agent utilities:

- `pkg/agent/base.go` - Common agent functionality
- Standardized health checks and configuration

## Enhancements

- **Orchestration agent improvements** - Better coordination logic and error handling
- **Research agent refactoring** - Cleaner separation of concerns
- **Eino orchestration updates** - Improved graph-based workflow
- **JSON extraction** - Enhanced `extractJSONFromMarkdown()` function

## Documentation

- `4_AGENT_ARCHITECTURE.md` - Complete architecture documentation
- `DOCKER.md` - Docker deployment guide
- `LLM_INTEGRATION.md` - LLM provider integration
- `MULTI_LLM_SUPPORT.md` - Multi-provider configuration
- `SEARCH_INTEGRATION.md` - Search provider setup
- Updated `README.md` with new features and examples

## Dependencies

- Added `github.com/grokify/gollm` for multi-LLM support
- Added `github.com/grokify/metasearch` for web search
- Various go.mod updates for compatibility

## Bug Fixes

- Fixed search functionality (`ad14475`)
- Fixed LLM support for synthesis and verification agents (`99c9447`)
- Various golangci-lint fixes

## Breaking Changes

- Removed pre-built binaries from `bin/` directory (now built from source)
- Research agent API changes for 4-agent architecture compatibility

## Migration Guide

### From v0.1.0

1. **Update environment variables:**
   ```bash
   # Search provider (required for web search)
   export SERPER_API_KEY="your-key"
   # or
   export SERPAPI_API_KEY="your-key"
   ```

2. **Docker deployment (recommended):**
   ```bash
   docker-compose up -d
   ```

3. **Local development:**
   ```bash
   make install
   make run-all-eino  # Recommended
   # or
   make run-all       # ADK orchestration
   ```

### LLM Provider Configuration

```bash
# Gemini (default)
export GOOGLE_API_KEY="your-key"

# Claude
export LLM_PROVIDER="claude"
export ANTHROPIC_API_KEY="your-key"

# OpenAI
export LLM_PROVIDER="openai"
export OPENAI_API_KEY="your-key"

# xAI Grok
export LLM_PROVIDER="xai"
export XAI_API_KEY="your-key"

# Ollama (local)
export LLM_PROVIDER="ollama"
export OLLAMA_URL="http://localhost:11434"
```

## Contributors

- John Wang (@grokify)

---

**Full Changelog:** https://github.com/grokify/stats-agent-team/compare/v0.1.0...v0.2.0

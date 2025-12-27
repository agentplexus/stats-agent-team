# Installation

## Prerequisites

- Go 1.21 or higher
- LLM API key for your chosen provider:
    - **Gemini** (default): Google API key (set as `GOOGLE_API_KEY` or `GEMINI_API_KEY`)
    - **Claude**: Anthropic API key (set as `ANTHROPIC_API_KEY` or `CLAUDE_API_KEY`)
    - **OpenAI**: OpenAI API key (set as `OPENAI_API_KEY`)
    - **xAI Grok**: xAI API key (set as `XAI_API_KEY`)
    - **Ollama**: Local Ollama installation (default: `http://localhost:11434`)
- Optional: API keys for search provider (Serper, SerpAPI)

## Setup

### 1. Clone the repository

```bash
git clone https://github.com/agentplexus/stats-agent-team.git
cd stats-agent-team
```

### 2. Install dependencies

```bash
make install
# or
go mod download
```

### 3. Configure environment variables

```bash
# For Gemini (default)
export GOOGLE_API_KEY="your-google-api-key"

# For Claude
export LLM_PROVIDER="claude"
export ANTHROPIC_API_KEY="your-anthropic-api-key"

# For OpenAI
export LLM_PROVIDER="openai"
export OPENAI_API_KEY="your-openai-api-key"

# For xAI Grok
export LLM_PROVIDER="xai"
export XAI_API_KEY="your-xai-api-key"

# For Ollama (local)
export LLM_PROVIDER="ollama"
export OLLAMA_URL="http://localhost:11434"
export LLM_MODEL="llama3:latest"

# Optional: Create .env file
cp .env.example .env
# Edit .env with your API keys
```

### 4. Build the agents

```bash
make build
```

## Verification

After installation, verify everything is working:

```bash
# Check build artifacts
ls -la bin/

# Run a quick test (requires API keys configured)
make run-all-eino &
sleep 5
curl http://localhost:8000/health
```

## Next Steps

- [Quick Start](quickstart.md) - Get started quickly with Docker or local development
- [Configuration](configuration.md) - Detailed configuration options

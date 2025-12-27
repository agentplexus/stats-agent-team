# Release Notes v0.1.0

**Release Date:** December 13, 2025

This is the initial release of Statistics Agent Team, a multi-agent system for finding and verifying statistics from reputable web sources.

## Overview

Statistics Agent Team is built with [Google ADK (Agent Development Kit)](https://github.com/google/adk-go) and [Eino](https://github.com/cloudwego/eino), implementing a sophisticated multi-agent architecture that leverages LLMs and web search to find verifiable statistics from well-known and respected publishers.

## Features

### 3-Agent Architecture

The system implements three specialized agents:

#### Research Agent (`agents/research/`)
- Built with Google ADK and Gemini 2.0 Flash model
- Executes web searches for statistics on given topics
- Prioritizes reputable sources (academic, government, research organizations)
- Extracts candidate statistics with context
- Returns structured data for verification
- Port: 8001

#### Verification Agent (`agents/verification/`)
- Built with Google ADK and Gemini 2.0 Flash model
- Fetches actual source content from URLs
- Searches for verbatim excerpts in source
- Validates numerical values match exactly
- Flags hallucinations and discrepancies
- Returns verification results with reasons
- Port: 8002

#### Orchestration Agents (Two Options)

**ADK Orchestration** (`agents/orchestration/`)
- Built with Google ADK and Gemini 2.0 Flash model
- LLM-based decision making for workflow coordination
- Implements adaptive retry logic
- Dynamic quality control
- Port: 8000

**Eino Orchestration** (`agents/orchestration-eino/`) - Recommended
- Deterministic graph-based workflow
- Type-safe orchestration with compile-time checks
- Predictable, reproducible behavior
- Faster and lower cost (no LLM for orchestration)
- Port: 8003

### MCP Server Integration
- Full MCP (Model Context Protocol) server implementation
- Integration with Claude Code and other MCP clients
- Exposes Eino orchestration via MCP tools
- `mcp/server/main.go` - Server implementation
- `MCP_SERVER.md` - Integration documentation

### Multi-LLM Provider Support
Configurable LLM providers with unified interface:
- **Gemini** (default) - Google's Gemini 2.0 Flash
- **Claude** - Anthropic Claude models
- **OpenAI** - GPT-4 and GPT-3.5
- **Ollama** - Local LLM deployment

Configuration via environment variables documented in `LLM_CONFIGURATION.md`.

### Structured Output

Statistics returned in JSON format with complete metadata:

```json
{
  "name": "Global temperature increase since pre-industrial times",
  "value": 1.1,
  "unit": "°C",
  "source": "IPCC Sixth Assessment Report",
  "source_url": "https://www.ipcc.ch/...",
  "excerpt": "Global surface temperature has increased by approximately 1.1°C...",
  "verified": true,
  "date_found": "2025-12-13T10:30:00Z"
}
```

### Core Capabilities

- Multi-agent orchestration with workflow coordination
- Google ADK integration for LLM-based agents
- Eino framework for deterministic graph orchestration
- Source verification to prevent hallucinations
- Reputable source prioritization
- HTTP APIs for all agents
- Retry logic for ensuring quality results
- Function tools for structured agent capabilities

## Project Structure

```
stats-agent-team/
├── agents/
│   ├── orchestration/       # ADK-based orchestration agent
│   ├── orchestration-eino/  # Eino-based orchestration agent
│   ├── research/            # Research agent
│   └── verification/        # Verification agent
├── mcp/
│   └── server/              # MCP server implementation
├── pkg/
│   ├── config/              # Configuration management
│   ├── httpclient/          # HTTP client utilities
│   ├── llm/                 # LLM factory and adapters
│   ├── models/              # Data models
│   └── orchestration/       # Eino orchestration logic
├── main.go                  # CLI entry point
└── Makefile                 # Build and run targets
```

## Documentation

- `README.md` - Project overview and quick start
- `README_EINO.md` - Eino orchestrator details
- `LLM_CONFIGURATION.md` - LLM provider setup
- `MCP_SERVER.md` - MCP integration guide

## Requirements

- Go 1.21 or higher
- LLM API key (Gemini, Claude, OpenAI, or Ollama)

## Quick Start

```bash
# Install dependencies
make install

# Set API key
export GOOGLE_API_KEY="your-api-key"

# Run with Eino orchestration (recommended)
make run-eino

# Or run all agents
make run-all
```

## CI/CD

- GitHub Actions workflows for build and lint
- Dependabot configuration for dependency updates
- golangci-lint integration for code quality

## Contributors

- John Wang (@grokify)

---

**Repository:** https://github.com/grokify/stats-agent-team

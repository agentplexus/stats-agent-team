# Quick Start

You can run the system either with Docker (containerized) or locally. Choose the method that best fits your needs.

| Method | Best For | Command |
|--------|----------|---------|
| **Docker** | Production, quick start, isolated environment | `docker-compose up -d` |
| **Local** | Development, debugging, customization | `make run-all-eino` |

## Quick Start with Docker

The fastest way to get started:

```bash
# Start all agents with Docker Compose
docker-compose up -d

# Test the orchestration endpoint
curl -X POST http://localhost:8000/orchestrate \
  -H "Content-Type: application/json" \
  -d '{"topic": "climate change", "min_verified_stats": 5}'

# View logs
docker-compose logs -f

# Stop
docker-compose down
```

See [Docker Deployment](../guides/docker.md) for complete Docker deployment guide.

## Local Development Setup

### Running the Agents Locally

#### Option 1: Run all agents with Eino orchestrator (Recommended)

```bash
make run-all-eino
```

#### Option 2: Run all agents with ADK orchestrator

```bash
make run-all
```

#### Option 3: Run each agent separately (in different terminals)

```bash
# Terminal 1: Research Agent (ADK)
make run-research

# Terminal 2: Verification Agent (ADK)
make run-verification

# Terminal 3: Orchestration Agent (choose one)
make run-orchestration       # ADK version (LLM-based)
make run-orchestration-eino  # Eino version (deterministic, recommended)
```

## Using the CLI

The CLI supports three modes: **Direct LLM search** (fast, like ChatGPT), **Direct + Verification** (hybrid), and **Multi-agent verification pipeline** (thorough, verified).

### Multi-Agent Pipeline Mode (Recommended)

For verified, web-scraped statistics (requires agents running):

```bash
# Start agents first
make run-all-eino

# Then in another terminal:
# Basic search with verification pipeline
./bin/stats-agent search "climate change"

# Request specific number of verified statistics
./bin/stats-agent search "global warming" --min-stats 15

# Increase candidate search space
./bin/stats-agent search "AI trends" --min-stats 10 --max-candidates 100

# Only reputable sources
./bin/stats-agent search "COVID-19 statistics" --reputable-only

# JSON output only
./bin/stats-agent search "renewable energy" --output json

# Text output only
./bin/stats-agent search "climate data" --output text
```

**Advantages of Multi-Agent Mode:**

- **Verified sources** - Actually fetches and checks web pages
- **Web search** - Finds current statistics from the web
- **Accuracy** - Validates excerpts and values match
- **Human-in-the-loop** - Prompts to continue if target not met

### Direct Mode (Not Recommended for Statistics)

Direct mode uses a single LLM call to find statistics from memory - similar to ChatGPT without web search:

```bash
# Start direct agent first
make run-direct

# Then query (fast but uses LLM memory)
./bin/stats-agent search "climate change" --direct
```

!!! warning "Why Not Recommended for Statistics"
    - **Uses LLM memory** - Not real-time web search (training data up to Jan 2025)
    - **Outdated URLs** - LLM guesses URLs where stats came from
    - **Low accuracy** - Pages may have moved, changed, or be paywalled
    - **0% verification rate** - When combined with `--direct-verify`, most claims fail

**When to Use:**

- General knowledge questions
- Concept explanations
- Quick brainstorming (accept unverified data)

### CLI Options

```bash
stats-agent search <topic> [options]

Options:
  -d, --direct              Use direct LLM search (fast, like ChatGPT)
      --direct-verify       Verify LLM claims with verification agent (requires --direct)
  -m, --min-stats <n>       Minimum statistics to find (default: 10)
  -c, --max-candidates <n>  Max candidates for pipeline mode (default: 50)
  -r, --reputable-only      Only use reputable sources
  -o, --output <format>     Output format: json, text, both (default: both)
      --orchestrator-url    Override orchestrator URL
  -v, --verbose             Show verbose debug information
      --version             Show version information
```

### Mode Comparison

| Mode | Speed | Accuracy | Agents Needed | Client Needs API Key? | Best For |
|------|-------|----------|---------------|----------------------|----------|
| `--direct` | Fastest | LLM-claimed | Direct agent only | No | Quick research, brainstorming |
| `--direct --direct-verify` | Fast | Web-verified | Direct + Verification | No | Balanced speed + accuracy |
| Pipeline (default) | Slower | Fully verified | All 4 agents | No | Maximum reliability |

## API Usage

You can also call the agents directly via HTTP (works with both Docker and local deployment):

```bash
# Call orchestration agent (port 8000 - supports both ADK and Eino)
curl -X POST http://localhost:8000/orchestrate \
  -H "Content-Type: application/json" \
  -d '{
    "topic": "climate change",
    "min_verified_stats": 10,
    "max_candidates": 30,
    "reputable_only": true
  }'
```

## Using with Claude Code (MCP Server)

The system can be used as an MCP server with Claude Code and other MCP clients:

```bash
# Build the MCP server
make build-mcp

# Configure in Claude Code's MCP settings (see MCP Server guide)
```

See [MCP Server Integration](../guides/mcp-server.md) for detailed setup instructions.

# Architecture Overview

The system implements a **4-agent architecture** with clear separation of concerns.

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────┐
│                   User Request                          │
│              "Find climate change statistics"           │
└───────────────────┬─────────────────────────────────────┘
                    │
                    ▼
┌─────────────────────────────────────────────────────────┐
│            ORCHESTRATION AGENT                          │
│              (Port 8000 - Both ADK/Eino)                │
│  • Coordinates 4-agent workflow                         │
│  • Manages retry logic                                  │
│  • Ensures quality standards                            │
└─────┬─────────────┬────────────────┬────────────────────┘
      │             │                │
      ▼             ▼                ▼
┌────────────┐ ┌──────────┐ ┌─────────────────┐
│  RESEARCH  │ │SYNTHESIS │ │  VERIFICATION   │
│   AGENT    │ │  AGENT   │ │     AGENT       │
│ Port 8001  │ │Port 8004 │ │   Port 8002     │
│            │ │          │ │                 │
│ • Search   │─│• Fetch   │─│• Re-fetch URLs  │
│   Serper   │ │  URLs    │ │• Validate text  │
│ • Filter   │ │• LLM     │ │• Check numbers  │
│   Sources  │ │  Extract │ │• Flag errors    │
└────────────┘ └──────────┘ └─────────────────┘
      │             │                │
      ▼             ▼                ▼
  URLs only     Statistics     Verified Stats
```

## Agent Responsibilities

### 1. Research Agent (`agents/research/`) - Web Search Only

- **No LLM required** - Pure search functionality
- Web search via Serper/SerpAPI integration
- Returns URLs with metadata (title, snippet, domain)
- Prioritizes reputable sources (`.gov`, `.edu`, research orgs)
- Output: List of `SearchResult` objects
- Port: **8001**

### 2. Synthesis Agent (`agents/synthesis/`) - Google ADK

- **LLM-heavy** extraction agent
- Built with Google ADK and LLM (Gemini/Claude/OpenAI/Ollama)
- Fetches webpage content from URLs
- Extracts numerical statistics using LLM analysis
- Finds verbatim excerpts containing statistics
- Creates `CandidateStatistic` objects with proper metadata
- Port: **8004**

### 3. Verification Agent (`agents/verification/`) - Google ADK

- **LLM-light** validation agent
- Re-fetches source URLs to verify content
- Checks excerpts exist verbatim in source
- Validates numerical values match exactly
- Flags hallucinations and discrepancies
- Returns verification results with pass/fail reasons
- Port: **8002**

### 4a. Orchestration Agent - Google ADK (`agents/orchestration/`)

- Built with Google ADK for LLM-driven workflow decisions
- Coordinates: Research → Synthesis → Verification
- Implements adaptive retry logic
- Dynamic quality control
- Port: **8000**

### 4b. Orchestration Agent - Eino (`agents/orchestration-eino/`) - RECOMMENDED

- **Deterministic graph-based workflow** (no LLM for orchestration)
- Type-safe orchestration with Eino framework
- Predictable, reproducible behavior
- Faster and lower cost
- Workflow: ValidateInput → Research → Synthesis → Verification → QualityCheck → Format
- Port: **8000** (same port as ADK, but they don't run simultaneously)
- **Recommended for production use**

## Reputable Sources

The research agent prioritizes these source types:

- **Government Agencies**: CDC, NIH, Census Bureau, EPA, etc.
- **Academic Institutions**: Universities, research journals
- **Research Organizations**: Pew Research, Gallup, McKinsey, etc.
- **International Organizations**: WHO, UN, World Bank, IMF, etc.
- **Respected Media**: With proper citations (NYT, WSJ, Economist, etc.)

## Error Handling

- **Source Unreachable**: Marked as failed with reason
- **Excerpt Not Found**: Verification fails with explanation
- **Value Mismatch**: Flagged as discrepancy
- **Insufficient Results**: Automatic retry with more candidates
- **Max Retries Exceeded**: Returns partial results with warning

## Learn More

- [4-Agent Architecture](4-agent-architecture.md) - Detailed agent implementation
- [Eino Orchestration](eino-orchestration.md) - Deterministic graph-based orchestration

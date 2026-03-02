# Roadmap

This document outlines the planned features and enhancements for the Statistics Agent Team project.

## Orchestrator Comparison (ADK vs Eino)

The project supports two orchestration approaches for comparison:

| Orchestrator | Protocol | Routing Style | Status |
|--------------|----------|---------------|--------|
| **ADK** | A2A (:900x) | LLM-driven (Hybrid) | ✅ Implemented |
| **Eino** | HTTP (:800x) | Code-driven (Graph) | ✅ Implemented |

### Planned Comparison Metrics

- 📊 **Response time** - ADK vs Eino orchestrator latency
- 📊 **Cost per query** - LLM calls for orchestration decisions
- 📊 **Verification rate** - Does routing strategy affect accuracy?
- 📊 **Predictability** - Variance in execution paths and timing
- 📊 **Error recovery** - How each handles failures

### Why Both?

- **ADK (Hybrid)**: Code-defined agent structure, LLM-driven routing/delegation
- **Eino (Code-driven)**: Deterministic graph, predictable execution, lower cost

Current recommendation: **Eino** for production (faster, cheaper, reproducible)

### Refined A2A Strategy: External Agent Services

**Key insight:** A2A is most valuable for **external interoperability**, not internal communication.

#### Agent Reusability Analysis

| Agent | Capability | External Value |
|-------|------------|----------------|
| **Verification** | Validate excerpt exists in URL | ✅ **High** - Universal problem |
| **Research** | Search web for topic | ✅ Medium - Generic capability |
| **Synthesis** | Extract statistics from pages | ⚠️ Low - Specific to statistics |
| **Orchestrator** | Coordinate pipeline | ❌ None - Internal only |

#### Recommended Architecture

```
┌─────────────────────────────────────────────────────────────┐
│  stats-agent-team (internal)                                │
│                                                             │
│  ┌─────────────────┐                                        │
│  │ Eino Orchestrator│──HTTP──→ Research, Synthesis          │
│  └────────┬────────┘          (internal, simple)            │
│           │                                                 │
│           │ HTTP or A2A                                     │
│           ▼                                                 │
│  ┌─────────────────┐                                        │
│  │ Verification    │←──A2A───── External A2A Clients        │
│  │ Agent           │           (other agent systems)        │
│  └─────────────────┘                                        │
└─────────────────────────────────────────────────────────────┘
```

#### Verification-as-a-Service

The Verification Agent solves a universal problem: **LLMs hallucinate URLs and citations**.

Any agent system that generates sourced content needs verification:
- Research assistants
- Content generators
- Fact-checking tools
- RAG systems

**Proposed A2A Agent Card:**

```yaml
name: "Verification Agent"
description: "Verify that excerpts and statistics exist in source URLs"
skills:
  - name: "verify_excerpt"
    description: "Check if text excerpt exists in URL"
    input: { url: string, excerpt: string }
    output: { verified: boolean, reason: string }
  - name: "verify_statistic"
    description: "Verify numerical statistic with context"
    input: { url: string, value: number, excerpt: string }
    output: { verified: boolean, value_match: boolean, excerpt_match: boolean }
```

#### Protocol Strategy

| Component | Protocol | Rationale |
|-----------|----------|-----------|
| **Eino → Internal agents** | HTTP | Simple, no overhead |
| **Verification Agent** | HTTP + A2A | A2A for external clients |
| **Research Agent** | HTTP + A2A | Optional external exposure |

**Don't add A2A client to Eino** unless there's a concrete need. HTTP is simpler for internal communication.

## Q1 2026

- ✨ **Perplexity API integration** - Built-in search without separate provider
- ✨ **Range statistics support** - Add `value_max` field for ranges like "79-96%"
- ✨ **Response streaming** - Faster perceived performance with streaming results
- 📊 **Orchestrator benchmarks** - Publish ADK vs Eino comparison results
- 🌐 **Verification-as-a-Service** - Document and promote Verification Agent as external A2A service

## Q2 2026

- ✨ **Multi-language support** - Spanish, French, German, Chinese sources
- ✨ **Caching layer** - Reduce redundant searches and API costs
- ✨ **GraphQL API** - Alternative query interface
- 🌐 **Research Agent external** - Expose Research Agent via A2A if demand exists

## Q3 2026

- ✨ **Browser extension** - Real-time fact-checking while browsing
- ✨ **Notion/Confluence integrations** - Embed verified statistics in docs
- ✨ **Advanced citation formats** - APA, MLA, Chicago styles

## Future Considerations

- 🔮 Academic database integration (PubMed, arXiv, JSTOR)
- 🔮 Paywall-aware fetching with institutional credentials
- 🔮 Historical statistics tracking and trend analysis
- 🔮 Confidence scoring based on source reputation

## Contributing

This roadmap is community-driven. Submit feature requests on [GitHub Issues](https://github.com/agentplexus/stats-agent-team/issues)!

To propose a new feature:
1. Check existing issues for duplicates
2. Open a new issue with the `enhancement` label
3. Describe the use case and proposed solution
4. Community feedback helps prioritize features

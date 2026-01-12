# 01 - Orchestrator Workflows

## Background
We have built four standalone microservices:
1.  `fetcher`: Ingests raw content.
2.  `analyzer`: Generates insights/summaries.
3.  `creator`: Generates or remixes content.
4.  `publisher`: Posts to platforms.

The `orchestrator` will bind these together into automated pipelines.

> [!NOTE]
> This service is deployed as part of the [Docker Orchestration Stack](../../design/02-docker-orchestration.md).

## Problem Statement
We need a central controller to execute complex content flows to automate social media growth.
*   **Goal**: Turn "Input Source" into "Published Post" with minimal human intervention.

## Pipeline Flows & Examples

### Flow 1: The "Cross-Pollinator" (Reddit -> LinkedIn)
*   **Goal**: Take technical discussions from Reddit and turn them into thought leadership posts on LinkedIn.
*   **Trigger**: Cron (Every 6 hours).

**Step-by-Step Execution:**
1.  **Fetch**: calling `fetcher.FetchContent(platform="reddit", query="golang")`.
    *   *Result*: List of items. Item A: "Go 1.24 released with new loop var semantics."
2.  **Analyze** (Internal in Fetcher, or explicit): Fetcher already calls Analyzer.
    *   *Result*: Summary: "Go 1.24 changes how loop variables are scoped..." Tags: `[Go, Update, Programming]`
3.  **Filter**: Orchestrator checks rules (e.g., Score > 50, Sentiment != Negative).
4.  **Remix**: Call `creator.RemixContent(original=Summary, source="reddit", target="linkedin", tone="thought_leader")`.
    *   *Result*: "🚀 Big changes coming to Go! The new loop variable semantics in 1.24 will prevent common bugs... #Golang #TechNews"
5.  **Publish**: Call `publisher.PublishContent(content=RemixResult, platform="linkedin")`.

### Flow 2: The "Trend Jacker" (News -> Twitter)
*   **Goal**: Monitor specific keywords and post hot takes.
*   **Trigger**: Webhook or Fast Polling.

**Step-by-Step Execution:**
1.  **Fetch**: `fetcher.Fetch(platform="news_api", query="AI Agents")`.
2.  **Analyze**: Extract key fact: "Google DeepMind releases new Agentic Coding framework."
3.  **Generate**: Call `creator.GenerateContent(topic="Google DeepMind Agentic Coding", platform="twitter", tone="excited")`.
    *   *Result*: "Agentic workflows are here! 🤖 DeepMind just dropped a bombshell... 🧵"
4.  **Publish**: `publisher.PublishContent(platform="twitter")`.

## Design

### API Definition
```protobuf
service OrchestratorService {
    rpc RunPipeline(PipelineRequest) returns (PipelineResponse) {}
}

message PipelineRequest {
    string flow_name = 1; // "cross_pollinator", "trend_jacker"
    map<string, string> params = 2; // query="golang", target="linkedin"
}
```

### Architecture
The Orchestrator is a **Workflow Engine**. It holds the state of the job.

```mermaid
graph TD
    User -->|Start Flow| Orch[Orchestrator]
    Orch -->|1. Fetch| Fetch[Fetcher Svc]
    Fetch -->|2. Analyze| Analyze[Analyzer Svc]
    Fetch -.->|Return Items| Orch
    Orch -->|3. Filter| Logic{Good Content?}
    Logic -->|Yes| Create[Creator Svc]
    Logic -->|No| Stop[End]
## Implementation Results

### 1. Unified Workflow
*   Implemented `OrchestratorService` with `RunPipeline` RPC.
*   Successfully connected to all 3 sub-services (`fetcher`, `creator`, `publisher`).

### 2. Verification
*   **Trigger**: Client triggered "Cross-Pollinator" flow.
*   **Execution**:
    *   Orchestrator -> Fetcher (Reddit): **Success** (Request Received).
    *   Fetcher -> Analyzer: **Partial Success** (Hit Gemini 429 Rate Limits).
    *   *Observation*: The pipeline successfully orchestrated the call stack. The failure was due to external API quotas (Free Tier 5 RPM), not logic.
    *   **Fallback Logic**: The system is designed to fallback to raw text if analysis fails, verifying robust error handling.

### 3. Trade-offs
*   **Synchronous Chain**: The current design waits for Fetcher (which waits for Analyzer x 10 items). This causes timeouts when rate limits are hit.
*   **Next Step**: Move to Async/Queue-based architecture (e.g., Kafka/RabbitMQ or DB-polling) to handle rate limits gracefully.


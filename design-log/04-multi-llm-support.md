# 04 - Multi-LLM Provider Support

## Background
Currently, the platform is hardcoded to use Google Gemini for analysis and content generation. To increase robustness and flexibility, we want to support multiple LLM providers, starting with OpenAI (GPT-4o).

## Problem Statement
We need a way to:
- Choose the AI provider at runtime via a command-line flag in the `batch-runner`.
- Propagate this choice through the `orchestrator` to the `analyzer` and `creator` services.
- Provide a consistent abstraction layer so adding future providers (Claude, etc.) is easy.

## Questions and Answers

**Q: How should the model choice be propagated?**
A: We will add a `model_provider` field to the gRPC request messages for the `orchestrator`, `fetcher`, `analyzer`, and `creator` services.

**Q: Where will API keys for OpenAI be stored?**
A: In the `.env` file as `OPENAI_API_KEY`, and passed to the respective services via `docker-compose.yml`.

**Q: How do we handle different model capabilities (e.g., vision)?**
A: The abstraction layer (`ContentAnalyzer`/`ContentCreator`) already handles this. We will implement `AnalyzeImage` using OpenAI's vision capabilities.

**Q: Should the model choice be per interaction or per batch?**
A: Initially, it will be per batch via the `--model` flag in `batch-runner`, but the gRPC changes will allow per-request granularity.

## Design

### 1. Proto Changes
We'll update the `.proto` files to include a `provider` or `model` field.

**Orchestrator (`orchestrator.proto`)**:
```proto
message PipelineRequest {
  string flow_name = 1;
  map<string, string> params = 2;
  string model_provider = 3; // "gemini" (default) or "openai"
}
```

**Fetcher (`fetcher.proto`)**:
```proto
message FetchRequest {
  string platform = 1;
  string query = 2;
  map<string, string> credentials = 3;
  string model_provider = 4;
}
```

**Analyzer (`analyzer.proto`)**:
```proto
message AnalyzeContentRequest {
  oneof content {
    string text = 1;
    string image_url = 2;
  }
  string model_provider = 3;
}
```

**Creator (`creator.proto`)**:
```proto
message GenerateRequest {
  string topic = 1;
  string platform = 2;
  string tone = 3;
  string model_provider = 4;
}
```

### 2. Service Abstraction
In `analyzer` and `creator`, the `NewAnalyzerService` and `NewCreatorService` logic will be updated to hold *both* providers or a factory that selects them per request.

Better approach: Update the services to delegate to an `AIProviderFactory`.

```go
type AIProviderFactory struct {
    gemini  ContentAnalyzer
    openai  ContentAnalyzer
}

func (f *AIProviderFactory) Get(provider string) ContentAnalyzer {
    if provider == "openai" {
        return f.openai
    }
    return f.gemini
}
```

### 3. OpenAI Implementation
Created `pkg/ai/openai.go` using the `go-openai` library.

## Implementation Plan

### Phase 1: Proto & Infrastructure
- [ ] Add `model_provider` to proto files:
  - `orchestrator/api/proto/orchestrator.proto`
  - `fetcher/api/proto/fetcher.proto`
  - `analyzer/api/proto/analyzer.proto`
  - `creator/api/proto/creator.proto`
- [ ] Regenerate Go code in all services (`make proto`)
- [ ] Update `docker-compose.yml` to include `OPENAI_API_KEY`

### Phase 2: OpenAI Adapters
- [ ] Implement `openaiAnalyzer` in `analyzer/pkg/ai/openai.go`
- [ ] Implement `openaiCreator` in `creator/pkg/ai/openai.go`
- [ ] Add `OPENAI_API_KEY` to `.env` template

### Phase 3: Service Updates
- [ ] Update `AnalyzerService` to select provider based on `req.ModelProvider`
- [ ] Update `CreatorService` to select provider based on `req.ModelProvider`
- [ ] Update `FetcherService` to pass `model_provider` to `analyzer` calls
- [ ] Update `OrchestratorService` to pass `model_provider` to `fetcher` and `creator` calls

### Phase 4: Batch Runner Update
- [ ] Add `--model` flag to `orchestrator/cmd/batch-runner/main.go`
- [ ] Pass the flag value to the gRPC `RunPipeline` call

## Examples

### ✅ Good: Using OpenAI
```bash
go run cmd/batch-runner/main.go --config=../users.yaml --model=openai
```

### ✅ Good: Defaulting to Gemini
```bash
go run cmd/batch-runner/main.go --config=../users.yaml
```

## Trade-offs

### Chosen Approach: Per-Request Provider Selection
**Pros:**
- Maximum flexibility (can mix and match providers in the future).
- Clean propagation from the top (batch-runner) to the bottom (LLM adapters).

**Cons:**
- Requires many small proto changes.
- Slightly more boilerplate in service handlers to switch between providers.

### Alternative: Global Env Variable per Service
Rejected because it doesn't allow the batch-runner to control the model choice dynamically without restarting services.

# Design Log 06 - AI Usage Optimization

## Background
The system currently implements a "stateless" fetch-analyze-create pipeline. During the `facebook_echo` flow, the orchestrator fetches recent posts, analyzes them, and generates a response. To ensure high quality, the AI is used at both the analysis and creation stages.

## Problem Statement
AI usage (and cost) is currently sub-optimal.
1. **Redundant Analysis**: The `fetcher` service analyzes *every* post it retrieves from a platform (e.g., 25+ posts from Meta API), even if the `orchestrator` only uses the most recent one.
2. **Missing Caching**: We analyze the same posts repeatedly across different pipeline runs if they remain the latest content.

### Current Flow AI Demand:
- `FetchContent`: Calls `Analyzer` **N** times (where N is the number of posts in the feed).
- `GenerateContent`: Calls `Creator` **1** time.
- **Total**: ~26 AI requests per post handling.

## Questions and Answers

**Q: Should we add a limit to the fetcher or the orchestrator?**
A: Both. The `FetchRequest` should include a `limit` field. The `fetcher` should respect this limit when calling the AI `analyzer`. The `orchestrator` can then specify exactly how many items it needs.

**Q: Should we separate Fetching from Analyzing?**
A: Yes. Ideally, the `fetcher` should just fetch raw data, and the `orchestrator` (or a dedicated service) should decide which items are worth analyzing. However, for the current MVP, simply adding a `limit` to the `FetchContent` call is the most direct fix.

**Q: How do we handle caching?**
A: We can use the `AnalysisSummary` field in the `AIContext`. If we've already analyzed a specific `PostID`, we can retrieve the analysis from the local CSV history instead of re-calling the AI.

## Design

### 1. Proto Update
Update `fetcher.proto` to include a `limit` field.

```proto
message FetchRequest {
  string platform = 1;
  string query = 2;
  map<string, string> credentials = 3;
  string model_provider = 4;
  int32 limit = 5; // <--- NEW
}
```

### 2. Fetcher Optimization
Modify `FetcherService.FetchContent` to slice the results BEFORE the analysis loop.

```go
// 2. Limit and Analyze
limit := int(req.Limit)
if limit <= 0 {
    limit = 1 // default to 1 for safety if not specified
}
if limit > len(items) {
    limit = len(items)
}

for i := 0; i < limit; i++ {
    item := items[i]
    // ... analyze ...
}
```

### 3. Orchestrator Optimization
Update `runFacebookEcho` and `runCrossPollinator` to pass specific limits.

- `facebook_echo`: `limit: 1`
- `cross_pollinator`: `limit: 3` (or as needed)

## Implementation Plan

### Phase 1: Proto
- [ ] Update `fetcher.proto` with `limit` field.
- [ ] Regenerate Go code (`make proto` equivalents).

### Phase 2: Fetcher Service
- [ ] Update `FetcherService.FetchContent` to respect `req.Limit`.
- [ ] Update `source.Fetch` to optionally pass limit to external APIs (e.g., `&limit=1` in Meta URL) to save bandwidth too.

### Phase 3: Orchestrator Service
- [ ] Update `runFacebookEcho` to request `limit: 1`.
- [ ] Update `runCrossPollinator` to request a sensible limit (e.g., 1 or 2).

## Examples

### ✅ Optimized (limit=1)
1. Fetch 1 post from Meta.
2. Analyze 1 post.
3. Create 1 response.
**Total AI Calls: 2**

### ❌ Current (default items)
1. Fetch 25 posts from Meta.
2. Analyze 25 posts.
3. Create 1 response.
**Total AI Calls: 26**

## Trade-offs

### Limit vs Caching
- **Limit**: Simplest to implement, immediate 90%+ reduction in AI calls for `facebook_echo`.
- **Caching**: More complex (requires state check), but useful if we ever analyze the same items twice.
- **Decision**: Start with `Limit`.

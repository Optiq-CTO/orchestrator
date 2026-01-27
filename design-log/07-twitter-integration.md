# Design Log 07 - X (Twitter) Integration

## Background
The platform currently supports Meta (Facebook) and Reddit. To expand our reach and monetization capabilities, we need to integrate with X (formerly Twitter). This includes both fetching content for analysis and publishing new content.

## Problem Statement
We need to add X as a first-class platform in our microservices architecture.
- **Fetcher**: Needs to retrieve tweets from a specific user or based on a query.
- **Publisher**: Needs to post text and media to X.
- **Orchestrator**: Needs to support X in pipelines (e.g., `twitter_echo` or as a destination in `cross_pollinator`).

## Questions and Answers

**Q: Which version of the X API should we use?**
A: We will use **X API v2**, as it is the current standard. We'll need to support OAuth 1.0a User Context for publishing and Bearer Token for some fetching operations.

**Q: What are the primary use cases for X in the first iteration?**
A: 
1. `twitter_echo`: Similar to `facebook_echo`, it listens to your own feed and responds/amplifies.
2. `cross_pollinator` target: Cross-posting high-performing content from Reddit or Facebook to X.

**Q: How do we handle X's strict rate limits and multi-part credentials?**
A: X requires: `API Key`, `API Secret`, `Access Token`, and `Access Token Secret`. We will update `Identity` and `orchestrator`'s `users.yaml` to store these securely.

**Q: Should we use a third-party library for X?**
A: For Go, `github.com/dghubble/go-twitter` is popular but mostly v1.1. We might prefer `github.com/g8rswimmer/go-twitter/v2` or direct HTTP calls for maximum control over v2 features.

## Design

### 1. Fetcher Service
**File**: `fetcher/pkg/source/twitter.go`
**Interface**: `source.Source`
```go
type TwitterSource struct {
    BearerToken string
}

func (s *TwitterSource) Fetch(ctx context.Context, query string, creds map[string]string) ([]*ContentItem, error)
```

### 2. Publisher Service
**File**: `publisher/pkg/publisher/twitter.go`
**Interface**: `publisher.Publisher`
```go
type TwitterPublisher struct {
    ApiKey            string
    ApiSecret         string
    AccessToken       string
    AccessTokenSecret string
}

func (p *TwitterPublisher) Publish(ctx context.Context, content string, mediaURLs []string, creds map[string]string) (string, error)
```

### 3. Orchestrator Service
**File**: `orchestrator/internal/service/orchestrator.go`
- Add `runTwitterEcho` flow.
- Update `runCrossPollinator` to accept `twitter` as a `target_platform`.

### 4. Configuration
**File**: `users.yaml`
```yaml
users:
  - id: "user-001"
    credentials:
      twitter_api_key: "..."
      twitter_api_secret: "..."
      twitter_access_token: "..."
      twitter_access_token_secret: "..."
```

## Implementation Plan

### Phase 1: Infrastructure & Fetcher
- [ ] Add `twitter` to `Fetcher` sources.
- [ ] Implement `TwitterSource.Fetch` (v2 search/user timeline).
- [ ] Add unit tests with mocks.

### Phase 2: Publisher
- [ ] Implement `TwitterPublisher.Publish` using OAuth 1.0a.
- [ ] Add unit tests with mocks.

### Phase 3: Orchestrator & E2E
- [ ] Update `orchestrator` logic to route `twitter` requests.
- [ ] Create `twitter_echo` pipeline.
- [ ] Verify using `batch-runner`.

## Examples

### ✅ Cross-Pollination to X
```bash
go run cmd/batch-runner/main.go --params="target_platform=twitter,query=golang"
```

## Trade-offs

### API v2 vs v1.1
- **v2**: Better support for modern features (Threads, Polls), but some endpoints are behind higher paywalls.
- **v1.1**: Legacy, being deprecated, but some media upload features still rely on it.
- **Decision**: Primary logic in v2, fallback to v1.1 only for media upload if necessary (common in X API).

## Implementation Results

### 1. Fetcher Implementation
- **File**: `fetcher/pkg/source/twitter.go`
- **Logic**: Implemented X API v2 standard. Supports User Timeline (via `id:<userid>`) and Recent Search.
- **Service**: Registered `twitter` in `FetcherService`.

### 2. Publisher Implementation
- **File**: `publisher/pkg/publisher/twitter.go`
- **Status**: Implemented structure and credential validation. 
- **⚠️ Note**: OAuth 1.0a signing logic is flagged as a TODO because it requires an external library (e.g., `dghubble/oauth1`) for HMAC-SHA1 signing which isn't currently vendored.

### 3. Orchestrator Implementation
- **File**: `orchestrator/internal/service/orchestrator.go`
- **Flow**: Added `twitter_echo` pipeline. This flow fetches the latest tweet, uses `AIContext` for personality consistency, and attempts to publish a response.

### 4. Verification
- **Code Audit**: Verified service registration and flow switching.
- **Next Steps**: User needs to provide a valid X Developer App with Bearer Token and User-level OAuth tokens in `users.yaml`.

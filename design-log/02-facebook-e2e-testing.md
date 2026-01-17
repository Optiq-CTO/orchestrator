# 02 - E2E Facebook Testing Strategy

## Background
The user wants to use a **real Facebook page** to verify the end-to-end (E2E) pipeline.
Currently:
*   `fetcher` has a `MetaSource` implementation (Read-only).
*   `publisher` has a Mock implementation but no real Meta/Facebook adapter (Write).
*   `orchestrator` can run flows but relies on these underlying services.

## Problem Statement
We need to demonstrate a full loop: `Fetch -> Analyze -> Create -> Publish` using a live platform (Facebook) to prove the system works in the real world.

## Questions and Answers

**Q: What prerequisites are missing?**
A: We need a **Meta Publisher Adapter** in the `publisher` service. Currently, it only has a mock. We need to implement `POST /v19.0/{page_id}/feed`.

**Q: How do we handle authentication?**
A: We will use the same User Access Token (with `pages_manage_posts` and `pages_read_engagement` scopes) for both fetching and publishing.
*   The Orchestrator request will need to accept these credentials and pass them down.

## Proposed Scenarios

### Scenario 1: The "Hello World" (Write-only)
**Goal**: Verify we can write to the Page.
1.  **Trigger**: User manually triggers default pipeline.
2.  **Action**: `creator` generates a "Hello World from AI Agents" post.
3.  **Publish**: `publisher` posts it to the Real Facebook Page.
4.  **Verify**: User checks Facebook Page URL to see the post.

### Scenario 2: The "Echo" Analysis (Read-Write)
**Goal**: Verify we can read, understand, and reply/react.
1.  **Setup**: User posts a photo/status on the Page manually (e.g., "Having coffee ☕").
2.  **Fetch**: `fetcher` grabs this latest post.
3.  **Analyze**: `analyzer` detects "Coffee", "Relaxation", "Morning".
4.  **Create**: `creator` generates a comment/reaction: "Enjoy your brew! ☕ goes well with coding."
5.  **Publish**: `publisher` posts this *as a comment* (or a new post referencing the old one).

## Implementation Plan

### Phase 1: Publisher Support
- [ ] Create `pkg/publisher/meta.go` in `publisher` service.
- [ ] Implement `PublishContent` using Facebook Graph API (`POST /feed`).

### Phase 2: Orchestrator Integration
- [ ] Define a new "Facebook E2E" workflow in Orchestrator.
- [ ] Update Client CLI to accept Facebook Page Token.

### Phase 3: Execution
- [ ] Run Scenario 1 (Write).
- [ ] Run Scenario 2 (Read-Write).

## Trade-offs
*   **Security**: We are pasting raw access tokens into CLI/Request. Acceptable for dev/testing.
*   **Scopes**: Requires a Token with broad permissions (`pages_*`). User must generate this via Graph API Explorer.

## Implementation Results

### 1. Meta Publisher Adapter
**File**: [`publisher/pkg/publisher/meta.go`](file:///Users/george/Documents/SideProjects/publisher/pkg/publisher/meta.go)

Implemented `MetaPublisher` struct with the following capabilities:
- **Endpoint**: `POST /v19.0/{page_id}/feed`
- **Validation**: Requires `page_id` and `access_token` in credentials
- **Payload**: Supports text content via `message` field
- **Media Support**: Basic support for `link` field (Phase 1 - external URLs)
- **Error Handling**: Parses and surfaces Meta API error messages

Registered in [`publisher/internal/service/publisher.go`](file:///Users/george/Documents/SideProjects/publisher/internal/service/publisher.go#L19-L30) as both `"meta"` and `"facebook"` platform strings.

### 2. Facebook Echo Bot Flow
**File**: [`orchestrator/internal/service/orchestrator.go`](file:///Users/george/Documents/SideProjects/orchestrator/internal/service/orchestrator.go#L119-L192)

Implemented `runFacebookEcho` workflow:
1. **Fetch**: Calls `fetcher.FetchContent` with `platform="meta"` and user credentials
2. **Analyze**: Uses analysis already performed by fetcher (tags, sentiment)
3. **Generate**: Calls `creator.GenerateContent` with contextual prompt including analysis
4. **Publish**: Calls `publisher.PublishContent` with same credentials to post response

**Flow Logic**:
- Processes the most recent post from the page
- Handles empty feed gracefully
- Generates friendly, context-aware responses

### 3. Test Client
**File**: [`orchestrator/cmd/client-facebook/main.go`](file:///Users/george/Documents/SideProjects/orchestrator/cmd/client-facebook/main.go)

Created dedicated client for Facebook testing:
```bash
go run orchestrator/cmd/client-facebook/main.go \
  -page_id=YOUR_PAGE_ID \
  -access_token=YOUR_TOKEN
```

### 4. Prerequisites for Testing
User needs to obtain:
1. **Facebook Page ID**: Found in Page Settings or via Graph API Explorer
2. **Access Token**: Generate via [Graph API Explorer](https://developers.facebook.com/tools/explorer/)
   - Required Permissions: `pages_read_engagement`, `pages_manage_posts`
   - Token Type: User Access Token (for your Page)

### 5. Running the Test
**Step-by-step**:
1. Start all services via docker-compose:
   ```bash
   cd /Users/george/Documents/SideProjects
   docker-compose up
   ```
2. Run the client:
   ```bash
   cd orchestrator
   go run cmd/client-facebook/main.go -page_id=XXX -access_token=YYY
   ```
3. Verify on Facebook that:
   - The orchestrator fetched your recent post
   - A new AI-generated response was published to your page


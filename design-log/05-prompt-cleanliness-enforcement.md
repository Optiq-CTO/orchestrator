# Design Log 05 - Prompt Cleanliness Enforcement

## Background
The content platform generates social media posts using Gemini and OpenAI. Users have reported that the generated content often includes "meta-talk" (e.g., "Here is your post..", "I hope this helps!", or explanations of why the post was written in a certain way). These explanations are intended for the developer/user but are being published to the end platforms as part of the post content.

## Problem Statement
The current instruction `- Do not include preamble.` is insufficient to prevent models from including conversational filler or post-generation analysis. 

### Example of Failed Output:
```text
We spotted this fantastic post about the power of daily stretching...
...
💬 THE PERFECT REPLY:
[Actual Content]
...
Why this works: It agrees with the original premise...
```

The system should only capture the `[Actual Content]` and discard everything else.

## Questions and Answers

**Q: Should we use JSON output mode?**
A: JSON would be highly reliable, but it increases token usage and can sometimes stifle creative formatting if the model gets confused by escaping. However, it's worth considering for a "content-only" field.

**Q: Should we use XML-style tags?**
A: Yes. Tagging the intended output with `<post>...</post>` is almost universally understood by modern LLMs and allows for easy server-side extraction using regular expressions.

**Q: How do we handle models that still fail?**
A: We will implement a robust extraction helper in the `pkg/ai` package that scans for tags and fallback to the whole text only if no tags are found.

## Design

### 1. Robust Tagging Strategy
Update all prompts in `creator` (and potentially `analyzer` if similar issues arise) to use explicit delimiters.

```text
Instruction: ...
Rules:
- Output ONLY the post content between <post> and </post> tags.
- ABSOLUTELY NO preamble, post-amble, or conversational filler.
- If you explain your response, do it OUTSIDE the tags (though ideally, don't explain at all).
```

### 2. Extraction Utility
Create a helper function in `pkg/ai/util.go` (or similar) to handle the parsing.

```go
func ExtractTaggedContent(text, tag string) string {
    re := regexp.MustCompile(fmt.Sprintf(`(?s)<%s>(.*?)</%s>`, tag, tag))
    match := re.FindStringSubmatch(text)
    if len(match) > 1 {
        return strings.TrimSpace(match[1])
    }
    // Fallback to original text if tags missing
    return strings.TrimSpace(text)
}
```

### 3. Updated Creator Adapters
Modify `gemini.go` and `openai.go` to use this extraction utility.

## Implementation Plan

### Phase 1: Infrastructure
- [ ] Create `pkg/ai/util.go` with extraction logic.
- [ ] Add unit tests for extraction.

### Phase 2: Creator Service
- [ ] Update `geminiCreator` prompts to use `<post>` tags.
- [ ] Update `openaiCreator` prompts to use `<post>` tags.
- [ ] Integrate `ExtractTaggedContent` into both adapters.

### Phase 3: Verification
- [ ] Test with "chatty" prompts to ensure meta-talk is discarded.

## Examples

### ✅ Good: Tagged Output
```
<post>
Feeling stiff? This 5-minute morning flow will change your day! 🧘‍♂️✨ #Health #Stretching
</post>
```

### ❌ Bad: Untagged commentary
```
I've created a post for you. It's optimized for engagement:
Feeling stiff? This 5-minute morning flow will change your day! 🧘‍♂️✨ #Health #Stretching
I hope your followers like it!
```

## Trade-offs

### Tagging vs JSON
- **Tagging**: Easier for the model to "just write," handles multi-line content naturally.
- **JSON**: More formal, but requires the model to manage string escaping (quotes, newlines) which can lead to invalid JSON or broken content.
- **Decision**: Go with Tagging for content generation as it's more flexible for raw text.

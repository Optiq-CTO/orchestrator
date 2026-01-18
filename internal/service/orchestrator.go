package service

import (
	"context"
	"fmt"
	"log"

	pb "github.com/Optiq-CTO/orchestrator/api/proto"
	aicontext "github.com/Optiq-CTO/orchestrator/api/proto/external/aicontext"
	creator "github.com/Optiq-CTO/orchestrator/api/proto/external/creator"
	fetcher "github.com/Optiq-CTO/orchestrator/api/proto/external/fetcher"
	publisher "github.com/Optiq-CTO/orchestrator/api/proto/external/publisher"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type OrchestratorService struct {
	pb.UnimplementedOrchestratorServiceServer
	fetcher   fetcher.FetcherServiceClient
	creator   creator.CreatorServiceClient
	publisher publisher.PublisherServiceClient
	aicontext aicontext.AIContextServiceClient
}

func NewOrchestratorService(f fetcher.FetcherServiceClient, c creator.CreatorServiceClient, p publisher.PublisherServiceClient, ac aicontext.AIContextServiceClient) *OrchestratorService {
	return &OrchestratorService{
		fetcher:   f,
		creator:   c,
		publisher: p,
		aicontext: ac,
	}
}

func (s *OrchestratorService) RunPipeline(ctx context.Context, req *pb.PipelineRequest) (*pb.PipelineResponse, error) {
	log.Printf("Running pipeline: %s", req.FlowName)

	switch req.FlowName {
	case "cross_pollinator":
		return s.runCrossPollinator(ctx, req.Params)
	case "facebook_echo":
		return s.runFacebookEcho(ctx, req.Params)
	case "trend_jacker":
		return nil, status.Error(codes.Unimplemented, "trend_jacker not implemented yet")
	default:
		return nil, status.Errorf(codes.InvalidArgument, "unknown flow: %s", req.FlowName)
	}
}

// Flow 1: Cross-Pollinator (Reddit -> LinkedIn/Twitter)
func (s *OrchestratorService) runCrossPollinator(ctx context.Context, params map[string]string) (*pb.PipelineResponse, error) {
	query := params["query"]
	targetPlatform := params["target_platform"]
	if query == "" || targetPlatform == "" {
		return nil, status.Error(codes.InvalidArgument, "missing params: query, target_platform")
	}

	// 1. Fetch from Reddit
	log.Printf("[Orchestrator] Step 1: Fetching from Reddit (query=%s)", query)
	fetchRes, err := s.fetcher.FetchContent(ctx, &fetcher.FetchRequest{
		Platform: "reddit",
		Query:    query,
	})
	if err != nil {
		return nil, fmt.Errorf("fetch failed: %w", err)
	}

	var outputURLs []string

	// 2. Process Items (Limit to top 1 for demo/MVP to avoid spamming)
	limit := 1
	for i, item := range fetchRes.Items {
		if i >= limit {
			break
		}

		log.Printf("[Orchestrator] Processing item: %s", item.ContentText[:min(50, len(item.ContentText))])

		// Use Summary if available, else raw text
		contentToRemix := item.ContentText
		if item.Analysis != nil && item.Analysis.Summary != "" {
			contentToRemix = item.Analysis.Summary
		}

		// 3. Remix Content
		log.Printf("[Orchestrator] Step 2: Remixing for %s", targetPlatform)
		remixRes, err := s.creator.RemixContent(ctx, &creator.RemixRequest{
			OriginalContent: contentToRemix,
			SourcePlatform:  "reddit",
			TargetPlatform:  targetPlatform,
			Tone:            "professional", // default for LinkedIn
		})
		if err != nil {
			log.Printf("Remix failed for item %s: %v", item.SourceId, err)
			continue
		}

		// 4. Publish
		log.Printf("[Orchestrator] Step 3: Publishing to %s", targetPlatform)
		pubRes, err := s.publisher.PublishContent(ctx, &publisher.PublishRequest{
			Content:  remixRes.Content,
			Platform: targetPlatform,
			// For MVP, passing dummy internal credential. In real world, Orchestrator might fetch this from Vault.
			Credentials: map[string]string{"internal_call": "true"},
		})
		if err != nil {
			log.Printf("Publish failed for item %s: %v", item.SourceId, err)
			continue
		}

		log.Printf("Successfully published: %s", pubRes.PostUrl)
		outputURLs = append(outputURLs, pubRes.PostUrl)
	}

	return &pb.PipelineResponse{
		PipelineId: "pipeline-123", // UUID in future
		Status:     "completed",
		OutputUrls: outputURLs,
	}, nil
}

// Flow 2: Facebook Echo Bot (Meta -> Analyze -> Create -> Meta)
func (s *OrchestratorService) runFacebookEcho(ctx context.Context, params map[string]string) (*pb.PipelineResponse, error) {
	pageID := params["page_id"]
	accessToken := params["access_token"]
	if pageID == "" || accessToken == "" {
		return nil, status.Error(codes.InvalidArgument, "missing params: page_id, access_token")
	}

	// 1. Fetch from Facebook
	log.Printf("[Orchestrator] Step 1: Fetching from Facebook page %s", pageID)
	fetchRes, err := s.fetcher.FetchContent(ctx, &fetcher.FetchRequest{
		Platform: "meta",
		Query:    pageID,
		Credentials: map[string]string{
			"access_token": accessToken,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("fetch failed: %w", err)
	}

	if len(fetchRes.Items) == 0 {
		return &pb.PipelineResponse{
			PipelineId:   "pipeline-fb-echo",
			Status:       "completed",
			ErrorMessage: "No posts found on the page",
		}, nil
	}

	// Get the most recent post
	latestPost := fetchRes.Items[0]
	log.Printf("[Orchestrator] Processing latest post: %s", latestPost.ContentText[:min(50, len(latestPost.ContentText))])

	// 2. Get AI Context
	log.Printf("[Orchestrator] Step 2: Fetching AI context for page %s", pageID)
	ctxRes, _ := s.aicontext.GetUserContext(ctx, &aicontext.GetUserContextRequest{
		User: &aicontext.User{Platform: "facebook", UserId: pageID},
	})

	var analysisContext string
	if latestPost.Analysis != nil {
		analysisContext = fmt.Sprintf("Tags: %v, Sentiment: %s",
			latestPost.Analysis.Tags,
			latestPost.Analysis.Sentiment)
	}

	prompt := fmt.Sprintf("Create a friendly response to this post: '%s'. Analysis: %s", latestPost.ContentText, analysisContext)
	if ctxRes != nil && ctxRes.Summary != "" {
		prompt = fmt.Sprintf("Last Context: %s. %s", ctxRes.Summary, prompt)
	}

	// 3. Generate contextual response
	log.Printf("[Orchestrator] Step 3: Generating response based on analysis and context")
	generateRes, err := s.creator.GenerateContent(ctx, &creator.GenerateRequest{
		Topic:    prompt,
		Platform: "facebook",
		Tone:     "friendly",
	})
	if err != nil {
		return nil, fmt.Errorf("content generation failed: %w", err)
	}

	// 4. Publish response to Facebook
	log.Printf("[Orchestrator] Step 4: Publishing response to Facebook")
	pubRes, err := s.publisher.PublishContent(ctx, &publisher.PublishRequest{
		Content:  generateRes.Content,
		Platform: "facebook",
		Credentials: map[string]string{
			"page_id":      pageID,
			"access_token": accessToken,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("publish failed: %w", err)
	}

	// 5. Update AI Context
	log.Printf("[Orchestrator] Step 5: Updating AI context with new interaction")
	s.aicontext.UpdateUserContext(ctx, &aicontext.UpdateUserContextRequest{
		User: &aicontext.User{Platform: "facebook", UserId: pageID},
		NewInteraction: &aicontext.Interaction{
			PostId:          pubRes.PostId,
			Content:         generateRes.Content,
			Direction:       "outbound",
			AnalysisSummary: analysisContext, // Or some other summary
		},
	})

	log.Printf("Successfully published echo response: %s", pubRes.PostUrl)

	return &pb.PipelineResponse{
		PipelineId: "pipeline-fb-echo",
		Status:     "completed",
		OutputUrls: []string{pubRes.PostUrl},
	}, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

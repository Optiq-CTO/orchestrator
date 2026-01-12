package service

import (
	"context"
	"fmt"
	"log"

	pb "github.com/Optiq-CTO/orchestrator/api/proto"
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
}

func NewOrchestratorService(f fetcher.FetcherServiceClient, c creator.CreatorServiceClient, p publisher.PublisherServiceClient) *OrchestratorService {
	return &OrchestratorService{
		fetcher:   f,
		creator:   c,
		publisher: p,
	}
}

func (s *OrchestratorService) RunPipeline(ctx context.Context, req *pb.PipelineRequest) (*pb.PipelineResponse, error) {
	log.Printf("Running pipeline: %s", req.FlowName)

	switch req.FlowName {
	case "cross_pollinator":
		return s.runCrossPollinator(ctx, req.Params)
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

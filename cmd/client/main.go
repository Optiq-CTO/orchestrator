package main

import (
	"context"
	"log"
	"time"

	pb "github.com/Optiq-CTO/orchestrator/api/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	conn, err := grpc.Dial("localhost:50056", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewOrchestratorServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second) // Long timeout for full pipeline
	defer cancel()

	log.Println("--- Triggering Cross-Pollinator Pipeline ---")
	log.Println("Goal: Fetch Reddit(golang) -> Analyze -> Remix -> Publish(Twitter)")

	res, err := c.RunPipeline(ctx, &pb.PipelineRequest{
		FlowName: "cross_pollinator",
		Params: map[string]string{
			"query":           "golang",
			"target_platform": "twitter",
		},
	})

	if err != nil {
		log.Fatalf("Pipeline failed: %v", err)
	}

	log.Printf("Pipeline Status: %s", res.Status)
	for i, url := range res.OutputUrls {
		log.Printf("Output %d: %s", i+1, url)
	}
}

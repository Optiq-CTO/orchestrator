package main

import (
	"context"
	"flag"
	"log"
	"time"

	pb "github.com/Optiq-CTO/orchestrator/api/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	pageID := flag.String("page_id", "", "Facebook Page ID")
	accessToken := flag.String("access_token", "", "Facebook Access Token")
	flag.Parse()

	if *pageID == "" || *accessToken == "" {
		log.Fatal("Usage: go run main.go -page_id=YOUR_PAGE_ID -access_token=YOUR_TOKEN")
	}

	conn, err := grpc.Dial("localhost:50056", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewOrchestratorServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	log.Println("--- Triggering Facebook Echo Bot Pipeline ---")
	log.Println("Goal: Fetch Facebook -> Analyze -> Generate Response -> Publish to Facebook")

	res, err := c.RunPipeline(ctx, &pb.PipelineRequest{
		FlowName: "facebook_echo",
		Params: map[string]string{
			"page_id":      *pageID,
			"access_token": *accessToken,
		},
	})

	if err != nil {
		log.Fatalf("Pipeline failed: %v", err)
	}

	log.Printf("Pipeline Status: %s", res.Status)
	if res.ErrorMessage != "" {
		log.Printf("Message: %s", res.ErrorMessage)
	}
	for i, url := range res.OutputUrls {
		log.Printf("Output %d: %s", i+1, url)
	}
}

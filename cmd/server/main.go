package main

import (
	"log"
	"net"
	"os"

	pb "github.com/Optiq-CTO/orchestrator/api/proto"
	aicontext "github.com/Optiq-CTO/orchestrator/api/proto/external/aicontext"
	creator "github.com/Optiq-CTO/orchestrator/api/proto/external/creator"
	fetcher "github.com/Optiq-CTO/orchestrator/api/proto/external/fetcher"
	publisher "github.com/Optiq-CTO/orchestrator/api/proto/external/publisher"
	"github.com/Optiq-CTO/orchestrator/internal/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "50056"
	}

	// Connect to Fetcher
	fetcherHost := os.Getenv("FETCHER_HOST")
	if fetcherHost == "" {
		fetcherHost = "localhost:50053"
	}
	connFetcher, err := grpc.Dial(fetcherHost, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect to fetcher: %v", err)
	}
	defer connFetcher.Close()
	fetcherClient := fetcher.NewFetcherServiceClient(connFetcher)

	// Connect to Creator
	creatorHost := os.Getenv("CREATOR_HOST")
	if creatorHost == "" {
		creatorHost = "localhost:50054"
	}
	connCreator, err := grpc.Dial(creatorHost, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect to creator: %v", err)
	}
	defer connCreator.Close()
	creatorClient := creator.NewCreatorServiceClient(connCreator)

	// Connect to Publisher
	publisherHost := os.Getenv("PUBLISHER_HOST")
	if publisherHost == "" {
		publisherHost = "localhost:50055"
	}
	connPub, err := grpc.Dial(publisherHost, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect to publisher: %v", err)
	}
	defer connPub.Close()
	pubClient := publisher.NewPublisherServiceClient(connPub)

	// Connect to AIContext
	aiContextHost := os.Getenv("AICONTEXT_HOST")
	if aiContextHost == "" {
		aiContextHost = "localhost:50057"
	}
	connAIContext, err := grpc.Dial(aiContextHost, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect to aicontext: %v", err)
	}
	defer connAIContext.Close()
	aiContextClient := aicontext.NewAIContextServiceClient(connAIContext)

	// Start Orchestrator
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	svc := service.NewOrchestratorService(fetcherClient, creatorClient, pubClient, aiContextClient)
	pb.RegisterOrchestratorServiceServer(s, svc)
	reflection.Register(s)

	log.Printf("Orchestrator service listening on port %s", port)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

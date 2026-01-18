package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	pb "github.com/Optiq-CTO/orchestrator/api/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gopkg.in/yaml.v3"
)

// Config represents the users.yaml structure
type Config struct {
	Version string `yaml:"version"`
	Users   []User `yaml:"users"`
}

type User struct {
	ID          string            `yaml:"id"`
	Name        string            `yaml:"name"`
	Platform    string            `yaml:"platform"`
	Credentials map[string]string `yaml:"credentials"`
	Pipelines   []Pipeline        `yaml:"pipelines"`
}

type Pipeline struct {
	Name    string `yaml:"name"`
	Enabled bool   `yaml:"enabled"`
}

// ExecutionResult tracks the outcome of running a pipeline for a user
type ExecutionResult struct {
	UserID   string
	UserName string
	Pipeline string
	Status   string // "success", "failed", "skipped"
	Error    error
	PostURLs []string
	Duration time.Duration
}

func main() {
	configPath := flag.String("config", "../../users.yaml", "Path to users.yaml configuration file")
	orchestratorAddr := flag.String("orchestrator", "localhost:50056", "Orchestrator service address")
	modelProvider := flag.String("model", "gemini", "AI model provider (gemini or openai)")
	flag.Parse()

	// 1. Load configuration
	log.Printf("Loading configuration from: %s", *configPath)
	config, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	log.Printf("Loaded %d users from configuration", len(config.Users))

	// 2. Connect to Orchestrator
	log.Printf("Connecting to Orchestrator at %s", *orchestratorAddr)
	conn, err := grpc.Dial(*orchestratorAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to orchestrator: %v", err)
	}
	defer conn.Close()
	client := pb.NewOrchestratorServiceClient(conn)

	// 3. Execute pipelines for each user
	results := []ExecutionResult{}

	log.Println("\n===== Starting Batch Pipeline Execution =====")
	for i, user := range config.Users {
		if i > 0 {
			log.Println("\n[Rate Limit] Waiting 15s between users to respect Gemini API limits...")
			time.Sleep(15 * time.Second)
		}

		log.Printf("\n[%d/%d] Processing user: %s (ID: %s)", i+1, len(config.Users), user.Name, user.ID)

		for _, pipeline := range user.Pipelines {
			if !pipeline.Enabled {
				log.Printf("  Pipeline '%s' is disabled, skipping", pipeline.Name)
				results = append(results, ExecutionResult{
					UserID:   user.ID,
					UserName: user.Name,
					Pipeline: pipeline.Name,
					Status:   "skipped",
				})
				continue
			}

			result := executePipeline(client, user, pipeline, *modelProvider)
			results = append(results, result)
		}
	}

	// 4. Print summary
	printSummary(results)
}

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parsing YAML: %w", err)
	}

	return &config, nil
}

func executePipeline(client pb.OrchestratorServiceClient, user User, pipeline Pipeline, modelProvider string) ExecutionResult {
	result := ExecutionResult{
		UserID:   user.ID,
		UserName: user.Name,
		Pipeline: pipeline.Name,
	}

	start := time.Now()
	defer func() {
		result.Duration = time.Since(start)
	}()

	log.Printf("  Executing pipeline: %s", pipeline.Name)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Map pipeline name to flow and prepare params
	flowName := pipeline.Name
	params := make(map[string]string)

	// Copy all credentials to params
	for k, v := range user.Credentials {
		params[k] = v
	}

	// Execute pipeline
	res, err := client.RunPipeline(ctx, &pb.PipelineRequest{
		FlowName:      flowName,
		Params:        params,
		ModelProvider: modelProvider,
	})

	if err != nil {
		result.Status = "failed"
		result.Error = err
		log.Printf("  ❌ FAILED: %v", err)
		return result
	}

	result.Status = "success"
	result.PostURLs = res.OutputUrls
	log.Printf("  ✅ SUCCESS: Pipeline completed with status '%s'", res.Status)
	for i, url := range res.OutputUrls {
		log.Printf("     Output %d: %s", i+1, url)
	}

	return result
}

func printSummary(results []ExecutionResult) {
	log.Println("\n\n===== Execution Summary =====")

	successCount := 0
	failedCount := 0
	skippedCount := 0

	for _, r := range results {
		switch r.Status {
		case "success":
			successCount++
		case "failed":
			failedCount++
		case "skipped":
			skippedCount++
		}
	}

	log.Printf("Total: %d pipelines", len(results))
	log.Printf("✅ Success: %d", successCount)
	log.Printf("❌ Failed: %d", failedCount)
	log.Printf("⏭️  Skipped: %d", skippedCount)

	// Detailed results
	log.Println("\nDetailed Results:")
	log.Println("--------------------------------------------------")
	for _, r := range results {
		icon := "✅"
		if r.Status == "failed" {
			icon = "❌"
		} else if r.Status == "skipped" {
			icon = "⏭️"
		}

		log.Printf("%s [%s] %s - %s (%.2fs)",
			icon,
			r.Status,
			r.UserName,
			r.Pipeline,
			r.Duration.Seconds())

		if r.Error != nil {
			log.Printf("   Error: %v", r.Error)
		}
		if len(r.PostURLs) > 0 {
			for _, url := range r.PostURLs {
				log.Printf("   Post: %s", url)
			}
		}
	}
	log.Println("--------------------------------------------------")

	if failedCount > 0 {
		log.Printf("\nWARNING: %d pipeline(s) failed. Check logs above for details.", failedCount)
		os.Exit(1)
	}
}

package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/GoogleCloudPlatform/cloud-run-release-operator/internal/metrics"
	"github.com/GoogleCloudPlatform/cloud-run-release-operator/internal/stackdriver"
)

func main() {
	client, err := stackdriver.NewProvider(context.Background(), "gtvo-test")
	if err != nil {
		log.Fatalf("failed to initialize metrics provider: %v", err)
	}

	ctx := context.Background()

	query := stackdriver.NewQuery("gtvo-test", "us-east1", "tester", "tester-00004-reg")
	requestCount, err := client.RequestCount(ctx, query, time.Hour)
	if err != nil {
		log.Fatalf("failed to retrieve request count: %v", err)
	}
	fmt.Println(requestCount)

	errorRate, err := client.ErrorRate(ctx, query, time.Hour)
	if err != nil {
		log.Fatalf("failed to retrieve server error rate: %v", err)
	}
	fmt.Println(errorRate)

	latency, err := client.Latency(ctx, query, time.Hour, metrics.Align99Reduce99)
	if err != nil {
		log.Fatalf("failed to retrieve latency: %v", err)
	}
	fmt.Println(latency)
}

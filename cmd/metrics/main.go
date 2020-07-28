package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/GoogleCloudPlatform/cloud-run-release-operator/internal/metrics"
	"github.com/GoogleCloudPlatform/cloud-run-release-operator/internal/metrics/sheets"
)

func main() {
	client, err := sheets.NewProvider(context.Background(), "1B3Ex-1KlcCn5HtsK3gbErc-8T4V_SDrBUPKjp3THJi0", "", "us-east1", "tester")
	if err != nil {
		log.Fatalf("failed to initialize metrics provider: %v", err)
	}

	ctx := context.Background()

	requestCount, err := client.RequestCount(ctx, time.Hour)
	if err != nil {
		log.Fatalf("failed to retrieve request count: %v", err)
	}
	fmt.Println(requestCount)

	errorRate, err := client.ErrorRate(ctx, time.Hour)
	if err != nil {
		log.Fatalf("failed to retrieve server error rate: %v", err)
	}
	fmt.Println(errorRate)

	latency, err := client.Latency(ctx, time.Hour, metrics.Align99Reduce99)
	if err != nil {
		log.Fatalf("failed to retrieve latency: %v", err)
	}
	fmt.Println(latency)
}

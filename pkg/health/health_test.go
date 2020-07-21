package health_test

import (
	"context"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/cloud-run-release-operator/internal/metrics"
	metricsMocker "github.com/GoogleCloudPlatform/cloud-run-release-operator/internal/metrics/mock"
	"github.com/GoogleCloudPlatform/cloud-run-release-operator/pkg/config"
	"github.com/GoogleCloudPlatform/cloud-run-release-operator/pkg/health"
	"github.com/stretchr/testify/assert"
)

func TestIsHealthy(t *testing.T) {
	metricsMock := metricsMocker.Metrics{}
	metricsMock.LatencyFn = func(ctx context.Context, query metrics.Query, offset time.Duration, alignReduceType metrics.AlignReduce) (float64, error) {
		return 500, nil
	}
	metricsMock.ErrorRateFn = func(ctx context.Context, query metrics.Query, offset time.Duration) (float64, error) {
		return 0.01, nil
	}

	tests := []struct {
		name     string
		query    metrics.Query
		offset   time.Duration
		metrics  []config.Metric
		expected bool
	}{
		{
			name:   "healthy revision",
			query:  metricsMocker.Query{},
			offset: 5 * time.Minute,
			metrics: []config.Metric{
				{Type: config.LatencyMetricsCheck, Percentile: 99, Max: 750},
				{Type: config.ErrorRateMetricsCheck, Max: 5},
			},
			expected: true,
		},
		{
			name:   "barely healthy revision",
			query:  metricsMocker.Query{},
			offset: 5 * time.Minute,
			metrics: []config.Metric{
				{Type: config.LatencyMetricsCheck, Percentile: 99, Max: 500},
				{Type: config.ErrorRateMetricsCheck, Max: 1},
			},
			expected: true,
		},
		{
			name:   "unhealthy revision, miss latency",
			query:  metricsMocker.Query{},
			offset: 5 * time.Minute,
			metrics: []config.Metric{
				{Type: config.LatencyMetricsCheck, Percentile: 99, Max: 499},
			},
			expected: false,
		},
		{
			name:   "unhealthy revision, miss error rate",
			query:  metricsMocker.Query{},
			offset: 5 * time.Minute,
			metrics: []config.Metric{
				{Type: config.ErrorRateMetricsCheck, Max: 0.95},
			},
			expected: false,
		},
	}

	health := health.New(&metricsMock)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			isHealthy, _ := health.IsHealthy(context.Background(), test.query, test.offset, test.metrics)
			assert.Equal(t, test.expected, isHealthy)
		})
	}
}

package health_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/cloud-run-release-operator/internal/metrics"
	metricsMocker "github.com/GoogleCloudPlatform/cloud-run-release-operator/internal/metrics/mock"
	"github.com/GoogleCloudPlatform/cloud-run-release-operator/pkg/config"
	"github.com/GoogleCloudPlatform/cloud-run-release-operator/pkg/health"
	"github.com/stretchr/testify/assert"
)

func TestDiagnose(t *testing.T) {
	metricsMock := &metricsMocker.Metrics{}
	metricsMock.LatencyFn = func(ctx context.Context, query metrics.Query, offset time.Duration, alignReduceType metrics.AlignReduce) (float64, error) {
		return 500, nil
	}
	metricsMock.ErrorRateFn = func(ctx context.Context, query metrics.Query, offset time.Duration) (float64, error) {
		return 0.01, nil
	}

	tests := []struct {
		name            string
		query           metrics.Query
		offset          time.Duration
		minRequests     int64
		metricsCriteria []config.Metric
		expected        *health.Diagnosis
	}{
		{
			name:        "healthy revision",
			query:       metricsMocker.Query{},
			offset:      5 * time.Minute,
			minRequests: 1000,
			metricsCriteria: []config.Metric{
				{Type: config.LatencyMetricsCheck, Percentile: 99, Max: 750},
				{Type: config.ErrorRateMetricsCheck, Max: 5},
			},
			expected: &health.Diagnosis{
				EnoughRequests: true,
				IsHealthy:      true,
				CheckResults: []*health.CheckResult{
					{
						MetricsType:   config.LatencyMetricsCheck,
						ActualValue:   500.0,
						MinOrMaxValue: 750.0,
						IsCriteriaMet: true,
						Reason:        fmt.Sprintf("actual value %.2f is less or equal than max allowed latency %.2f", 500.0, 750.0),
					},
					{
						MetricsType:   config.ErrorRateMetricsCheck,
						ActualValue:   1.0,
						MinOrMaxValue: 5.0,
						IsCriteriaMet: true,
						Reason:        fmt.Sprintf("actual value %.2f is less or equal than max allowed error rate %.2f", 1.0, 5.0),
					},
				},
			},
		},
		{
			name:        "barely healthy revision",
			query:       metricsMocker.Query{},
			offset:      5 * time.Minute,
			minRequests: 1000,
			metricsCriteria: []config.Metric{
				{Type: config.LatencyMetricsCheck, Percentile: 99, Max: 500},
				{Type: config.ErrorRateMetricsCheck, Max: 1},
			},
			expected: &health.Diagnosis{
				EnoughRequests: true,
				IsHealthy:      true,
				CheckResults: []*health.CheckResult{
					{
						MetricsType:   config.LatencyMetricsCheck,
						ActualValue:   500.0,
						MinOrMaxValue: 500.0,
						IsCriteriaMet: true,
						Reason:        fmt.Sprintf("actual value %.2f is less or equal than max allowed latency %.2f", 500.0, 500.0),
					},
					{
						MetricsType:   config.ErrorRateMetricsCheck,
						ActualValue:   1.0,
						MinOrMaxValue: 1.0,
						IsCriteriaMet: true,
						Reason:        fmt.Sprintf("actual value %.2f is less or equal than max allowed error rate %.2f", 1.0, 1.0),
					},
				},
			},
		},
		{
			name:        "unhealthy revision, miss latency",
			query:       metricsMocker.Query{},
			offset:      5 * time.Minute,
			minRequests: 1000,
			metricsCriteria: []config.Metric{
				{Type: config.LatencyMetricsCheck, Percentile: 99, Max: 499},
			},
			expected: &health.Diagnosis{
				EnoughRequests: true,
				IsHealthy:      false,
				CheckResults: []*health.CheckResult{
					{
						MetricsType:   config.LatencyMetricsCheck,
						ActualValue:   500.0,
						MinOrMaxValue: 499.0,
						IsCriteriaMet: false,
						Reason:        fmt.Sprintf("actual value %.2f is greater than max allowed latency %.2f", 500.0, 499.0),
					},
				},
				FailedCheckResults: []*health.CheckResult{
					{
						MetricsType:   config.LatencyMetricsCheck,
						ActualValue:   500.0,
						MinOrMaxValue: 499.0,
						IsCriteriaMet: false,
						Reason:        fmt.Sprintf("actual value %.2f is greater than max allowed latency %.2f", 500.0, 499.0),
					},
				},
			},
		},
		{
			name:        "unhealthy revision, miss error rate",
			query:       metricsMocker.Query{},
			offset:      5 * time.Minute,
			minRequests: 1000,
			metricsCriteria: []config.Metric{
				{Type: config.ErrorRateMetricsCheck, Max: 0.95},
			},
			expected: &health.Diagnosis{
				EnoughRequests: true,
				IsHealthy:      false,
				CheckResults: []*health.CheckResult{
					{
						MetricsType:   config.ErrorRateMetricsCheck,
						ActualValue:   1.0,
						MinOrMaxValue: 0.95,
						IsCriteriaMet: false,
						Reason:        fmt.Sprintf("actual value %.2f is greater than max allowed error rate %.2f", 1.0, 0.95),
					},
				},
				FailedCheckResults: []*health.CheckResult{
					{
						MetricsType:   config.ErrorRateMetricsCheck,
						ActualValue:   1.0,
						MinOrMaxValue: 0.95,
						IsCriteriaMet: false,
						Reason:        fmt.Sprintf("actual value %.2f is greater than max allowed error rate %.2f", 1.0, 0.95),
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			isHealthy, _ := health.Diagnose(ctx, metricsMock, test.query, test.offset, test.minRequests, test.metricsCriteria)
			assert.Equal(t, test.expected, isHealthy)
		})
	}
}

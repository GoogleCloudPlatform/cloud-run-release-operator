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

func TestDiagnosis(t *testing.T) {
	tests := []struct {
		name           string
		healthCriteria []config.Metric
		results        []float64
		expected       health.Diagnosis
		shouldErr      bool
	}{
		{
			name: "healthy revision",
			healthCriteria: []config.Metric{
				{Type: config.LatencyMetricsCheck, Percentile: 99, Threshold: 750},
				{Type: config.ErrorRateMetricsCheck, Threshold: 5},
			},
			results: []float64{500.0, 1.0},
			expected: health.Diagnosis{
				OverallResult: health.Healthy,
				CheckResults: []health.CheckResult{
					{
						Threshold:     750.0,
						ActualValue:   500.0,
						IsCriteriaMet: true,
					},
					{
						Threshold:     5.0,
						ActualValue:   1.0,
						IsCriteriaMet: true,
					},
				},
			},
		},
		{
			name: "barely healthy revision",
			healthCriteria: []config.Metric{
				{Type: config.LatencyMetricsCheck, Percentile: 99, Threshold: 500},
				{Type: config.ErrorRateMetricsCheck, Threshold: 1},
			},
			results: []float64{500.0, 1.0},
			expected: health.Diagnosis{
				OverallResult: health.Healthy,
				CheckResults: []health.CheckResult{
					{
						Threshold:     500.0,
						ActualValue:   500.0,
						IsCriteriaMet: true,
					},
					{
						Threshold:     1.0,
						ActualValue:   1.0,
						IsCriteriaMet: true,
					},
				},
			},
		},
		{
			name: "unhealthy revision, miss latency",
			healthCriteria: []config.Metric{
				{Type: config.LatencyMetricsCheck, Percentile: 99, Threshold: 499},
			},
			results: []float64{500.0},
			expected: health.Diagnosis{
				OverallResult: health.Unhealthy,
				CheckResults: []health.CheckResult{
					{
						Threshold:     499.0,
						ActualValue:   500.0,
						IsCriteriaMet: false,
					},
				},
			},
		},
		{
			name: "unhealthy revision, miss error rate",
			healthCriteria: []config.Metric{
				{Type: config.ErrorRateMetricsCheck, Threshold: 0.95},
			},
			results: []float64{1.0},
			expected: health.Diagnosis{
				OverallResult: health.Unhealthy,
				CheckResults: []health.CheckResult{
					{
						Threshold:     0.95,
						ActualValue:   1.0,
						IsCriteriaMet: false,
					},
				},
			},
		},
		{
			name:      "should err, empty health criteria",
			shouldErr: true,
		},
		{
			name: "should err, different sizes for criteria and results",
			healthCriteria: []config.Metric{
				{Type: config.ErrorRateMetricsCheck, Threshold: 0.95},
			},
			results:   []float64{},
			shouldErr: true,
		},
		{
			name:      "should err, empty health criteria",
			shouldErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			diagnosis, err := health.Diagnose(ctx, test.healthCriteria, test.results)
			if test.shouldErr {
				assert.NotNil(t, err)
			} else {
				assert.Equal(t, test.expected, diagnosis)
			}
		})
	}
}

// TestCollectMetrics tests that the health.CollectMetrics returns the correct
// values using the metrics provider.
func TestCollectMetrics(t *testing.T) {
	metricsMock := &metricsMocker.Metrics{}
	metricsMock.LatencyFn = func(ctx context.Context, offset time.Duration, alignReduceType metrics.AlignReduce) (float64, error) {
		return 500, nil
	}
	metricsMock.ErrorRateFn = func(ctx context.Context, offset time.Duration) (float64, error) {
		return 0.01, nil
	}

	ctx := context.Background()
	offset := 5 * time.Minute
	healthCriteria := []config.Metric{
		{Type: config.LatencyMetricsCheck, Percentile: 99},
		{Type: config.ErrorRateMetricsCheck},
	}
	expected := []float64{500.0, 1.0}
	results, _ := health.CollectMetrics(ctx, metricsMock, offset, healthCriteria)
	assert.Equal(t, expected, results)

	assert.Equal(t, expected, results)
}
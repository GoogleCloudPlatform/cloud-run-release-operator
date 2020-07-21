package health

import (
	"context"
	"time"

	"github.com/GoogleCloudPlatform/cloud-run-release-operator/internal/metrics"
	"github.com/GoogleCloudPlatform/cloud-run-release-operator/pkg/config"
	"github.com/pkg/errors"
)

// Checker validates that revision is healthy.
type Checker interface {
	IsHealthy(ctx context.Context, query metrics.Query, healthCheckOffset time.Duration, metricsChecks []config.Metric) (bool, error)
}

// Health is the candidate's health status checker.
type Health struct {
	Metrics metrics.Metrics
}

// New initializes a health checker.
func New(metrics metrics.Metrics) *Health {
	return &Health{
		Metrics: metrics,
	}
}

// IsHealthy checks if a revision is healthy based on the metrics configuration.
func (h *Health) IsHealthy(ctx context.Context, query metrics.Query, healthCheckOffset time.Duration, metricsChecks []config.Metric) (bool, error) {
	healthy := true
	for _, check := range metricsChecks {
		var err error
		switch check.Type {
		case config.LatencyMetricsCheck:
			healthy, err = h.checkLatency(ctx, query, healthCheckOffset, check.Percentile, check.Max)
			break
		case config.ErrorRateMetricsCheck:
			healthy, err = h.checkErrorRate(ctx, query, healthCheckOffset, check.Max)
			break
		}

		if err != nil {
			return false, errors.Wrapf(err, "failed to check metrics %q", check.Type)
		}
	}

	return healthy, nil
}

// checkLatency checks that the threshold for latency is not exceeded.
func (h *Health) checkLatency(ctx context.Context, query metrics.Query, offset time.Duration, percentile, max float64) (bool, error) {
	alignerReducer, err := metrics.PercentileToAlignReduce(percentile)
	if err != nil {
		return false, errors.Wrap(err, "invalid percentile")
	}

	latency, err := h.Metrics.Latency(ctx, query, offset, alignerReducer)
	if err != nil {
		return false, errors.Wrap(err, "failed to get latency metrics")
	}

	return latency <= max, nil
}

// checkErrorRate checks that the threshold for error rate is not exceeded.
func (h *Health) checkErrorRate(ctx context.Context, query metrics.Query, offset time.Duration, max float64) (bool, error) {
	rate, err := h.Metrics.ErrorRate(ctx, query, offset)
	if err != nil {
		return false, errors.Wrap(err, "failed to get error rate metrics")
	}

	return rate <= max, nil
}

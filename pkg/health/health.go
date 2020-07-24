package health

import (
	"context"
	"time"

	"github.com/GoogleCloudPlatform/cloud-run-release-operator/internal/metrics"
	"github.com/GoogleCloudPlatform/cloud-run-release-operator/pkg/config"
	"github.com/pkg/errors"
)

// DiagnosisResult is a possible result after a diagnosis.
type DiagnosisResult int32

// Possible diagnosis results.
const (
	Unknown      DiagnosisResult = 0
	Inconclusive DiagnosisResult = 1
	Healthy      DiagnosisResult = 2
	Unhealthy    DiagnosisResult = 3
)

// Diagnosis is the information about the health of the revision.
type Diagnosis struct {
	OverallResult DiagnosisResult
	CheckResults  []*CheckResult
}

// CheckResult is information about a metrics criteria check.
type CheckResult struct {
	Threshold     float64
	ActualValue   float64
	IsCriteriaMet bool
}

// Diagnose attempts to determine the health of a revision.
//
// If the minimum number of requests is not met, then health cannot be
// determined and Diagnosis.EnoughRequests is set to false.
//
// Otherwise, all metrics criteria are checked to determine if the revision is
// healthy.
func Diagnose(ctx context.Context, provider metrics.Metrics, query metrics.Query,
	offset time.Duration, minRequests int64, healthCriteria []config.Metric) (*Diagnosis, error) {

	metricsValues, err := CollectMetrics(ctx, provider, query, offset, healthCriteria)
	if err != nil {
		return nil, errors.Wrap(err, "could not collect metrics")
	}

	overallResult := Healthy
	var results []*CheckResult
	for i, criteria := range healthCriteria {
		result := determineResult(criteria.Type, criteria.Threshold, metricsValues[i])
		results = append(results, result)

		if !result.IsCriteriaMet {
			overallResult = Unhealthy
		}
	}

	return &Diagnosis{
		OverallResult: overallResult,
		CheckResults:  results,
	}, nil
}

// CollectMetrics returns an array of values collected for each of the specified
// metrics criteria.
func CollectMetrics(ctx context.Context, provider metrics.Metrics, query metrics.Query, offset time.Duration, healthCriteria []config.Metric) ([]float64, error) {
	var values []float64
	for _, criteria := range healthCriteria {
		var value float64
		var err error

		switch criteria.Type {
		case config.LatencyMetricsCheck:
			value, err = latency(ctx, provider, query, offset, criteria.Percentile)
			break
		case config.ErrorRateMetricsCheck:
			value, err = errorRatePercent(ctx, provider, query, offset)
			break
		default:
			return nil, errors.Errorf("unimplemented metrics %q", criteria.Type)
		}

		if err != nil {
			return nil, errors.Wrapf(err, "failed to obtain metrics %q", criteria.Type)
		}
		values = append(values, value)
	}

	return values, nil
}

// determineResult concludes if metrics criteria was met.
//
// The returned value also includes a string with details of why the criteria
// was met or not.
func determineResult(metricsType config.MetricsCheck, threshold float64, actualValue float64) *CheckResult {
	result := &CheckResult{ActualValue: actualValue, Threshold: threshold}

	// As of now, the supported health criteria (latency and error rate) need to
	// be less than the threshold. So, this is sufficient for now but might need
	// to change to a switch statement when criteria with a minimum threshold is
	// added.
	if actualValue <= threshold {
		result.IsCriteriaMet = true
	}
	return result
}

// latency returns the latency for the given offset and percentile.
func latency(ctx context.Context, provider metrics.Metrics, query metrics.Query, offset time.Duration, percentile float64) (float64, error) {
	alignerReducer, err := metrics.PercentileToAlignReduce(percentile)
	if err != nil {
		return 0, errors.Wrap(err, "invalid percentile")
	}

	latency, err := provider.Latency(ctx, query, offset, alignerReducer)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get latency metrics")
	}

	return latency, nil
}

// errorRatePercent returns the percentage of errors during the given offset.
func errorRatePercent(ctx context.Context, provider metrics.Metrics, query metrics.Query, offset time.Duration) (float64, error) {
	rate, err := provider.ErrorRate(ctx, query, offset)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get error rate metrics")
	}

	// Multiply rate by 100 to have a percentage.
	return rate * 100, nil
}

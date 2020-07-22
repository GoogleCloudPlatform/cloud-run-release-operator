package health

import (
	"context"
	"fmt"
	"time"

	"github.com/GoogleCloudPlatform/cloud-run-release-operator/internal/metrics"
	"github.com/GoogleCloudPlatform/cloud-run-release-operator/pkg/config"
	"github.com/pkg/errors"
)

// Diagnosis is the information about the health of the revision.
type Diagnosis struct {
	EnoughRequests     bool
	CheckResults       []*CheckResult
	FailedCheckResults []*CheckResult
	IsHealthy          bool
}

// CheckResult is information about a metrics criteria check.
type CheckResult struct {
	MetricsType   config.MetricsCheck
	MinOrMaxValue interface{}
	ActualValue   interface{}
	IsCriteriaMet bool
	Reason        string
}

// Diagnose attempts to determine the health of a revision.
//
// If the minimum number of requests is not met, then health cannot be
// determined and Diagnosis.EnoughRequests is set to false.
//
// Otherwise, all metrics criteria are checked to determine if the revision is
// healthy.
func Diagnose(ctx context.Context, provider metrics.Metrics, query metrics.Query,
	offset time.Duration, minRequests int64, metricsCriteria []config.Metric) (*Diagnosis, error) {

	metricsValues, err := CollectMetrics(ctx, provider, query, offset, metricsCriteria)
	if err != nil {
		return nil, errors.Wrap(err, "could not collect metrics")
	}

	isHealthy := true
	var results, failedResults []*CheckResult
	for i, criteria := range metricsCriteria {
		result := determineResult(criteria.Type, criteria.Min, criteria.Max, metricsValues[i])
		results = append(results, result)

		if !result.IsCriteriaMet {
			isHealthy = false
			failedResults = append(failedResults, result)
		}
	}

	return &Diagnosis{
		EnoughRequests:     true,
		CheckResults:       results,
		FailedCheckResults: failedResults,
		IsHealthy:          isHealthy,
	}, nil
}

// CollectMetrics returns an array of values collected for each of the specified
// metrics criteria.
func CollectMetrics(ctx context.Context, provider metrics.Metrics, query metrics.Query, offset time.Duration, metricsCriteria []config.Metric) ([]interface{}, error) {

	var values []interface{}
	for _, criteria := range metricsCriteria {
		var value interface{}
		var err error

		switch criteria.Type {
		case config.LatencyMetricsCheck:
			value, err = latency(ctx, provider, query, offset, criteria.Percentile)
			break
		case config.ErrorRateMetricsCheck:
			value, err = errorRate(ctx, provider, query, offset)
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
func determineResult(metricsType config.MetricsCheck, min interface{}, max interface{}, actualValue interface{}) *CheckResult {
	result := &CheckResult{MetricsType: metricsType, ActualValue: actualValue}

	switch metricsType {
	case config.LatencyMetricsCheck:
		actual := actualValue.(float64)
		reasonFormat := "actual value %.2f is %s than max allowed latency %.2f"

		if actual <= max.(float64) {
			result.IsCriteriaMet = true
			result.Reason = fmt.Sprintf(reasonFormat, actual, "less or equal", max)
		} else {
			result.IsCriteriaMet = false
			result.Reason = fmt.Sprintf(reasonFormat, actual, "greater", max)
		}
		result.MinOrMaxValue = max
		break

	case config.ErrorRateMetricsCheck:
		actual := actualValue.(float64)
		reasonFormat := "actual value %.2f is %s than max allowed error rate %.2f"

		if actual <= max.(float64) {
			result.IsCriteriaMet = true
			result.Reason = fmt.Sprintf(reasonFormat, actual, "less or equal", max)
		} else {
			result.IsCriteriaMet = false
			result.Reason = fmt.Sprintf(reasonFormat, actual, "greater", max)
		}
		result.MinOrMaxValue = max
		break
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

// errorRate returns the percentage of errors during the given offset.
func errorRate(ctx context.Context, provider metrics.Metrics, query metrics.Query, offset time.Duration) (float64, error) {
	rate, err := provider.ErrorRate(ctx, query, offset)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get error rate metrics")
	}

	// Multiply rate by 100 to have a percentage.
	return rate * 100, nil
}

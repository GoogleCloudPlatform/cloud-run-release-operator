package health

import (
	"context"
	"time"

	"github.com/GoogleCloudPlatform/cloud-run-release-operator/internal/metrics"
	"github.com/GoogleCloudPlatform/cloud-run-release-operator/internal/util"
	"github.com/GoogleCloudPlatform/cloud-run-release-operator/pkg/config"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Diagnosis is a possible result after a diagnosis.
type Diagnosis int

// Possible diagnosis results.
const (
	Unknown Diagnosis = iota
	Inconclusive
	Healthy
	Unhealthy
)

// Diagnose attempts to determine the health of a revision.
//
// If no health criteria is specified or the size of the health criteria and the
// actual values are not the same, the diagnosis is Unknown and an error is
// returned.
//
// If the minimum number of requests is not met, then health cannot be
// determined and diagnosis is Inconclusive.
//
// Otherwise, all metrics criteria are checked to determine whether the revision
// is healthy or not.
func Diagnose(ctx context.Context, healthCriteria []config.Metric, actualValues []float64) (Diagnosis, error) {
	logger := util.LoggerFromContext(ctx)
	if len(healthCriteria) != len(actualValues) {
		return Unknown, errors.New("the size of health criteria is not the same to the size of the actual metrics values")
	}
	if len(healthCriteria) == 0 {
		return Unknown, errors.New("health criteria must be specified")
	}

	diagnosis := Healthy
	for i, value := range actualValues {
		criteria := healthCriteria[i]
		logger := logger.WithFields(logrus.Fields{
			"metricsType": criteria.Type,
			"percentile":  criteria.Percentile,
			"threshold":   criteria.Threshold,
			"actualValue": value,
		})

		if !isCriteriaMet(criteria.Type, criteria.Threshold, value) {
			logger.Debug("unmet criterion")
			diagnosis = Unhealthy
			continue
		}
		logger.Debug("met criterion")
	}

	return diagnosis, nil
}

// CollectMetrics gets a metrics value for each of the given health criteria and
// returns a result for each criterion.
func CollectMetrics(ctx context.Context, provider metrics.Provider, offset time.Duration, healthCriteria []config.Metric) ([]float64, error) {
	if len(healthCriteria) == 0 {
		return nil, errors.New("health criteria must be specified")
	}
	var results []float64
	for _, criteria := range healthCriteria {
		var value float64
		var err error

		switch criteria.Type {
		case config.LatencyMetricsCheck:
			value, err = latency(ctx, provider, offset, criteria.Percentile)
			break
		case config.ErrorRateMetricsCheck:
			value, err = errorRatePercent(ctx, provider, offset)
			break
		default:
			return nil, errors.Errorf("unimplemented metrics %q", criteria.Type)
		}

		if err != nil {
			return nil, errors.Wrapf(err, "failed to obtain metrics %q", criteria.Type)
		}
		results = append(results, value)
	}

	return results, nil
}

// isCriteriaMet concludes if metrics criteria was met.
func isCriteriaMet(metricsType config.MetricsCheck, threshold float64, actualValue float64) bool {
	// As of now, the supported health criteria (latency and error rate) need to
	// be less than the threshold. So, this is sufficient for now but might need
	// to change to a switch statement when criteria with a minimum threshold is
	// added.
	if actualValue <= threshold {
		return true
	}
	return false
}

// latency returns the latency for the given offset and percentile.
func latency(ctx context.Context, provider metrics.Provider, offset time.Duration, percentile float64) (float64, error) {
	alignerReducer, err := metrics.PercentileToAlignReduce(percentile)
	if err != nil {
		return 0, errors.Wrap(err, "failed to parse percentile")
	}

	logger := util.LoggerFromContext(ctx).WithField("percentile", percentile)
	logger.Debug("querying for latency metrics")
	latency, err := provider.Latency(ctx, offset, alignerReducer)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get latency metrics")
	}
	logger.WithField("value", latency).Debug("latency successfully retrieved")

	return latency, nil
}

// errorRatePercent returns the percentage of errors during the given offset.
func errorRatePercent(ctx context.Context, provider metrics.Provider, offset time.Duration) (float64, error) {
	logger := util.LoggerFromContext(ctx)
	logger.Debug("querying for error rate metrics")
	rate, err := provider.ErrorRate(ctx, offset)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get error rate metrics")
	}

	// Multiply rate by 100 to have a percentage.
	rate *= 100
	logger.WithField("value", rate).Debug("error rate successfully retrieved")
	return rate, nil
}

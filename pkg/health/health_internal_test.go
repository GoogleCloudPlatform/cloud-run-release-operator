package health

import (
	"testing"

	"github.com/GoogleCloudPlatform/cloud-run-release-operator/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestIsCriteriaMet(t *testing.T) {
	tests := []struct {
		name        string
		metricsType config.MetricsCheck
		threshold   float64
		actualValue float64
		expected    bool
	}{
		{
			name:        "met request count",
			metricsType: config.RequestCountMetricsCheck,
			threshold:   1000,
			actualValue: 1000,
			expected:    true,
		},
		{
			name:        "met latency",
			metricsType: config.LatencyMetricsCheck,
			threshold:   750,
			actualValue: 500,
			expected:    true,
		},
		{
			name:        "met error rate",
			metricsType: config.ErrorRateMetricsCheck,
			threshold:   1,
			actualValue: 1,
			expected:    true,
		},
		{
			name:        "unmet request count",
			metricsType: config.RequestCountMetricsCheck,
			threshold:   1000,
			actualValue: 700,
		},
		{
			name:        "unmet latency",
			metricsType: config.LatencyMetricsCheck,
			threshold:   750,
			actualValue: 751,
		},
		{
			name:        "unmet error rate",
			metricsType: config.ErrorRateMetricsCheck,
			threshold:   1,
			actualValue: 1.01,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			isMet := isCriteriaMet(test.metricsType, test.threshold, test.actualValue)
			assert.Equal(t, test.expected, isMet)
		})
	}
}

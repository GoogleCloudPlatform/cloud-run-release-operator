package health_test

import (
	"testing"

	"github.com/GoogleCloudPlatform/cloud-run-release-operator/pkg/config"
	"github.com/GoogleCloudPlatform/cloud-run-release-operator/pkg/health"
	"github.com/stretchr/testify/assert"
)

func TestHealthReportAnnotation(t *testing.T) {
	tests := []struct {
		name           string
		healthCriteria []config.Metric
		diagnosis      health.Diagnosis
		expected       string
	}{
		{
			name: "single metrics",
			healthCriteria: []config.Metric{
				{Type: config.LatencyMetricsCheck, Percentile: 99, Threshold: 750},
			},
			diagnosis: health.Diagnosis{
				OverallResult: health.Unhealthy,
				CheckResults: []health.CheckResult{
					{Threshold: 750, ActualValue: 1000, IsCriteriaMet: true},
				},
			},
			expected: `{"diagnosis":"unhealthy","checkResults":[` +
				`{"actualValue":1000,"isCriteriaMet":true,"metricsType":"request-latency","percentile":99,"threshold":750}]}`,
		},
		{
			name: "more than one metrics",
			healthCriteria: []config.Metric{
				{Type: config.LatencyMetricsCheck, Percentile: 99, Threshold: 750},
				{Type: config.ErrorRateMetricsCheck, Threshold: 5},
			},
			diagnosis: health.Diagnosis{
				OverallResult: health.Healthy,
				CheckResults: []health.CheckResult{
					{Threshold: 750, ActualValue: 500, IsCriteriaMet: true},
					{Threshold: 5, ActualValue: 2, IsCriteriaMet: true},
				},
			},
			expected: `{"diagnosis":"healthy","checkResults":[` +
				`{"actualValue":500,"isCriteriaMet":true,"metricsType":"request-latency","percentile":99,"threshold":750},` +
				`{"actualValue":2,"isCriteriaMet":true,"metricsType":"error-rate-percent","threshold":5}]}`,
		},
		{
			name:     "no metrics",
			expected: `{"diagnosis":"unknown","checkResults":null}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			report, _ := health.JSONReport(test.healthCriteria, test.diagnosis)
			assert.Equal(t, test.expected, report)
		})
	}
}

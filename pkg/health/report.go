package health

import (
	"fmt"

	"github.com/GoogleCloudPlatform/cloud-run-release-operator/pkg/config"
)

// StringReport returns a human-readable report of the diagnosis.
//
// The returned string has the format:
// status: healthy
// metrics:
// - request-latency[p99]: 500.00 (needs 750.0)
// - request-count: 800 (needs 1000)
func StringReport(healthCriteria []config.Metric, diagnosis Diagnosis) string {
	report := fmt.Sprintf("status: %s\n", diagnosis.OverallResult.String())
	report += "metrics:"
	for i, result := range diagnosis.CheckResults {
		criteria := healthCriteria[i]

		// Include percentile value for latency criteria.
		if criteria.Type == config.LatencyMetricsCheck {
			report += fmt.Sprintf("\n- %s[p%.0f]: %.2f (needs %.2f)", criteria.Type, criteria.Percentile, result.ActualValue, criteria.Threshold)
			continue
		}

		format := "\n- %s: %.2f (needs %.2f)"
		if criteria.Type == config.RequestCountMetricsCheck {
			// No decimals for request count.
			format = "\n- %s: %.0f (needs %.0f)"
		}
		report += fmt.Sprintf(format, criteria.Type, result.ActualValue, criteria.Threshold)
	}

	return report
}

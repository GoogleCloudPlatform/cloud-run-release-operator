package health

import (
	"encoding/json"
	"fmt"

	"github.com/GoogleCloudPlatform/cloud-run-release-operator/pkg/config"
	"github.com/pkg/errors"
)

// StringReport report generates a human-readable report of the diagnosis.
//
// The returned string has the format:
//
// last status: healthy
// metrics:
// - request-latency[p99]: 500.00 (threshold 750.0)
// - request-count: 800 (threshold 1000)
func StringReport(healthCriteria []config.Metric, diagnosis Diagnosis) string {
	report := fmt.Sprintf("last status: %s\n", diagnosis.OverallResult.String())
	report += "metrics:"
	for i, result := range diagnosis.CheckResults {
		criteria := healthCriteria[i]

		// Include percentile value for latency criteria.
		if criteria.Type == config.LatencyMetricsCheck {
			report += fmt.Sprintf("\n- %s[p%.0f]: %.2f (threshold %.2f)", criteria.Type, criteria.Percentile, result.ActualValue, criteria.Threshold)
			continue
		}

		format := "\n- %s: %.2f (threshold %.2f)"
		if criteria.Type == config.RequestCountMetricsCheck {
			// No decimals for request count.
			format = "\n- %s: %.0f (threshold %.0f)"
		}
		report += fmt.Sprintf(format, criteria.Type, result.ActualValue, criteria.Threshold)
	}

	return report
}

// JSONReport returns a JSON representation of the diagnosis.
//
// TODO: consider if this function is useful (e.g. for a dashboard) since it's
// not used anywhere yet.
func JSONReport(healthCriteria []config.Metric, diagnosis Diagnosis) (string, error) {
	var resultsMap []map[string]interface{}
	for i, result := range diagnosis.CheckResults {
		criteria := healthCriteria[i]
		report := map[string]interface{}{
			"metricsType":   criteria.Type,
			"threshold":     result.Threshold,
			"actualValue":   result.ActualValue,
			"isCriteriaMet": result.IsCriteriaMet,
		}
		if criteria.Type == config.LatencyMetricsCheck {
			report["percentile"] = criteria.Percentile
		}
		resultsMap = append(resultsMap, report)
	}

	report := struct {
		Diagnosis    string                   `json:"diagnosis"`
		CheckResults []map[string]interface{} `json:"checkResults"`
	}{
		Diagnosis:    diagnosis.OverallResult.String(),
		CheckResults: resultsMap,
	}

	reportJSON, err := json.Marshal(report)
	return string(reportJSON), errors.Wrap(err, "failed to marshal report")
}

package health

import (
	"encoding/json"

	"github.com/GoogleCloudPlatform/cloud-run-release-operator/pkg/config"
	"github.com/pkg/errors"
)

// JSONReport returns a JSON representation of the diagnosis.
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

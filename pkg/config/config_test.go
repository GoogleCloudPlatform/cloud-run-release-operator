package config_test

import (
	"testing"

	"github.com/GoogleCloudPlatform/cloud-run-release-operator/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestIsValid(t *testing.T) {
	tests := []struct {
		name     string
		config   *config.Config
		cliMode  bool
		expected bool
	}{
		{
			name: "correct config with label selector",
			config: config.WithValues([]*config.Target{
				config.NewTarget("myproject", []string{"us-east1", "us-west1"}, "team=backend"),
			}, []int64{5, 30, 60}, 20, nil),
			expected: true,
		},
		{
			name: "missing project",
			config: config.WithValues([]*config.Target{
				config.NewTarget("", []string{"us-east1", "us-west1"}, "team=backend"),
			}, []int64{5, 30, 60}, 20, nil),
			expected: false,
		},
		{
			name: "missing steps",
			config: config.WithValues([]*config.Target{
				config.NewTarget("myproject", []string{"us-east1", "us-west1"}, "team=backend"),
			}, []int64{}, 20, nil),
			cliMode:  true,
			expected: false,
		},
		{
			name: "steps not in order",
			config: config.WithValues([]*config.Target{
				config.NewTarget("myproject", []string{"us-east1", "us-west1"}, "team=backend"),
			}, []int64{30, 30, 60}, 20, nil),
			cliMode:  true,
			expected: false,
		},
		{
			name: "step greater than 100",
			config: config.WithValues([]*config.Target{
				config.NewTarget("myproject", []string{"us-east1", "us-west1"}, "team=backend"),
			}, []int64{5, 30, 101}, 20, nil),
			expected: false,
		},
		{
			name: "no interval for cli mode",
			config: config.WithValues([]*config.Target{
				config.NewTarget("myproject", []string{"us-east1", "us-west1"}, "team=backend"),
			}, []int64{5, 30, 60}, 0, nil),
			cliMode:  true,
			expected: false,
		},
		{
			name: "empty label selector",
			config: config.WithValues([]*config.Target{
				config.NewTarget("myproject", []string{"us-east1", "us-west1"}, ""),
			}, []int64{5, 30, 60}, 20, nil),
			expected: false,
		},
		{
			name: "invalid error rate in metrics",
			config: config.WithValues([]*config.Target{
				config.NewTarget("myproject", []string{"us-east1", "us-west1"}, "team=backend"),
			}, []int64{5, 30, 60}, 20,
				[]config.Metric{
					{Type: config.ErrorRateMetricsCheck, Threshold: 101},
				},
			),
			expected: false,
		},
		{
			name: "invalid latency percentile",
			config: config.WithValues([]*config.Target{
				config.NewTarget("myproject", []string{"us-east1", "us-west1"}, "team=backend"),
			}, []int64{5, 30, 60}, 20,
				[]config.Metric{
					{Type: config.LatencyMetricsCheck, Percentile: 98},
				},
			),
			expected: false,
		},
		{
			name: "invalid latency value",
			config: config.WithValues([]*config.Target{
				config.NewTarget("myproject", []string{"us-east1", "us-west1"}, "team=backend"),
			}, []int64{5, 30, 60}, 20,
				[]config.Metric{
					{Type: config.LatencyMetricsCheck, Percentile: 99, Threshold: -1},
				},
			),
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			isValid, _ := test.config.IsValid(test.cliMode)
			assert.Equal(t, test.expected, isValid)
		})
	}
}

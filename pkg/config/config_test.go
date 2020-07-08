package config_test

import (
	"testing"

	"github.com/GoogleCloudPlatform/cloud-run-release-operator/pkg/config"
	"github.com/stretchr/testify/assert"
)

var yaml1 = `
metadata:
  project: "test"
  service: "hello"
  region: "us-east1"
rollout:
  steps: [5, 30, 60]
  interval: 300`

var yaml2 = `
metadata:
  service: "hello"
  region: "us-east1"
rollout:
  steps: [5, 30, 60]
  interval: 300`

var yaml3 = `
metadata:
  project: "test"
  service: "hello"
  region: "us-east1"
rollout:
  interval: 300`

var yaml4 = `
metadata:
  project: "test"
  service: "hello"
  region: "us-east1"
rollout:
  steps: [30, 30, 60]
  interval: 300`

var yaml5 = `
metadata:
  project: "test"
service: "hello"
  region: "us-east1"
  rollout:
steps: [5, 30, 105]`

var yaml6 = `
metadata:
  project: "test"
  service: "hello"
  region: "us-east1"
rollout:
  steps: [5, 30, 60]`

func TestDecode(t *testing.T) {
	tests := []struct {
		name      string
		data      []byte
		cliMode   bool
		expected  *config.Config
		shouldErr bool
	}{
		{
			name:    "correct config file",
			data:    []byte(yaml1),
			cliMode: true,
			expected: &config.Config{
				Metadata: &config.Metadata{
					Project: "test",
					Service: "hello",
					Region:  "us-east1",
				},
				Rollout: &config.Rollout{
					Steps:    []int64{5, 30, 60},
					Interval: 300,
				},
			},
			shouldErr: false,
		},
		// No project.
		{
			name:      "missing project",
			data:      []byte(yaml2),
			cliMode:   true,
			expected:  nil,
			shouldErr: true,
		},
		// No steps
		{
			name:      "missing steps",
			data:      []byte(yaml3),
			cliMode:   true,
			expected:  nil,
			shouldErr: true,
		},
		// Steps are not in ascending order.
		{
			name:      "steps not in order",
			data:      []byte(yaml4),
			cliMode:   true,
			expected:  nil,
			shouldErr: true,
		},
		// A step is greater than 100.
		{
			name:      "step greater than 100",
			data:      []byte(yaml5),
			cliMode:   true,
			expected:  nil,
			shouldErr: true,
		},
		// No interval for CLI mode.
		{
			name:      "no interval for cli mode",
			data:      []byte(yaml6),
			cliMode:   true,
			expected:  nil,
			shouldErr: true,
		},
	}

	for _, test := range tests {
		config, err := config.Decode(test.data, test.cliMode)
		if test.shouldErr {
			assert.NotNil(t, err)
			continue
		}

		assertConfig(t, test.expected, config)
	}
}

func assertConfig(t *testing.T, expected, actual *config.Config) {
	assert.Equal(t, expected.Metadata.Project, actual.Metadata.Project)
	assert.Equal(t, expected.Metadata.Service, actual.Metadata.Service)
	assert.Equal(t, expected.Metadata.Region, actual.Metadata.Region)

	assert.Equal(t, expected.Rollout.Steps, actual.Rollout.Steps)
	assert.Equal(t, expected.Rollout.Interval, actual.Rollout.Interval)
}

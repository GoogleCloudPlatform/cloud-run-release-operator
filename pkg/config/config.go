package config

import (
	"encoding/json"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

// Metadata is the information on the service to be managed.
type Metadata struct {
	Project string `json:"project" yaml:"project"`
	Service string `json:"service" yaml:"service"`
	Region  string `json:"region"  yaml:"region"`
}

// Rollout is the steps and configuration for rollout.
type Rollout struct {
	Steps    []int64 `json:"steps" yaml:"steps"`
	Interval int64   `yaml:"interval"`
}

// Config contains the configuration for a managed rollout.
type Config struct {
	Metadata *Metadata `json:"metadata" yaml:"metadata"`
	Rollout  *Rollout  `json:"rollout" yaml:"rollout"`
}

// Decode returns the configuration struct based on the data provided by the
// operator.
func Decode(data []byte, cliMode bool) (*Config, error) {
	config := &Config{}

	if cliMode {
		err := yaml.Unmarshal(data, config)
		if err != nil {
			return nil, errors.Wrap(err, "could not unmarshal data")
		}
	} else {
		err := json.Unmarshal(data, &config)
		if err != nil {
			return nil, errors.Wrap(err, "could not unmarshal data")
		}
	}

	if !isValid(config, cliMode) {
		return nil, errors.New("invalid configuration")
	}

	return config, nil
}

func isValid(config *Config, cliMode bool) bool {
	if config.Metadata.Project == "" ||
		config.Metadata.Service == "" ||
		config.Metadata.Region == "" {

		return false
	}

	if cliMode && config.Rollout.Interval <= 0 {
		return false
	}

	if len(config.Rollout.Steps) == 0 {
		return false
	}

	// Steps must be in ascending order.
	var previous int64
	for _, step := range config.Rollout.Steps {
		if previous >= step {
			return false
		}
		previous = step
	}

	return true
}

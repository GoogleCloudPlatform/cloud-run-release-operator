package service

import (
	runapi "github.com/GoogleCloudPlatform/cloud-run-release-operator/internal/run"
	"github.com/GoogleCloudPlatform/cloud-run-release-operator/pkg/config"
)

// Config is the configuration for a single service in a specific region.
type Config struct {
	Project     string
	ServiceName string
	Region      string
	Strategy    *config.Strategy
}

// Client is the connection to update and obtain data about a single service.
type Client struct {
	RunClient runapi.Client
	Config    *Config
}

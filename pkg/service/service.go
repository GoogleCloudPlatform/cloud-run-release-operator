package service

import (
	runapi "github.com/GoogleCloudPlatform/cloud-run-release-operator/internal/run"
)

// Service is the information about a single service.
type Service struct {
	Project string
	Name    string
	Region  string
}

// Client is the connection to update and obtain data about a single service.
type Client struct {
	RunClient runapi.Client
	Service   *Service
}

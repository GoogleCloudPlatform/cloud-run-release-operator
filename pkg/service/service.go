package service

import (
	"context"

	runapi "github.com/GoogleCloudPlatform/cloud-run-release-operator/internal/run"
	"github.com/GoogleCloudPlatform/cloud-run-release-operator/pkg/config"
	"google.golang.org/api/run/v1"
)

// Client is the connection to update and obtain data about a single service. It
// contains a client to the Cloud Run API, information about the service and the
// service object.
type Client struct {
	RunClient   runapi.Client
	Service     *run.Service
	Project     string
	ServiceName string
	Region      string
}

// NewClient initializes a service client.
func NewClient(runclient runapi.Client, svc *run.Service, project, serviceName, region string) *Client {
	return &Client{
		RunClient:   runclient,
		Service:     svc,
		Project:     project,
		ServiceName: serviceName,
		Region:      region,
	}
}

// Filter retrieves all the services that match the filter in the configuration.
func Filter(cfg *config.Config) []*Client {
	clientsCh := make(chan []*Client)
	numberOfFilterByRegionCalls := 0
	for _, target := range cfg.Targets {
		regions := target.Regions
		if len(target.Regions) == 0 {
			regions = runapi.Regions
		}

		for _, region := range regions {
			go filterByRegion(clientsCh, target.Project, region, target.Selector)
			numberOfFilterByRegionCalls++
		}
	}

	var svcClients []*Client
	for numberOfFilterByRegionCalls > 0 {
		clients := <-clientsCh
		svcClients = append(svcClients, clients...)

		numberOfFilterByRegionCalls--
	}

	close(clientsCh)

	return svcClients
}

// filterByRegion searches for a service name or label selector in an specific
// regional endpoint.
func filterByRegion(clientsCh chan []*Client, project, region string, selector config.TargetSelector) {
	runclient, err := runapi.NewAPIClient(context.Background(), region)
	if err != nil {
		clientsCh <- nil
		return
	}

	switch selector.Type {
	case config.ServiceNameType:
		svc, err := runclient.Service(project, selector.Filter)
		if svc == nil || err != nil {
			clientsCh <- nil
			return
		}

		clientsCh <- []*Client{
			NewClient(runclient, svc, project, svc.Metadata.Name, region),
		}
		break
	case config.LabelSelectorType:
		var clients []*Client
		svcs, err := runclient.ListServices(project, selector.Filter)
		if svcs == nil || err != nil {
			clientsCh <- nil
			return
		}

		for _, svc := range svcs.Items {
			client := NewClient(runclient, svc, project, svc.Metadata.Name, region)
			clients = append(clients, client)
		}
		clientsCh <- clients
	default:
		clientsCh <- nil
	}
}

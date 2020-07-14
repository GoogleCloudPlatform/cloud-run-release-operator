package service

import (
	"context"

	runapi "github.com/GoogleCloudPlatform/cloud-run-release-operator/internal/run"
	"github.com/GoogleCloudPlatform/cloud-run-release-operator/pkg/config"
	"github.com/sirupsen/logrus"
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
func Filter(logger *logrus.Logger, cfg *config.Config) []*Client {
	clientsCh := make(chan []*Client)
	numberOfFilterByRegionCalls := 0
	for _, target := range cfg.Targets {
		regions := target.Regions

		// If regions are not specified in configuration, query all regions.
		if len(target.Regions) == 0 {
			var err error
			regions, err = runapi.Locations(target.Project)
			if err != nil {
				logger.Errorf("Cannot get all regions: %v", err)
				continue
			}
		}

		for _, region := range regions {
			go filterByRegion(clientsCh, logger, target.Project, region, target.Selector)
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
func filterByRegion(clientsCh chan []*Client, logger *logrus.Logger, project, region string, selector config.TargetSelector) {
	lg := logger.WithFields(logrus.Fields{
		"project":        project,
		"region":         region,
		"selectorType":   selector.Type,
		"selectorFilter": selector.Filter,
	})

	runclient, err := runapi.NewAPIClient(context.Background(), region)
	if err != nil {
		lg.Errorf("can not initialize Cloud Run client: %v", err)
		clientsCh <- nil
		return
	}

	switch selector.Type {
	case config.ServiceNameType:
		svc, err := runclient.Service(project, selector.Filter)
		if err != nil {
			lg.Errorf("failed to obtain information on service %q: %v", selector.Filter, err)
			clientsCh <- nil
			return
		}
		if svc == nil {
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
		if err != nil {
			lg.Errorf("failed to filter services with label %q: %v", selector.Filter, err)
			clientsCh <- nil
			return
		}
		if svcs == nil {
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

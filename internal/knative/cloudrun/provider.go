package cloudrun

import (
	"context"
	"fmt"

	"github.com/GoogleCloudPlatform/cloud-run-release-manager/internal/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/option"
	"google.golang.org/api/run/v1"
)

// Provider is a KServiceProvider for Cloud Run (gke or fully managed).
type Provider struct {
	apiService *run.APIService
	project    string
	platform   string

	// Managed
	region string

	// GKE
	clusterLocation string
	clusterName     string
}

// regions are the available regions.
//
// TODO: caching regions might be unnecessary if we are querying them once during
// the lifespan of the process.
var regions = []string{}

// NewFullyManagedProvider returns a provider for Cloud Run fully managed.
func NewFullyManagedProvider(ctx context.Context, project, region string) (*Provider, error) {
	regionalEndpoint := fmt.Sprintf("https://%s-run.googleapis.com/", region)
	apiService, err := run.NewService(ctx, option.WithEndpoint(regionalEndpoint))
	if err != nil {
		return nil, errors.Wrap(err, "could not initialize client for the Cloud Run API")
	}

	return &Provider{
		apiService: apiService,
		project:    project,
		platform:   "managed",
		region:     region,
	}, nil
}

// NewGKEProvider returns a provider for Cloud Run for Anthos.
func NewGKEProvider(ctx context.Context, project, zone, clusterName string) (*Provider, error) {
	hClient, endpoint, err := newGKEClient(ctx, project, zone, clusterName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to initialize client to GKE cluster %s, zone %s", clusterName, zone)
	}

	apiService, err := run.NewService(ctx,
		option.WithHTTPClient(hClient),
		option.WithEndpoint(endpoint),
	)
	if err != nil {
		return nil, errors.Wrap(err, "could not initialize client for the Cloud Run API")
	}

	return &Provider{
		apiService:      apiService,
		project:         project,
		platform:        "gke",
		clusterLocation: zone,
		clusterName:     clusterName,
	}, nil
}

// ReplaceService replaces an existing service.
func (p *Provider) ReplaceService(namespace, serviceID string, svc *run.Service) (*run.Service, error) {
	serviceName := serviceName(namespace, serviceID)
	return p.apiService.Namespaces.Services.ReplaceService(serviceName, svc).Do()
}

// ServicesWithLabelSelector gets services filtered by a label selector.
func (p *Provider) ServicesWithLabelSelector(namespace string, labelSelector string) ([]*run.Service, error) {
	parent := fmt.Sprintf("namespaces/%s", namespace)
	servicesList, err := p.apiService.Namespaces.Services.List(parent).LabelSelector(labelSelector).Do()
	if err != nil {
		return nil, errors.Wrap(err, "failed to filter services by label selector")
	}

	return servicesList.Items, nil
}

// LoggingFields returns the logging fields related to this provider.
func (p *Provider) LoggingFields() logrus.Fields {
	switch p.platform {
	case "gke":
		return logrus.Fields{
			"project":         p.project,
			"clusterLocation": p.clusterLocation,
			"clusterName":     p.clusterName,
		}
	case "managed":
		return logrus.Fields{
			"project": p.project,
			"region":  p.region,
		}
	default:
		return logrus.Fields{}
	}
}

// Regions gets the supported regions for the project (for Cloud Run fully
// managed).
func Regions(ctx context.Context, project string) ([]string, error) {
	logger := util.LoggerFrom(ctx)
	if len(regions) != 0 {
		logger.Debug("using cached regions, skip querying from API")
		return regions, nil
	}

	client, err := run.NewService(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not initialize client for the Cloud Run API")
	}

	name := fmt.Sprintf("projects/%s", project)
	resp, err := client.Projects.Locations.List(name).Do()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get locations")
	}

	for _, location := range resp.Locations {
		regions = append(regions, location.LocationId)
	}
	return regions, nil
}

// serviceName returns the name of the specified service. It returns the
// form namespaces/{namespace_id}/services/{service_id}.
//
// For Cloud Run (fully managed), the namespace is the project ID or number.
func serviceName(namespace, serviceID string) string {
	return fmt.Sprintf("namespaces/%s/services/%s", namespace, serviceID)
}

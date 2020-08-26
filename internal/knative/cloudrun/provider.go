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

// provider contains shared properties between Cloud Run providers.
type provider struct {
	apiService *run.APIService
	project    string
}

// FullyManagedProvider is a Knative Provider for Cloud Run fully managed.
type FullyManagedProvider struct {
	provider
	region string
}

// GKEProvider is a Knative Provider for Cloud Run for Anthos.
type GKEProvider struct {
	provider
	clusterLocation string
	clusterName     string
}

// fullyManagedRegionsCache are the available regions.
//
// TODO: caching regions might be unnecessary if we are querying them once during
// the lifespan of the process.
var fullyManagedRegionsCache = []string{}

// NewFullyManagedProvider returns a provider for Cloud Run fully managed.
func NewFullyManagedProvider(ctx context.Context, project, region string) (*FullyManagedProvider, error) {
	regionalEndpoint := fmt.Sprintf("https://%s-run.googleapis.com/", region)
	apiService, err := run.NewService(ctx, option.WithEndpoint(regionalEndpoint))
	if err != nil {
		return nil, errors.Wrap(err, "could not initialize client for the Cloud Run API")
	}

	return &FullyManagedProvider{
		provider: provider{
			apiService: apiService,
			project:    project,
		},
		region: region,
	}, nil
}

// NewGKEProvider returns a provider for Cloud Run for Anthos.
func NewGKEProvider(ctx context.Context, project, zone, clusterName string) (*GKEProvider, error) {
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

	return &GKEProvider{
		provider: provider{
			apiService: apiService,
			project:    project,
		},
		clusterLocation: zone,
		clusterName:     clusterName,
	}, nil
}

// ReplaceService replaces an existing service.
func (p *FullyManagedProvider) ReplaceService(namespace, serviceID string, svc *run.Service) (*run.Service, error) {
	return replaceService(p.apiService, namespace, serviceID, svc)
}

// ListServices gets services filtered by a label selector.
func (p *FullyManagedProvider) ListServices(namespace, labelSelector string) ([]*run.Service, error) {
	return listServices(p.apiService, namespace, labelSelector)
}

// LoggingFields returns the logging fields related to this provider.
func (p *FullyManagedProvider) LoggingFields() logrus.Fields {
	return logrus.Fields{
		"project": p.project,
		"region":  p.region,
	}
}

// ReplaceService replaces an existing service.
func (p *GKEProvider) ReplaceService(namespace, serviceID string, svc *run.Service) (*run.Service, error) {
	return replaceService(p.apiService, namespace, serviceID, svc)
}

// ListServices gets services filtered by a label selector.
func (p *GKEProvider) ListServices(namespace, labelSelector string) ([]*run.Service, error) {
	return listServices(p.apiService, namespace, labelSelector)
}

// LoggingFields returns the logging fields related to this provider.
func (p *GKEProvider) LoggingFields() logrus.Fields {
	return logrus.Fields{
		"project":         p.project,
		"clusterLocation": p.clusterLocation,
		"clusterName":     p.clusterName,
	}
}

func replaceService(apiSvc *run.APIService, namespace, serviceID string, svc *run.Service) (*run.Service, error) {
	serviceName := serviceName(namespace, serviceID)
	return apiSvc.Namespaces.Services.ReplaceService(serviceName, svc).Do()
}

func listServices(apiSvc *run.APIService, namespace, labelSelector string) ([]*run.Service, error) {
	parent := fmt.Sprintf("namespaces/%s", namespace)
	servicesList, err := apiSvc.Namespaces.Services.List(parent).LabelSelector(labelSelector).Do()
	if err != nil {
		return nil, errors.Wrap(err, "failed to filter services by label selector")
	}

	return servicesList.Items, nil
}

// serviceName returns the name of the specified service. It returns the
// form namespaces/{namespace_id}/services/{service_id}.
//
// For Cloud Run (fully managed), the namespace is the project ID or number.
func serviceName(namespace, serviceID string) string {
	return fmt.Sprintf("namespaces/%s/services/%s", namespace, serviceID)
}

// FullyManagedRegions gets the supported regions for the project.
func FullyManagedRegions(ctx context.Context, project string) ([]string, error) {
	logger := util.LoggerFrom(ctx)
	if len(fullyManagedRegionsCache) != 0 {
		logger.Debug("using cached regions, skip querying from API")
		return fullyManagedRegionsCache, nil
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
		fullyManagedRegionsCache = append(fullyManagedRegionsCache, location.LocationId)
	}
	return fullyManagedRegionsCache, nil
}

package stackdriver

import (
	"context"
	"fmt"
	"time"

	"github.com/GoogleCloudPlatform/cloud-run-release-manager/internal/metrics"
	"github.com/pkg/errors"

	// TODO: Migrate to cloud.google.com/go/monitoring/apiv3/v2 once RPC for MQL
	// query is added (https://cloud.google.com/monitoring/api/ref_v3/rest/v3/projects.timeSeries/query).
	monitoring "google.golang.org/api/monitoring/v3"
)

// query is the filter used to retrieve metrics data.
type query string

// Provider is a metrics provider for Cloud Monitoring.
type Provider struct {
	metricsClient *monitoring.Service
	project       string

	// The platform the Cloud Run service is running on.
	platform

	// query is used to filter the metrics for the wanted resource.
	//
	// TODO: Use a different data structure to keep track of the filters and
	// build the query just before calling the API.
	query
}

// Metric types.
const (
	cloudRunManagedRequestLatencies = "run.googleapis.com/request_latencies"
	cloudRunManagedRequestCount     = "run.googleapis.com/request_count"
	gkeRequestLatencies             = "knative.dev/serving/revision/request_latencies"
	gkeRequestCount                 = "knative.dev/serving/revision/request_count"
)

// NewProvider initializes the provider for Cloud Monitoring.
func NewProvider(ctx context.Context, project string, location string, serviceName string) (Provider, error) {
	client, err := monitoring.NewService(ctx)
	if err != nil {
		return Provider{}, errors.Wrap(err, "could not initialize Cloud Metics client")
	}

	return Provider{
		metricsClient: client,
		project:       project,
		platform:      managed{},
		query:         newQuery(project, location, serviceName),
	}, nil
}

// WithGKEPlatform updates the Cloud Run platform to GKE.
func (p Provider) WithGKEPlatform(namespace, clusterName string) Provider {
	p.query = p.addFilter("resource.label.namespace_name", namespace).
		addFilter("resource.label.cluster_name", clusterName)
	p.platform = gke{}
	return p
}

// SetCandidateRevision sets the candidate revision name for which the provider
// should get metrics.
func (p Provider) SetCandidateRevision(revisionName string) {
	p.query = p.query.addFilter("resource.labels.revision_name", revisionName)
}

// RequestCount count returns the number of requests for the given offset.
func (p Provider) RequestCount(ctx context.Context, offset time.Duration) (int64, error) {
	query := p.addFilter("metric.type", p.platform.RequestCountMetricType())
	return requestCount(ctx, p, query, offset)
}

// Latency returns the latency for the resource for the given offset.
// It returns 0 if no request was made during the interval.
func (p Provider) Latency(ctx context.Context, offset time.Duration, alignReduceType metrics.AlignReduce) (float64, error) {
	query := p.query.addFilter("metric.type", p.platform.RequestLatenciesMetricType())
	return latency(ctx, p, query, offset, alignReduceType)
}

// ErrorRate returns the rate of 5xx errors for the resource in the given offset.
// It returns 0 if no request was made during the interval.
func (p Provider) ErrorRate(ctx context.Context, offset time.Duration) (float64, error) {
	query := p.query.addFilter("metric.type", p.platform.RequestCountMetricType())
	return errorRate(ctx, p, query, offset)
}

// newQuery initializes a query.
func newQuery(project, location, serviceName string) query {
	var q query
	return q.addFilter("resource.labels.project_id", project).
		addFilter("resource.labels.location", location).
		addFilter("resource.labels.service_name", serviceName)
}

// addFilter adds a filter to the query.
//
// TODO: Support field-based filters, so the query string is generated based on
// the fields instead of appending a filter everytime this method is called.
func (q query) addFilter(key, value string) query {
	if q != "" {
		q += " AND "
	}
	q += query(fmt.Sprintf("%s=%q", key, value))

	return q
}

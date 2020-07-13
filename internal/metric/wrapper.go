package metric

import (
	"context"
	"fmt"
	"time"

	monitoring "cloud.google.com/go/monitoring/apiv3"
	"github.com/pkg/errors"
	"google.golang.org/api/iterator"
	monitoringpb "google.golang.org/genproto/googleapis/monitoring/v3"
	duration "google.golang.org/protobuf/types/known/durationpb"
	timestamp "google.golang.org/protobuf/types/known/timestamppb"
)

// Client represents a wrapper around the Cloud Monitoring package.
type Client interface {
	Latency(ctx context.Context, filter Filter, startTime time.Time) (float64, error)
}

// API is a wrapper for the Cloud Monitoring package.
type API struct {
	Client  *monitoring.MetricClient
	Project string
}

// Filter is the filter for the query.
type Filter struct {
	service  string
	revision string
	query    string
}

// Metric types.
const (
	RequestLatencies = "run.googleapis.com/request_latencies"
	RequestCount     = "run.googleapis.com/request_count"
)

type alignReduce int32

// Series aligner and cross series reducer types (for latency).
const (
	Align99Reduce99 alignReduce = 1
	Align95Reduce95             = 2
	Align50Reduce50             = 3
)

// NewAPIClient initializes
func NewAPIClient(ctx context.Context, project string) (*API, error) {
	client, err := monitoring.NewMetricClient(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not initialize Cloud Metics client")
	}

	return &API{
		Client:  client,
		Project: project,
	}, nil
}

// Latency returns the latency for the resource matching the filter.
func (a *API) Latency(ctx context.Context, filter Filter, startTime time.Time, alignReduceType alignReduce) (float64, error) {
	filter = filter.Add("metric.type", RequestLatencies)
	endTime := time.Now()
	aligner, reducer := alignerAndReducer(alignReduceType)

	it := a.Client.ListTimeSeries(ctx, &monitoringpb.ListTimeSeriesRequest{
		Name:   "projects/" + a.Project,
		Filter: filter.query,
		Interval: &monitoringpb.TimeInterval{
			StartTime: timestamp.New(startTime),
			EndTime:   timestamp.New(endTime),
		},
		Aggregation: &monitoringpb.Aggregation{
			AlignmentPeriod:    duration.New(endTime.Sub(startTime)),
			PerSeriesAligner:   aligner,
			GroupByFields:      []string{"metric.labels.response_code_class"},
			CrossSeriesReducer: reducer,
		},
	})

	return latencyForCodeClass(it, "2xx")
}

// ServerErrorRate returns the rate of 5xx errors for the resource matching the
// filter.
func (a *API) ServerErrorRate(ctx context.Context, filter Filter, startTime time.Time) (float64, error) {
	filter = filter.Add("metric.type", RequestCount)
	endTime := time.Now()

	it := a.Client.ListTimeSeries(ctx, &monitoringpb.ListTimeSeriesRequest{
		Name:   "projects/" + a.Project,
		Filter: filter.query,
		Interval: &monitoringpb.TimeInterval{
			StartTime: timestamp.New(startTime),
			EndTime:   timestamp.New(endTime),
		},
		Aggregation: &monitoringpb.Aggregation{
			AlignmentPeriod:    duration.New(endTime.Sub(startTime)),
			PerSeriesAligner:   monitoringpb.Aggregation_ALIGN_DELTA,
			GroupByFields:      []string{"metric.labels.response_code_class"},
			CrossSeriesReducer: monitoringpb.Aggregation_REDUCE_SUM,
		},
	})

	return calculateErrorResponseRate(it)
}

// latencyForCodeClass retrieves the latency for a given response code class
// (e.g. 2xx, 5xx, etc.)
func latencyForCodeClass(it *monitoring.TimeSeriesIterator, codeClass string) (float64, error) {
	var latency float64
	for {
		series, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return 0, errors.Wrap(err, "error when iterating through time series")
		}

		// Because the interval and the series aligner are the same, only one
		// point is returned per time series.
		if series.Metric.Labels["response_code_class"] == codeClass {
			latency = series.Points[0].Value.GetDoubleValue()
		}
	}

	return latency, nil
}

// calculateErrorResponseRate calculates the percentage of 5xx error response.
//
// It obtains all the successful responses (2xx) and the error responses (5xx),
// add them up to form a 'total'. Then, it divides the number of error responses
// by the total.
// It ignores any other type of responses (e.g. 4xx).
func calculateErrorResponseRate(it *monitoring.TimeSeriesIterator) (float64, error) {
	var errorResponseCount, successfulResponseCount int64
	for {
		series, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return 0, errors.Wrap(err, "error when iterating through time series")
		}

		// Because the interval and the series aligner are the same, only one
		// point is returned per time series.
		switch series.Metric.Labels["response_code_class"] {
		case "2xx":
			successfulResponseCount += series.Points[0].Value.GetInt64Value()
			break
		case "5xx":
			errorResponseCount += series.Points[0].Value.GetInt64Value()
			break
		}
	}

	totalResponses := errorResponseCount + successfulResponseCount
	if totalResponses == 0 {
		return 0, errors.New("no requests in interval")
	}

	rate := float64(errorResponseCount) / float64(totalResponses)

	return rate, nil
}

func alignerAndReducer(alignReduceType alignReduce) (aligner monitoringpb.Aggregation_Aligner, reducer monitoringpb.Aggregation_Reducer) {
	switch alignReduceType {
	case Align99Reduce99:
		aligner = monitoringpb.Aggregation_ALIGN_PERCENTILE_99
		reducer = monitoringpb.Aggregation_REDUCE_PERCENTILE_99
		break
	case Align95Reduce95:
		aligner = monitoringpb.Aggregation_ALIGN_PERCENTILE_95
		reducer = monitoringpb.Aggregation_REDUCE_PERCENTILE_95
		break
	case Align50Reduce50:
		aligner = monitoringpb.Aggregation_ALIGN_PERCENTILE_50
		reducer = monitoringpb.Aggregation_REDUCE_PERCENTILE_50
		break
	}

	return aligner, reducer
}

// NewFilter initializes a filter for a query.
func NewFilter(serviceName, revisionName string) Filter {
	filter := Filter{}
	filter = filter.Add("resource.labels.service_name", serviceName)
	filter = filter.Add("resource.labels.revision_name", revisionName)

	return filter
}

// Add adds a filter for the query.
func (f Filter) Add(key string, value string) Filter {
	if f.query != "" {
		f.query += " AND "
	}
	f.query += fmt.Sprintf("%s=%q", key, value)

	return f
}

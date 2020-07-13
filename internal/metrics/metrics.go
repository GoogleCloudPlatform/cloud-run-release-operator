package metrics

import (
	"context"
	"time"
)

// Query is the filter used to retrieve metrics data.
type Query interface {
	Filter(key, value string)
	Query() string
}

// AlignReduce is the type to enumerate allowed combinations of per series
// aligner and cross series reducer.
type AlignReduce int32

// Series aligner and cross series reducer types (for latency).
const (
	Align99Reduce99 AlignReduce = 1
	Align95Reduce95             = 2
	Align50Reduce50             = 3
)

// Client represents a wrapper around the Cloud Monitoring package.
type Client interface {
	Latency(ctx context.Context, query Query, startTime time.Time, alignReduceType AlignReduce) (float64, error)
	ServerErrorRate(ctx context.Context, query Query, startTime time.Time) (float64, error)
}

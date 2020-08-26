package stackdriver

type platform interface {
	RequestLatenciesMetricType() string
	RequestCountMetricType() string
}

type managed struct{}
type gke struct{}

func (managed) RequestLatenciesMetricType() string {
	return "run.googleapis.com/request_latencies"
}

func (managed) RequestCountMetricType() string {
	return "run.googleapis.com/request_count"
}

func (gke) RequestLatenciesMetricType() string {
	return "knative.dev/serving/revision/request_latencies"
}

func (gke) RequestCountMetricType() string {
	return "knative.dev/serving/revision/request_count"
}

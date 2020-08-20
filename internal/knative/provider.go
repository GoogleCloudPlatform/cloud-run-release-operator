package knative

import (
	"github.com/sirupsen/logrus"
	"google.golang.org/api/run/v1"
)

// Provider represents a Knative client.
type Provider interface {
	ServicesWithLabelSelector(namespace string, labelSelector string) ([]*run.Service, error)
	ReplaceService(namespace, serviceID string, svc *run.Service) (*run.Service, error)

	// Returns the logging fields related to this provider.
	LoggingFields() logrus.Fields
}

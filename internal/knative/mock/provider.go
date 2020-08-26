package mock

import (
	"github.com/sirupsen/logrus"
	"google.golang.org/api/run/v1"
)

// Provider represents a mock implementation of knative.Provider.
type Provider struct {
	ListServicesFn      func(namespace, labelSelector string) ([]*run.Service, error)
	ListServicesInvoked bool

	ReplaceServiceFn      func(namespace, serviceID string, svc *run.Service) (*run.Service, error)
	ReplaceServiceInvoked bool

	LoggingFieldsFn      func() logrus.Fields
	LoggingFieldsInvoked bool
}

// ListServices invokes the mock implementation and marks the function as invoked.
func (p *Provider) ListServices(namespace, labelSelector string) ([]*run.Service, error) {
	p.ListServicesInvoked = true
	return p.ListServicesFn(namespace, labelSelector)
}

// ReplaceService invokes the mock implementation and marks the function as invoked.
func (p *Provider) ReplaceService(namespace, serviceID string, svc *run.Service) (*run.Service, error) {
	p.ReplaceServiceInvoked = true
	return p.ReplaceServiceFn(namespace, serviceID, svc)
}

// LoggingFields invokes the mock implementation and marks the function as invoked.
func (p *Provider) LoggingFields() logrus.Fields {
	p.LoggingFieldsInvoked = true
	return p.LoggingFieldsFn()
}

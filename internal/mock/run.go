package mock

import "google.golang.org/api/run/v1"

// Run represents a mock implementation of run.API.
type Run struct {
	ServiceFn      func(namespace, serviceID string) (*run.Service, error)
	ServiceInvoked bool

	ReplaceServiceFn      func(namespace, serviceID string, svc *run.Service) (*run.Service, error)
	ReplaceServiceInvoked bool
}

// Service invokes the mock implementation and marks the function as invoked.
func (r *Run) Service(namespace, service string) (*run.Service, error) {
	r.ServiceInvoked = true
	return r.ServiceFn(namespace, service)
}

// ReplaceService invokes the mock implementation and marks the function as invoked.
func (r *Run) ReplaceService(namespace, serviceID string, svc *run.Service) (*run.Service, error) {
	r.ReplaceServiceInvoked = true
	return r.ReplaceServiceFn(namespace, serviceID, svc)
}

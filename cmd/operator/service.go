package main

import (
	"context"
	"sync"

	"github.com/GoogleCloudPlatform/cloud-run-release-manager/internal/config"
	"github.com/GoogleCloudPlatform/cloud-run-release-manager/internal/knative"
	"github.com/GoogleCloudPlatform/cloud-run-release-manager/internal/knative/cloudrun"
	"github.com/GoogleCloudPlatform/cloud-run-release-manager/internal/rollout"
	"github.com/GoogleCloudPlatform/cloud-run-release-manager/internal/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/run/v1"
)

// getTargetedServices returned a list of service records that match the target
// configuration.
func getTargetedServices(ctx context.Context, target config.Target) ([]*rollout.ServiceRecord, error) {
	logger := util.LoggerFrom(ctx)
	logger.Debug("querying Cloud Run API to get all targeted services")
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var (
		retServices []*rollout.ServiceRecord
		retError    error
		mu          sync.Mutex
		wg          sync.WaitGroup
	)

	regions, err := determineRegions(ctx, target)
	if err != nil {
		return nil, errors.Wrap(err, "cannot determine regions")
	}

	for _, region := range regions {
		wg.Add(1)

		go func(ctx context.Context, logger *logrus.Entry, project, region, labelSelector string) {
			defer wg.Done()

			provider, err := cloudrun.NewFullyManagedProvider(ctx, project, region)
			if err != nil {
				retError = errors.Wrap(err, "failed to initialize Cloud Run fully managed client")
				cancel()
				return
			}

			svcs, err := getServicesByLabel(ctx, provider, project, labelSelector)
			if err != nil {
				retError = err
				cancel()
				return
			}

			for _, svc := range svcs {
				mu.Lock()
				retServices = append(retServices, newServiceRecord(svc, provider, project, region))
				mu.Unlock()
			}
		}(ctx, logger, target.Project, region, target.LabelSelector)
	}

	wg.Wait()
	return retServices, retError
}

// getServicesByLabel returns all the service records that match the label
// selector.
func getServicesByLabel(ctx context.Context, provider knative.Provider, namespace, labelSelector string) ([]*run.Service, error) {
	logger := util.LoggerFrom(ctx)
	lg := logger.WithFields(logrus.Fields{
		"labelSelector": labelSelector,
	})

	lg.Debug("querying for services in provider")
	svcs, err := provider.ServicesWithLabelSelector(namespace, labelSelector)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get services with label %q", labelSelector)
	}

	lg.WithField("n", len(svcs)).Debug("finished retrieving services from the API")
	return svcs, nil
}

// determineRegions gets the regions the label selector should be searched at.
//
// If the target configuration does not specify any regions, the entire list of
// regions is retrieved from API.
func determineRegions(ctx context.Context, target config.Target) ([]string, error) {
	logger := util.LoggerFrom(ctx)
	regions := target.Regions
	if len(regions) != 0 {
		logger.Debug("using predefined list of regions, skip querying from API")
		return regions, nil
	}

	logger.Debug("retrieving all regions from the API")
	regions, err := cloudrun.Regions(ctx, target.Project)
	if err != nil {
		return nil, errors.Wrap(err, "cannot get list of regions from Cloud Run API")
	}

	logger.WithField("n", len(regions)).Debug("finished retrieving regions from the API")
	return regions, nil
}

// newServiceRecord creates a new service record.
func newServiceRecord(svc *run.Service, provider knative.Provider, namespace, region string) *rollout.ServiceRecord {
	return &rollout.ServiceRecord{
		Service:   svc,
		KProvider: provider,
		Namespace: namespace,
		Region:    region,
	}
}

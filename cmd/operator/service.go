package main

import (
	"context"
	"sync"

	runapi "github.com/GoogleCloudPlatform/cloud-run-release-operator/internal/run"
	"github.com/GoogleCloudPlatform/cloud-run-release-operator/pkg/config"
	"github.com/GoogleCloudPlatform/cloud-run-release-operator/pkg/rollout"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// getTargetedServices returned a list of service records that match the target
// configuration.
func getTargetedServices(ctx context.Context, logger *logrus.Logger, targets []*config.Target) ([]*rollout.ServiceRecord, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var (
		retServices []*rollout.ServiceRecord
		retError    error
		mu          sync.Mutex
		wg          sync.WaitGroup
	)

	for _, target := range targets {
		regions, err := getRegions(target)
		if err != nil {
			logger.Errorf("Cannot get all regions: %v", err)
			continue
		}

		for _, region := range regions {
			wg.Add(1)

			go func(region, labelSelector string) {
				defer wg.Done()
				svcs, err := getServicesByRegionAndLabel(ctx, target.Project, region, target.LabelSelector)
				if err != nil {
					retError = err
					cancel()
					return
				}

				mu.Lock()
				retServices = append(retServices, svcs...)
				mu.Unlock()

			}(region, target.LabelSelector)
		}
	}

	wg.Wait()
	return retServices, retError
}

// getServicesByRegionAndLabel returns all the service records that match the
// labelSelector in a specific region.
func getServicesByRegionAndLabel(ctx context.Context, project, region, labelSelector string) ([]*rollout.ServiceRecord, error) {
	runclient, err := runapi.NewAPIClient(ctx, region)
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize Cloud Run client")
	}

	svcs, err := runclient.ServicesWithLabelSelector(project, labelSelector)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get services with label %q in region %q", labelSelector, region)
	}

	var records []*rollout.ServiceRecord
	for _, svc := range svcs {
		records = append(records, &rollout.ServiceRecord{
			Service: svc,
			Project: project,
			Region:  region,
		})
	}

	return records, nil
}

func getRegions(target *config.Target) ([]string, error) {
	regions := target.Regions
	if len(regions) == 0 {
		var err error
		regions, err = runapi.Regions(target.Project)
		if err != nil {
			return nil, errors.Wrap(err, "cannot get list of regions from Cloud Run API")
		}
	}

	return regions, nil
}

package rollout

import (
	"google.golang.org/api/run/v1"
)

// Annotations name for information related to the rollout.
const (
	StableRevisionAnnotation    = "rollout.cloud.run/stableRevision"
	CandidateRevisionAnnotation = "rollout.cloud.run/candidateRevision"
)

// StableRevision returns the stable revision of the Cloud Run service.
func StableRevision(svc *run.Service) string {
	stableRevision, ok := svc.Metadata.Annotations[StableRevisionAnnotation]
	if !ok {
		stableRevision = detectTrafficHandler(svc)
		if stableRevision == "" {
			return ""
		}

		return stableRevision
	}

	// In case the stable Revision in the annotation is not the one handling
	// 100% of the traffic, this recovers from this unexpected situation.
	trafficHandler := detectTrafficHandler(svc)
	if trafficHandler != "" && trafficHandler != stableRevision {
		stableRevision = trafficHandler
	}

	return stableRevision
}

// CandidateRevision attempts to deduce what revision could be considered
// a candidate.
func CandidateRevision(svc *run.Service, stable string) string {
	latestRevision := svc.Status.LatestReadyRevisionName
	if stable == latestRevision {
		return ""
	}

	return latestRevision
}

// detectTrafficHandler scans the service and retrieves a revision with 100%
// traffic.
func detectTrafficHandler(svc *run.Service) string {
	candidate := svc.Metadata.Annotations[CandidateRevisionAnnotation]
	for _, target := range svc.Status.Traffic {
		if target.Percent == 100 && target.RevisionName != candidate {
			return target.RevisionName
		}
	}

	return ""
}

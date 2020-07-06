package rollout

import (
	"fmt"

	runapi "github.com/GoogleCloudPlatform/cloud-run-release-operator/internal/run"
	"github.com/GoogleCloudPlatform/cloud-run-release-operator/pkg/config"
	format "github.com/logrusorgru/aurora"
	"github.com/pkg/errors"
	"google.golang.org/api/run/v1"
)

// Logger describes a logger printing information to the user.
type Logger interface {
	Println(args ...interface{})
	Printf(format string, args ...interface{})
}

// Rollout is the rollout manager.
type Rollout struct {
	RunClient        runapi.Client
	Config           *config.Config
	Log              Logger
	promoteCandidate bool
}

// Automatic tags.
const (
	StableTag    = "stable"
	CandidateTag = "candidate"
	LatestTag    = "latest"
)

// New returns a new rollout manager.
func New(client runapi.Client, config *config.Config, log Logger) *Rollout {
	return &Rollout{
		RunClient:        client,
		Config:           config,
		Log:              log,
		promoteCandidate: false,
	}
}

// Manage handles the gradual rollout.
func (r *Rollout) Manage() (*run.Service, error) {
	serviceName := generateServiceName(r.Config.Metadata.Project, r.Config.Metadata.Service)
	svc, err := r.RunClient.Service(serviceName)
	if err != nil {
		return nil, errors.Wrap(err, "could not get service")
	}
	if svc == nil {
		return nil, errors.Errorf("service %s does not exist", serviceName)
	}

	stable := DetectStableRevisionName(svc)
	if stable == "" {
		r.Log.Println("Could not determine stable revision")
		return nil, nil
	}

	candidate := DetectCandidateRevisionName(svc, stable)
	if candidate == "" {
		r.Log.Println("Could not determine candidate revision")
		return nil, nil
	}

	svc = r.SplitTraffic(svc, stable, candidate)
	svc = r.updateAnnotations(svc, stable, candidate)
	svc, err = r.RunClient.ReplaceService(serviceName, svc)
	if err != nil {
		return nil, errors.Wrap(err, "could not update service")
	}

	return svc, nil
}

// SplitTraffic changes the traffic configuration of the service.
//
// It creates a new traffic slice in case there's a new candidate revision.
// If we only drop the traffic of the previous candidate to 0, Cloud Run would
// still consider it as serving traffic and the slice of traffic targets would
// grow very large over time.
func (r *Rollout) SplitTraffic(svc *run.Service, stable, candidate string) *run.Service {
	candidateTarget, candidatePercent := r.candidateTraffic(svc, candidate)
	if candidateTarget == nil {
		candidatePercent = r.Config.Rollout.Steps[0]
	} else if candidatePercent == candidateTarget.Percent {
		// If the traffic share did not change, candidate already handled 100%
		// and is now ready to become stable.
		r.promoteCandidate = true
	}

	var traffic []*run.TrafficTarget
	candidateTarget = newTrafficTarget(candidate, candidatePercent, CandidateTag)
	if r.promoteCandidate {
		candidateTarget.Tag = StableTag
	}

	traffic = append(traffic, candidateTarget)
	for _, target := range svc.Spec.Traffic {
		// Respect tags manually introduced by the user.
		if target.Tag != "" && !target.LatestRevision &&
			target.Tag != StableTag && target.Tag != CandidateTag {

			traffic = append(traffic, target)
			continue
		}

		// When the user introduces a tag manually (UI/gcloud), Cloud Run breaks
		// all targets with (Name, Tag, Percent) into two different targets
		// (Name, Tag) and (Name, Percent).
		// This recovers from this by ignoring the (Name, Tag) target.
		if (target.Tag == StableTag || target.Tag == CandidateTag) && target.Percent == 0 {
			continue
		}

		if target.RevisionName == stable && !r.promoteCandidate {
			traffic = append(traffic, newTrafficTarget(target.RevisionName, 100-candidatePercent, StableTag))
		}
	}

	// Always assign latest tag to the latest revision.
	traffic = append(traffic, &run.TrafficTarget{LatestRevision: true, Tag: LatestTag})

	if !r.promoteCandidate {
		r.Log.Printf("Assigning %d%% of the traffic to stable revision %s", 100-candidatePercent, format.Bold(stable))
		r.Log.Printf("Assigning %d%% of the traffic to candidate revision %s", candidatePercent, format.Bold(candidate))
	} else {
		r.Log.Printf("Making revision %s stable\n", format.Bold(candidate))
	}

	svc.Spec.Traffic = traffic

	return svc
}

// candidateTraffic returns the traffic configuration for the candidate and the
// next traffic share for the candidate.
func (r *Rollout) candidateTraffic(svc *run.Service, candidate string) (*run.TrafficTarget, int64) {
	for _, target := range svc.Status.Traffic {
		if target.RevisionName == candidate && target.Percent > 0 {
			return target, r.nextCandidateTraffic(target.Percent)
		}
	}

	return nil, 0
}

// nextCandidateTraffic calculates the next traffic share for the candidate.
func (r *Rollout) nextCandidateTraffic(current int64) int64 {
	for _, step := range r.Config.Rollout.Steps {
		if step > current {
			return step
		}
	}

	return 100
}

// updateAnnotations updates the annotations to keep some state about the rollout.
func (r *Rollout) updateAnnotations(svc *run.Service, stable, candidate string) *run.Service {
	// The candidate has become the stable revision.
	if r.promoteCandidate {
		svc.Metadata.Annotations[StableRevisionAnnotation] = candidate
		delete(svc.Metadata.Annotations, CandidateRevisionAnnotation)

		return svc
	}

	svc.Metadata.Annotations[StableRevisionAnnotation] = stable
	svc.Metadata.Annotations[CandidateRevisionAnnotation] = candidate

	return svc
}

// newTrafficTarget returns a new traffic target instance.
func newTrafficTarget(revision string, percent int64, tag string) *run.TrafficTarget {
	return &run.TrafficTarget{
		RevisionName: revision,
		Percent:      percent,
		Tag:          tag,
	}
}

// generateServiceName returns the name of the specified service. It returns the
// form namespaces/{namespace_id}/services/{service_id}.
//
// For Cloud Run (fully managed), the namespace is the project ID or number.
func generateServiceName(namespace, serviceID string) string {
	return fmt.Sprintf("namespaces/%s/services/%s", namespace, serviceID)
}

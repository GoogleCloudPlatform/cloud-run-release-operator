package rollout_test

import (
	"io/ioutil"
	"testing"

	"github.com/GoogleCloudPlatform/cloud-run-release-operator/internal/run/mock"
	"github.com/GoogleCloudPlatform/cloud-run-release-operator/pkg/config"
	"github.com/GoogleCloudPlatform/cloud-run-release-operator/pkg/rollout"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/run/v1"
)

type ServiceOpts struct {
	LatestReadyRevision   string
	LatestCreatedRevision string
	Annotations           map[string]string
	Traffic               []*run.TrafficTarget
}

func generateService(opts *ServiceOpts) *run.Service {
	return &run.Service{
		Metadata: &run.ObjectMeta{
			Annotations: opts.Annotations,
		},
		Spec: &run.ServiceSpec{
			Traffic: opts.Traffic,
		},
		Status: &run.ServiceStatus{
			Traffic:                 opts.Traffic,
			LatestReadyRevisionName: opts.LatestReadyRevision,
		},
	}
}

func TestManage(t *testing.T) {
	client := &mock.RunAPI{}
	config := &config.Config{
		Metadata: &config.Metadata{
			Project: "test",
			Service: "hello",
		},
		Rollout: &config.Rollout{
			Steps: []int64{10, 40, 70},
		},
	}

	var tests = []struct {
		annotations    map[string]string
		traffic        []*run.TrafficTarget
		lastReady      string
		outAnnotations map[string]string
		outTraffic     []*run.TrafficTarget
		shouldErr      bool
		nilService     bool
	}{
		// There is a revision with 100% of traffic different from stable and candidate.
		// Make it the stable revision.
		{
			annotations: map[string]string{
				rollout.StableRevisionAnnotation:    "test-001",
				rollout.CandidateRevisionAnnotation: "test-003",
			},
			traffic: []*run.TrafficTarget{
				{RevisionName: "test-002", Percent: 100},
				{RevisionName: "test-003", Percent: 0, Tag: rollout.CandidateTag},
			},
			lastReady: "test-003",
			outAnnotations: map[string]string{
				rollout.StableRevisionAnnotation:    "test-002",
				rollout.CandidateRevisionAnnotation: "test-003",
			},
			outTraffic: []*run.TrafficTarget{
				{RevisionName: "test-003", Percent: config.Rollout.Steps[0], Tag: rollout.CandidateTag},
				{RevisionName: "test-002", Percent: 100 - config.Rollout.Steps[0], Tag: rollout.StableTag},
				{LatestRevision: true, Tag: rollout.LatestTag},
			},
		},
		// There's no a stable revision nor a revision handling 100% of traffic.
		{
			annotations: map[string]string{},
			traffic: []*run.TrafficTarget{
				{RevisionName: "test-002", Percent: 50},
				{RevisionName: "test-001", Percent: 50},
			},
			lastReady:      "test-002",
			outAnnotations: map[string]string{},
			outTraffic:     []*run.TrafficTarget{},
			nilService:     true,
		},
		// Stable revision is the same as the latest revision. There's no candidate.
		{
			annotations: map[string]string{},
			traffic: []*run.TrafficTarget{
				{RevisionName: "test-001", Percent: 100},
			},
			lastReady:      "test-001",
			outAnnotations: map[string]string{},
			outTraffic:     []*run.TrafficTarget{},
			nilService:     true,
		},
		// Candidate is new with non-existing previous candidate.
		{
			annotations: map[string]string{
				rollout.StableRevisionAnnotation: "test-001",
			},
			traffic: []*run.TrafficTarget{
				{RevisionName: "test-001", Percent: 100 - config.Rollout.Steps[1], Tag: rollout.StableTag},
				{LatestRevision: true, Tag: rollout.LatestTag},
			},
			lastReady: "test-002",
			outAnnotations: map[string]string{
				rollout.StableRevisionAnnotation:    "test-001",
				rollout.CandidateRevisionAnnotation: "test-002",
			},
			outTraffic: []*run.TrafficTarget{
				{RevisionName: "test-002", Percent: config.Rollout.Steps[0], Tag: rollout.CandidateTag},
				{RevisionName: "test-001", Percent: 100 - config.Rollout.Steps[0], Tag: rollout.StableTag},
				{LatestRevision: true, Tag: rollout.LatestTag},
			},
		},
		// Candidate is the same as before, keep rolling forward.
		{
			annotations: map[string]string{
				rollout.StableRevisionAnnotation:    "test-001",
				rollout.CandidateRevisionAnnotation: "test-002",
			},
			traffic: []*run.TrafficTarget{
				{RevisionName: "test-002", Percent: config.Rollout.Steps[1], Tag: rollout.CandidateTag},
				{RevisionName: "test-001", Percent: 100 - config.Rollout.Steps[1], Tag: rollout.StableTag},
				{LatestRevision: true, Tag: rollout.LatestTag},
			},
			lastReady: "test-002",
			outAnnotations: map[string]string{
				rollout.StableRevisionAnnotation:    "test-001",
				rollout.CandidateRevisionAnnotation: "test-002",
			},
			outTraffic: []*run.TrafficTarget{
				{RevisionName: "test-002", Percent: config.Rollout.Steps[2], Tag: rollout.CandidateTag},
				{RevisionName: "test-001", Percent: 100 - config.Rollout.Steps[2], Tag: rollout.StableTag},
				{LatestRevision: true, Tag: rollout.LatestTag},
			},
		},
		// Candidate is not the same as before, restart rollout with new candidate.
		{
			annotations: map[string]string{
				rollout.StableRevisionAnnotation:    "test-001",
				rollout.CandidateRevisionAnnotation: "test-002",
			},
			traffic: []*run.TrafficTarget{
				{RevisionName: "test-002", Percent: config.Rollout.Steps[2], Tag: rollout.CandidateTag},
				{RevisionName: "test-001", Percent: 100 - config.Rollout.Steps[2], Tag: rollout.StableTag},
				{LatestRevision: true, Tag: rollout.LatestTag},
			},
			lastReady: "test-003",
			outAnnotations: map[string]string{
				rollout.StableRevisionAnnotation:    "test-001",
				rollout.CandidateRevisionAnnotation: "test-003",
			},
			outTraffic: []*run.TrafficTarget{
				{RevisionName: "test-003", Percent: config.Rollout.Steps[0], Tag: rollout.CandidateTag},
				{RevisionName: "test-001", Percent: 100 - config.Rollout.Steps[0], Tag: rollout.StableTag},
				{LatestRevision: true, Tag: rollout.LatestTag},
			},
		},
		// Candidate was handling 100% of traffic. It's now ready to become stable.
		{
			annotations: map[string]string{
				rollout.StableRevisionAnnotation:    "test-001",
				rollout.CandidateRevisionAnnotation: "test-002",
			},
			traffic: []*run.TrafficTarget{
				{RevisionName: "test-002", Percent: 100, Tag: rollout.CandidateTag},
				{RevisionName: "test-001", Percent: 0, Tag: rollout.StableTag},
			},
			lastReady: "test-002",
			outAnnotations: map[string]string{
				rollout.StableRevisionAnnotation: "test-002",
			},
			outTraffic: []*run.TrafficTarget{
				{RevisionName: "test-002", Percent: 100, Tag: rollout.StableTag},
				{LatestRevision: true, Tag: rollout.LatestTag},
			},
		},
	}

	for _, test := range tests {

		client.ServiceFn = func(name string) (*run.Service, error) {
			opts := &ServiceOpts{
				LatestReadyRevision: test.lastReady,
				Annotations:         test.annotations,
				Traffic:             test.traffic,
			}
			return generateService(opts), nil
		}
		client.ReplaceServiceFn = func(name string, svc *run.Service) (*run.Service, error) {
			return svc, nil
		}

		lg := logrus.New()
		lg.Out = ioutil.Discard
		r := rollout.New(client, config, lg)

		svc, err := r.Manage()
		if test.shouldErr {
			assert.NotNil(t, err)
		} else if test.nilService {
			assert.Nil(t, svc)
		} else {
			assertAnnotations(t, test.outAnnotations, svc.Metadata.Annotations)
			assertTraffic(t, test.outTraffic, svc.Spec.Traffic)
		}
	}
}

func TestSplitTraffic(t *testing.T) {
	client := &mock.RunAPI{}
	config := &config.Config{
		Metadata: &config.Metadata{
			Project: "test",
			Service: "hello",
		},
		Rollout: &config.Rollout{
			Steps: []int64{5, 30, 60},
		},
	}

	var tests = []struct {
		stable    string
		candidate string
		traffic   []*run.TrafficTarget
		expected  []*run.TrafficTarget
	}{
		// There's a new candidate. Restart rollout process
		{
			stable:    "test-001",
			candidate: "test-003",
			traffic: []*run.TrafficTarget{
				{RevisionName: "test-001", Percent: 50},
				{RevisionName: "test-001", Tag: "tag1"},
				{RevisionName: "test-002", Percent: 50, Tag: rollout.CandidateTag},
				{RevisionName: "test-002", Tag: "tag2"},
				{LatestRevision: true, Tag: rollout.LatestTag},
			},
			expected: []*run.TrafficTarget{
				{RevisionName: "test-001", Percent: 95, Tag: rollout.StableTag},
				{RevisionName: "test-001", Tag: "tag1"},
				{RevisionName: "test-003", Percent: 5, Tag: rollout.CandidateTag},
				{RevisionName: "test-002", Tag: "tag2"},
				{LatestRevision: true, Tag: rollout.LatestTag},
			},
		},
		// Candidate is the same. Continue rolling forward.
		{
			stable:    "test-001",
			candidate: "test-003",
			traffic: []*run.TrafficTarget{
				{RevisionName: "test-001", Percent: 70, Tag: rollout.StableTag},
				{RevisionName: "test-002", Tag: "tag1"},
				{RevisionName: "test-003", Percent: 30, Tag: rollout.CandidateTag},
				{RevisionName: "test-003", Tag: "tag2"},
				{LatestRevision: true, Tag: rollout.LatestTag},
			},
			expected: []*run.TrafficTarget{
				{RevisionName: "test-001", Percent: 40, Tag: rollout.StableTag},
				{RevisionName: "test-002", Tag: "tag1"},
				{RevisionName: "test-003", Percent: 60, Tag: rollout.CandidateTag},
				{RevisionName: "test-003", Tag: "tag2"},
				{LatestRevision: true, Tag: rollout.LatestTag},
			},
		},
		// Candidate is the same. Continue rolling forward to 100%.
		{
			stable:    "test-001",
			candidate: "test-003",
			traffic: []*run.TrafficTarget{
				{RevisionName: "test-001", Percent: 40, Tag: rollout.StableTag},
				{RevisionName: "test-002", Tag: "tag1"},
				{RevisionName: "test-003", Percent: 60, Tag: rollout.CandidateTag},
				{RevisionName: "test-003", Tag: "tag2"},
			},
			expected: []*run.TrafficTarget{
				{RevisionName: "test-001", Percent: 0, Tag: rollout.StableTag},
				{RevisionName: "test-002", Tag: "tag1"},
				{RevisionName: "test-003", Percent: 100, Tag: rollout.CandidateTag},
				{RevisionName: "test-003", Tag: "tag2"},
				{LatestRevision: true, Tag: rollout.LatestTag},
			},
		},
		// Candidate has proven able to handle 100%, make it stable.
		{
			stable:    "test-001",
			candidate: "test-003",
			traffic: []*run.TrafficTarget{
				{RevisionName: "test-001", Percent: 0, Tag: rollout.StableTag},
				{RevisionName: "test-002", Tag: "tag1"},
				{RevisionName: "test-003", Percent: 100, Tag: rollout.CandidateTag},
				{RevisionName: "test-003", Tag: "tag2"},
				{LatestRevision: true, Tag: rollout.LatestTag},
			},
			expected: []*run.TrafficTarget{
				{RevisionName: "test-002", Tag: "tag1"},
				{RevisionName: "test-003", Percent: 100, Tag: rollout.StableTag},
				{RevisionName: "test-003", Tag: "tag2"},
				{LatestRevision: true, Tag: rollout.LatestTag},
			},
		},
		// Two targets for the same stable and candidate revisions.
		{
			stable:    "test-001",
			candidate: "test-003",
			traffic: []*run.TrafficTarget{
				{RevisionName: "test-001", Percent: 70},
				{RevisionName: "test-001", Tag: rollout.StableTag},
				{RevisionName: "test-002", Tag: "tag1"},
				{RevisionName: "test-003", Percent: 30},
				{RevisionName: "test-003", Tag: rollout.CandidateTag},
				{LatestRevision: true, Tag: rollout.LatestTag},
			},
			expected: []*run.TrafficTarget{
				{RevisionName: "test-001", Percent: 40, Tag: rollout.StableTag},
				{RevisionName: "test-002", Tag: "tag1"},
				{RevisionName: "test-003", Percent: 60, Tag: rollout.CandidateTag},
				{LatestRevision: true, Tag: rollout.LatestTag},
			},
		},
	}

	for _, test := range tests {
		lg := logrus.New()
		lg.Out = ioutil.Discard
		r := rollout.New(client, config, lg)

		opts := &ServiceOpts{
			Traffic: test.traffic,
		}
		svc := generateService(opts)

		svc = r.SplitTraffic(svc, test.stable, test.candidate)
		assertTraffic(t, test.expected, svc.Spec.Traffic)
	}
}

// TestManageServiceFailed tests Manage when retrieving information on a service fails.
func TestManageServiceFailed(t *testing.T) {
	client := &mock.RunAPI{}
	config := &config.Config{
		Metadata: &config.Metadata{
			Project: "test",
			Service: "hello",
		},
		Rollout: &config.Rollout{
			Steps: []int64{5, 30, 60},
		},
	}
	lg := logrus.New()
	lg.Out = ioutil.Discard
	r := rollout.New(client, config, lg)

	// When retrieving service fails, an error should be returned.
	client.ServiceInvoked = false
	client.ServiceFn = func(name string) (*run.Service, error) {
		return nil, errors.New("bad request")
	}
	_, err := r.Manage()
	assert.True(t, client.ServiceInvoked, "Service method was not called")
	assert.NotNil(t, err)

	// When Service returns nil, an error should be returned since service does not exist.
	client.ServiceInvoked = false
	client.ServiceFn = func(name string) (*run.Service, error) {
		return nil, nil
	}
	_, err = r.Manage()
	assert.True(t, client.ServiceInvoked, "Service method was not called")
	assert.NotNil(t, err)
}

// equalAnnotations checks that maps of annotations are equivalent.
func assertAnnotations(t *testing.T, expected, actual map[string]string) {
	assert.Equal(t, len(expected), len(actual), "The size of the annotation maps are not equal.")
	for key, value := range expected {
		assert.Equal(t, value, actual[key])
	}
}

// equalTraffic checks that arrays of traffic targets are equivalent.
func assertTraffic(t *testing.T, expected, actual []*run.TrafficTarget) {
	assert.Equal(t, len(expected), len(actual), "The size of the traffic target arrays are not equal.")
	for _, t1 := range expected {
		found := false
		for _, t2 := range actual {
			if equalTrafficTarget(t1, t2) {
				found = true
				break
			}
		}

		assert.True(t, found, "Expected target traffic not found.", t1)
	}

}

// equalArray checks that the values of two arrays are equivalent.
func assertArray(t *testing.T, expected, actual []string) {
	assert.Equal(t, len(expected), len(actual), "The size of the arrays are not equal.")
	for _, v1 := range expected {
		found := false
		for _, v2 := range actual {
			if v1 == v2 {
				found = true
			}
		}

		assert.True(t, found, "Expected value not found.", v1)
	}
}

func equalTrafficTarget(t1 *run.TrafficTarget, t2 *run.TrafficTarget) bool {
	return t1.RevisionName == t2.RevisionName &&
		t1.Percent == t2.Percent &&
		t1.Tag == t2.Tag &&
		t1.LatestRevision == t2.LatestRevision
}

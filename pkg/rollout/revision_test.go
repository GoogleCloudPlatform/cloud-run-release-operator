package rollout_test

import (
	"testing"

	"github.com/GoogleCloudPlatform/cloud-run-release-operator/pkg/rollout"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/run/v1"
)

func TestStableRevision(t *testing.T) {
	var tests = []struct {
		annotations map[string]string
		traffic     []*run.TrafficTarget
		expected    string
	}{
		// There's no a stable revision nor a Revision handling all the traffic.
		{
			annotations: map[string]string{},
			traffic: []*run.TrafficTarget{
				{RevisionName: "test-001", Percent: 50},
				{RevisionName: "test-002", Percent: 50},
			},
			expected: "",
		},
		// There is no annotation but there's a Revision handling all the traffic.
		{
			annotations: map[string]string{},
			traffic: []*run.TrafficTarget{
				{RevisionName: "test-002", Tag: "new"},
				{RevisionName: "test-001", Percent: 100},
			},
			expected: "test-001",
		},
		// There's a stable Revision according the annotations.
		{
			annotations: map[string]string{rollout.StableRevisionAnnotation: "test-001"},
			traffic: []*run.TrafficTarget{
				{RevisionName: "test-003", Tag: "candidate"},
				{RevisionName: "test-003", Percent: 50},
				{RevisionName: "test-001", Percent: 50},
			},
			expected: "test-001",
		},
		// The same Revision is annotated as stable and receive all the traffic.
		{
			annotations: map[string]string{rollout.StableRevisionAnnotation: "test-001"},
			traffic: []*run.TrafficTarget{
				{RevisionName: "test-003", Tag: "new"},
				{RevisionName: "test-002", Percent: 100},
			},
			expected: "test-002",
		},
	}

	for _, test := range tests {
		opts := &ServiceOpts{Annotations: test.annotations, Traffic: test.traffic}
		svc := generateService(opts)
		stable := rollout.StableRevision(svc)

		assert.Equal(t, test.expected, stable)
	}
}

func TestGetCandidateRevision(t *testing.T) {
	var tests = []struct {
		lastestReady   string
		stableRevision string
		expected       string
	}{
		// Latest Revision is the same as the stable one.
		{
			lastestReady:   "test-001",
			stableRevision: "test-001",
			expected:       "",
		},
		// Latest Revision is not the same as the stable one.
		{
			lastestReady:   "test-002",
			stableRevision: "test-001",
			expected:       "test-002",
		},
	}

	for _, test := range tests {
		opts := &ServiceOpts{LatestReadyRevision: test.lastestReady}
		svc := generateService(opts)
		candidate := rollout.CandidateRevision(svc, test.stableRevision)

		assert.Equal(t, test.expected, candidate)
	}
}

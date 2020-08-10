package rollout_test

import (
	"context"
	"testing"

	metricsmock "github.com/GoogleCloudPlatform/cloud-run-release-operator/internal/metrics/mock"
	runmock "github.com/GoogleCloudPlatform/cloud-run-release-operator/internal/run/mock"
	"github.com/GoogleCloudPlatform/cloud-run-release-operator/pkg/config"
	"github.com/GoogleCloudPlatform/cloud-run-release-operator/pkg/rollout"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/run/v1"
)

func TestRollForwardTraffic(t *testing.T) {
	runclient := &runmock.RunAPI{}
	metricsMock := &metricsmock.Metrics{}
	strategy := config.Strategy{
		Steps: []int64{5, 30, 60},
	}

	var tests = []struct {
		name      string
		stable    string
		candidate string
		traffic   []*run.TrafficTarget
		expected  []*run.TrafficTarget
	}{
		{
			name:      "new candidate, restart rollout",
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
				{RevisionName: "test-003", Percent: 5, Tag: rollout.CandidateTag},
				{LatestRevision: true, Tag: rollout.LatestTag},
				{RevisionName: "test-001", Tag: "tag1"},
				{RevisionName: "test-002", Tag: "tag2"},
			},
		},
		{
			name:      "continue rolling out candidate",
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
				{RevisionName: "test-003", Percent: 60, Tag: rollout.CandidateTag},
				{LatestRevision: true, Tag: rollout.LatestTag},
				{RevisionName: "test-002", Tag: "tag1"},
				{RevisionName: "test-003", Tag: "tag2"},
			},
		},
		{
			name:      "roll out to 100%",
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
				{RevisionName: "test-003", Percent: 100, Tag: rollout.CandidateTag},
				{LatestRevision: true, Tag: rollout.LatestTag},
				{RevisionName: "test-002", Tag: "tag1"},
				{RevisionName: "test-003", Tag: "tag2"},
			},
		},
		{
			name:      "make candidate stable",
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
				{RevisionName: "test-003", Percent: 100, Tag: rollout.StableTag},
				{LatestRevision: true, Tag: rollout.LatestTag},
				{RevisionName: "test-002", Tag: "tag1"},
				{RevisionName: "test-003", Tag: "tag2"},
			},
		},
		{
			name:      "two targets for same revision",
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
				{RevisionName: "test-003", Percent: 60, Tag: rollout.CandidateTag},
				{LatestRevision: true, Tag: rollout.LatestTag},
				{RevisionName: "test-002", Tag: "tag1"},
			},
		},
	}

	for _, test := range tests {
		opts := &ServiceOpts{
			Traffic: test.traffic,
		}
		svc := generateService(opts)
		svcRecord := &rollout.ServiceRecord{Service: svc}

		r := rollout.New(context.TODO(), metricsMock, svcRecord, strategy).WithClient(runclient)

		t.Run(test.name, func(tt *testing.T) {
			traffic := r.RollForwardTraffic(svc.Spec.Traffic, test.stable, test.candidate)
			assert.Equal(tt, test.expected, traffic)
		})
	}
}

func TestRollbackTraffic(t *testing.T) {
	metricsMock := &metricsmock.Metrics{}

	stable := "test-001"
	candidate := "test-003"
	traffic := []*run.TrafficTarget{
		{RevisionName: "test-001", Percent: 40, Tag: rollout.StableTag},
		{RevisionName: "test-002", Tag: "tag1"},
		{RevisionName: "test-003", Percent: 60, Tag: rollout.CandidateTag},
		{RevisionName: "test-003", Tag: "tag2"},
	}
	expectedTraffic := []*run.TrafficTarget{
		{RevisionName: "test-001", Percent: 100, Tag: rollout.StableTag},
		{RevisionName: "test-003", Percent: 0, Tag: rollout.CandidateTag},
		{LatestRevision: true, Tag: rollout.LatestTag},
		{RevisionName: "test-002", Tag: "tag1"},
		{RevisionName: "test-003", Tag: "tag2"},
	}
	svc := generateService(&ServiceOpts{Traffic: traffic})
	svcRecord := &rollout.ServiceRecord{Service: svc}

	r := rollout.New(context.TODO(), metricsMock, svcRecord, config.Strategy{})
	traffic = r.RollbackTraffic(svc.Spec.Traffic, stable, candidate)
	assert.Equal(t, expectedTraffic, traffic)
}

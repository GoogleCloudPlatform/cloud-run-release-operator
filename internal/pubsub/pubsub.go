package pubsub

import (
	"context"
	"encoding/json"
	"net/url"
	"regexp"

	cloudpubsub "cloud.google.com/go/pubsub"
	"github.com/GoogleCloudPlatform/cloud-run-release-operator/internal/health"
	"github.com/GoogleCloudPlatform/cloud-run-release-operator/internal/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/run/v1"
)

// Client represents a client to Google Cloud Pub/Sub.
type Client interface {
	Publish(ctx context.Context, msg Message) error
}

// PubSub is a Google Cloud Pub/Sub client to publish messages.
type PubSub struct {
	client *cloudpubsub.Client
	topic  *cloudpubsub.Topic
}

// Message is the format when publishing to Pub/Sub.
type Message struct {
	Event                        string       `json:"event"`
	CandidateRevisionName        string       `json:"candidateRevisionName"`
	CandidateRevisionPercent     int          `json:"candidateRevisionPercent"`
	CandidateRevisionURL         string       `json:"candidateRevisionURL"`
	CandidateWasPromotedToStable bool         `json:"candidateWasPromotedToStable"`
	Service                      *run.Service `json:"service"`
}

// New initializes a PubSub client.
func New(ctx context.Context, projectID string, topicName string) (ps PubSub, err error) {
	logger := util.LoggerFrom(ctx)
	client, err := cloudpubsub.NewClient(ctx, projectID)
	if err != nil {
		return ps, errors.Wrap(err, "failed to initialize Pub/Sub client")
	}

	match := regexp.MustCompile(`projects/([^/]*)/topics/([^/]*)`).FindStringSubmatch(topicName)
	if len(match) != 3 {
		return ps, errors.Errorf("invalid topic name %s", topicName)
	}
	project := match[1]
	topicID := match[2]
	logger.WithFields(logrus.Fields{"topicProject": project, "topicID": topicID}).Debug("parsed topic name value")

	return PubSub{
		client: client,
		topic:  client.TopicInProject(topicID, project),
	}, nil
}

// NewMessage initializes a message.
func NewMessage(svc *run.Service, diagnosis health.DiagnosisResult, candidateWasPromoted bool) (Message, error) {
	event := "rollout"
	if diagnosis == health.Unhealthy {
		event = "rollback"
	}

	var candidateRevision *run.TrafficTarget
	var err error
	if candidateWasPromoted {
		// The last candidate is now the stable.
		candidateRevision, err = findRevisionWithTag(svc, "stable")
	} else {
		candidateRevision, err = findRevisionWithTag(svc, "candidate")
	}
	if err != nil {
		return Message{}, errors.New("failed to find candidate revision traffic target")
	}

	return Message{
		Event:                        event,
		CandidateRevisionName:        candidateRevision.RevisionName,
		CandidateRevisionPercent:     int(candidateRevision.Percent),
		CandidateRevisionURL:         candidateRevision.Url,
		CandidateWasPromotedToStable: candidateWasPromoted,
		Service:                      svc,
	}, nil
}

// Publish publishes message to the topic.
func (ps PubSub) Publish(ctx context.Context, msg Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return errors.Wrap(err, "failed to marshal message")
	}

	logger := util.LoggerFrom(ctx)
	ps.topic.Publish(ctx, &cloudpubsub.Message{
		Data: data,
	})
	logger.WithField("size", len(data)).Debug("published data to Pub/Sub")
	return nil
}

// findRevisionWithTag scans the service's traffic configuration and returns the
// revision that has the given tag.
//
// Since the update of a service occurs asynchronously, the changes in the
// traffic in the Service spec is not reflected in the Service's status at the
// time of publishing.
//
// However, the traffic targets in the Service spec do not have a URL associated
// to them since the URL field is read-only and available only in the status
// traffic configuration.
//
// Because the Service spec contains the updated traffic configuration, the
// traffic targets in the spec are the ones that are scanned. The URL to the
// revision is generated based on the service's URL and the tag value.
func findRevisionWithTag(svc *run.Service, tag string) (*run.TrafficTarget, error) {
	var target *run.TrafficTarget
	for _, t := range svc.Spec.Traffic {
		if t.Tag == tag {
			target = t
			break
		}
	}
	if target == nil {
		return nil, errors.Errorf("could not find revision with tag %s", tag)
	}

	url, err := url.Parse(svc.Status.Url)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse the service's url %s", svc.Status.Url)
	}
	url.Host = tag + "---" + url.Host
	target.Url = url.String()

	return target, nil
}

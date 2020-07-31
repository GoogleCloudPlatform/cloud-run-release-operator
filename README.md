# Cloud Run Progressive Delivery Operator

A gradual rollout operator for Cloud Run services.

## How does it work?

The operator provides an automated way to gradually roll out new versions of
your code by increasing the traffic to your newly deployed revision. The
operator periodically checks for new revisions in the services that opted-in for
gradual rollouts. If a new revision with no traffic is found, the operator
automatically assigns it some initial traffic. This new revision is labeled
*candidate* while the previous revision serving traffic is labeled *stable*.

After some time, the operator checks for metrics for the new revision. Using the
set health criteria, it evaluates what to do next:

1) If the metrics show a *healthy* revision, traffic to the *candidate* is
increased.
2) If the metrics show an *unhealthy* revision, traffic to the *candidate* is
dropped and the *stable* revision gets 100% of the traffic.

Once a healthy *candidate* handles 100% of the traffic, it becomes the new
*stable* revision.

## Usage

`crun-release-operator -cli -project=myproject -interval=600`

## Configuration

Currently, all the configuration arguments must be specified using command line
flags

### Choosing services

The tool can manage the rollout of several services at the same time. To opt-in
a service, the service must have the configured label selector. By default, the
tool looks for services with the label `rollout-strategy=gradual` in all
regions. However, a project must be specified.

- `project`: Google Cloud project in which the Cloud Run services are deployed
- `regions`: Regions where to look for opted-in services (default: [all
available Cloud Run regions](https://cloud.google.com/run/docs/locations))
- `label`: The label selector that the opted-in services must have (default:
`rollout-strategy=rollout`)

### Rollout strategy

The rollout strategy consists of the steps and health criteria.

- `steps`: Percentages of traffic the candidate should go through (default:
`5,20,50,80`)
- `max-error-rate`: Expected maximum rate (in percent) of server errors
(default: 1)
- `interval`: The time between each health check (in seconds)
- `latency-p99`: Expected maximum latency for 99th percentile of requests, 0 to
ignore (default: 0)
- `latency-p95`: Expected maximum latency for 95th percentile of requests, 0 to
ignore (default: 0)
- `latency-p50`: Expected maximum latency for 50th percentile of requests, 0 to
ignore (default: 0)

---

This is not an official Google project. See [LICENSE](./LICENSE).

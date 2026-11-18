# Pub/Sub

Production Pub/Sub resources are managed by Terraform in
[zone-ops PR #27](https://github.com/initia-labs/zone-ops/pull/27).
This chart is disabled by default (`enabled: false`) and should remain disabled
for those networks. Worker KSAs have publisher/subscriber permissions, not resource
creation permissions. See [worker configuration](../../docs/pubsub.md) for the
project, endpoint, service accounts, and exact topic/subscription IDs.

For a separate project without Terraform provisioning, enable this chart. It uses
the same `image`, `env`, `envFrom`, `resources`, `annotations`, and
`serviceAccountName` values as the worker charts. A Helm install/upgrade Job creates
Google Cloud Pub/Sub resources through `gcloud`; it does not run a message broker.
Enable the Pub/Sub API and supply an existing Kubernetes service account with
Workload Identity and `roles/pubsub.editor` in the target project.

```yaml
# pubsub-staging.yaml
enabled: true
serviceAccountName: pubsub-setup
env:
  PUBSUB_PROJECT_ID: your-staging-project
  PUBSUB_REGION: asia-southeast1
  CHAIN: initiation-2
```

```sh
helm upgrade --install pubsub ./charts/pubsub \
  --namespace scan --create-namespace \
  --values pubsub-staging.yaml --wait --timeout 10m
```

Install this release before starting workers. It mirrors the zone-ops topology:

| Topic | Subscriptions | Ordering |
| --- | --- | --- |
| `${CHAIN}-sequential-indexer-block-results` | `${CHAIN}-informative-indexer` | Enabled |
| `${CHAIN}-concurrent-indexer-block-results` | `${CHAIN}-generic-indexer`, `${CHAIN}-event-indexer` | Disabled |
| `${CHAIN}-tx-responses` | `${CHAIN}-tx-response-uploader` | Disabled |

Subscriptions have a 60-second ack deadline, seven-day message retention, and never
expire. All topics restrict persistence to `PUBSUB_REGION`. The three application
DLQ topics `dlq-${CHAIN}-{generic-indexer-block-results,event-indexer-block-results,tx-responses}`
retain messages for 15 days without pre-created subscriptions. No native dead-letter
policy is installed. Names have no environment suffix; use separate projects for
staging and production when sharing a chain ID.

Existing resources are reused without changing their configuration. An upgrade
reruns the setup Job and creates missing resources. Uninstalling this chart or
disabling it leaves cloud topics, subscriptions, and their backlog in place. Failed
Jobs remain available for logs until the next install/upgrade attempt.

# Pub/Sub

Production Pub/Sub resources are managed by Terraform in
[zone-ops PR #27](https://github.com/initia-labs/zone-ops/pull/27).
This chart is disabled by default (`enabled: false`) and should remain disabled
for those networks. Worker KSAs have publisher/subscriber permissions, not resource
creation permissions. See [worker configuration](../../docs/pubsub.md) for the
project, endpoint, service accounts, and exact topic/subscription IDs.

For a separate project without Terraform provisioning, enable this chart. It uses
the same `image`, `env`, `envFrom`, `resources`, `annotations`, and
`serviceAccountName` values as the worker charts. An Argo CD `PreSync` Job creates
Google Cloud Pub/Sub resources through `gcloud`; it does not run a message broker.
Enable the Pub/Sub API and supply an existing Kubernetes service account with
Workload Identity and `roles/pubsub.editor` in the target project.
The service account and any ConfigMaps or Secrets referenced by `envFrom` must
exist before the sync starts.

```yaml
# pubsub-staging.yaml
enabled: true
serviceAccountName: pubsub-setup
env:
  PUBSUB_PROJECT_ID: your-staging-project
  PUBSUB_REGION: asia-southeast1
  CHAIN: initiation-2
```

Configure the Argo CD Application to render `charts/pubsub` with these values.
The [PreSync hook](https://argo-cd.readthedocs.io/en/stable/user-guide/sync-waves/)
must succeed before Argo CD applies the application's regular resources. If the
workers are in separate Applications, sync provisioning successfully before
syncing the workers; the hook does not order separate Applications.

The chart mirrors the zone-ops topology:

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

Existing resources are reused without changing their configuration. A full Argo CD
sync reruns the setup Job and creates missing resources. Hooks do not run during
selective sync. Removing or disabling the chart leaves cloud topics, subscriptions,
and their backlog in place. Successful Jobs are cleaned up; failed Jobs remain
available for logs until the next sync replaces them.

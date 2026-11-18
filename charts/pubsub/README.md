# Pub/Sub

This chart uses the same `image`, `env`, `envFrom`, `resources`, `annotations`, and
`serviceAccountName` values as the worker charts. A Helm install/upgrade Job creates
Google Cloud Pub/Sub resources through `gcloud`; it does not run a message broker.

Enable the Pub/Sub API in the target GCP project. Supply an existing Kubernetes
service account configured for Workload Identity with `roles/pubsub.editor` in
that project. Worker service accounts only need their publisher/subscriber roles.

```yaml
# pubsub-staging.yaml
serviceAccountName: pubsub-setup
env:
  PUBSUB_PROJECT_ID: your-staging-project
  CHAIN: initiation-2
  ENVIRONMENT: staging
```

```sh
helm upgrade --install pubsub ./charts/pubsub \
  --namespace scan --create-namespace \
  --values pubsub-staging.yaml --wait --timeout 10m
```

Install this release before starting the workers. The Job creates:

- `${CHAIN}-${ENVIRONMENT}-block-results` with separate ordered subscriptions
  named `${CHAIN}-${ENVIRONMENT}-{generic-indexer,informative-indexer,event-indexer}`.
- `${CHAIN}-${ENVIRONMENT}-tx-responses` with subscription
  `${CHAIN}-${ENVIRONMENT}-tx-response-uploader`.
- `dlq-${CHAIN}-{generic-indexer-block-results,event-indexer-block-results,tx-responses}`
  topics, each with a same-name subscription.

New subscriptions never expire due to inactivity, including DLQ subscriptions
that may go unused for long periods. Names follow the worker configuration in
[docs/pubsub.md](../../docs/pubsub.md). DLQ names do not include the environment;
use separate projects for staging and production when sharing a chain ID.

Existing resources are reused without changing their configuration. For existing
block subscriptions, enable ordering at creation as described in the migration
guide. No native dead-letter policy is installed on the block subscriptions.

An upgrade reruns the setup Job and creates any missing resources. Uninstalling
this chart leaves cloud topics, subscriptions, and their backlog in place. Failed
Jobs remain available for logs until the next install/upgrade attempt.

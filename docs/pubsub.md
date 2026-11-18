# Pub/Sub

All five workers use Google Cloud Pub/Sub. JSON payloads and GCS claim-check
object paths are unchanged. The old Kafka key is carried in the `key` message
attribute; `height`, `tx_hash`, and `service` are also attributes.

## Configuration

| Worker | Required messaging environment variables |
| --- | --- |
| All | `PUBSUB_PROJECT_ID` |
| Sweeper | `BLOCK_RESULTS_TOPICS` (comma-separated topic IDs) |
| Generic / Informative / Event | `BLOCK_RESULTS_SUBSCRIPTION` |
| Generic | `TX_TOPIC` |
| Informative | `TX_RESPONSE_TOPIC` |
| TX response uploader | `TX_RESPONSE_SUBSCRIPTION` |

Set `PUBSUB_ENDPOINT` to the same regional endpoint for publishers of a chain,
for example `asia-southeast1-pubsub.googleapis.com:443`. Set it on all workers for
consistent regional routing. Authentication uses Application Default Credentials;
on GKE use the workload's existing service account identity with Pub/Sub permissions.
`PUBSUB_EMULATOR_HOST` is only for local testing and must be unset on staging/prod.

Remove `BOOTSTRAP_SERVER`, `KAFKA_API_KEY`, `KAFKA_API_SECRET`, and consumer-group
environment variables. Consumers no longer need an input topic variable. CLI
replacements are `--pubsub-project-id`, `--pubsub-endpoint`, and
`--block-results-subscription`. GCS, database, RPC, and claim-check settings stay
as before. Claim-check payloads are capped at 9 MiB even if a larger threshold is
configured, to fit Pub/Sub publish requests with metadata.

## Resources and zone-ops configuration

Production resources are owned by Terraform in
[zone-ops PR #27](https://github.com/initia-labs/zone-ops/pull/27)
(reviewed at `84a60b447b58fe997fc658e63ce8b1b03cd332d3`). The project is
`apps-prod-514512`, with persistence in `asia-southeast1`. Use
`PUBSUB_PROJECT_ID=apps-prod-514512` and
`PUBSUB_ENDPOINT=asia-southeast1-pubsub.googleapis.com:443` on every worker.

| Network | `CHAIN` | Kubernetes namespace | `serviceAccountName` |
| --- | --- | --- | --- |
| Testnet | `initiation-2` | `testnet` | `core-indexer-service-testnet` |
| Mainnet | `interwoven-1` | `mainnet` | `core-indexer-service-mainnet` |

Set these worker values, replacing `<chain>` with the network's chain ID:

| Worker | Environment variable | Value |
| --- | --- | --- |
| Sweeper | `BLOCK_RESULTS_TOPICS` | `<chain>-concurrent-indexer-block-results,<chain>-sequential-indexer-block-results` |
| Generic | `BLOCK_RESULTS_SUBSCRIPTION` | `<chain>-generic-indexer` |
| Generic | `TX_TOPIC` | `<chain>-tx-responses` |
| Informative | `BLOCK_RESULTS_SUBSCRIPTION` | `<chain>-informative-indexer` |
| Informative | `TX_RESPONSE_TOPIC` | `<chain>-tx-responses` |
| Event | `BLOCK_RESULTS_SUBSCRIPTION` | `<chain>-event-indexer` |
| TX response uploader | `TX_RESPONSE_SUBSCRIPTION` | `<chain>-tx-response-uploader` |

For example, merge this messaging configuration into the existing testnet sweeper
values (retain its database, RPC, chain, and claim-check settings):

```yaml
serviceAccountName: core-indexer-service-testnet
env:
  PUBSUB_PROJECT_ID: apps-prod-514512
  PUBSUB_ENDPOINT: asia-southeast1-pubsub.googleapis.com:443
  BLOCK_RESULTS_TOPICS: initiation-2-concurrent-indexer-block-results,initiation-2-sequential-indexer-block-results
```

The sweeper publishes every block to both topics, waiting for confirmation from
each before advancing. Keep the concurrent topic first: the sweeper resumes from
the informative height after restart, so the sequential publish must follow the
concurrent publish. It sets the chain ID as the ordering key. The informative
subscription consumes the sequential topic with ordering enabled; generic and
event subscriptions independently consume the concurrent topic without ordering.
Replicas of each worker share its subscription. Keep one sweeper and one informative
indexer per chain, as the informative indexer retains mutable chain state in memory.
All indexers must use the sweeper's claim-check bucket.

The four subscriptions use a 60-second ack deadline, retain messages for seven days,
and never expire on inactivity. The SDK extends ack leases during processing.
DLQ topics are `dlq-<chain>-generic-indexer-block-results`,
`dlq-<chain>-event-indexer-block-results`, and `dlq-<chain>-tx-responses`, with
15-day topic retention and no pre-created subscriptions. Recovery subscriptions
must seek to a timestamp in the retained window to replay earlier DLQ messages.
These are application-managed DLQs: generic/event/uploader publish failures there
before acknowledging their input. There is no native dead-letter policy.
Informative-indexer stops on processing failure so it cannot skip ahead.

Resource names have no environment suffix. Separate projects are needed for
staging/prod if they share the same chain ID. The network KSAs receive direct
Workload Identity Federation grants for publishing to their network's topics and
consuming its subscriptions. Use the existing KSA with Application Default
Credentials; no GSA impersonation or key file is required. The PR removes the
Google service accounts, so remove any old `iam.gke.io/gcp-service-account`
annotation from these KSAs during deployment.

Apply Terraform before starting workers. Leave the [Pub/Sub Helm chart](../charts/pubsub/README.md)
disabled for these networks: worker identities cannot create resources, and the
chart must not provision alongside Terraform. For separate projects, its Argo CD
`PreSync` Job uses the same topic names, subscription routing, and retention
policies. Complete provisioning before syncing workers in separate Applications.

## Delivery and cutover

Consumers process one message at a time per process, with SDK-managed ack lease
extension. Successful processing is acknowledged. Failed processing or failed DLQ
publishing stops the receiver without acknowledging the message. Tx responses are queued for SDK batching, then every publish result is confirmed
before the block database transaction commits. Informative validator signatures
and the indexed height commit in the same transaction.
Delivery remains at least once; a crash between database commit and ack can replay
work, just as with the former manual Kafka commits.

For a short-stop cutover after staging passes:

1. Create Pub/Sub topics, subscriptions, and IAM grants; prepare the new worker env.
2. Stop the Kafka sweeper and let all Kafka consumers, including the uploader, drain.
3. Stop Kafka consumers. Deploy Pub/Sub consumers, then the Pub/Sub sweeper.
4. The sweeper resumes from the database's latest informative block height.

Kafka offsets are not transferable to Pub/Sub. Draining every consumer before the
switch avoids leaving a slower consumer's backlog behind. For rollback, stop the
Pub/Sub sweeper and drain Pub/Sub consumers before switching back; do not run two
sweepers against the same chain during cutover.

## Local verification

From `pkg`, run `go test ./mq`. The tests use Google's in-process Pub/Sub test server
and cover ordered fanout, claim-check payloads, DLQ metadata, publish failures, and
redelivery without acknowledging failed work. No cloud credentials are required.
See [local setup](../local/README.md) for the Docker Compose emulator stack.

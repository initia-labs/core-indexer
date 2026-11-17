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

## Resources

Create subscriptions before starting the sweeper. Each indexer type needs its own
subscription so each receives every block; replicas of the same type share that
subscription. The sweeper publishes sequentially with the chain ID as ordering key.
Block subscriptions must enable ordering. Keep one sweeper and one informative
indexer per chain, as the informative indexer retains mutable chain state in memory.

Example resource setup (set `PROJECT`, `CHAIN`, and `STAGE` first):

```sh
BLOCK_TOPIC="${CHAIN}-${STAGE}-block-results"
TX_TOPIC="${CHAIN}-${STAGE}-tx-responses"
gcloud pubsub topics create "$BLOCK_TOPIC" "$TX_TOPIC" --project="$PROJECT"
for component in generic-indexer informative-indexer event-indexer; do
  gcloud pubsub subscriptions create "${CHAIN}-${STAGE}-${component}" \
    --project="$PROJECT" --topic="$BLOCK_TOPIC" --enable-message-ordering
done
gcloud pubsub subscriptions create "${CHAIN}-${STAGE}-tx-response-uploader" \
  --project="$PROJECT" --topic="$TX_TOPIC"
for component in generic-indexer-block-results event-indexer-block-results tx-responses; do
  gcloud pubsub topics create "dlq-${CHAIN}-${component}" --project="$PROJECT"
  gcloud pubsub subscriptions create "dlq-${CHAIN}-${component}" \
    --project="$PROJECT" --topic="dlq-${CHAIN}-${component}"
done
```

The DLQ naming convention is unchanged and does not include the stage; use separate
projects for staging/prod if both use the same chain ID. These are application-managed
DLQs: generic/event/uploader publish failures there before acknowledging their input.
Informative-indexer stops on processing failure so it cannot skip ahead to a later block.
Do not attach a native dead-letter policy that skips failed informative blocks.

Grant workers `roles/pubsub.publisher` on topics they publish to and
`roles/pubsub.subscriber` on their subscriptions. Resource creation happens during
setup; worker identities do not need topic/subscription administration permissions.
Use each chart's existing `env`, `envFrom`, and `serviceAccountName` fields.
All consumers of a shared block topic must point to the sweeper's claim-check bucket.

## Delivery and cutover

Consumers process one message at a time per process, with SDK-managed ack lease
extension. Successful processing is acknowledged. Failed processing or failed DLQ
publishing stops the receiver without acknowledging the message. Publishing waits
for server confirmation, including tx responses emitted before a database commit.
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

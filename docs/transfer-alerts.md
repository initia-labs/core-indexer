# Transfer alerts

`event-indexer alerts --config /etc/alerts/alerts.json` runs independently of the
event-indexing worker. It reads the same raw block results and sends Telegram
messages for the configured assets and routes. It does not query flattened
`transaction_events`, change the event schema, or interrupt the indexing subscription.

## Rules

Copy `event-indexer/alerts.example.json` and replace its placeholders using the
target chain's deployed channels, denoms, decimals, escrow addresses and OFT type
tags. The example is a template, not a production watch list.

- `assets` names are display labels/references. `threshold` is a decimal string in
  display tokens, **not USD**. The comparison is strictly greater than the threshold,
  using integer base units. Exactly 100,000 tokens does not alert. Totals are per
  successful transaction and configured asset ID: IBC across watched channels,
  LayerZero across watched sends, and escrow debits per watched address.
- `ibc_routes` bind a local source port/channel to an explicit asset list. Each
  asset's `packet_denoms` lists exact ICS-20 packet denomination traces, which can
  differ from its local bank `denom`. A matching `send_packet` alerts on source
  submission, not destination completion. Both packet_data and packet_data_hex
  encodings are supported. Matching packets are summed within the transaction,
  with source-channel totals included in the alert.
- `escrows` bind addresses to asset lists. Bank `coin_spent` events whose spender is
  a watched escrow alert on actual debit, including releases/refunds triggered by
  incoming IBC packets and finalize-block transfers. Inflows do not alert. This
  does not infer insolvency or classify an outflow as malicious. `coin_spent` also
  covers MultiSend, whose transfer events can omit the sender on Initia. One
  MultiSend debit is evaluated as a whole; its individual recipients are not in
  this event. The alert reports the spender and transaction, not a guessed recipient.
  Separate `transfer` events are ignored to avoid counting the debit twice.
- `layerzero` binds one deployed full `::oft_core::OftSent` type tag to an asset.
  It compares `amount_sent_ld` in local decimals; `amount_received_ld` is not the
  source debit. An empty `destinations` list monitors all destination EIDs.
  Schema reference: [LayerZero's Initia OFT implementation](https://github.com/LayerZero-Labs/devtools/blob/main/examples/oft-initia/sources/shared_oft/oft_core.move).
  The event does not contain the destination recipient; the alert includes the
  sender, EID and GUID. Verify this schema against the deployed bridge before rollout.

Failed transactions are ignored. One Telegram message per transaction contains
all totals exceeding their thresholds, followed by the contributing event details.
For example, two 60,000 USDC packets on watched channels in one transaction produce
a 120,000 USDC IBC total. Two separate transactions of 60,000 USDC do not alert.
Different assets, bridge types and escrow addresses are never summed together:
one transfer can emit both bridge and escrow events, so those totals remain
separate observations rather than a combined volume. Overlapping watch rules do
not count the same event twice. Events outside a transaction (finalize-block
transfers) are checked individually, with one message per triggering event.
Rolling-window aggregation, ERC-20/CW20 internal transfers, burn-only withdrawals
and destination-delivery tracking are not implemented by these rules.

## Runtime

Apply the normal core-indexer migrations, including
`20261118000000_transfer_alerts.up.sql`, before starting the worker. The new tables
hold delivery records and RPC cursors; existing event tables are unchanged.

Required environment variables (supply credentials through a Secret):

| Variable | Purpose |
| --- | --- |
| `DB_CONNECTION_STRING` | PostgreSQL database with the alert tables |
| `TELEGRAM_BOT_TOKEN` | Bot permitted to send to the target group |
| `TELEGRAM_CHAT_ID` | New alert group's chat ID |

For the existing L1 Pub/Sub feed, leave `rpc` empty and also set:

| Variable | Purpose |
| --- | --- |
| `PUBSUB_PROJECT_ID` | Feed's Google Cloud project |
| `PUBSUB_ENDPOINT` | Optional regional endpoint |
| `BLOCK_RESULTS_SUBSCRIPTION` | **Dedicated** alert subscription on the block-results topic |
| `BLOCK_RESULTS_CLAIM_CHECK_BUCKET` | Same claim-check bucket as the sweeper |

Do not reuse the event indexer's subscription: subscribers on one subscription
share messages instead of receiving independent copies. Provision subscription
and IAM through the existing infrastructure owner. The worker implements the
repository's Pub/Sub transport; it is not a Kafka consumer. Confirm the production
cutover before deployment, or use RPC mode while Kafka is still active.

For an L2 or another Cosmos chain without a feed, set `rpc` to its CometBFT HTTP
endpoint and `start_height` to the first block to inspect. Run one worker per
chain. It polls every two seconds, processes every block and saves its cursor only
after alerts succeed. On restart it resumes from PostgreSQL; changing
`start_height` does not overwrite an existing cursor. Historical backfills require
block and block-results retention at the RPC endpoint. The same worker can cover
L1/L2 Cosmos bank escrows without duplicating transfer logic in rollup-monitoring.

The worker exits on RPC, decoding, database or Telegram errors. Use the normal
Deployment restart policy; failed Pub/Sub messages are not acknowledged, and RPC
cursors do not advance. Successfully sent events are deduplicated in PostgreSQL.
Telegram has no idempotency key, so a crash after Telegram accepts a message but
before the database commit can still produce a duplicate. Alert IDs in messages
make that case identifiable. Delivery is at least once, not exactly once.

Use `charts/event-indexer/values.alerts.example.yaml` as an overlay for a **separate
Helm release**. In the release namespace, create the `transfer-alerts-env` Secret
with the database and Telegram credentials, the `transfer-alerts-env` ConfigMap
with the nonsecret source settings above, and the `transfer-alerts-rules` ConfigMap
with the rules stored under the `alerts.json` key. The latter mounts as
`/etc/alerts/alerts.json`. In RPC mode, remove the environment ConfigMap's
`envFrom` entry if it is unused; retain the Secret and rules ConfigMap.
The overlay uses `core-indexer-service-mainnet`; select the testnet KSA for testnet.
Grant that KSA access to the dedicated subscription and the sweeper's claim-check
bucket through the infrastructure configuration.

Monitor restart counts and Pub/Sub
backlog (or RPC cursor progress) to detect interrupted monitoring. Retain delivery
records for at least the supported replay/backfill window; deleting them allows
old alerts to be sent again. Transaction alert IDs use
`<chain>/<height>/tx/<hash>`; finalize-block alerts use
`<chain>/<height>/block/<event-index>`. The transaction-total implementation uses
new IDs compared with the earlier per-event implementation. Replaying blocks
already notified by that earlier version can therefore send a new summary;
existing per-event delivery records do not suppress it.

## Verification

```sh
cd event-indexer
go test ./alerts ./cmd/alerts
# Optional durable-delivery integration test against a disposable PostgreSQL DB:
ALERT_TEST_DATABASE_URL='postgres://...' go test ./alerts -run TestDeliveryRetryAndReplay -count=1
```

Before enabling production notifications, replay representative real transactions
for each configured route, including a multi-transfer transaction, escrow release,
failed transaction and LZ send. Create the Telegram group and add the team and bot
before supplying its chat ID. No production configuration or group is created by
this code change.

# Transfer alerts

`event-indexer alerts --config /etc/alerts/alerts.json` runs independently of the
event-indexing worker and polls PostgreSQL for complete event groups. It does not
connect to Pub/Sub, GCS or RPC. The existing event-indexer continues consuming its
feed and writes groups into `transfer_alert_batches` in the same DB transaction as
its normal indexed events. The bot evaluates those groups and queues Telegram
messages in `transfer_alerts`.

The additional table preserves original event boundaries and successful execution
status without reconstructing the flattened `transaction_events` attributes. It
stores event arrays for successful transactions containing `send_packet`,
`coin_spent` or `move` events, plus finalize-block event groups containing those
types. Other transaction data is not copied. Rules run only in the alert worker.

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

Deploy in this order:

1. Apply the normal core-indexer migrations, including
   `20261118000000_transfer_alerts.up.sql` and
   `20261118010000_transfer_alert_batches.up.sql`.
2. Deploy the updated **indexing worker** so it starts writing complete event
   groups. Existing flattened rows do not automatically populate the new table.
3. Start the separate alert release against the same chain's **read-write DB**.
   Historical backfills require replay through the updated indexing worker.

Only these environment variables are required (supply credentials through a Secret):

| Variable | Purpose |
| --- | --- |
| `DB_CONNECTION_STRING` | Same chain's PostgreSQL database with indexed groups and alert tables |
| `TELEGRAM_BOT_TOKEN` | Bot permitted to send to the target group |
| `TELEGRAM_CHAT_ID` | Alert group's chat ID |

No dedicated Pub/Sub subscription, GCS bucket grant or RPC endpoint is needed for
the bot. Existing indexer cloud permissions remain unchanged. To monitor an L2,
run its updated indexing worker and point an alert release at that chain's DB.

Rules also accept these polling settings:

| Field | Default | Purpose |
| --- | --- | --- |
| `workers` | `4` | Concurrent DB evaluation workers |
| `batch_size` | `100` | Maximum event groups claimed by each worker per DB transaction |
| `start_height` | `0` | Inclusive lower bound for groups to evaluate; zero includes all captured groups |

Workers claim pending rows using `FOR UPDATE SKIP LOCKED`, so they process distinct
batches concurrently. They drain available batches immediately and poll every two
seconds when idle. A transaction's events are never split between batches. Late
blocks remain pending regardless of which higher heights have already been
processed; there is no high-water cursor. The earlier `transfer_alert_cursors`
table remains for migration compatibility but is no longer used.

Each batch atomically inserts its pending alerts and marks its groups evaluated.
An evaluation or DB error rolls the batch back. Successful evaluation is recorded
even when no threshold is exceeded. Changing rules does not automatically
reevaluate completed groups. Indexer replay does not reset completed groups.

A separate sender delivers one pending notification at a time, paced three
seconds apart, so Telegram latency does not block DB evaluation. Keep one bot
replica for this chat; `workers` controls evaluation concurrency. Failed sends stay
pending and the process exits for the normal Deployment restart policy to retry.
Telegram has no idempotency key: a crash after Telegram accepts a message but
before the DB commit can still duplicate it. Delivery is at least once.

Use `charts/event-indexer/values.alerts.example.yaml` as an overlay for a **separate
Helm release**. Create the `transfer-alerts-env` Secret with DB and Telegram
credentials and the `transfer-alerts-rules` ConfigMap with the rules under the
`alerts.json` key. The file mounts at `/etc/alerts/alerts.json`. The bot needs no
cloud service account. Both chart and images must include these changes.

For testnet, use namespace `testnet`, chain `initiation-2`, and the existing
`scan_initiation_2` database. Its existing DB Secret can supply `DBPASS` via
`PGPASSWORD`; keep Telegram credentials in a separate Secret if preferred.
The rule template still requires reviewed testnet denoms, source channels, escrow
addresses and OFT type tags.

Monitor pending unevaluated groups, unsent alerts, and event-indexer lag. Retain
evaluated groups and delivery records for at least the supported replay window;
deleting them lets old data be processed/sent again. Event arrays add DB storage
in exchange for removing the bot's messaging/cloud dependencies.

Transaction alert IDs use `<chain>/<height>/tx/<hash>`; finalize-block alerts use
`<chain>/<height>/block/<event-index>`. IDs match the previous transaction-total
worker, so retained delivery records deduplicate a source switch. The older
per-event implementation used different IDs; replay from that version can send a
new transaction summary.

## Verification

```sh
cd event-indexer
go test -race ./...
# Optional durable-delivery integration test against a disposable PostgreSQL DB:
ALERT_TEST_DATABASE_URL='postgres://...' go test -race ./alerts -run TestDB -count=1
```

The DB integration tests cover commit visibility, batch rollback, late blocks,
replay, row locking and delivery retries against a disposable database.

Before enabling production notifications, replay representative real transactions
for each configured route, including a multi-transfer transaction, escrow release,
failed transaction and LZ send. Create the Telegram group and add the team and bot
before supplying its chat ID. No production configuration or group is created by
this code change.

# Event alerts moved to rollup-monitoring

Event alerts now run as the independent `event_monitor` service in
[rollup-monitoring](https://github.com/initia-labs/rollup-monitoring/tree/main/internal/event_monitor).
One process reads RPC blocks, evaluates USD whale and swap-divergence rules,
and delivers queued Telegram alerts. Core-indexer no longer captures alert
batches or exposes the `alerts` and `alerts-rpc` commands.

For cutover:

1. Build and publish the new `event_monitor` image and prepare its configuration.
2. Stop the old `event-indexer alerts` and `event-indexer alerts-rpc` releases.
3. Start one `event_monitor` instance with the same `DB_CONNECTION_STRING`, chain,
   and Telegram credentials. Configure its RPC endpoints and USD thresholds.
4. Deploy this core-indexer version to stop its former alert-batch writer.

Keep the existing `transfer_alerts`, `transfer_alert_batches`, and
`transfer_alert_rpc_cursors` data. The new service retains their identities and
continues the RPC checkpoint, pending evaluations, and queued deliveries.
Existing sent-alert records continue deduplicating replay; Telegram delivery
remains at least once. A missing RPC checkpoint starts at the configured
`start_height`, or the current head when zero.

The historical alert migrations and table declarations remain in core-indexer
for schema-history compatibility. They do not start workers or write new alert
batches. Do not drop or recreate these tables during cutover.

See the new service documentation for its configuration, build, deployment,
and verification instructions.

# Event alerts moved to rollup-monitoring

Event alerts now run as the independent `event_monitor` service in
[rollup-monitoring](https://github.com/initia-labs/rollup-monitoring/tree/main/internal/event_monitor).
One process reads RPC blocks, evaluates USD whale and swap-divergence rules,
and sends Telegram alerts directly. The service has no database dependency.
Core-indexer no longer captures alert batches or exposes the `alerts` and
`alerts-rpc` commands.

For cutover:

1. Build and publish the new `event_monitor` image and prepare its configuration.
2. If existing pending database alerts must be delivered, let the old sender
   drain them before stopping the old `alerts` and `alerts-rpc` releases.
3. Start one `event_monitor` instance with RPC endpoints, USD thresholds and the
   existing Telegram credentials. No database credentials are needed.
4. Deploy this core-indexer version to stop its former alert-batch writer.

The monitor keeps progress and pending sends in memory only. It retries RPC,
price and Telegram failures while running. On restart, `start_height: 0` starts
at the current RPC head; an explicit height replays from there and can resend
alerts. Previous database queues and checkpoints are not imported.

Historical alert migrations and table declarations remain in core-indexer for
schema-history compatibility. The new monitor does not access those tables,
and this change does not drop existing data.

See the new service documentation for configuration, deployment and tests.

# Core Indexer

## Informative Indexer (Event Indexer)

### Indexed Data & Database Schema

The indexer tracks three main types of blockchain events:

**Transaction Events** (`transaction_events` table)
- Records events emitted during transaction execution
- Fields:
  - `transaction_hash` - Hash of the transaction
  - `block_height` - Block height where the event occurred
  - `event_key` - Event type and attribute key (format: "event_type.attribute_key")
  - `event_value` - Value associated with the event
  - `event_index` - Order of the event within the transaction

**Block Events** (`finalize_block_events` table)
- Captures events that occur during block finalization
- Fields:
  - `block_height` - Height of the block
  - `event_key` - Event type and attribute key
  - `event_value` - Value associated with the event
  - `event_index` - Order of the event within the block
  - `mode` - Either "BeginBlock" or "EndBlock" indicating when the event occurred

**Move Events** (`move_events` table)
- Tracks events from Move smart contract execution
- Fields:
  - `type_tag` - Move event type identifier
  - `data` - Event payload in JSON format
  - `block_height` - Block height where the event occurred
  - `transaction_hash` - Hash of the transaction that emitted the event
  - `event_index` - Order of the event within the transaction

### Components

**Sweeper** - Polls the RPC for new block data and submits it to the message queue.

**Flusher** - Consumes messages from the queue and processes them into the database.

**Prunner** - Periodically prunes the database and stores backups in a cloud storage service.

### Workflow Summary

Each service runs continuously and must be executed together for the indexer to function properly.

**Sweeper**

1. Retrieves the latest indexed block from the database.
2. Fetches data for the next block using the RPC methods `/block` and `/block_results`.
3. Publishes the block data as a message to the message queue.

**Flusher**

1. Subscribes to and reads messages from the queue.
2. Processes each message and inserts the data into the database.

**Prunner**

1. Triggers at predefined intervals.
2. Checks whether the database requires pruning.
3. If pruning is needed:
   1. Fetches prunable rows from the database.
   2. Uploads the data to a cloud storage service.
   3. Deletes the fetched rows from the database.

### Running Locally

To run the Informative Indexer with Docker locally, follow this [guide](local/README.md).

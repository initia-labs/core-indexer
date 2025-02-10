# Core Indexer

## Informative Indexer

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

Follow the guide for running the Informative Indexer with Docker: [dockerfiles/README.md](dockerfiles/README.md).

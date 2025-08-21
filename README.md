# Core Indexer

## Components

### API
REST API service that provides endpoints for querying indexed blockchain data. Built with Fiber framework and includes comprehensive Swagger documentation.

**Features:**
- Account data queries
- Block information retrieval
- Module transaction tracking
- NFT data and transaction history
- Governance proposal information
- Transaction details and history
- Validator information and metrics
- Health check endpoints
- CORS support and request logging

### Event Indexer
Specialized indexer for blockchain events with comprehensive event tracking capabilities. Processes transaction events, block events, and Move smart contract events.

**Features:**
- Transaction event processing and indexing
- Block finalization event capture
- Move smart contract event tracking
- Database migration management
- Data pruning with cloud storage backup
- Command-line interface with multiple modes

**Commands:**
- `indexer` - Main indexing process
- `migrate` - Database migration operations
- `prunner` - Data pruning and archival

### Generic Indexer
General-purpose blockchain data indexer with both continuous and scheduled processing capabilities. Handles comprehensive blockchain state tracking and account management.

**Features:**
- Continuous block processing
- Cron-based batch operations
- Account data indexing
- Block result processing
- Validator state tracking
- Batch insertion optimization
- Multi-mode operation support

**Commands:**
- `indexer` - Continuous indexing mode
- `indexercron` - Scheduled batch processing mode

### Informative Indexer
Comprehensive blockchain data processor with specialized module processors for different blockchain components. Features advanced state tracking and caching mechanisms.

**Features:**
- Modular processor architecture (auth, bank, IBC, move, OPinit, gov, staking)
- State tracking and management
- Data caching for performance optimization
- Genesis block processing
- Event processing utilities
- Validator uptime tracking
- Batch state updates

**Commands:**
- `indexer` - Main processing engine
- `migrate` - Database schema management

### Sweeper
High-performance data collection service that polls RPC endpoints for new blockchain data and distributes it via message queues.

**Features:**
- RPC endpoint polling
- Block data retrieval
- Message queue publishing
- Database migration on startup
- Error handling and retry logic
- Configurable polling intervals

### TX Response Uploader
Message queue consumer that processes transaction response data and uploads it to cloud storage systems with support for large message handling.

**Features:**
- Kafka message consumption
- Cloud storage integration (GCS)
- Claim check pattern support for large messages
- Dead letter queue error handling
- Retry mechanisms with exponential backoff
- Transaction response archival
- Sentry integration for monitoring

### Workflow Summary

The indexer ecosystem consists of multiple specialized services that work together to provide comprehensive blockchain data indexing and querying capabilities.

**Core Data Flow:**

1. **Sweeper** retrieves new block data from RPC endpoints and publishes it to message queues
2. **Indexers** (Event, Generic, Informative) consume messages and process blockchain data into the database
3. **API** serves the indexed data through REST endpoints
4. **TX Response Uploader** handles transaction response storage in cloud services
5. **Pruners** manage data lifecycle and storage optimization

**Sweeper**

1. On startup, applies any pending database migrations.
2. Retrieves the latest indexed block from the database.
3. Fetches data for the next block using the RPC methods `/block` and `/block_results`.
4. Publishes the block data as a message to the message queue.

**Indexers**

1. Subscribe to and read messages from the queue.
2. Process each message using specialized processors for different blockchain modules.
3. Insert processed data into the database with batch operations for efficiency.
4. Handle state tracking and caching for optimized performance.

**Prunner**

1. Triggers at predefined intervals.
2. Checks whether the database requires pruning.
3. If pruning is needed:
   1. Fetches prunable rows from the database.
   2. Uploads the data to a cloud storage service.
   3. Deletes the fetched rows from the database.

### Running Locally

To run the Informative Indexer with Docker locally, follow this [guide](local/README.md).

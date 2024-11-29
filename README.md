# Core Indexer

## Generic Indexer

### Terms

Sweeper = service that polls block to be indexed from RPC

Flusher = service that polls block to be indexed from Kafka and indexes them in the DB

Sw = num of Sweeper workers

Fw = num of Flusher workers

### Pseudocode

1. Get last indexed block from DB
2. Query next Sw blocks - each Sweeper worker queries one
3. Each Sweeper worker calls rpc's `block` method, then enqueue the response, may be in the form of

```json
{
  "height": "123456",
  "hash": "wfwefwefw",
  "timestamp": "fwefwefweffw",
  "txs": ["tx1_hash", "tx2_hash", ...]
}
```

1. Each Flusher worker subscribes to a Kafka partition
   1. Reads a message from the partition
   2. Queries RPC to get all tx data from that block (should have multiple goroutine for this task)
   3. Check if all calls return successfully, if not retry 3.2
   4. Open a DB transaction, insert into `blocks` and insert all txs into `transactions` (make sure block timestamp and indexed timestamp is in both schemas). For duplicate txs -> ignore, other errors -> rollback
   5. Repeat 3.1

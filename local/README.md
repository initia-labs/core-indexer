# Informative Indexer (Event Indexer) Local Setup Guide

## Overview
By default, it's configured for the `initiation-2` (Initia testnet), but can be configured for any Initia's Move rollup chain.

## Prerequisites
- Docker and Docker Compose installed
- Basic understanding of blockchain concepts
- Access to an RPC endpoint of the chain you want to index

## Setup Instructions

### 1. Environment Configuration
1. Create a `.env` file in the `local` directory
   - Use the provided `.env.example` file as a reference
   - Example configuration:
   ```env
   POSTGRES_USER="postgres"
   POSTGRES_PASSWORD="postgres"
   POSTGRES_DB="core_indexer"
   CHAIN="initiation-2"
   ENVIRONMENT="local"
   ```

### 2. Database initialization
The `migrate` service applies the repository SQL migrations before either worker
starts. The indexer initializes tracking from genesis; the sweeper retries until
that tracking row is available. The PostgreSQL init script creates only the Hasura
database and the `readonly` role required by the migrations.

If your `pg_data` volume was created by the previous init script, reset the local
stack before the first start with this version. Its existing tables conflict with
the migrations (`relation already exists`), and PostgreSQL does not rerun the new
init script on an existing volume.

**This deletes the local database and mock GCS volumes, including indexed data
and claim-check objects. Back up anything you need before running it.**

```sh
docker compose --env-file local/.env -f local/docker-compose-informative.yml down -v
```

Then start the stack using the command below to initialize a fresh database.
Fresh volumes and databases already managed by these migrations do not need this
reset.

### 3. RPC Configuration
1. Open `docker-compose-informative.yml`
2. Update the `RPC_ENDPOINTS` environment variable under the `sweeper` service:
   ```yaml
   sweeper:
     environment:
       RPC_ENDPOINTS: '{"rpcs":[{"url": "YOUR_RPC_ENDPOINT"}]}'
   ```
   - For rollup indexing: Use your rollup's RPC endpoint
   - For `initiation-2`: leave it as `https://rpc.testnet.initia.xyz`

## Running the Service

### Starting the Indexer
```shell
docker compose --env-file local/.env -f local/docker-compose-informative.yml up --build
```

### Verifying the Setup
1. Check if all containers are running:
   ```shell
   docker compose --env-file local/.env -f local/docker-compose-informative.yml ps
   ```
2. Monitor logs:
   ```shell
   docker compose --env-file local/.env -f local/docker-compose-informative.yml logs -f
   ```

## Pub/Sub emulator

The Compose stack starts Pub/Sub on `localhost:8085` and creates the same six topics
and four subscriptions as zone-ops. The sweeper publishes to the concurrent topic
first, then the sequential topic consumed by the informative indexer. Only the
informative subscription enables ordering. No GCP credentials are required.
For processes outside Docker, set
`PUBSUB_PROJECT_ID=core-indexer-local` and `PUBSUB_EMULATOR_HOST=localhost:8085`.
The emulator is ephemeral; restarting it clears messages and subscriptions. Stop
both workers and restart them after recreating the resources so the sweeper reloads
the database height instead of continuing past messages lost from the emulator:

```sh
docker compose --env-file local/.env -f local/docker-compose-informative.yml stop sweeper indexer
docker compose --env-file local/.env -f local/docker-compose-informative.yml restart pubsub
docker compose --env-file local/.env -f local/docker-compose-informative.yml run --rm pubsub-init
docker compose --env-file local/.env -f local/docker-compose-informative.yml start indexer sweeper
```

This local stack does not run the tx response uploader.

## Components

### Storage System (Mock-GCS)
The indexer uses [mock-gcs-server](https://github.com/fsouza/fake-gcs-server) for local storage simulation.

- Access the storage interface: `http://localhost:9184/storage/v1/b/{bucket_name}`
- Default bucket format: `<chain-id>-local-informative-indexer-large-block-results`
- Example bucket information:
  ```json
  {
    "kind": "storage#bucket",
    "id": "<chain-id>-local-informative-indexer-large-block-results",
    "name": "<chain-id>-local-informative-indexer-large-block-results",
    "versioning": { "enabled": false },
    "timeCreated": "2025-01-23T06:11:01.205566Z",
    "updated": "2025-01-23T06:11:01.205566Z",
    "location": "US-CENTRAL1",
    "storageClass": "STANDARD"
  }
  ```

### GraphQL Interface (Hasura)
The indexer exposes a GraphQL API through Hasura for querying indexed data.

1. Initialize table tracking:
   ```shell
   curl -X POST -H 'Content-Type: application/json' \
     --data '{
       "type": "bulk",
       "source": "default",
       "resource_version": 1,
       "args": [
         {
           "type": "postgres_track_tables",
           "args": {
             "allow_warnings": true,
             "tables": [
               {"table": {"name": "finalize_block_events", "schema": "public"}, "source": "default"},
               {"table": {"name": "move_events", "schema": "public"}, "source": "default"},
               {"table": {"name": "transaction_events", "schema": "public"}, "source": "default"}
             ]
           }
         }
       ]
     }' \
     http://localhost:8080/v1/metadata
   ```
2. Access the GraphQL console: `http://localhost:8080/console`

## Troubleshooting

### Common Issues
1. Connection refused to RPC endpoint
   - Verify RPC endpoint is accessible
   - Check network connectivity
   - Ensure RPC endpoint is correctly formatted in configuration

2. Database connection issues
   - Verify PostgreSQL credentials in .env file
   - Check if PostgreSQL container is running
   - Ensure no port conflicts on 5432 (can be changed in docker-compose-informative.yml under postgres service ports).

3. Hasura metadata issues
   - Reset metadata and reapply if tracking fails
   - Verify PostgreSQL connection in Hasura console
   - Check Hasura logs for detailed error messages

## Support and Contribution
For issues and feature requests, please create an issue in the repository.

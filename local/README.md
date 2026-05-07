# Informative Indexer (Event Indexer) Local Setup Guide

## Overview
By default, it's configured for the `initiation-2` (Initia testnet), but can be configured for any Initia's Move rollup chain.

## Prerequisites
- Docker and Docker Compose installed
- Basic understanding of blockchain concepts
- Access to an RPC endpoint of the chain you want to index

## Setup Instructions

### 1. Environment Configuration
1. Create a `.env` file in `dockerfiles` directory
   - Use the provided `.env.example` file as a reference
   - Example configuration:
   ```env
   POSTGRES_USER="postgres"
   POSTGRES_PASSWORD="postgres"
   POSTGRES_DB="core_indexer"
   CHAIN="initiation-2"
   ENVIRONMENT="local"
   ```

### 2. Initial Block Height Configuration
1. Navigate to `init/init.sql`
2. Modify the `block_height` value to your desired starting point
   - This determines from which block the indexer will start processing

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
docker compose -f local/docker-compose-informative.yml up --build
```

### Verifying the Setup
1. Check if all containers are running:
   ```shell
   docker compose -f local/docker-compose-informative.yml ps
   ```
2. Monitor logs:
   ```shell
   docker compose -f local/docker-compose-informative.yml logs -f
   ```

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

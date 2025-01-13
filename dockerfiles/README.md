## Docker compose for informative indexer

### Prerequisite
1. Create a .env file.
- Add the following content to your `.env` file.
```json
  POSTGRES_USER="postgres"
  POSTGRES_PASSWORD="postgres"
  POSTGRES_DB="core_indexer"
  CHAIN="initiation-2"
  ENVIRONMENT="local"
  CLAIM_CHECK_BUCKET="claim_check_bucket"
```
2. Update the storage path for the claim_check_bucket
- If you need to change `CLAIM_CHECK_BUCKET` value in the `.env` file, update the `.storage/claim_check_bucket` directory accordingly.

3. Update the `block_height` value
- Modify the `block_height` value if you want to change the starting block height.
- Default: 2797040

### Run
Run the following command to start the service:
```shell
docker compose -f dockerfiles/docker-compose-informative.yml up --build
```

### Fake-GCS
We are using [fake-gcs-server](https://github.com/fsouza/fake-gcs-server) to enable the storage feature.
You can view your fake GCS information at: `http://localhost:9184/storage/v1/b/{bucket_name}`
```json
{
  "kind": "storage#bucket",
  "id": "claim_check_bucket",
  "defaultEventBasedHold": false,
  "name": "claim_check_bucket",
  "versioning": {
    "enabled": false
  },
  "timeCreated": "2025-01-10T05:33:37.457352Z",
  "updated": "2025-01-10T05:33:37.457352Z",
  "location": "US-CENTRAL1",
  "storageClass": "STANDARD",
  "projectNumber": "0",
  "metageneration": "1",
  "etag": "RVRhZw==",
  "locationType": "region"
}
```
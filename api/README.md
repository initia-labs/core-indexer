# Core Indexer API

This is the API service for the Core Indexer project, built with Go Fiber.

## Prerequisites

- Go 1.21 or higher
- Git
- Air (for hot reloading)

## Setup

1. Install dependencies:

```bash
go mod download
```

2. Create a `.env` file in the root directory (optional):

```bash
PORT=8080
```

## Running the Application

### Development Mode (with Hot Reload)

To run the application with hot reload (recommended for development):

```bash
air
```

The server will automatically restart when you make changes to the code.

### Production Mode

To run the application without hot reload:

```bash
go run main.go
```

The server will start on port 3000 by default (or the port specified in your .env file).

## API Endpoints

- `GET /indexer/`: Welcome message
- `GET /indexer/health`: Health check endpoint
- `GET /indexer/swagger/*`: Swagger documentation

## Project Structure

```
api/
├── main.go           # Application entry point
├── routes/           # Route definitions
│   └── routes.go
├── dto/             # Data Transfer Objects
│   └── nft.go
├── go.mod           # Go module file
├── .air.toml        # Air configuration
└── README.md        # This file
```

## Pagination limits

Paginated list endpoints accept `pagination.offset` from 0 through 10,000,
including offsets decoded from `pagination.key`. Larger effective offsets return
HTTP 400 before querying the database. When a key is present, its decoded offset
takes precedence over the explicit offset.

`pagination.limit` remains 1–1,000. Responses omit `next_key` when the next page's
offset would exceed 10,000, even if more records exist. `total`, when requested,
still describes all matching records; a missing `next_key` can indicate the
pagination limit rather than the end of the dataset.

## Validator uptime window

`GET /indexer/validator/v1/validators/{operatorAddr}/uptime` requires a positive
`blocks` value. The effective window is capped at 10,000 blocks and the latest
indexed height before querying block votes or allocating history. Larger values
are accepted and capped; `uptime.total` reports the effective window, while
`recent_100_blocks` contains at most its latest 100 blocks.

## Proposal type catalog

The proposal type catalog query uses `REPOSITORY_COUNT_QUERY_TIMEOUT` (default
`5s`) as its execution deadline. PostgreSQL extracts distinct type names, and
the API retains its existing ordering. Query failures, including timeouts,
return an error rather than a partial catalog.

## Development

To add new routes:

1. Create new handler functions in the appropriate package
2. Add the routes in `routes/routes.go`
3. Import and use the handlers in your routes

### Transaction lookup caller monitoring

For `GET /indexer/tx/v1/txs/{tx_hash}`, the API logs `Transaction lookup requested`
and adds the same caller fields to storage lookup logs, including
`Found latest transaction across all buckets`:

- `caller`: `scan-api` when `X-Scan-Api-Path` is present and nonempty; otherwise `direct`.
- `scan_api_path`: the originating scan API request path, when supplied, without its query string.
- `indexer_path`: the requested indexer path.

The scan API populates `X-Scan-Api-Path` from its own incoming request path on
outbound indexer requests, including transaction detail lookups during list
enrichment. Both services must be deployed for automatic attribution. The header
is monitoring metadata, not authenticated caller identity; `direct` also includes
older scan API instances or proxies that omit the header.

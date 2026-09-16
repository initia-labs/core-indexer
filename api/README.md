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

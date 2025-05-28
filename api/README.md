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

- `GET /`: Welcome message
- `GET /swagger/*`: Swagger documentation

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

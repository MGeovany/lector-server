# PDF Text Reader

HTTP server in Go for processing and reading text from PDF files.

## Requirements

- Go 1.25.5 or higher

## Installation

```bash
# Install dependencies
go mod download

# Install Air for hot-reload (optional but recommended)
go install github.com/air-verse/air@latest
```

**Note:** If `air` is not in your PATH after installation, add `~/go/bin` to your PATH or use `~/go/bin/air` directly.

## Running

### Hot-reload using Make

```bash
make dev
```

This will automatically detect changes and restart the server. The Makefile handles the `air` path automatically.

### Option 1b: With hot-reload directly

```bash
# Use the full path to avoid conflicts with other 'air' tools
~/go/bin/air

# Or if ~/go/bin is in your PATH:
air
```

**Note:** If you get an error about "R language server", it means there's another `air` tool in your PATH. Use `make dev` or `~/go/bin/air` directly.

### Option 2: Run directly with Make

```bash
make run
```

### Option 3: Run directly with `go run`

```bash
go run cmd/server/main.go
```

### Option 4: Build and run

```bash
# Build
make build

# Run
./bin/server
```

Or use Make:

```bash
make build && ./bin/server
```

### Other useful Make commands

```bash
make help      # Show all available commands
make test      # Run tests
make vet       # Run go vet
make fmt       # Format code
make clean     # Clean build artifacts
```

## Environment Variables

The server uses the following environment variables (all are optional):

- `SERVER_PORT`: Server port (default: `8080`)
- `UPLOAD_PATH`: Directory for uploading files (default: `./uploads`)
- `MAX_FILE_SIZE`: Maximum file size in bytes (default: `52428800` = 50MB)
- `LOG_LEVEL`: Logging level (default: `info`)
- `DATABASE_PATH`: Database path (default: `./data`)

### Example with environment variables:

```bash
SERVER_PORT=3000 LOG_LEVEL=debug go run cmd/server/main.go
```

## Endpoints

- `GET /health` - Server health check

## Stopping the server

Press `Ctrl+C` to stop the server gracefully.

# Development Guide

## Requirements

- Go version specified in `src/go.mod`
- Git
- golangci-lint v2.1.6 for lint checks

## Setup

Clone the repository and enter the Go module directory:

```bash
cd src
```

Copy the example environment file:

```bash
cp .env.example .env
```

Download dependencies:

```bash
go mod download
```

## Run the server

```bash
make run
```

The server listens on port 8000 by default.

Override configuration using environment variables:

```bash
PORT=9000 make run
```

Existing process environment variables take precedence over values
loaded from `.env`.

## Verify the application

Check the health endpoint:

```bash
curl -i http://localhost:8000/health
```

Expected response body:

```json
{"status":"healthy"}
```

## Quality checks

Run all checks:

```bash
make check
```

Run individual checks:

```bash
make fmt-check
make test
make vet
make lint
```

To format Go files:

```bash
make fmt
```

## Troubleshooting

### Invalid PORT

PORT must be an integer between 1 and 65535.

### Port already in use

Set PORT to another available port.

### Missing golangci-lint

Install the version specified in this guide before running `make lint`.

### Tests fail

Run `go test ./...` from the `src` directory and inspect the
reported package and test name.

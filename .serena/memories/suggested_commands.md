# Suggested Commands for Router Manager Go

## Service Execution

```bash
# Run API service
cd services/api
go run ./cmd/app

# Run Batch service  
cd services/batch
go run ./cmd/batch
```

## Debugging (VS Code)

1. Press `Ctrl+Shift+D` to open "RUN AND DEBUG" menu
2. Select service from dropdown
3. Press `F5` to start debugging

## Linting and Formatting

```bash
# Run golangci-lint (from any Go module directory)
golangci-lint run ./...

# Auto-fix some issues
golangci-lint run --fix

# Format Go code with gofumpt
gofumpt -w .

# Format other files (JSON, Markdown, YAML, TOML)
dprint fmt

# Check formatting without changes
dprint check
```

## Module Management

```bash
# Update dependencies for a service
cd services/api
go mod tidy

# Update all modules
find . -name go.mod -exec dirname {} \; | xargs -I {} sh -c 'cd {} && go mod tidy'
```

## Git Hooks

```bash
# Install git hooks (run from repo root)
lefthook install

# Run hooks manually
lefthook run pre-commit
lefthook run pre-push
```

## Database Access (Development)

```bash
# Login to development PostgreSQL
docker exec -it router-manager-go_devcontainer-postgres-1 psql -U postgres -d router_manager
```

## Build Commands

```bash
# Build API service
cd services/api
go build ./cmd/app

# Build Batch service
cd services/batch
go build ./cmd/batch
```

## Testing

```bash
# Run tests for a service
cd services/api
go test ./...

# Run tests with coverage
go test -cover ./...
```

## System Utilities (Linux)

- `git`: Version control
- `ls`: List files and directories
- `cd`: Change directory
- `grep`/`rg`: Search in files (prefer ripgrep)
- `find`: Find files and directories

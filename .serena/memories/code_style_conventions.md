# Code Style and Conventions

## Go Code Style

- **Go Version**: 1.24
- **Formatter**: gofumpt (via golangci-lint)
- **Import Organizer**: goimports

## Linting Rules (golangci-lint)

### Error Checking

- `errcheck`: Detect unchecked errors
- `errorlint`: Error handling best practices
- `govet`: Detect suspicious code patterns
- `staticcheck`: Advanced static analysis
- `rowserrcheck`: Database operation error checking
- `sqlclosecheck`: SQL close leak detection

### Code Quality

- `gocritic`: Code improvement suggestions
- `gosec`: Security vulnerability checks (excluding G112 Slowloris)
- `ineffassign`: Detect ineffective assignments
- `noctx`: Context usage enforcement
- `revive`: Modern golint replacement (with use-any rule enabled)

### Performance

- `perfsprint`: String concatenation optimization

### Test File Exceptions

- `errcheck` and `gosec` are disabled for test files (**test.go, test**.go)

## Architecture Principles

1. **Clean Architecture**: Business logic isolation
2. **Internal Packages**: Each service uses `internal/` to prevent cross-service imports
3. **Module Boundaries**: Each service has its own go.mod with replace directives for local packages
4. **Shared Code**: Common functionality in `pkg/` directory

## Naming Conventions

- Follow standard Go naming conventions
- Use meaningful variable and function names
- Prefer clarity over brevity

## Important Notes from CLAUDE.md

- Do NOT remove Japanese comments while keeping corresponding source code
- Build should succeed after editing *.go files
- Test nftables/dnsmasq operations in test environment
- Format with gofumpt after editing: `gofumpt -w .`
- Run `golangci-lint run ./...` and ensure no warnings
- Public methods should have unit tests (except very simple ones)

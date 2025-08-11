# Task Completion Checklist

When completing any coding task in this project, ensure the following steps are performed:

## 1. Code Quality Checks

```bash
# Run golangci-lint from the module directory
cd services/<service_name>
golangci-lint run ./...
```

- Ensure no warnings or errors are reported
- If there are fixable issues, run: `golangci-lint run --fix`

## 2. Code Formatting

```bash
# Format Go code with gofumpt
gofumpt -w .

# Format other files (JSON, Markdown, YAML, TOML)
dprint fmt
```

## 3. Build Verification

```bash
# Verify the code builds successfully
go build ./cmd/app  # for API service
# or
go build ./cmd/batch  # for Batch service
```

## 4. Testing

```bash
# Run tests if they exist
go test ./...
```

- Implement unit tests for public methods (except trivial ones)

## 5. Module Dependencies

```bash
# Update dependencies if needed
go mod tidy
```

## 6. Pre-commit Hooks

The following will run automatically on commit (via lefthook):

- dprint formatting for non-Go files
- Files will be automatically staged after formatting

## 7. Pre-push Hooks

The following will run automatically on push:

- golangci-lint for all Go modules
- Auto-fix will be attempted

## Special Considerations

- **Japanese Comments**: Do NOT remove Japanese comments if the corresponding code remains
- **System Operations**: Test nftables/dnsmasq operations in a test environment
- **Security**: Never commit secrets or API keys
- **Clean Architecture**: Maintain separation of concerns

## Important File Guidelines

- NEVER create files unless absolutely necessary
- ALWAYS prefer editing existing files
- NEVER proactively create documentation files unless explicitly requested

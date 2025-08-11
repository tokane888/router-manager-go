# Development Environment Setup

## Prerequisites

- Linux environment (required for dnsmasq and nftables)
- Go 1.24+
- Docker and VS Code with DevContainer support

## Environment Variables (Local)

For MCP server (cipher) usage, set on host:

- `ANTHROPIC_API_KEY`: Anthropic Claude API key
- `OLLAMA_BASE_URL`: Ollama local LLM URL (optional)

Example (~/.zshrc or ~/.bashrc):

```bash
export ANTHROPIC_API_KEY="sk-ant-xxx..."
```

## DevContainer Features

- Pre-installed tools:
  - Go 1.24
  - golangci-lint
  - gofumpt
  - dprint
  - lefthook
  - PostgreSQL client
  - Git hooks support

## Initial Setup Steps

1. Start DevContainer in VS Code
2. Install git hooks: `lefthook install`
3. Create .env files for configuration
4. Verify permissions for nftables/dnsmasq operations

## Database

- Development PostgreSQL runs in Docker container
- Database name: `router_manager`
- Access: `docker exec -it router-manager-go_devcontainer-postgres-1 psql -U postgres -d router_manager`

## Configuration Files

- `.golangci.yml`: Comprehensive linting rules
- `dprint.json`: Formatting for non-Go files
- `lefthook.yml`: Git hooks configuration
- `.devcontainer/devcontainer.json`: VS Code dev environment

## Project Layout

Follows [Standard Go Project Layout](https://github.com/golang-standards/project-layout)

- `/services`: Microservices (api, batch)
- `/pkg`: Shared packages
- `/internal`: Private implementation (per service)
- `/cmd`: Entry points for services

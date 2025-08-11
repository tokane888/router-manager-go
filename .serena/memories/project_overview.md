# Router Manager Go - Project Overview

## Purpose

This is a router management system providing domain blocking functionality using dnsmasq and nftables.

## Architecture

- **Monorepo structure**: Multiple Go services in a single repository
- **Clean Architecture**: Each service follows clean architecture principles
- **Microservices**:
  - **API Service**: Manages blocked domains (dnsmasq settings and nftables IP blocking)
  - **Batch Service**: Periodically resolves domain names and updates nftables rules

## Main Functionality

### API Service

- Edits dnsmasq configurations to add domains for name resolution blocking
- Registers domains in DB for IP blocking via nftables

### Batch Service

- Runs periodically to:
  - Resolve domain names (2-5 times at 30-second intervals to handle round-robin DNS)
  - Register new IP addresses in DB and block packet forwarding
  - Compare resolved IPs with registered ones and update nftables rules accordingly

## Tech Stack

- **Language**: Go 1.24
- **Web Framework**: Gin (for API service)
- **Logging**: uber/zap with shared logger package
- **Configuration**: godotenv (.env files)
- **Database**: PostgreSQL
- **System Tools**: dnsmasq, nftables (Linux-specific)

## Directory Structure

```
├── services/           # Microservices
│   ├── api/           # API service
│   └── batch/         # Batch service
├── pkg/               # Shared packages
│   └── logger/        # Shared logging with zap
├── db/                # Database related files
├── memAgent/          # Memory agent files
└── .devcontainer/     # VS Code DevContainer settings
```

## Development Environment

- DevContainer with all tools pre-installed
- Git hooks via lefthook
- Comprehensive linting and formatting setup

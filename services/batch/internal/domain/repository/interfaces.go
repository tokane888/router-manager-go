package repository

import (
	"context"

	"github.com/tokane888/router-manager-go/pkg/db"
)

// DNSResolver defines the interface for DNS resolution operations
type DNSResolver interface {
	ResolveIPs(ctx context.Context, domain string) ([]string, error)
}

// NFTablesManager defines the interface for nftables rule management
type NFTablesManager interface {
	AddBlockRule(ctx context.Context, ip string) error
	RemoveBlockRule(ctx context.Context, ip string) error
}

// DomainRepository defines the interface for domain data operations
type DomainRepository interface {
	// Domain operations
	GetAllDomains(ctx context.Context) ([]db.Domain, error)
	CreateDomain(ctx context.Context, domainName string) error

	// Domain IP operations
	GetDomainIPs(ctx context.Context, domainName string) ([]db.DomainIP, error)
	CreateDomainIP(ctx context.Context, domainName, ipAddress string) error
	DeleteDomainIP(ctx context.Context, domainName, ipAddress string) error
	GetAllDomainIPs(ctx context.Context) ([]db.DomainIP, error)
}

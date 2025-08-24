package repository

import (
	"context"
)

// DNSResolver defines the interface for DNS resolution operations
type DNSResolver interface {
	ResolveIPs(ctx context.Context, domain string) ([]string, error)
}

// FirewallManager defines the interface for firewall rule management
type FirewallManager interface {
	AddBlockRule(ctx context.Context, ip string) error
	RemoveBlockRule(ctx context.Context, ip string) error
}

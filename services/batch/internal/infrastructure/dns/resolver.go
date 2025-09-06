package dns

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/tokane888/router-manager-go/services/batch/internal/domain/repository"
	"go.uber.org/zap"
)

// NetResolver is an interface for DNS resolution (useful for testing)
type NetResolver interface {
	LookupIP(ctx context.Context, network, host string) ([]net.IP, error)
}

// dnsResolverImpl implements the DNSResolver interface
type dnsResolverImpl struct {
	resolver      NetResolver
	logger        *zap.Logger
	timeout       time.Duration
	retryAttempts int
}

// NewDNSResolver creates a new DNS resolver implementation
// resolver parameter should be net.DefaultResolver for production use,
// or a mock implementation for testing

// DNSConfig contains DNS resolution configuration
type DNSConfig struct {
	Timeout       time.Duration
	RetryAttempts int
}

func NewDNSResolver(cfg DNSConfig, resolver NetResolver, logger *zap.Logger) repository.DNSResolver {
	return &dnsResolverImpl{
		resolver:      resolver,
		logger:        logger,
		timeout:       cfg.Timeout,
		retryAttempts: cfg.RetryAttempts,
	}
}

// ResolveIPs resolves domain name to IPv4 addresses
func (r *dnsResolverImpl) ResolveIPs(ctx context.Context, domain string) ([]string, error) {
	r.logger.Debug("Starting DNS resolution",
		zap.String("domain", domain),
		zap.Duration("timeout", r.timeout),
		zap.Int("retryAttempts", r.retryAttempts))

	var lastErr error
	for attempt := 0; attempt <= r.retryAttempts; attempt++ {
		if attempt > 0 {
			r.logger.Debug("Retrying DNS resolution",
				zap.String("domain", domain),
				zap.Int("attempt", attempt))
		}

		ips, err := r.resolveWithTimeout(ctx, domain)
		if err != nil {
			lastErr = err
			r.logger.Warn("DNS resolution attempt failed",
				zap.String("domain", domain),
				zap.Int("attempt", attempt),
				zap.Error(err))

			// Don't sleep after the last attempt
			if attempt < r.retryAttempts {
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(time.Second * time.Duration(attempt+1)):
					// Exponential backoff: 1s, 2s, 3s, etc.
				}
			}
			continue
		}

		// Success
		r.logger.Info("Successfully resolved domain",
			zap.String("domain", domain),
			zap.Int("ipCount", len(ips)),
			zap.Strings("ips", ips))
		return ips, nil
	}

	r.logger.Error("All DNS resolution attempts failed",
		zap.String("domain", domain),
		zap.Int("totalAttempts", r.retryAttempts+1),
		zap.Error(lastErr))

	return nil, fmt.Errorf("failed to resolve domain %s after %d attempts: %w",
		domain, r.retryAttempts+1, lastErr)
}

// resolveWithTimeout performs DNS resolution with a timeout (IPv4 only)
func (r *dnsResolverImpl) resolveWithTimeout(ctx context.Context, domain string) ([]string, error) {
	// Create a context with timeout
	resolveCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	// Resolve IPv4 addresses only
	ipv4Addrs, err := r.resolver.LookupIP(resolveCtx, "ip4", domain)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve IPv4 addresses for domain %s: %w", domain, err)
	}

	if len(ipv4Addrs) == 0 {
		return nil, fmt.Errorf("no IPv4 addresses found for domain %s", domain)
	}

	// Convert to string slice
	var ips []string
	for _, ip := range ipv4Addrs {
		ips = append(ips, ip.String())
	}

	r.logger.Debug("IPv4 addresses resolved",
		zap.String("domain", domain),
		zap.Int("count", len(ipv4Addrs)))

	return ips, nil
}

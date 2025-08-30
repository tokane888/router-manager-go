package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/tokane888/router-manager-go/services/batch/internal/domain/repository"
	"go.uber.org/zap"
)

// ProcessingConfig contains domain processing configuration
type ProcessingConfig struct {
	MaxConcurrency int // Configurable via environment variable, default 10
	DomainTimeout  time.Duration
}

type DomainBlockerUseCase struct {
	domainRepo      repository.DomainRepository
	dnsResolver     repository.DNSResolver
	firewallManager repository.FirewallManager
	logger          *zap.Logger
}

// NewDomainBlockerUseCase creates a new instance of DomainBlockerUseCase
func NewDomainBlockerUseCase(
	domainRepo repository.DomainRepository,
	dnsResolver repository.DNSResolver,
	firewallManager repository.FirewallManager,
	logger *zap.Logger,
) *DomainBlockerUseCase {
	return &DomainBlockerUseCase{
		domainRepo:      domainRepo,
		dnsResolver:     dnsResolver,
		firewallManager: firewallManager,
		logger:          logger,
	}
}

// ProcessAllDomains processes all domains from the database
func (uc *DomainBlockerUseCase) ProcessAllDomains(ctx context.Context) error {
	// Retrieve all domains from the database
	domains, err := uc.domainRepo.GetAllDomains(ctx)
	if err != nil {
		uc.logger.Error("Failed to retrieve domains from database", zap.Error(err))
		return err
	}

	uc.logger.Info("Retrieved domains from database", zap.Int("count", len(domains)))

	// Process each domain
	for _, domain := range domains {
		uc.logger.Info("Processing domain", zap.String("domain", domain.DomainName))

		if err := uc.processDomain(ctx, domain.DomainName); err != nil {
			uc.logger.Error("Failed to process domain",
				zap.String("domain", domain.DomainName),
				zap.Error(err))
			// Continue processing other domains even if one fails
			continue
		}
	}

	return nil
}

// processDomain processes a single domain
func (uc *DomainBlockerUseCase) processDomain(ctx context.Context, domain string) error {
	uc.logger.Info("Processing single domain", zap.String("domain", domain))

	// Discover all IPs for the domain
	discoveredIPs, err := uc.discoverAllIPs(ctx, domain)
	if err != nil {
		return fmt.Errorf("failed to discover IPs for domain %s: %w", domain, err)
	}

	uc.logger.Info("Discovered IPs for domain",
		zap.String("domain", domain),
		zap.Int("ip_count", len(discoveredIPs)))

	// Update firewall rules based on discovered IPs
	if err := uc.updateFirewallRules(ctx, domain, discoveredIPs); err != nil {
		return fmt.Errorf("failed to update firewall rules for domain %s: %w", domain, err)
	}

	uc.logger.Info("Successfully processed domain", zap.String("domain", domain))
	return nil
}

// discoverAllIPs discovers all IP addresses for a domain
func (uc *DomainBlockerUseCase) discoverAllIPs(ctx context.Context, domain string) ([]string, error) {
	uc.logger.Info("Discovering IPs for domain", zap.String("domain", domain))

	// Use the DNS resolver to get actual IP addresses
	ips, err := uc.dnsResolver.ResolveIPs(ctx, domain)
	if err != nil {
		uc.logger.Error("Failed to resolve IPs for domain",
			zap.String("domain", domain),
			zap.Error(err))
		return nil, fmt.Errorf("DNS resolution failed for domain %s: %w", domain, err)
	}

	if len(ips) == 0 {
		uc.logger.Warn("No IPs discovered for domain", zap.String("domain", domain))
		return []string{}, nil
	}

	uc.logger.Info("IP discovery completed",
		zap.String("domain", domain),
		zap.Strings("ips", ips))

	return ips, nil
}

// updateFirewallRules updates firewall rules based on discovered IPs
func (uc *DomainBlockerUseCase) updateFirewallRules(ctx context.Context, domain string, newIPs []string) error {
	// Get existing IPs
	existingIPs, err := uc.getExistingIPs(ctx, domain)
	if err != nil {
		return fmt.Errorf("failed to get existing IPs for domain %s: %w", domain, err)
	}

	// Calculate changes
	ipsToAdd, ipsToRemove := uc.calculateIPChanges(existingIPs, newIPs)

	// Apply changes
	if err := uc.applyIPChanges(ctx, domain, ipsToAdd, ipsToRemove); err != nil {
		return fmt.Errorf("failed to apply IP changes for domain %s: %w", domain, err)
	}

	uc.logger.Info("Completed firewall rules update",
		zap.String("domain", domain),
		zap.Int("added", len(ipsToAdd)),
		zap.Int("removed", len(ipsToRemove)))

	return nil
}

// getExistingIPs retrieves existing IPs for a domain
// nolint: unused
func (uc *DomainBlockerUseCase) getExistingIPs(ctx context.Context, domain string) ([]string, error) {
	domainIPs, err := uc.domainRepo.GetDomainIPs(ctx, domain)
	if err != nil {
		return nil, err
	}

	existingIPs := make([]string, 0, len(domainIPs))
	for _, domainIP := range domainIPs {
		existingIPs = append(existingIPs, domainIP.IPAddress)
	}
	return existingIPs, nil
}

// calculateIPChanges determines which IPs need to be added or removed
// nolint: unused
func (uc *DomainBlockerUseCase) calculateIPChanges(existingIPs, newIPs []string) ([]string, []string) {
	// Convert to maps for O(1) lookup
	existingIPsMap := make(map[string]bool)
	for _, ip := range existingIPs {
		existingIPsMap[ip] = true
	}

	newIPsMap := make(map[string]bool)
	for _, ip := range newIPs {
		newIPsMap[ip] = true
	}

	// Find IPs to add (exist in newIPs but not in existing)
	var ipsToAdd []string
	for ip := range newIPsMap {
		if !existingIPsMap[ip] {
			ipsToAdd = append(ipsToAdd, ip)
		}
	}

	// Find IPs to remove (exist in existing but not in newIPs)
	var ipsToRemove []string
	for ip := range existingIPsMap {
		if !newIPsMap[ip] {
			ipsToRemove = append(ipsToRemove, ip)
		}
	}

	return ipsToAdd, ipsToRemove
}

// applyIPChanges applies the calculated IP changes to the database
// nolint: unused
func (uc *DomainBlockerUseCase) applyIPChanges(ctx context.Context, domain string, ipsToAdd, ipsToRemove []string) error {
	// Add new IPs
	for _, ip := range ipsToAdd {
		uc.logger.Info("Adding firewall rule and domain IP",
			zap.String("domain", domain),
			zap.String("ip", ip))

		// Add firewall rule first
		if err := uc.firewallManager.AddBlockRule(ctx, ip); err != nil {
			uc.logger.Warn("Failed to add firewall rule, continuing with others",
				zap.String("domain", domain),
				zap.String("ip", ip),
				zap.Error(err))
			continue
		}

		// Then add to database
		if err := uc.domainRepo.CreateDomainIP(ctx, domain, ip); err != nil {
			// If database insertion fails, try to remove the firewall rule
			if rollbackErr := uc.firewallManager.RemoveBlockRule(ctx, ip); rollbackErr != nil {
				uc.logger.Error("Failed to rollback firewall rule after database error",
					zap.String("domain", domain),
					zap.String("ip", ip),
					zap.Error(rollbackErr))
			}

			uc.logger.Warn("Failed to create domain IP, continuing with others",
				zap.String("domain", domain),
				zap.String("ip", ip),
				zap.Error(err))
			continue
		}

		uc.logger.Info("Successfully added firewall rule and domain IP",
			zap.String("domain", domain),
			zap.String("ip", ip))
	}

	// Remove obsolete IPs
	for _, ip := range ipsToRemove {
		uc.logger.Info("Removing firewall rule and domain IP",
			zap.String("domain", domain),
			zap.String("ip", ip))

		// Remove from database first
		if err := uc.domainRepo.DeleteDomainIP(ctx, domain, ip); err != nil {
			uc.logger.Warn("Failed to delete domain IP, continuing with others",
				zap.String("domain", domain),
				zap.String("ip", ip),
				zap.Error(err))
			continue
		}

		// Then remove firewall rule
		if err := uc.firewallManager.RemoveBlockRule(ctx, ip); err != nil {
			uc.logger.Warn("Failed to remove firewall rule, but database entry was deleted",
				zap.String("domain", domain),
				zap.String("ip", ip),
				zap.Error(err))
			continue
		}

		uc.logger.Info("Successfully removed firewall rule and domain IP",
			zap.String("domain", domain),
			zap.String("ip", ip))
	}

	return nil
}

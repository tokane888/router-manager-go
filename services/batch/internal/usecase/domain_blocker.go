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
	MaxConcurrency   int // Configurable via environment variable, default 10
	DomainTimeout    time.Duration
	MaxDNSIterations int // Configurable via environment variable, default 5
}

type DomainBlockerUseCase struct {
	domainRepo      repository.DomainRepository
	dnsResolver     repository.DNSResolver
	firewallManager repository.FirewallManager
	rebootDetector  repository.RebootDetector
	logger          *zap.Logger
	config          ProcessingConfig
}

// NewDomainBlockerUseCase creates a new instance of DomainBlockerUseCase
func NewDomainBlockerUseCase(
	domainRepo repository.DomainRepository,
	dnsResolver repository.DNSResolver,
	firewallManager repository.FirewallManager,
	rebootDetector repository.RebootDetector,
	logger *zap.Logger,
	config ProcessingConfig,
) *DomainBlockerUseCase {
	return &DomainBlockerUseCase{
		domainRepo:      domainRepo,
		dnsResolver:     dnsResolver,
		firewallManager: firewallManager,
		logger:          logger,
		config:          config,
		rebootDetector:  rebootDetector,
	}
}

// ProcessAllDomains processes all domains from the database
func (uc *DomainBlockerUseCase) ProcessAllDomains(ctx context.Context) error {
	// Check if system has rebooted and cleanup is needed
	cleanupNeeded, err := uc.rebootDetector.CheckAndHandleReboot(ctx)
	if err != nil {
		uc.logger.Error("Failed to check reboot status", zap.Error(err))
		// Continue processing even if reboot detection fails
	} else if cleanupNeeded {
		uc.logger.Info("System reboot detected - cleaning up domain IPs table")
		if err := uc.domainRepo.DeleteAllDomainIPs(ctx); err != nil {
			uc.logger.Error("Failed to cleanup domain IPs table after reboot", zap.Error(err))
			// Continue processing even if cleanup fails
		} else {
			uc.logger.Info("Successfully cleaned up domain IPs table after reboot")
		}
	}

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

	// Update nftables rules based on discovered IPs
	if err := uc.updateFirewallRules(ctx, domain, discoveredIPs); err != nil {
		return fmt.Errorf("failed to update nftables rules for domain %s: %w", domain, err)
	}

	uc.logger.Info("Successfully processed domain", zap.String("domain", domain))
	return nil
}

// discoverAllIPs discovers all IP addresses for a domain
// 短時間でipが切り替わるサイトへの対応のため、30秒間隔で一定回数名前解決実行
func (uc *DomainBlockerUseCase) discoverAllIPs(ctx context.Context, domain string) ([]string, error) {
	uc.logger.Info("Discovering IPs for domain", zap.String("domain", domain))

	// 1. 最初の名前解決を行い、解決結果をips変数に保持
	initialIPs, err := uc.dnsResolver.ResolveIPs(ctx, domain)
	if err != nil {
		uc.logger.Error("Failed to resolve IPs for domain",
			zap.String("domain", domain),
			zap.Error(err))
		return nil, fmt.Errorf("DNS resolution failed for domain %s: %w", domain, err)
	}

	if len(initialIPs) == 0 {
		uc.logger.Warn("No IPs discovered for domain", zap.String("domain", domain))
		return []string{}, nil
	}

	// IPの重複を避けるためにmapを使用
	ipsMap := make(map[string]bool)
	for _, ip := range initialIPs {
		ipsMap[ip] = true
	}

	uc.logger.Info("Initial IP discovery completed",
		zap.String("domain", domain),
		zap.Strings("ips", initialIPs))

	// 最大反復回数を設定（デフォルトは5回）
	maxIterations := uc.config.MaxDNSIterations
	if maxIterations <= 0 {
		maxIterations = 5 // デフォルト値
	}

	for iteration := 1; iteration < maxIterations; iteration++ {
		// 2. 30秒待機
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(30 * time.Second):
		}

		// 3. 名前解決を再度行う
		currentIPs, err := uc.dnsResolver.ResolveIPs(ctx, domain)
		if err != nil {
			uc.logger.Warn("DNS resolution failed during iteration",
				zap.String("domain", domain),
				zap.Int("iteration", iteration),
				zap.Error(err))
			// エラーが発生した場合は現在のIPリストを返す
			break
		}

		// 4. 新しいIPがあるかチェック
		hasNewIPs := false
		for _, ip := range currentIPs {
			if !ipsMap[ip] {
				// 5. 新しいIPがある場合は追加して次のループへ
				ipsMap[ip] = true
				hasNewIPs = true
				uc.logger.Info("New IP discovered",
					zap.String("domain", domain),
					zap.String("new_ip", ip),
					zap.Int("iteration", iteration))
			}
		}

		// 4. 新しいIPがない場合は終了
		if !hasNewIPs {
			uc.logger.Info("No new IPs found, IP discovery stabilized",
				zap.String("domain", domain),
				zap.Int("iteration", iteration))
			break
		}
	}

	// 最大反復回数に達した場合のログ
	if maxIterations > 1 {
		uc.logger.Info("IP discovery completed",
			zap.String("domain", domain),
			zap.Int("max_iterations", maxIterations))
	}

	// 最終的なIPリストを作成
	finalIPs := make([]string, 0, len(ipsMap))
	for ip := range ipsMap {
		finalIPs = append(finalIPs, ip)
	}

	uc.logger.Info("IP discovery completed with iterative resolution",
		zap.String("domain", domain),
		zap.Strings("final_ips", finalIPs),
		zap.Int("total_ips", len(finalIPs)))

	return finalIPs, nil
}

// updateFirewallRules updates nftables rules based on discovered IPs
func (uc *DomainBlockerUseCase) updateFirewallRules(ctx context.Context, domain string, newIPs []string) error {
	// Get existing IPs
	existingIPs, err := uc.getExistingIPs(ctx, domain)
	if err != nil {
		return fmt.Errorf("failed to get existing IPs for domain %s: %w", domain, err)
	}

	// Calculate changes
	ipsToAdd := uc.calculateIPChanges(existingIPs, newIPs)

	// Add new IPs
	for _, ip := range ipsToAdd {
		uc.addIP(ctx, domain, ip)
	}

	uc.logger.Info("Completed nftables rules update",
		zap.String("domain", domain),
		zap.Int("added", len(ipsToAdd)))

	return nil
}

// getExistingIPs retrieves existing IPs for a domain
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

// calculateIPChanges determines which IPs need to be added
func (uc *DomainBlockerUseCase) calculateIPChanges(existingIPs, newIPs []string) []string {
	// Convert to map for O(1) lookup
	existingIPsMap := make(map[string]bool)
	for _, ip := range existingIPs {
		existingIPsMap[ip] = true
	}

	// Find IPs to add (exist in newIPs but not in existing)
	var ipsToAdd []string
	for _, ip := range newIPs {
		if !existingIPsMap[ip] {
			ipsToAdd = append(ipsToAdd, ip)
		}
	}

	return ipsToAdd
}

// addIP adds a new IP address to both nftables and database
func (uc *DomainBlockerUseCase) addIP(ctx context.Context, domain, ip string) {
	uc.logger.Info("Adding nftables rule and domain IP",
		zap.String("domain", domain),
		zap.String("ip", ip))

	// Add nftables rule first
	if err := uc.firewallManager.AddBlockRule(ctx, ip); err != nil {
		uc.logger.Warn("Failed to add nftables rule, continuing with others",
			zap.String("domain", domain),
			zap.String("ip", ip),
			zap.Error(err))
		return
	}

	// Then add to database
	if err := uc.domainRepo.CreateDomainIP(ctx, domain, ip); err != nil {
		// If database insertion fails, try to remove the nftables rule
		if rollbackErr := uc.firewallManager.RemoveBlockRule(ctx, ip); rollbackErr != nil {
			uc.logger.Error("Failed to rollback nftables rule after database error",
				zap.String("domain", domain),
				zap.String("ip", ip),
				zap.Error(rollbackErr))
		}

		uc.logger.Warn("Failed to create domain IP, continuing with others",
			zap.String("domain", domain),
			zap.String("ip", ip),
			zap.Error(err))
		return
	}

	uc.logger.Info("Successfully added nftables rule and domain IP",
		zap.String("domain", domain),
		zap.String("ip", ip))
}

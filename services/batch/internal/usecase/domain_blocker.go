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
	MaxDNSIterations int           // Configurable via environment variable, default 5
	DNSRetryInterval time.Duration // Configurable via environment variable, default 60 seconds
	IPExpiryDuration time.Duration // Configurable via environment variable, default 24h
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
		rebootDetector:  rebootDetector,
		logger:          logger,
		config:          config,
	}
}

// ProcessAllDomains processes all domains from the database
func (uc *DomainBlockerUseCase) ProcessAllDomains(ctx context.Context) error {
	// On first run after reboot, re-apply all existing DB rules to nftables immediately.
	// nftables resets on reboot, so rules must be re-added from DB before DNS resolution begins.
	// Subsequent runs skip this to avoid duplicate nftables rules.
	isReboot, err := uc.rebootDetector.CheckAndHandleReboot(ctx)
	if err != nil {
		uc.logger.Error("Failed to check reboot status", zap.Error(err))
	} else if isReboot {
		uc.logger.Info("System reboot detected - applying existing IP blocks from database")
		if applyErr := uc.applyExistingIPBlocks(ctx); applyErr != nil {
			uc.logger.Error("Failed to apply existing IP blocks after reboot", zap.Error(applyErr))
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

	// Remove IPs that have not appeared in DNS results for longer than IPExpiryDuration
	if err := uc.cleanupExpiredIPs(ctx); err != nil {
		uc.logger.Error("Failed to cleanup expired IPs", zap.Error(err))
	}

	return nil
}

// applyExistingIPBlocks loads all domain IPs from the database and applies nftables rules for each.
// Called only on first run after reboot since nftables rules are lost on system restart.
func (uc *DomainBlockerUseCase) applyExistingIPBlocks(ctx context.Context) error {
	allIPs, err := uc.domainRepo.GetAllDomainIPs(ctx)
	if err != nil {
		return fmt.Errorf("failed to get all domain IPs: %w", err)
	}

	uc.logger.Info("Applying existing IP blocks from database", zap.Int("count", len(allIPs)))

	for _, domainIP := range allIPs {
		if err := uc.firewallManager.AddBlockRule(ctx, domainIP.IPAddress); err != nil {
			uc.logger.Warn("Failed to apply existing nftables rule",
				zap.String("domain", domainIP.DomainName),
				zap.String("ip", domainIP.IPAddress),
				zap.Error(err))
			// Continue with remaining IPs
		}
	}

	uc.logger.Info("Finished applying existing IP blocks")
	return nil
}

// cleanupExpiredIPs removes IPs from DB and nftables that have not been seen in DNS results
// for longer than IPExpiryDuration.
func (uc *DomainBlockerUseCase) cleanupExpiredIPs(ctx context.Context) error {
	cutoff := time.Now().Add(-uc.config.IPExpiryDuration)
	expiredIPs, err := uc.domainRepo.DeleteExpiredDomainIPs(ctx, cutoff)
	if err != nil {
		return fmt.Errorf("failed to delete expired domain IPs: %w", err)
	}

	if len(expiredIPs) == 0 {
		return nil
	}

	uc.logger.Info("Removing nftables rules for expired IPs", zap.Int("count", len(expiredIPs)))

	for _, domainIP := range expiredIPs {
		if err := uc.firewallManager.RemoveBlockRule(ctx, domainIP.IPAddress); err != nil {
			uc.logger.Warn("Failed to remove nftables rule for expired IP",
				zap.String("domain", domainIP.DomainName),
				zap.String("ip", domainIP.IPAddress),
				zap.Error(err))
			// Continue with remaining IPs
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
// 短時間でipが切り替わるサイトへの対応のため、設定可能な間隔（デフォルト60秒）で一定回数名前解決実行
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
		// 2. 設定可能な間隔で待機
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(uc.config.DNSRetryInterval):
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

// updateFirewallRules updates nftables rules and database based on discovered IPs.
// Existing IPs found in DNS results have their updated_at refreshed.
// New IPs are added to both nftables and the database.
func (uc *DomainBlockerUseCase) updateFirewallRules(ctx context.Context, domain string, resolvedIPs []string) error {
	existingIPs, err := uc.getExistingIPs(ctx, domain)
	if err != nil {
		return fmt.Errorf("failed to get existing IPs for domain %s: %w", domain, err)
	}

	existingIPsMap := make(map[string]bool, len(existingIPs))
	for _, ip := range existingIPs {
		existingIPsMap[ip] = true
	}

	var added, refreshed int
	for _, ip := range resolvedIPs {
		if existingIPsMap[ip] {
			if err := uc.domainRepo.UpdateDomainIPUpdatedAt(ctx, domain, ip); err != nil {
				uc.logger.Warn("Failed to refresh domain IP timestamp",
					zap.String("domain", domain),
					zap.String("ip", ip),
					zap.Error(err))
			} else {
				refreshed++
			}
		} else {
			uc.addIP(ctx, domain, ip)
			added++
		}
	}

	uc.logger.Info("Completed nftables rules update",
		zap.String("domain", domain),
		zap.Int("added", added),
		zap.Int("refreshed", refreshed))

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

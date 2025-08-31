package firewall

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"go.uber.org/zap"
)

// NFTablesManager implements the FirewallManager interface for nftables

// NFTablesManagerConfig contains firewall management configuration
type NFTablesManagerConfig struct {
	DryRun         bool
	CommandTimeout time.Duration
	Family         string // nftables address family (ip, ip6, inet, etc.)
	Table          string // nftables table name
	Chain          string // nftables chain name
}

type NFTablesManager struct {
	logger    *zap.Logger
	dryRun    bool   // For development environments
	family    string // nftables address family
	tableName string // nftables table name
	chainName string // nftables chain name
}

// NewNFTablesManager creates a new nftables manager implementation
func NewNFTablesManager(cfg NFTablesManagerConfig, logger *zap.Logger) *NFTablesManager {
	return &NFTablesManager{
		logger:    logger,
		dryRun:    cfg.DryRun,
		family:    cfg.Family,
		tableName: cfg.Table,
		chainName: cfg.Chain,
	}
}

// AddBlockRule adds a blocking rule for the specified IP
func (n *NFTablesManager) AddBlockRule(ctx context.Context, ip string) error {
	if n.dryRun {
		n.logger.Info("DRY RUN: Would add firewall rule", zap.String("ip", ip))
		return nil
	}

	n.logger.Info("Adding firewall rule", zap.String("ip", ip))

	// Check if table and chain exist (do not create)
	if err := n.ensureTableAndChainExist(ctx); err != nil {
		return fmt.Errorf("failed to check table and chain: %w", err)
	}

	// Add the blocking rule
	args := []string{"add", "rule", n.family, n.tableName, n.chainName, "ip", "daddr", ip, "drop"}
	if err := n.executeCommand(ctx, args); err != nil {
		return fmt.Errorf("failed to add blocking rule for IP %s: %w", ip, err)
	}

	n.logger.Info("Successfully added firewall rule", zap.String("ip", ip))
	return nil
}

// RemoveBlockRule removes a blocking rule for the specified IP
func (n *NFTablesManager) RemoveBlockRule(ctx context.Context, ip string) error {
	if n.dryRun {
		n.logger.Info("DRY RUN: Would remove firewall rule", zap.String("ip", ip))
		return nil
	}

	n.logger.Info("Removing firewall rule", zap.String("ip", ip))

	// Delete the specific rule (nftables will find and remove the matching rule)
	args := []string{"delete", "rule", n.family, n.tableName, n.chainName, "ip", "daddr", ip, "drop"}
	if err := n.executeCommand(ctx, args); err != nil {
		// If the rule doesn't exist, nftables will return an error, but we can log and continue
		n.logger.Warn("Failed to remove firewall rule (may not exist)",
			zap.String("ip", ip),
			zap.Error(err))
		return nil // Don't return error to continue processing other IPs
	}

	n.logger.Info("Successfully removed firewall rule", zap.String("ip", ip))
	return nil
}

// executeCommand executes nftables commands
func (n *NFTablesManager) executeCommand(ctx context.Context, args []string) error {
	n.logger.Debug("Executing nft command", zap.Strings("args", args))

	cmd := exec.CommandContext(ctx, "nft", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		n.logger.Error("nft command failed",
			zap.Strings("args", args),
			zap.String("output", string(output)),
			zap.Error(err))
		return fmt.Errorf("nft command failed: %s: %w", string(output), err)
	}

	n.logger.Debug("nft command executed successfully",
		zap.Strings("args", args),
		zap.String("output", string(output)))

	return nil
}

// ensureTableAndChainExist ensures the nftables table and chain exist
func (n *NFTablesManager) ensureTableAndChainExist(ctx context.Context) error {
	// Check if table exists
	checkTableArgs := []string{"list", "table", n.family, n.tableName}
	if err := n.executeCommand(ctx, checkTableArgs); err != nil {
		n.logger.Error("Table does not exist",
			zap.String("family", n.family),
			zap.String("table", n.tableName),
			zap.Error(err))
		return fmt.Errorf("table %s in family %s does not exist: %w", n.tableName, n.family, err)
	}

	// Check if chain exists
	checkChainArgs := []string{"list", "chain", n.family, n.tableName, n.chainName}
	if err := n.executeCommand(ctx, checkChainArgs); err != nil {
		n.logger.Error("Chain does not exist",
			zap.String("family", n.family),
			zap.String("table", n.tableName),
			zap.String("chain", n.chainName),
			zap.Error(err))
		return fmt.Errorf("chain %s in table %s (family %s) does not exist: %w", n.chainName, n.tableName, n.family, err)
	}

	return nil
}

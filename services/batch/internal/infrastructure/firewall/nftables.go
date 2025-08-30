package firewall

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"go.uber.org/zap"
)

// NFTablesManager implements the FirewallManager interface for nftables

// FirewallConfig contains firewall management configuration
type FirewallConfig struct {
	DryRun         bool
	CommandTimeout time.Duration
	Table          string
	Chain          string
}

type NFTablesManager struct {
	logger    *zap.Logger
	dryRun    bool   // For development environments
	tableName string // nftables table name
	chainName string // nftables chain name
}

// NewNFTablesManager creates a new nftables manager implementation
func NewNFTablesManager(cfg FirewallConfig, logger *zap.Logger) *NFTablesManager {
	return &NFTablesManager{
		logger:    logger,
		dryRun:    cfg.DryRun,
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

	// Ensure the table and chain exist
	if err := n.ensureTableAndChainExist(ctx); err != nil {
		return fmt.Errorf("failed to ensure table and chain exist: %w", err)
	}

	// Add the blocking rule
	args := []string{"add", "rule", "ip", n.tableName, n.chainName, "ip", "daddr", ip, "drop"}
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
	args := []string{"delete", "rule", "ip", n.tableName, n.chainName, "ip", "daddr", ip, "drop"}
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
	// Create table if it doesn't exist (this will not error if table already exists)
	args := []string{"add", "table", "ip", n.tableName}
	if err := n.executeCommand(ctx, args); err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	// Create chain if it doesn't exist with OUTPUT hook and policy accept
	args = []string{"add", "chain", "ip", n.tableName, n.chainName, "{", "type", "filter", "hook", "output", "priority", "0", ";", "policy", "accept", ";", "}"}
	if err := n.executeCommand(ctx, args); err != nil {
		// If chain already exists, this might fail, but we can continue
		n.logger.Debug("Chain creation command result (may already exist)",
			zap.Strings("args", args),
			zap.Error(err))
	}

	return nil
}

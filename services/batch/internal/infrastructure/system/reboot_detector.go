package system

import (
	"context"
	"fmt"
	"os"

	"go.uber.org/zap"
)

const (
	// Flag directory and file paths
	flagDir  = "/run/router-manager-batch"
	flagFile = "/run/router-manager-batch/executed"
)

// RebootDetector handles reboot detection and cleanup
type RebootDetector struct {
	logger   *zap.Logger
	flagDir  string
	flagFile string
}

// NewRebootDetector creates a new reboot detector
func NewRebootDetector(logger *zap.Logger) *RebootDetector {
	return &RebootDetector{
		logger:   logger,
		flagDir:  flagDir,
		flagFile: flagFile,
	}
}

// CheckAndHandleReboot checks if this is first run after reboot and returns true if cleanup is needed
func (rd *RebootDetector) CheckAndHandleReboot(ctx context.Context) (bool, error) {
	// Check if flag file exists
	if _, err := os.Stat(rd.flagFile); err == nil {
		// Flag file exists - not first run after reboot
		rd.logger.Info("Flag file exists - not first run after reboot")
		return false, nil
	} else if !os.IsNotExist(err) {
		// Stat failed for reason other than file not existing
		rd.logger.Error("Failed to check flag file", zap.Error(err))
		return false, fmt.Errorf("failed to check flag file: %w", err)
	}

	// Flag file doesn't exist - first run after reboot
	rd.logger.Info("Flag file not found - first run after reboot, cleanup needed")

	// Create flag directory if it doesn't exist
	if err := os.MkdirAll(rd.flagDir, 0755); err != nil {
		rd.logger.Error("Failed to create flag directory", zap.Error(err))
		return true, fmt.Errorf("failed to create flag directory: %w", err)
	}

	// Create flag file
	if err := rd.createFlagFile(); err != nil {
		rd.logger.Error("Failed to create flag file", zap.Error(err))
		// Return true anyway - cleanup is still needed
		return true, nil
	}

	return true, nil
}

// createFlagFile creates the flag file to indicate batch has been executed
func (rd *RebootDetector) createFlagFile() error {
	file, err := os.Create(rd.flagFile)
	if err != nil {
		return fmt.Errorf("failed to create flag file: %w", err)
	}
	defer file.Close()

	// Write a simple timestamp for reference (optional)
	if _, err := file.WriteString(fmt.Sprintf("executed at: %s\n", os.Args[0])); err != nil {
		return fmt.Errorf("failed to write to flag file: %w", err)
	}

	rd.logger.Info("Created flag file", zap.String("file", rd.flagFile))
	return nil
}
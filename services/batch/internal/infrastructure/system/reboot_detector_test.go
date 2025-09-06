package system

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// newRebootDetectorWithPaths creates a new reboot detector with custom paths for testing
func newRebootDetectorWithPaths(logger *zap.Logger, flagDir, flagFile string) *RebootDetector {
	return &RebootDetector{
		logger:   logger,
		flagDir:  flagDir,
		flagFile: flagFile,
	}
}

func TestRebootDetector_CheckAndHandleReboot(t *testing.T) {
	logger := zap.NewNop()

	// Use temporary directory for testing
	tempDir := t.TempDir()
	testFlagDir := tempDir + "/test-flag"
	testFlagFile := testFlagDir + "/executed"

	detector := newRebootDetectorWithPaths(logger, testFlagDir, testFlagFile)

	tests := []struct {
		name               string
		setupPreConditions func(t *testing.T)
		cleanupPostActions func(t *testing.T)
		expectedCleanup    bool
		expectError        bool
	}{
		{
			name: "first run - no flag file exists",
			setupPreConditions: func(t *testing.T) {
				// Ensure flag file doesn't exist
				_ = os.RemoveAll(testFlagDir)
			},
			cleanupPostActions: func(t *testing.T) {
				// No cleanup needed - tempDir will be cleaned automatically
			},
			expectedCleanup: true,
			expectError:     false,
		},
		{
			name: "subsequent run - flag file exists",
			setupPreConditions: func(t *testing.T) {
				// Create flag file
				err := os.MkdirAll(testFlagDir, 0o755)
				require.NoError(t, err)
				file, err := os.Create(testFlagFile)
				require.NoError(t, err)
				file.Close()
			},
			cleanupPostActions: func(t *testing.T) {
				// No cleanup needed - tempDir will be cleaned automatically
			},
			expectedCleanup: false,
			expectError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			if tt.setupPreConditions != nil {
				tt.setupPreConditions(t)
			}

			// Cleanup setup
			if tt.cleanupPostActions != nil {
				defer tt.cleanupPostActions(t)
			}

			// Execute
			cleanupNeeded, err := detector.CheckAndHandleReboot(context.Background())

			// Verify
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedCleanup, cleanupNeeded)

				// If cleanup was needed, flag file should now exist
				if tt.expectedCleanup {
					_, err := os.Stat(testFlagFile)
					assert.NoError(t, err, "Flag file should be created")
				}
			}
		})
	}
}

func TestRebootDetector_CreateFlagFile(t *testing.T) {
	logger := zap.NewNop()

	// Use temporary directory for testing
	tempDir := t.TempDir()
	testFlagDir := tempDir + "/test-flag"
	testFlagFile := testFlagDir + "/executed"

	detector := newRebootDetectorWithPaths(logger, testFlagDir, testFlagFile)

	// Test creating flag file when directory doesn't exist
	err := detector.createFlagFile()
	assert.Error(t, err, "Should fail when directory doesn't exist")

	// Create directory and test again
	err = os.MkdirAll(testFlagDir, 0o755)
	require.NoError(t, err)

	err = detector.createFlagFile()
	assert.NoError(t, err, "Should succeed when directory exists")

	// Verify file was created
	_, err = os.Stat(testFlagFile)
	assert.NoError(t, err, "Flag file should exist")
}

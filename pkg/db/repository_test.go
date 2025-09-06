package db

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_CreateDomain(t *testing.T) {
	testDB := SetupTestDB(t)
	defer testDB.Cleanup(t)

	tests := []struct {
		name              string
		domainName        string
		expectError       bool
		expectedErrorType error
		shouldExistInDB   bool
	}{
		{
			name:            "valid domain",
			domainName:      "example.com",
			expectError:     false,
			shouldExistInDB: true,
		},
		{
			name:              "duplicate domain",
			domainName:        "example.com",
			expectError:       true,
			expectedErrorType: ErrDomainAlreadyExists,
			shouldExistInDB:   true, // Domain should still exist from first creation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := testDB.DB.CreateDomain(context.Background(), tt.domainName)

			// Check error expectation
			if tt.expectError {
				assert.Error(t, err)
				if tt.expectedErrorType != nil {
					assert.True(t, errors.Is(err, tt.expectedErrorType),
						"Expected error type %v, got %v", tt.expectedErrorType, err)
				}
			} else {
				assert.NoError(t, err)
			}

			// Check DB state - verify domain exists if expected
			if tt.shouldExistInDB {
				domains, err := testDB.DB.GetAllDomains(context.Background())
				require.NoError(t, err)

				found := false
				for _, domain := range domains {
					if domain.DomainName == tt.domainName {
						found = true
						break
					}
				}
				assert.True(t, found, "Domain %s should exist in DB", tt.domainName)
			}
		})
	}
}

func Test_GetAllDomains(t *testing.T) {
	testDB := SetupTestDB(t)
	defer testDB.Cleanup(t)

	// Test empty result
	domains, err := testDB.DB.GetAllDomains(context.Background())
	require.NoError(t, err)
	assert.Empty(t, domains)

	// Insert test data
	testDomains := []string{"example.com", "test.com", "google.com"}
	for _, domain := range testDomains {
		err = testDB.DB.CreateDomain(context.Background(), domain)
		require.NoError(t, err)
	}

	// Test with data
	domains, err = testDB.DB.GetAllDomains(context.Background())
	require.NoError(t, err)
	assert.Len(t, domains, 3)

	// Verify domains are sorted by created_at DESC
	for _, domain := range domains {
		assert.Contains(t, testDomains, domain.DomainName)
		assert.NotZero(t, domain.CreatedAt)
		assert.NotZero(t, domain.UpdatedAt)
	}
}

func Test_CreateDomainIP(t *testing.T) {
	testDB := SetupTestDB(t)
	defer testDB.Cleanup(t)

	// First create a domain
	domainName := "example.com"
	err := testDB.DB.CreateDomain(context.Background(), domainName)
	require.NoError(t, err)

	tests := []struct {
		name              string
		domainName        string
		ipAddress         string
		expectError       bool
		expectedErrorType error
	}{
		{
			name:        "valid IPv4",
			domainName:  domainName,
			ipAddress:   "192.168.1.1",
			expectError: false,
		},
		{
			name:              "duplicate IP for same domain",
			domainName:        domainName,
			ipAddress:         "192.168.1.1",
			expectError:       true, // Should not be allowed
			expectedErrorType: ErrDomainIPAlreadyExists,
		},
		{
			name:        "invalid domain",
			domainName:  "nonexistent.com",
			ipAddress:   "192.168.1.2",
			expectError: true, // Foreign key constraint violation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := testDB.DB.CreateDomainIP(context.Background(), tt.domainName, tt.ipAddress)
			if tt.expectError {
				assert.Error(t, err)
				if tt.expectedErrorType != nil {
					assert.True(t, errors.Is(err, tt.expectedErrorType),
						"Expected error type %v, got %v", tt.expectedErrorType, err)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_GetDomainIPs(t *testing.T) {
	testDB := SetupTestDB(t)
	defer testDB.Cleanup(t)

	domainName := "example.com"
	// Create domain
	err := testDB.DB.CreateDomain(context.Background(), domainName)
	require.NoError(t, err)

	// Test empty result
	domainIPs, err := testDB.DB.GetDomainIPs(context.Background(), domainName)
	require.NoError(t, err)
	assert.Empty(t, domainIPs)

	// Insert test IPs
	testIPs := []string{"192.168.1.1", "192.168.1.2", "2001:db8::1"}
	for _, ip := range testIPs {
		err = testDB.DB.CreateDomainIP(context.Background(), domainName, ip)
		require.NoError(t, err)
	}

	// Test with data
	domainIPs, err = testDB.DB.GetDomainIPs(context.Background(), domainName)
	require.NoError(t, err)
	assert.Len(t, domainIPs, 3)

	// Verify all fields are populated
	for _, domainIP := range domainIPs {
		assert.NotZero(t, domainIP.ID)
		assert.Equal(t, domainName, domainIP.DomainName)
		assert.Contains(t, testIPs, domainIP.IPAddress)
		assert.NotZero(t, domainIP.CreatedAt)
		assert.NotZero(t, domainIP.UpdatedAt)
	}

	// Test non-existent domain
	domainIPs, err = testDB.DB.GetDomainIPs(context.Background(), "nonexistent.com")
	require.NoError(t, err)
	assert.Empty(t, domainIPs)
}

func Test_DeleteDomainIP(t *testing.T) {
	testDB := SetupTestDB(t)
	defer testDB.Cleanup(t)

	domainName := "example.com"
	ipAddress := "192.168.1.1"

	// Create domain and IP
	err := testDB.DB.CreateDomain(context.Background(), domainName)
	require.NoError(t, err)
	err = testDB.DB.CreateDomainIP(context.Background(), domainName, ipAddress)
	require.NoError(t, err)

	// Verify IP exists
	domainIPs, err := testDB.DB.GetDomainIPs(context.Background(), domainName)
	require.NoError(t, err)
	assert.Len(t, domainIPs, 1)

	// Delete IP
	err = testDB.DB.DeleteDomainIP(context.Background(), domainName, ipAddress)
	assert.NoError(t, err)

	// Verify IP is deleted
	domainIPs, err = testDB.DB.GetDomainIPs(context.Background(), domainName)
	require.NoError(t, err)
	assert.Empty(t, domainIPs)

	// Test deleting non-existent IP
	err = testDB.DB.DeleteDomainIP(context.Background(), domainName, "192.168.1.2")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func Test_IntegrationWorkflow(t *testing.T) {
	testDB := SetupTestDB(t)
	defer testDB.Cleanup(t)

	// Scenario: Add a domain, add multiple IPs, then remove some IPs
	domainName := "integration-test.com"

	// Step 1: Create domain
	err := testDB.DB.CreateDomain(context.Background(), domainName)
	require.NoError(t, err)

	// Step 2: Add multiple IPs
	ips := []string{"192.168.1.10", "192.168.1.11"}
	for _, ip := range ips {
		err = testDB.DB.CreateDomainIP(context.Background(), domainName, ip)
		require.NoError(t, err)
	}

	// Step 3: Verify all IPs are stored
	domainIPs, err := testDB.DB.GetDomainIPs(context.Background(), domainName)
	require.NoError(t, err)
	assert.Len(t, domainIPs, 2)

	// Step 4: Remove one IP
	err = testDB.DB.DeleteDomainIP(context.Background(), domainName, "192.168.1.10")
	require.NoError(t, err)

	// Step 5: Verify IP count decreased
	domainIPs, err = testDB.DB.GetDomainIPs(context.Background(), domainName)
	require.NoError(t, err)
	assert.Len(t, domainIPs, 1)

	// Step 6: Verify correct IPs remain
	remainingIPs := make([]string, len(domainIPs))
	for i, domainIP := range domainIPs {
		remainingIPs[i] = domainIP.IPAddress
	}
	assert.Contains(t, remainingIPs, "192.168.1.11")
	assert.NotContains(t, remainingIPs, "192.168.1.10")
}

func Test_DeleteAllDomainIPs(t *testing.T) {
	testDB := SetupTestDB(t)
	defer testDB.Cleanup(t)

	tests := []struct {
		name          string
		setupDomains  []string
		setupIPs      map[string][]string // domain -> IPs
		expectError   bool
		expectedCount int64 // Expected rows affected
	}{
		{
			name:          "delete from empty table",
			setupDomains:  []string{},
			setupIPs:      map[string][]string{},
			expectError:   false,
			expectedCount: 0,
		},
		{
			name:         "delete single domain with single IP",
			setupDomains: []string{"example.com"},
			setupIPs: map[string][]string{
				"example.com": {"192.168.1.1"},
			},
			expectError:   false,
			expectedCount: 1,
		},
		{
			name:         "delete multiple domains with multiple IPs",
			setupDomains: []string{"example.com", "test.com", "hoge.com"},
			setupIPs: map[string][]string{
				"example.com": {"192.168.1.1", "192.168.1.2"},
				"test.com":    {"10.0.0.1", "10.0.0.2", "10.0.0.3"},
				"hoge.com":    {"8.8.8.8"},
			},
			expectError:   false,
			expectedCount: 6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear tables before each test case
			_, err := testDB.DB.pool.Exec(context.Background(), "TRUNCATE TABLE domain_ips, domains CASCADE")
			require.NoError(t, err)

			// Setup test data
			for _, domain := range tt.setupDomains {
				createErr := testDB.DB.CreateDomain(context.Background(), domain)
				require.NoError(t, createErr)
			}

			totalInsertedIPs := int64(0)
			for domain, ips := range tt.setupIPs {
				for _, ip := range ips {
					createIPErr := testDB.DB.CreateDomainIP(context.Background(), domain, ip)
					require.NoError(t, createIPErr)
					totalInsertedIPs++
				}
			}

			// Verify setup - check that IPs were inserted
			if totalInsertedIPs > 0 {
				allIPs, getErr := testDB.DB.GetAllDomainIPs(context.Background())
				require.NoError(t, getErr)
				assert.Len(t, allIPs, int(totalInsertedIPs))
			}

			// Execute DeleteAllDomainIPs
			err = testDB.DB.DeleteAllDomainIPs(context.Background())

			// Check error expectation
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// Verify all domain IPs are deleted
			if !tt.expectError {
				allIPs, err := testDB.DB.GetAllDomainIPs(context.Background())
				require.NoError(t, err)
				assert.Empty(t, allIPs, "All domain IPs should be deleted")

				// Verify domains still exist (should not be affected)
				for _, domain := range tt.setupDomains {
					domains, err := testDB.DB.GetAllDomains(context.Background())
					require.NoError(t, err)
					found := false
					for _, d := range domains {
						if d.DomainName == domain {
							found = true
							break
						}
					}
					assert.True(t, found, "Domain %s should still exist", domain)
				}
			}
		})
	}
}

func Test_DeleteAllDomainIPsIdempotent(t *testing.T) {
	testDB := SetupTestDB(t)
	defer testDB.Cleanup(t)

	domainName := "idempotent-test.com"

	// Create domain and IP
	err := testDB.DB.CreateDomain(context.Background(), domainName)
	require.NoError(t, err)
	err = testDB.DB.CreateDomainIP(context.Background(), domainName, "192.168.1.1")
	require.NoError(t, err)

	// First deletion
	err = testDB.DB.DeleteAllDomainIPs(context.Background())
	assert.NoError(t, err)

	// Verify table is empty
	allIPs, err := testDB.DB.GetAllDomainIPs(context.Background())
	require.NoError(t, err)
	assert.Empty(t, allIPs)

	// Second deletion on empty table - should not error
	err = testDB.DB.DeleteAllDomainIPs(context.Background())
	assert.NoError(t, err, "DeleteAllDomainIPs should be idempotent")

	// Table should still be empty
	allIPs, err = testDB.DB.GetAllDomainIPs(context.Background())
	require.NoError(t, err)
	assert.Empty(t, allIPs)
}

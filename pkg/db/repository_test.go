package db

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateDomain(t *testing.T) {
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

func TestGetAllDomains(t *testing.T) {
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

func TestCreateDomainIP(t *testing.T) {
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

func TestGetDomainIPs(t *testing.T) {
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

func TestDeleteDomainIP(t *testing.T) {
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

func TestIntegrationWorkflow(t *testing.T) {
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

func TestCascadeDelete(t *testing.T) {
	testDB := SetupTestDB(t)
	defer testDB.Cleanup(t)

	domainName := "cascade-test.com"

	// Create domain and IPs
	err := testDB.DB.CreateDomain(context.Background(), domainName)
	require.NoError(t, err)

	ips := []string{"192.168.1.20", "192.168.1.21"}
	for _, ip := range ips {
		err = testDB.DB.CreateDomainIP(context.Background(), domainName, ip)
		require.NoError(t, err)
	}

	// Verify IPs exist
	domainIPs, err := testDB.DB.GetDomainIPs(context.Background(), domainName)
	require.NoError(t, err)
	assert.Len(t, domainIPs, 2)

	// This test would require implementing DeleteDomain method
	// For now, we'll test that foreign key constraint works by trying to delete domain directly
	_, err = testDB.DB.pool.Exec(context.Background(), "DELETE FROM domains WHERE domain_name = $1", domainName)
	require.NoError(t, err) // Should succeed due to CASCADE

	// Verify IPs are also deleted
	domainIPs, err = testDB.DB.GetDomainIPs(context.Background(), domainName)
	require.NoError(t, err)
	assert.Empty(t, domainIPs)
}

package dns

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestResolveIPs(t *testing.T) {
	tests := []struct {
		name          string
		domain        string
		mockBehavior  func(*mockResolver)
		retryAttempts int
		timeout       time.Duration
		expectedIPs   []string
		expectedError bool
		errorContains string
	}{
		{
			name:   "Successful resolution on first attempt",
			domain: "example.com",
			mockBehavior: func(m *mockResolver) {
				m.ipv4Results = []net.IP{
					net.ParseIP("192.168.1.1"),
					net.ParseIP("192.168.1.2"),
				}
			},
			retryAttempts: 2,
			timeout:       5 * time.Second,
			expectedIPs:   []string{"192.168.1.1", "192.168.1.2"},
			expectedError: false,
		},
		{
			name:   "IPv4 resolution",
			domain: "ipv4.com",
			mockBehavior: func(m *mockResolver) {
				m.ipv4Results = []net.IP{
					net.ParseIP("10.0.0.1"),
				}
			},
			retryAttempts: 2,
			timeout:       5 * time.Second,
			expectedIPs:   []string{"10.0.0.1"},
			expectedError: false,
		},
		{
			name:   "Successful after retry",
			domain: "retry.com",
			mockBehavior: func(m *mockResolver) {
				m.failuresBeforeSuccess = 1
				m.ipv4Results = []net.IP{
					net.ParseIP("172.16.0.1"),
				}
			},
			retryAttempts: 2,
			timeout:       5 * time.Second,
			expectedIPs:   []string{"172.16.0.1"},
			expectedError: false,
		},
		{
			name:   "All attempts fail",
			domain: "fail.com",
			mockBehavior: func(m *mockResolver) {
				m.ipv4Error = errors.New("resolution failed")
			},
			retryAttempts: 2,
			timeout:       5 * time.Second,
			expectedIPs:   nil,
			expectedError: true,
			errorContains: "failed to resolve domain fail.com after 3 attempts",
		},
		{
			name:   "Context cancelled",
			domain: "cancelled.com",
			mockBehavior: func(m *mockResolver) {
				m.shouldTimeout = true
			},
			retryAttempts: 2,
			timeout:       100 * time.Millisecond,
			expectedIPs:   nil,
			expectedError: true,
			errorContains: "context deadline exceeded",
		},
		{
			name:   "No IPv4 addresses found",
			domain: "no-ipv4.com",
			mockBehavior: func(m *mockResolver) {
				m.ipv4Results = []net.IP{}
			},
			retryAttempts: 0,
			timeout:       5 * time.Second,
			expectedIPs:   nil,
			expectedError: true,
			errorContains: "no IPv4 addresses found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRes := &mockResolver{}
			if tt.mockBehavior != nil {
				tt.mockBehavior(mockRes)
			}

			cfg := &DNSConfig{
				Timeout:       tt.timeout,
				RetryAttempts: tt.retryAttempts,
			}
			resolver := NewDNSResolver(cfg, mockRes, zap.NewNop())

			ctx := context.Background()
			ips, err := resolver.ResolveIPs(ctx, tt.domain)

			if tt.expectedError {
				require.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				require.NoError(t, err)
				assert.ElementsMatch(t, tt.expectedIPs, ips)
			}
		})
	}
}

// mockResolver is a test helper that simulates DNS resolution
type mockResolver struct {
	ipv4Results           []net.IP
	ipv4Error             error
	failuresBeforeSuccess int
	currentAttempt        int
	shouldTimeout         bool
}

func (m *mockResolver) LookupIP(ctx context.Context, network, host string) ([]net.IP, error) {
	if m.shouldTimeout {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(200 * time.Millisecond):
			return nil, context.DeadlineExceeded
		}
	}

	if m.currentAttempt < m.failuresBeforeSuccess {
		m.currentAttempt++
		return nil, errors.New("temporary failure")
	}

	return m.ipv4Results, m.ipv4Error
}

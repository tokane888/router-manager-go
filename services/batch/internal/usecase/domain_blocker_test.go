package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tokane888/router-manager-go/pkg/db"
	"go.uber.org/zap"
)

// --- mock implementations ---

type mockDomainRepo struct {
	domains            []db.Domain
	domainIPs          map[string][]db.DomainIP // key: domainName
	allIPs             []db.DomainIP
	updatedIPs         []string // "domain/ip" pairs that had updated_at refreshed
	deletedExpiredIPs  []db.DomainIP
	getDomainsErr      error
	getDomainIPsErr    error
	getAllDomainIPsErr error
	createDomainIPErr  error
	updateTimestampErr error
	deleteExpiredErr   error
}

func (m *mockDomainRepo) GetAllDomains(_ context.Context) ([]db.Domain, error) {
	return m.domains, m.getDomainsErr
}

func (m *mockDomainRepo) CreateDomain(_ context.Context, _ string) error { return nil }

func (m *mockDomainRepo) GetDomainIPs(_ context.Context, domainName string) ([]db.DomainIP, error) {
	if m.getDomainIPsErr != nil {
		return nil, m.getDomainIPsErr
	}
	return m.domainIPs[domainName], nil
}

func (m *mockDomainRepo) CreateDomainIP(_ context.Context, _, _ string) error {
	return m.createDomainIPErr
}

func (m *mockDomainRepo) DeleteDomainIP(_ context.Context, _, _ string) error { return nil }

func (m *mockDomainRepo) GetAllDomainIPs(_ context.Context) ([]db.DomainIP, error) {
	return m.allIPs, m.getAllDomainIPsErr
}

func (m *mockDomainRepo) UpdateDomainIPUpdatedAt(_ context.Context, domain, ip string) error {
	if m.updateTimestampErr != nil {
		return m.updateTimestampErr
	}
	m.updatedIPs = append(m.updatedIPs, domain+"/"+ip)
	return nil
}

func (m *mockDomainRepo) DeleteExpiredDomainIPs(_ context.Context, _ time.Time) ([]db.DomainIP, error) {
	return m.deletedExpiredIPs, m.deleteExpiredErr
}

type mockFirewallManager struct {
	addedRules   []string
	removedRules []string
	addErr       error
	removeErr    error
}

func (m *mockFirewallManager) AddBlockRule(_ context.Context, ip string) error {
	if m.addErr != nil {
		return m.addErr
	}
	m.addedRules = append(m.addedRules, ip)
	return nil
}

func (m *mockFirewallManager) RemoveBlockRule(_ context.Context, ip string) error {
	if m.removeErr != nil {
		return m.removeErr
	}
	m.removedRules = append(m.removedRules, ip)
	return nil
}

type mockDNSResolver struct {
	ips []string
	err error
}

func (m *mockDNSResolver) ResolveIPs(_ context.Context, _ string) ([]string, error) {
	return m.ips, m.err
}

type mockRebootDetector struct {
	isReboot bool
	err      error
}

func (m *mockRebootDetector) CheckAndHandleReboot(_ context.Context) (bool, error) {
	return m.isReboot, m.err
}

// --- helpers ---

func newTestUseCase(
	repo *mockDomainRepo,
	fw *mockFirewallManager,
	dns *mockDNSResolver,
	reboot *mockRebootDetector,
	cfg ProcessingConfig,
) *DomainBlockerUseCase {
	return NewDomainBlockerUseCase(repo, dns, fw, reboot, zap.NewNop(), cfg)
}

func defaultConfig() ProcessingConfig {
	return ProcessingConfig{
		MaxConcurrency:   1,
		DomainTimeout:    time.Second,
		MaxDNSIterations: 1,
		DNSRetryInterval: time.Millisecond,
		IPExpiryDuration: 24 * time.Hour,
	}
}

// --- tests ---

func Test_applyExistingIPBlocks(t *testing.T) {
	tests := []struct {
		name      string
		allIPs    []db.DomainIP
		getAllErr error
		addErr    error
		wantAdded []string
		wantErr   bool
	}{
		{
			name: "applies all IPs from DB to nftables",
			allIPs: []db.DomainIP{
				{DomainName: "example.com", IPAddress: "1.2.3.4"},
				{DomainName: "example.com", IPAddress: "5.6.7.8"},
			},
			wantAdded: []string{"1.2.3.4", "5.6.7.8"},
			wantErr:   false,
		},
		{
			name:      "empty DB returns no error",
			allIPs:    []db.DomainIP{},
			wantAdded: nil,
			wantErr:   false,
		},
		{
			name:      "GetAllDomainIPs error is propagated",
			getAllErr: errors.New("db error"),
			wantErr:   true,
		},
		{
			name: "AddBlockRule failure is logged and skipped, not returned",
			allIPs: []db.DomainIP{
				{DomainName: "example.com", IPAddress: "1.2.3.4"},
			},
			addErr:    errors.New("nft error"),
			wantAdded: nil,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockDomainRepo{allIPs: tt.allIPs, getAllDomainIPsErr: tt.getAllErr}
			fw := &mockFirewallManager{addErr: tt.addErr}
			uc := newTestUseCase(repo, fw, &mockDNSResolver{}, &mockRebootDetector{}, defaultConfig())

			err := uc.applyExistingIPBlocks(context.Background())

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantAdded, fw.addedRules)
			}
		})
	}
}

func Test_cleanupExpiredIPs(t *testing.T) {
	tests := []struct {
		name        string
		expiredIPs  []db.DomainIP
		deleteErr   error
		removeErr   error
		wantRemoved []string
		wantErr     bool
	}{
		{
			name: "removes nftables rules for expired IPs",
			expiredIPs: []db.DomainIP{
				{DomainName: "example.com", IPAddress: "1.2.3.4"},
				{DomainName: "example.com", IPAddress: "5.6.7.8"},
			},
			wantRemoved: []string{"1.2.3.4", "5.6.7.8"},
			wantErr:     false,
		},
		{
			name:        "no expired IPs is a no-op",
			expiredIPs:  []db.DomainIP{},
			wantRemoved: nil,
			wantErr:     false,
		},
		{
			name:      "DeleteExpiredDomainIPs error is propagated",
			deleteErr: errors.New("db error"),
			wantErr:   true,
		},
		{
			name: "RemoveBlockRule failure is logged and skipped",
			expiredIPs: []db.DomainIP{
				{DomainName: "example.com", IPAddress: "1.2.3.4"},
			},
			removeErr:   errors.New("nft error"),
			wantRemoved: nil,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockDomainRepo{deletedExpiredIPs: tt.expiredIPs, deleteExpiredErr: tt.deleteErr}
			fw := &mockFirewallManager{removeErr: tt.removeErr}
			uc := newTestUseCase(repo, fw, &mockDNSResolver{}, &mockRebootDetector{}, defaultConfig())

			err := uc.cleanupExpiredIPs(context.Background())

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantRemoved, fw.removedRules)
			}
		})
	}
}

func Test_updateFirewallRules(t *testing.T) {
	tests := []struct {
		name          string
		existingIPs   []db.DomainIP
		resolvedIPs   []string
		wantAdded     []string
		wantRefreshed []string
	}{
		{
			name: "new IPs are added, existing IPs have timestamp refreshed",
			existingIPs: []db.DomainIP{
				{DomainName: "example.com", IPAddress: "1.2.3.4"},
			},
			resolvedIPs:   []string{"1.2.3.4", "5.6.7.8"},
			wantAdded:     []string{"5.6.7.8"},
			wantRefreshed: []string{"example.com/1.2.3.4"},
		},
		{
			name:          "all IPs are new",
			existingIPs:   []db.DomainIP{},
			resolvedIPs:   []string{"1.2.3.4", "5.6.7.8"},
			wantAdded:     []string{"1.2.3.4", "5.6.7.8"},
			wantRefreshed: nil,
		},
		{
			name: "all IPs already exist",
			existingIPs: []db.DomainIP{
				{DomainName: "example.com", IPAddress: "1.2.3.4"},
				{DomainName: "example.com", IPAddress: "5.6.7.8"},
			},
			resolvedIPs:   []string{"1.2.3.4", "5.6.7.8"},
			wantAdded:     nil,
			wantRefreshed: []string{"example.com/1.2.3.4", "example.com/5.6.7.8"},
		},
		{
			name:          "no resolved IPs is a no-op",
			existingIPs:   []db.DomainIP{},
			resolvedIPs:   []string{},
			wantAdded:     nil,
			wantRefreshed: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockDomainRepo{
				domainIPs: map[string][]db.DomainIP{
					"example.com": tt.existingIPs,
				},
			}
			fw := &mockFirewallManager{}
			uc := newTestUseCase(repo, fw, &mockDNSResolver{}, &mockRebootDetector{}, defaultConfig())

			err := uc.updateFirewallRules(context.Background(), "example.com", tt.resolvedIPs)

			assert.NoError(t, err)
			assert.Equal(t, tt.wantAdded, fw.addedRules)
			assert.Equal(t, tt.wantRefreshed, repo.updatedIPs)
		})
	}
}

func TestProcessAllDomains_rebootAppliesExistingBlocks(t *testing.T) {
	existingIP := db.DomainIP{DomainName: "example.com", IPAddress: "1.2.3.4"}
	repo := &mockDomainRepo{
		allIPs:  []db.DomainIP{existingIP},
		domains: []db.Domain{{DomainName: "example.com"}},
		domainIPs: map[string][]db.DomainIP{
			"example.com": {existingIP},
		},
	}
	fw := &mockFirewallManager{}
	dns := &mockDNSResolver{ips: []string{"1.2.3.4"}}
	reboot := &mockRebootDetector{isReboot: true}

	uc := newTestUseCase(repo, fw, dns, reboot, defaultConfig())
	err := uc.ProcessAllDomains(context.Background())

	assert.NoError(t, err)
	// applyExistingIPBlocks should have added the rule
	assert.Contains(t, fw.addedRules, "1.2.3.4")
}

func TestProcessAllDomains_noRebootSkipsApplyExistingBlocks(t *testing.T) {
	repo := &mockDomainRepo{
		allIPs:  []db.DomainIP{{DomainName: "example.com", IPAddress: "1.2.3.4"}},
		domains: []db.Domain{{DomainName: "example.com"}},
		domainIPs: map[string][]db.DomainIP{
			"example.com": {{DomainName: "example.com", IPAddress: "1.2.3.4"}},
		},
	}
	fw := &mockFirewallManager{}
	dns := &mockDNSResolver{ips: []string{"1.2.3.4"}}
	reboot := &mockRebootDetector{isReboot: false}

	uc := newTestUseCase(repo, fw, dns, reboot, defaultConfig())
	err := uc.ProcessAllDomains(context.Background())

	assert.NoError(t, err)
	// applyExistingIPBlocks should NOT have been called — no rules added for existing IPs
	assert.Empty(t, fw.addedRules)
}

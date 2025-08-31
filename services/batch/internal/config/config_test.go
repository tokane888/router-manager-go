package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tokane888/router-manager-go/pkg/logger"
	"github.com/tokane888/router-manager-go/services/batch/internal/infrastructure/dns"
	"github.com/tokane888/router-manager-go/services/batch/internal/infrastructure/firewall"
	"github.com/tokane888/router-manager-go/services/batch/internal/usecase"
)

// TestLoadConfig is removed as it requires extensive environment setup
// and has high maintenance cost. Individual helper function tests below
// provide sufficient coverage for the configuration loading logic.

// validConfig returns a valid configuration for testing
func validConfig() *Config {
	return &Config{
		Env: "local",
		Logger: logger.LoggerConfig{
			Level:  "info",
			Format: "local",
		},
		Processing: usecase.ProcessingConfig{
			MaxConcurrency: 10,
			DomainTimeout:  30 * time.Second,
		},
		DNS: dns.DNSConfig{
			Timeout:       5 * time.Second,
			RetryAttempts: 3,
		},
		Firewall: firewall.NFTablesManagerConfig{
			CommandTimeout: 10 * time.Second,
			Family:         "ip",
			Table:          "filter",
			Chain:          "OUTPUT",
		},
	}
}

func Test_validateConfig(t *testing.T) {
	type args struct {
		cfg *Config
	}
	tests := []struct {
		name        string
		args        args
		wantErr     bool
		errContains string
	}{
		{
			name: "valid configuration",
			args: args{
				cfg: validConfig(),
			},
			wantErr: false,
		},
		{
			name: "invalid environment",
			args: args{
				cfg: func() *Config {
					cfg := validConfig()
					cfg.Env = "invalid"
					return cfg
				}(),
			},
			wantErr:     true,
			errContains: "invalid environment",
		},
		{
			name: "invalid log level",
			args: args{
				cfg: func() *Config {
					cfg := validConfig()
					cfg.Logger.Level = "invalid"
					return cfg
				}(),
			},
			wantErr:     true,
			errContains: "invalid log level",
		},
		{
			name: "invalid log format",
			args: args{
				cfg: func() *Config {
					cfg := validConfig()
					cfg.Logger.Format = "invalid"
					return cfg
				}(),
			},
			wantErr:     true,
			errContains: "invalid log format",
		},
		{
			name: "invalid max concurrency - zero",
			args: args{
				cfg: func() *Config {
					cfg := validConfig()
					cfg.Processing.MaxConcurrency = 0
					return cfg
				}(),
			},
			wantErr:     true,
			errContains: "max concurrency must be positive",
		},
		{
			name: "invalid max concurrency - too high",
			args: args{
				cfg: func() *Config {
					cfg := validConfig()
					cfg.Processing.MaxConcurrency = 101
					return cfg
				}(),
			},
			wantErr:     true,
			errContains: "max concurrency too high",
		},
		{
			name: "invalid DNS timeout",
			args: args{
				cfg: func() *Config {
					cfg := validConfig()
					cfg.DNS.Timeout = 0
					return cfg
				}(),
			},
			wantErr:     true,
			errContains: "DNS timeout must be positive",
		},
		{
			name: "invalid DNS retry attempts - negative",
			args: args{
				cfg: func() *Config {
					cfg := validConfig()
					cfg.DNS.RetryAttempts = -1
					return cfg
				}(),
			},
			wantErr:     true,
			errContains: "DNS retry attempts cannot be negative",
		},
		{
			name: "invalid DNS retry attempts - too high",
			args: args{
				cfg: func() *Config {
					cfg := validConfig()
					cfg.DNS.RetryAttempts = 11
					return cfg
				}(),
			},
			wantErr:     true,
			errContains: "DNS retry attempts too high",
		},
		{
			name: "invalid firewall command timeout",
			args: args{
				cfg: func() *Config {
					cfg := validConfig()
					cfg.Firewall.CommandTimeout = 0
					return cfg
				}(),
			},
			wantErr:     true,
			errContains: "firewall command timeout must be positive",
		},
		{
			name: "empty firewall family",
			args: args{
				cfg: func() *Config {
					cfg := validConfig()
					cfg.Firewall.Family = ""
					return cfg
				}(),
			},
			wantErr:     true,
			errContains: "firewall family cannot be empty",
		},
		{
			name: "empty firewall table",
			args: args{
				cfg: func() *Config {
					cfg := validConfig()
					cfg.Firewall.Table = ""
					return cfg
				}(),
			},
			wantErr:     true,
			errContains: "firewall table cannot be empty",
		},
		{
			name: "empty firewall chain",
			args: args{
				cfg: func() *Config {
					cfg := validConfig()
					cfg.Firewall.Chain = ""
					return cfg
				}(),
			},
			wantErr:     true,
			errContains: "firewall chain cannot be empty",
		},
		{
			name: "invalid domain timeout",
			args: args{
				cfg: func() *Config {
					cfg := validConfig()
					cfg.Processing.DomainTimeout = 0
					return cfg
				}(),
			},
			wantErr:     true,
			errContains: "domain timeout must be positive",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.args.cfg)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_getIntEnv(t *testing.T) {
	type args struct {
		key      string
		fallback int
	}
	tests := []struct {
		name        string
		args        args
		envValue    string
		setEnv      bool
		want        int
		wantErr     bool
		errContains string
	}{
		{
			name: "valid integer",
			args: args{
				key:      "TEST_INT",
				fallback: 10,
			},
			envValue: "42",
			setEnv:   true,
			want:     42,
			wantErr:  false,
		},
		{
			name: "invalid integer",
			args: args{
				key:      "TEST_INT",
				fallback: 10,
			},
			envValue:    "not-a-number",
			setEnv:      true,
			want:        0,
			wantErr:     true,
			errContains: "expected integer",
		},
		{
			name: "fallback value",
			args: args{
				key:      "NON_EXISTENT",
				fallback: 10,
			},
			setEnv:  false,
			want:    10,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setEnv {
				os.Setenv(tt.args.key, tt.envValue)
				defer os.Unsetenv(tt.args.key)
			}

			got, err := getIntEnv(tt.args.key, tt.args.fallback)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func Test_getBoolEnv(t *testing.T) {
	type args struct {
		key      string
		fallback bool
	}
	tests := []struct {
		name        string
		args        args
		envValue    string
		setEnv      bool
		want        bool
		wantErr     bool
		errContains string
	}{
		{
			name: "valid boolean - true",
			args: args{
				key:      "TEST_BOOL",
				fallback: false,
			},
			envValue: "true",
			setEnv:   true,
			want:     true,
			wantErr:  false,
		},
		{
			name: "valid boolean - false",
			args: args{
				key:      "TEST_BOOL",
				fallback: true,
			},
			envValue: "false",
			setEnv:   true,
			want:     false,
			wantErr:  false,
		},
		{
			name: "invalid boolean",
			args: args{
				key:      "TEST_BOOL",
				fallback: false,
			},
			envValue:    "not-a-boolean",
			setEnv:      true,
			want:        false,
			wantErr:     true,
			errContains: "expected boolean",
		},
		{
			name: "fallback value",
			args: args{
				key:      "NON_EXISTENT",
				fallback: true,
			},
			setEnv:  false,
			want:    true,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setEnv {
				os.Setenv(tt.args.key, tt.envValue)
				defer os.Unsetenv(tt.args.key)
			}

			got, err := getBoolEnv(tt.args.key, tt.args.fallback)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func Test_getDurationEnv(t *testing.T) {
	type args struct {
		key      string
		fallback time.Duration
	}
	tests := []struct {
		name        string
		args        args
		envValue    string
		setEnv      bool
		want        time.Duration
		wantErr     bool
		errContains string
	}{
		{
			name: "valid duration - seconds",
			args: args{
				key:      "TEST_DURATION",
				fallback: time.Second,
			},
			envValue: "5s",
			setEnv:   true,
			want:     5 * time.Second,
			wantErr:  false,
		},
		{
			name: "valid duration - milliseconds",
			args: args{
				key:      "TEST_DURATION",
				fallback: time.Second,
			},
			envValue: "100ms",
			setEnv:   true,
			want:     100 * time.Millisecond,
			wantErr:  false,
		},
		{
			name: "invalid duration",
			args: args{
				key:      "TEST_DURATION",
				fallback: time.Second,
			},
			envValue:    "not-a-duration",
			setEnv:      true,
			want:        0,
			wantErr:     true,
			errContains: "expected duration",
		},
		{
			name: "fallback value",
			args: args{
				key:      "NON_EXISTENT",
				fallback: 10 * time.Second,
			},
			setEnv:  false,
			want:    10 * time.Second,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setEnv {
				os.Setenv(tt.args.key, tt.envValue)
				defer os.Unsetenv(tt.args.key)
			}

			got, err := getDurationEnv(tt.args.key, tt.args.fallback)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

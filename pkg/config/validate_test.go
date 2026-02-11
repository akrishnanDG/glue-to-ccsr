package config

import (
	"testing"
)

// validConfig returns a minimal config that passes validation.
// DryRun is true to skip Confluent Cloud credential checks.
// RegistryAll is true to satisfy the registry specification requirement.
func validConfig() *Config {
	cfg := NewDefaultConfig()
	cfg.Output.DryRun = true
	cfg.AWS.RegistryAll = true
	return cfg
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(cfg *Config)
		wantErr bool
	}{
		{
			name:    "valid config passes",
			modify:  func(cfg *Config) {},
			wantErr: false,
		},
		{
			name: "empty region fails",
			modify: func(cfg *Config) {
				cfg.AWS.Region = ""
			},
			wantErr: true,
		},
		{
			name: "no registry spec fails",
			modify: func(cfg *Config) {
				cfg.AWS.RegistryAll = false
				cfg.AWS.RegistryNames = []string{}
			},
			wantErr: true,
		},
		{
			name: "invalid subject strategy fails",
			modify: func(cfg *Config) {
				cfg.Naming.SubjectStrategy = "invalid"
			},
			wantErr: true,
		},
		{
			name: "custom strategy without template fails",
			modify: func(cfg *Config) {
				cfg.Naming.SubjectStrategy = "custom"
				cfg.Naming.SubjectTemplate = ""
			},
			wantErr: true,
		},
		{
			name: "invalid context mapping fails",
			modify: func(cfg *Config) {
				cfg.Naming.ContextMapping = "invalid"
			},
			wantErr: true,
		},
		{
			name: "invalid dot strategy fails",
			modify: func(cfg *Config) {
				cfg.Normalization.NormalizeDots = "invalid"
			},
			wantErr: true,
		},
		{
			name: "invalid case strategy fails",
			modify: func(cfg *Config) {
				cfg.Normalization.NormalizeCase = "invalid"
			},
			wantErr: true,
		},
		{
			name: "invalid default role fails",
			modify: func(cfg *Config) {
				cfg.KeyValue.DefaultRole = "invalid"
			},
			wantErr: true,
		},
		{
			name: "invalid version strategy fails",
			modify: func(cfg *Config) {
				cfg.Migration.VersionStrategy = "invalid"
			},
			wantErr: true,
		},
		{
			name: "workers less than 1 fails",
			modify: func(cfg *Config) {
				cfg.Concurrency.Workers = 0
			},
			wantErr: true,
		},
		{
			name: "invalid format fails",
			modify: func(cfg *Config) {
				cfg.Output.Format = "xml"
			},
			wantErr: true,
		},
		{
			name: "invalid log level fails",
			modify: func(cfg *Config) {
				cfg.Output.LogLevel = "trace"
			},
			wantErr: true,
		},
		{
			name: "custom context mapping without file fails",
			modify: func(cfg *Config) {
				cfg.Naming.ContextMapping = "custom"
				cfg.Naming.ContextMappingFile = ""
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			tt.modify(cfg)

			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

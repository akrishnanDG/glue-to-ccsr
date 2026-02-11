package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewDefaultConfig(t *testing.T) {
	cfg := NewDefaultConfig()

	tests := []struct {
		name     string
		got      interface{}
		expected interface{}
	}{
		{"Region", cfg.AWS.Region, "us-east-1"},
		{"SubjectStrategy", cfg.Naming.SubjectStrategy, "topic"},
		{"Workers", cfg.Concurrency.Workers, 10},
		{"NormalizeCase", cfg.Normalization.NormalizeCase, "kebab"},
		{"DefaultRole", cfg.KeyValue.DefaultRole, "value"},
		{"InputTokenCost", cfg.LLM.InputTokenCost, 0.000005},
		{"OutputTokenCost", cfg.LLM.OutputTokenCost, 0.000015},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			switch got := tt.got.(type) {
			case string:
				if got != tt.expected.(string) {
					t.Errorf("%s = %q, expected %q", tt.name, got, tt.expected)
				}
			case int:
				if got != tt.expected.(int) {
					t.Errorf("%s = %d, expected %d", tt.name, got, tt.expected)
				}
			case float64:
				if got != tt.expected.(float64) {
					t.Errorf("%s = %v, expected %v", tt.name, got, tt.expected)
				}
			}
		})
	}
}

func TestLoadFromFile_ValidConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	content := []byte("aws:\n  region: eu-west-1\n")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := LoadFromFile(path)
	if err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	// Overridden field
	if cfg.AWS.Region != "eu-west-1" {
		t.Errorf("Region = %q, expected %q", cfg.AWS.Region, "eu-west-1")
	}

	// Fields that retain defaults
	if cfg.Naming.SubjectStrategy != "topic" {
		t.Errorf("SubjectStrategy = %q, expected default %q", cfg.Naming.SubjectStrategy, "topic")
	}
	if cfg.Concurrency.Workers != 10 {
		t.Errorf("Workers = %d, expected default %d", cfg.Concurrency.Workers, 10)
	}
	if cfg.Normalization.NormalizeCase != "kebab" {
		t.Errorf("NormalizeCase = %q, expected default %q", cfg.Normalization.NormalizeCase, "kebab")
	}
}

func TestLoadFromFile_NonExistent(t *testing.T) {
	_, err := LoadFromFile("/nonexistent/path/config.yaml")
	if err == nil {
		t.Fatal("expected error loading non-existent config file")
	}
}

func TestLoadFromFile_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	if err := os.WriteFile(path, []byte("{{{"), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	_, err := LoadFromFile(path)
	if err == nil {
		t.Fatal("expected error loading invalid YAML config file")
	}
}

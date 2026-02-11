package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds all configuration for the migration tool
type Config struct {
	// AWS Glue source configuration
	AWS AWSConfig `yaml:"aws"`

	// Confluent Cloud target configuration
	ConfluentCloud ConfluentCloudConfig `yaml:"confluent_cloud"`

	// Naming strategy configuration
	Naming NamingConfig `yaml:"naming"`

	// Normalization configuration
	Normalization NormalizationConfig `yaml:"normalization"`

	// Key/Value detection configuration
	KeyValue KeyValueConfig `yaml:"key_value"`

	// Version and reference handling
	Migration MigrationConfig `yaml:"migration"`

	// Metadata handling
	Metadata MetadataConfig `yaml:"metadata"`

	// LLM configuration
	LLM LLMConfig `yaml:"llm"`

	// Concurrency configuration
	Concurrency ConcurrencyConfig `yaml:"concurrency"`

	// Checkpoint configuration
	Checkpoint CheckpointConfig `yaml:"checkpoint"`

	// Output configuration
	Output OutputConfig `yaml:"output"`
}

// AWSConfig holds AWS Glue Schema Registry configuration
type AWSConfig struct {
	Region           string   `yaml:"region"`
	RegistryNames    []string `yaml:"registry_names"`
	RegistryAll      bool     `yaml:"registry_all"`
	RegistryExclude  []string `yaml:"registry_exclude"`
	SchemaFilter     string   `yaml:"schema_filter"`
	Profile          string   `yaml:"profile"`
	AccessKeyID      string   `yaml:"access_key_id"`
	SecretAccessKey  string   `yaml:"secret_access_key"`
}

// ConfluentCloudConfig holds Confluent Cloud Schema Registry configuration
type ConfluentCloudConfig struct {
	URL       string `yaml:"url"`
	APIKey    string `yaml:"api_key"`
	APISecret string `yaml:"api_secret"`
}

// NamingConfig holds naming strategy configuration
type NamingConfig struct {
	SubjectStrategy    string `yaml:"subject_strategy"`    // topic, record, llm, custom
	SubjectTemplate    string `yaml:"subject_template"`    // for custom strategy
	ContextMapping     string `yaml:"context_mapping"`     // registry, flat, custom
	ContextMappingFile string `yaml:"context_mapping_file"`
	NameMappingFile    string `yaml:"name_mapping_file"`   // explicit schema-to-subject mappings
}

// NormalizationConfig holds name normalization configuration
type NormalizationConfig struct {
	NormalizeDots          string `yaml:"normalize_dots"`           // keep, replace, extract-last
	DotReplacement         string `yaml:"dot_replacement"`          // character to replace dots with
	NormalizeCase          string `yaml:"normalize_case"`           // keep, kebab, snake, lower
	InvalidCharReplacement string `yaml:"invalid_char_replacement"` // for invalid chars
	CollisionCheck         bool   `yaml:"collision_check"`
	CollisionResolution    string `yaml:"collision_resolution"`     // fail, suffix, registry-prefix, prefer-shorter, skip
}

// KeyValueConfig holds key/value detection configuration
type KeyValueConfig struct {
	KeyRegex              []string `yaml:"key_regex"`
	ValueRegex            []string `yaml:"value_regex"`
	DefaultRole           string   `yaml:"default_role"` // key or value
	RoleOverrideFile      string   `yaml:"role_override_file"`
	DisableBuiltinPatterns bool    `yaml:"disable_builtin_patterns"`
}

// MigrationConfig holds migration behavior configuration
type MigrationConfig struct {
	VersionStrategy    string `yaml:"version_strategy"`     // all, latest
	ReferenceStrategy  string `yaml:"reference_strategy"`   // rewrite, skip, fail
	CrossRegistryRefs  string `yaml:"cross_registry_refs"`  // resolve, fail, warn
}

// MetadataConfig holds metadata migration configuration
type MetadataConfig struct {
	Strategy           string `yaml:"strategy"` // migrate, skip
	MigrateTags        bool   `yaml:"migrate_tags"`
	MigrateDescription bool   `yaml:"migrate_description"`
}

// LLMConfig holds LLM configuration
type LLMConfig struct {
	Provider        string  `yaml:"provider"`          // openai, anthropic, bedrock, ollama, local
	Model           string  `yaml:"model"`
	APIKey          string  `yaml:"api_key"`
	BaseURL         string  `yaml:"base_url"`          // for local LLMs
	CacheFile       string  `yaml:"cache_file"`
	MaxCost         float64 `yaml:"max_cost"`
	RateLimit       int     `yaml:"rate_limit"`
	InputTokenCost  float64 `yaml:"input_token_cost"`  // cost per token for input/prompt
	OutputTokenCost float64 `yaml:"output_token_cost"` // cost per token for output/completion
}

// ConcurrencyConfig holds concurrency configuration
type ConcurrencyConfig struct {
	Workers       int           `yaml:"workers"`
	BatchSize     int           `yaml:"batch_size"`
	AWSRateLimit  int           `yaml:"aws_rate_limit"`
	CCRateLimit   int           `yaml:"cc_rate_limit"`
	LLMRateLimit  int           `yaml:"llm_rate_limit"`
	RetryAttempts int           `yaml:"retry_attempts"`
	RetryDelay    time.Duration `yaml:"retry_delay"`
}

// CheckpointConfig holds checkpoint/resume configuration
type CheckpointConfig struct {
	File   string `yaml:"file"`
	Resume bool   `yaml:"resume"`
}

// OutputConfig holds output configuration
type OutputConfig struct {
	DryRun       bool   `yaml:"dry_run"`
	ReportFile   string `yaml:"report_file"`
	Format       string `yaml:"format"` // table, json, csv
	Progress     bool   `yaml:"progress"`
	LogFile      string `yaml:"log_file"`
	LogLevel     string `yaml:"log_level"` // debug, info, warn, error
}

// NewDefaultConfig returns a Config with default values
func NewDefaultConfig() *Config {
	return &Config{
		AWS: AWSConfig{
			Region: "us-east-1",
		},
		Naming: NamingConfig{
			SubjectStrategy: "topic",
			ContextMapping:  "flat",
		},
		Normalization: NormalizationConfig{
			NormalizeDots:          "replace",
			DotReplacement:         "-",
			NormalizeCase:          "kebab",
			InvalidCharReplacement: "-",
			CollisionCheck:         true,
			CollisionResolution:    "suffix", // Default: add -1, -2, etc.
		},
		KeyValue: KeyValueConfig{
			DefaultRole:            "value",
			DisableBuiltinPatterns: false,
		},
		Migration: MigrationConfig{
			VersionStrategy:   "all",
			ReferenceStrategy: "rewrite",
			CrossRegistryRefs: "resolve",
		},
		Metadata: MetadataConfig{
			Strategy:           "migrate",
			MigrateTags:        true,
			MigrateDescription: true,
		},
		LLM: LLMConfig{
			Provider:        "openai",
			Model:           "gpt-4o",
			RateLimit:       5,
			InputTokenCost:  0.000005,  // $5 per million input tokens (gpt-4o)
			OutputTokenCost: 0.000015,  // $15 per million output tokens (gpt-4o)
		},
		Concurrency: ConcurrencyConfig{
			Workers:       10,
			BatchSize:     100,
			AWSRateLimit:  10,
			CCRateLimit:   10,
			LLMRateLimit:  5,
			RetryAttempts: 3,
			RetryDelay:    5 * time.Second,
		},
		Output: OutputConfig{
			Format:   "table",
			Progress: true,
			LogLevel: "info",
		},
	}
}

// LoadFromFile loads configuration from a YAML file
func LoadFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	config := NewDefaultConfig()
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return config, nil
}

// SaveToFile saves configuration to a YAML file
func (c *Config) SaveToFile(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

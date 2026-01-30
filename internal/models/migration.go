package models

import (
	"time"
)

// MigrationState represents the state of a migration for checkpointing
type MigrationState struct {
	// Metadata
	StartedAt  time.Time `json:"started_at"`
	ConfigHash string    `json:"config_hash"`
	
	// Progress
	TotalSchemas    int `json:"total_schemas"`
	CompletedCount  int `json:"completed_count"`
	FailedCount     int `json:"failed_count"`
	SkippedCount    int `json:"skipped_count"`
	
	// Migration order
	MigrationOrder []string `json:"migration_order"`
	
	// Completed schemas
	CompletedSchemas map[string]CompletedSchema `json:"completed_schemas"`
	
	// Failed schemas
	FailedSchemas map[string]FailedSchema `json:"failed_schemas"`
	
	// LLM cache state
	LLMCacheState *LLMCacheState `json:"llm_cache_state,omitempty"`
}

// CompletedSchema represents a successfully migrated schema
type CompletedSchema struct {
	SourceRegistry string    `json:"source_registry"`
	SourceSchema   string    `json:"source_schema"`
	TargetSubject  string    `json:"target_subject"`
	Versions       int       `json:"versions"`
	CompletedAt    time.Time `json:"completed_at"`
}

// FailedSchema represents a failed schema migration
type FailedSchema struct {
	SourceRegistry string    `json:"source_registry"`
	SourceSchema   string    `json:"source_schema"`
	Error          string    `json:"error"`
	Attempts       int       `json:"attempts"`
	LastAttempt    time.Time `json:"last_attempt"`
}

// LLMCacheState represents the state of LLM caching
type LLMCacheState struct {
	Entries   int     `json:"entries"`
	CostSoFar float64 `json:"cost_so_far"`
}

// DependencyLevel represents a level in the dependency graph
type DependencyLevel struct {
	Level   int              `json:"level"`
	Schemas []SchemaMapping  `json:"schemas"`
}

// MigrationPlan represents the complete plan for migration
type MigrationPlan struct {
	// Source info
	SourceRegistries []string `json:"source_registries"`
	TotalSchemas     int      `json:"total_schemas"`
	TotalVersions    int      `json:"total_versions"`
	TotalReferences  int      `json:"total_references"`
	
	// Mappings
	Mappings []SchemaMapping `json:"mappings"`
	
	// Dependency levels for ordered migration
	Levels []DependencyLevel `json:"levels"`
	
	// Collisions detected
	Collisions []Collision `json:"collisions,omitempty"`
	
	// Warnings
	Warnings []Warning `json:"warnings,omitempty"`
	
	// Errors
	Errors []Error `json:"errors,omitempty"`
	
	// Summary
	Summary MigrationSummary `json:"summary"`
}

// Collision represents a naming collision
type Collision struct {
	NormalizedName string   `json:"normalized_name"`
	SourceSchemas  []string `json:"source_schemas"`
}

// Warning represents a migration warning
type Warning struct {
	Schema  string `json:"schema"`
	Message string `json:"message"`
}

// Error represents a migration error
type Error struct {
	Schema  string `json:"schema"`
	Message string `json:"message"`
}

// MigrationSummary represents a summary of the migration plan
type MigrationSummary struct {
	Registries       int `json:"registries"`
	Schemas          int `json:"schemas"`
	Versions         int `json:"versions"`
	References       int `json:"references"`
	Ready            int `json:"ready"`
	Warnings         int `json:"warnings"`
	Errors           int `json:"errors"`
	Collisions       int `json:"collisions"`
	LLMCalls         int `json:"llm_calls"`
	EstimatedLLMCost float64 `json:"estimated_llm_cost"`
}

// NewMigrationState creates a new migration state
func NewMigrationState(configHash string) *MigrationState {
	return &MigrationState{
		StartedAt:        time.Now(),
		ConfigHash:       configHash,
		CompletedSchemas: make(map[string]CompletedSchema),
		FailedSchemas:    make(map[string]FailedSchema),
	}
}

// IsComplete checks if the migration is complete
func (s *MigrationState) IsComplete() bool {
	return s.CompletedCount + s.FailedCount + s.SkippedCount >= s.TotalSchemas
}

// Progress returns the progress percentage
func (s *MigrationState) Progress() float64 {
	if s.TotalSchemas == 0 {
		return 0
	}
	return float64(s.CompletedCount+s.FailedCount+s.SkippedCount) / float64(s.TotalSchemas) * 100
}

package models

import (
	"time"
)

// MigrationReport represents the complete report of a migration
type MigrationReport struct {
	// Metadata
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	Duration    string    `json:"duration"`
	DryRun      bool      `json:"dry_run"`
	
	// Source
	Source SourceReport `json:"source"`
	
	// Target
	Target TargetReport `json:"target"`
	
	// Configuration used
	Config ConfigReport `json:"config"`
	
	// Results
	Results ResultsReport `json:"results"`
	
	// Schema details
	Schemas []SchemaReport `json:"schemas"`
	
	// Errors and warnings
	Errors   []ErrorReport   `json:"errors,omitempty"`
	Warnings []WarningReport `json:"warnings,omitempty"`
}

// SourceReport represents source information
type SourceReport struct {
	Type       string   `json:"type"` // "aws_glue"
	Region     string   `json:"region"`
	Registries []string `json:"registries"`
}

// TargetReport represents target information
type TargetReport struct {
	Type string `json:"type"` // "confluent_cloud"
	URL  string `json:"url"`
}

// ConfigReport represents configuration used
type ConfigReport struct {
	SubjectStrategy    string `json:"subject_strategy"`
	ContextMapping     string `json:"context_mapping"`
	VersionStrategy    string `json:"version_strategy"`
	ReferenceStrategy  string `json:"reference_strategy"`
	NormalizeDots      string `json:"normalize_dots"`
	NormalizeCase      string `json:"normalize_case"`
	LLMProvider        string `json:"llm_provider,omitempty"`
	LLMModel           string `json:"llm_model,omitempty"`
}

// ResultsReport represents migration results
type ResultsReport struct {
	RegistriesProcessed int     `json:"registries_processed"`
	SchemasProcessed    int     `json:"schemas_processed"`
	VersionsProcessed   int     `json:"versions_processed"`
	Successful          int     `json:"successful"`
	Failed              int     `json:"failed"`
	Skipped             int     `json:"skipped"`
	LLMCalls            int     `json:"llm_calls"`
	LLMCost             float64 `json:"llm_cost"`
}

// SchemaReport represents details about a single schema migration
type SchemaReport struct {
	// Source
	SourceRegistry string `json:"source_registry"`
	SourceSchema   string `json:"source_schema"`
	
	// Target
	TargetContext string `json:"target_context"`
	TargetSubject string `json:"target_subject"`
	
	// Details
	SchemaType       string     `json:"schema_type"`
	DetectedRole     SchemaRole `json:"detected_role"`
	RoleReason       string     `json:"role_reason"`
	NamingStrategy   string     `json:"naming_strategy"`
	Transformations  []string   `json:"transformations,omitempty"`
	
	// Versions
	VersionsMigrated int `json:"versions_migrated"`
	
	// References
	References []string `json:"references,omitempty"`
	
	// Status
	Status  string `json:"status"` // success, failed, skipped
	Error   string `json:"error,omitempty"`
	Warning string `json:"warning,omitempty"`
}

// ErrorReport represents an error in the report
type ErrorReport struct {
	Schema  string `json:"schema"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// WarningReport represents a warning in the report
type WarningReport struct {
	Schema  string `json:"schema"`
	Message string `json:"message"`
}

// RegistryReport represents a registry in the dry-run output
type RegistryReport struct {
	Name              string `json:"name"`
	Context           string `json:"context"`
	SchemaCount       int    `json:"schema_count"`
	VersionCount      int    `json:"version_count"`
	CrossRegistryRefs int    `json:"cross_registry_refs"`
}

// LLMSuggestionReport represents an LLM suggestion in the dry-run output
type LLMSuggestionReport struct {
	OriginalName  string `json:"original_name"`
	SuggestedName string `json:"suggested_name"`
	Reasoning     string `json:"reasoning"`
}

// KeyValueReport represents key/value detection in the dry-run output
type KeyValueReport struct {
	Schema          string `json:"schema"`
	Role            string `json:"role"`
	DetectionMethod string `json:"detection_method"`
	Subject         string `json:"subject"`
}

// NormalizationReport represents normalization in the dry-run output
type NormalizationReport struct {
	Schema         string   `json:"schema"`
	Transformations []string `json:"transformations"`
	NormalizedName string   `json:"normalized_name"`
}

// CollisionReport represents a collision in the dry-run output
type CollisionReport struct {
	Schema1       string `json:"schema1"`
	Schema2       string `json:"schema2"`
	NormalizedTo  string `json:"normalized_to"`
}

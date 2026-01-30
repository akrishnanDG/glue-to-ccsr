package models

import (
	"time"
)

// SchemaType represents the type of schema
type SchemaType string

const (
	SchemaTypeAvro     SchemaType = "AVRO"
	SchemaTypeJSON     SchemaType = "JSON"
	SchemaTypeProtobuf SchemaType = "PROTOBUF"
)

// SchemaRole represents whether a schema is a key or value schema
type SchemaRole string

const (
	SchemaRoleKey   SchemaRole = "key"
	SchemaRoleValue SchemaRole = "value"
)

// GlueRegistry represents an AWS Glue Schema Registry
type GlueRegistry struct {
	Name        string            `json:"name"`
	ARN         string            `json:"arn"`
	Description string            `json:"description"`
	Tags        map[string]string `json:"tags"`
	CreatedTime time.Time         `json:"created_time"`
	UpdatedTime time.Time         `json:"updated_time"`
}

// GlueSchema represents a schema from AWS Glue Schema Registry
type GlueSchema struct {
	Name              string            `json:"name"`
	RegistryName      string            `json:"registry_name"`
	ARN               string            `json:"arn"`
	Description       string            `json:"description"`
	DataFormat        SchemaType        `json:"data_format"`
	Compatibility     string            `json:"compatibility"`
	Tags              map[string]string `json:"tags"`
	LatestVersion     int64             `json:"latest_version"`
	CreatedTime       time.Time         `json:"created_time"`
	UpdatedTime       time.Time         `json:"updated_time"`
	Versions          []GlueSchemaVersion `json:"versions"`
}

// GlueSchemaVersion represents a version of a schema in AWS Glue
type GlueSchemaVersion struct {
	VersionNumber   int64     `json:"version_number"`
	SchemaVersionID string    `json:"schema_version_id"` // UUID in Glue
	Definition      string    `json:"definition"`
	Status          string    `json:"status"`
	CreatedTime     time.Time `json:"created_time"`
}

// ConfluentSubject represents a subject in Confluent Cloud Schema Registry
type ConfluentSubject struct {
	Name             string       `json:"name"`
	Context          string       `json:"context"` // e.g., ".payments"
	Compatibility    string       `json:"compatibility"`
	SchemaType       SchemaType   `json:"schema_type"`
	Versions         []ConfluentSchemaVersion `json:"versions"`
	Metadata         *SubjectMetadata `json:"metadata,omitempty"`
}

// ConfluentSchemaVersion represents a version of a schema in Confluent Cloud
type ConfluentSchemaVersion struct {
	Version    int                `json:"version"`
	SchemaID   int                `json:"schema_id"`
	Schema     string             `json:"schema"`
	SchemaType SchemaType         `json:"schema_type"`
	References []SchemaReference  `json:"references,omitempty"`
}

// SchemaReference represents a reference to another schema
type SchemaReference struct {
	Name    string `json:"name"`    // Local name in the schema
	Subject string `json:"subject"` // Subject name in CC SR
	Version int    `json:"version"` // Version number
}

// SubjectMetadata represents metadata for a Confluent Cloud subject
type SubjectMetadata struct {
	Properties map[string]string `json:"properties,omitempty"`
	Tags       []string          `json:"tags,omitempty"`
}

// ParsedSchema represents a parsed schema with extracted metadata
type ParsedSchema struct {
	// Original schema
	GlueSchema *GlueSchema `json:"glue_schema"`
	
	// Extracted metadata
	RecordName    string   `json:"record_name"`
	Namespace     string   `json:"namespace"`
	Documentation string   `json:"documentation"`
	Fields        []Field  `json:"fields"`
	References    []string `json:"references"`
	
	// Computed properties
	DetectedRole   SchemaRole `json:"detected_role"`
	RoleReason     string     `json:"role_reason"`
	NormalizedName string     `json:"normalized_name"`
	TargetSubject  string     `json:"target_subject"`
	TargetContext  string     `json:"target_context"`
}

// Field represents a field extracted from a schema
type Field struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	IsRequired bool   `json:"is_required"`
	Doc        string `json:"doc,omitempty"`
}

// SchemaMapping represents the mapping from a Glue schema to a Confluent subject
type SchemaMapping struct {
	// Source
	SourceRegistry   string `json:"source_registry"`
	SourceSchemaName string `json:"source_schema_name"`
	SourceVersions   int    `json:"source_versions"`
	
	// Target
	TargetContext    string     `json:"target_context"`
	TargetSubject    string     `json:"target_subject"`
	DetectedRole     SchemaRole `json:"detected_role"`
	
	// Naming
	NamingStrategy   string `json:"naming_strategy"`
	NamingReason     string `json:"naming_reason,omitempty"`
	
	// Normalization
	Transformations  []string `json:"transformations,omitempty"`
	
	// References
	References       []string `json:"references,omitempty"`
	DependencyLevel  int      `json:"dependency_level"`
	
	// Status
	Status           MappingStatus `json:"status"`
	Warning          string        `json:"warning,omitempty"`
	Error            string        `json:"error,omitempty"`
}

// MappingStatus represents the status of a schema mapping
type MappingStatus string

const (
	MappingStatusReady    MappingStatus = "ready"
	MappingStatusWarning  MappingStatus = "warning"
	MappingStatusError    MappingStatus = "error"
	MappingStatusSkipped  MappingStatus = "skipped"
)

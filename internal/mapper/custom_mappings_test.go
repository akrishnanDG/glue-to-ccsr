package mapper

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/akrishnanDG/glue-to-ccsr/internal/keyvalue"
	"github.com/akrishnanDG/glue-to-ccsr/internal/models"
	"github.com/akrishnanDG/glue-to-ccsr/internal/normalizer"
	"github.com/akrishnanDG/glue-to-ccsr/pkg/config"
)

func writeTempFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "mappings.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	return path
}

func TestLoadCustomMappings_SimpleMappings(t *testing.T) {
	path := writeTempFile(t, `
mappings:
  "UserEvent": "user-event-value"
  "OrderPlaced": "order-placed-value"
`)

	loaded, err := loadCustomMappings(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(loaded.simple) != 2 {
		t.Errorf("expected 2 simple mappings, got %d", len(loaded.simple))
	}

	if loaded.simple["UserEvent"].Subject != "user-event-value" {
		t.Errorf("expected subject 'user-event-value', got %q", loaded.simple["UserEvent"].Subject)
	}
}

func TestLoadCustomMappings_QualifiedMappings(t *testing.T) {
	path := writeTempFile(t, `
qualified_mappings:
  "payments:PaymentEvent": "payment-event-value"
  "orders:OrderEvent": "order-event-value"
`)

	loaded, err := loadCustomMappings(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(loaded.qualified) != 2 {
		t.Errorf("expected 2 qualified mappings, got %d", len(loaded.qualified))
	}

	if loaded.qualified["payments:PaymentEvent"].Subject != "payment-event-value" {
		t.Errorf("expected subject 'payment-event-value', got %q", loaded.qualified["payments:PaymentEvent"].Subject)
	}
}

func TestLoadCustomMappings_ExtendedMappings(t *testing.T) {
	path := writeTempFile(t, `
extended_mappings:
  - source: "UserKey"
    subject: "user-key"
    role: "key"
    context: ".users"
  - source: "payments:RefundEvent"
    subject: "payment-refund-value"
    role: "value"
`)

	loaded, err := loadCustomMappings(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// "UserKey" has no colon, goes to simple
	if len(loaded.simple) != 1 {
		t.Errorf("expected 1 simple mapping from extended, got %d", len(loaded.simple))
	}
	if loaded.simple["UserKey"].Role != "key" {
		t.Errorf("expected role 'key', got %q", loaded.simple["UserKey"].Role)
	}
	if loaded.simple["UserKey"].Context != ".users" {
		t.Errorf("expected context '.users', got %q", loaded.simple["UserKey"].Context)
	}

	// "payments:RefundEvent" has colon, goes to qualified
	if len(loaded.qualified) != 1 {
		t.Errorf("expected 1 qualified mapping from extended, got %d", len(loaded.qualified))
	}
	if loaded.qualified["payments:RefundEvent"].Subject != "payment-refund-value" {
		t.Errorf("expected subject 'payment-refund-value', got %q", loaded.qualified["payments:RefundEvent"].Subject)
	}
}

func TestLoadCustomMappings_EmptyFile(t *testing.T) {
	path := writeTempFile(t, "")

	loaded, err := loadCustomMappings(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(loaded.simple) != 0 || len(loaded.qualified) != 0 {
		t.Errorf("expected empty mappings for empty file")
	}
}

func TestLoadCustomMappings_FileNotFound(t *testing.T) {
	_, err := loadCustomMappings("/nonexistent/path/mappings.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadCustomMappings_InvalidYAML(t *testing.T) {
	path := writeTempFile(t, "{{invalid yaml")

	_, err := loadCustomMappings(path)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestLookupCustomMapping_QualifiedOverSimple(t *testing.T) {
	path := writeTempFile(t, `
mappings:
  "PaymentEvent": "generic-payment-value"
qualified_mappings:
  "payments:PaymentEvent": "specific-payment-value"
`)

	cfg := config.NewDefaultConfig()
	cfg.Naming.NameMappingFile = path

	norm := normalizer.New(cfg)
	kvDet, _ := keyvalue.New(cfg)

	m, err := New(cfg, norm, kvDet, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Qualified match should win
	result, found := m.lookupCustomMapping("payments", "PaymentEvent")
	if !found {
		t.Fatal("expected match")
	}
	if result.Subject != "specific-payment-value" {
		t.Errorf("expected qualified match, got %q", result.Subject)
	}

	// Different registry should fall back to simple match
	result, found = m.lookupCustomMapping("other-registry", "PaymentEvent")
	if !found {
		t.Fatal("expected match")
	}
	if result.Subject != "generic-payment-value" {
		t.Errorf("expected simple match, got %q", result.Subject)
	}
}

func TestLookupCustomMapping_NoMatch(t *testing.T) {
	path := writeTempFile(t, `
mappings:
  "UserEvent": "user-event-value"
`)

	cfg := config.NewDefaultConfig()
	cfg.Naming.NameMappingFile = path

	norm := normalizer.New(cfg)
	kvDet, _ := keyvalue.New(cfg)

	m, err := New(cfg, norm, kvDet, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, found := m.lookupCustomMapping("any-registry", "NonExistentSchema")
	if found {
		t.Error("expected no match")
	}
}

func TestMapSchema_CustomMapping(t *testing.T) {
	path := writeTempFile(t, `
mappings:
  "UserEvent": "my-custom-subject-value"
`)

	cfg := config.NewDefaultConfig()
	cfg.Naming.NameMappingFile = path

	norm := normalizer.New(cfg)
	kvDet, _ := keyvalue.New(cfg)

	m, err := New(cfg, norm, kvDet, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	schema := &models.GlueSchema{
		Name:         "UserEvent",
		RegistryName: "test-registry",
		DataFormat:   models.SchemaTypeAvro,
		Versions: []models.GlueSchemaVersion{
			{VersionNumber: 1, Definition: `{"type":"record","name":"UserEvent","fields":[{"name":"id","type":"string"}]}`},
		},
	}

	mapping, err := m.MapSchema(context.Background(), schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mapping.TargetSubject != "my-custom-subject-value" {
		t.Errorf("expected subject 'my-custom-subject-value', got %q", mapping.TargetSubject)
	}
	if mapping.NamingStrategy != "custom-mapping" {
		t.Errorf("expected strategy 'custom-mapping', got %q", mapping.NamingStrategy)
	}
}

func TestMapSchema_CustomMappingWithRoleOverride(t *testing.T) {
	path := writeTempFile(t, `
extended_mappings:
  - source: "UserSchema"
    subject: "user-key"
    role: "key"
`)

	cfg := config.NewDefaultConfig()
	cfg.Naming.NameMappingFile = path

	norm := normalizer.New(cfg)
	kvDet, _ := keyvalue.New(cfg)

	m, err := New(cfg, norm, kvDet, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	schema := &models.GlueSchema{
		Name:         "UserSchema",
		RegistryName: "test-registry",
		DataFormat:   models.SchemaTypeAvro,
	}

	mapping, err := m.MapSchema(context.Background(), schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mapping.DetectedRole != models.SchemaRoleKey {
		t.Errorf("expected role 'key', got %q", mapping.DetectedRole)
	}
}

func TestMapSchema_CustomMappingWithContextOverride(t *testing.T) {
	path := writeTempFile(t, `
extended_mappings:
  - source: "UserEvent"
    subject: "user-event-value"
    context: ".custom-context"
`)

	cfg := config.NewDefaultConfig()
	cfg.Naming.NameMappingFile = path

	norm := normalizer.New(cfg)
	kvDet, _ := keyvalue.New(cfg)

	m, err := New(cfg, norm, kvDet, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	schema := &models.GlueSchema{
		Name:         "UserEvent",
		RegistryName: "test-registry",
		DataFormat:   models.SchemaTypeAvro,
	}

	mapping, err := m.MapSchema(context.Background(), schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mapping.TargetContext != ".custom-context" {
		t.Errorf("expected context '.custom-context', got %q", mapping.TargetContext)
	}
}

func TestMapSchema_UnmappedUsesNormalPipeline(t *testing.T) {
	path := writeTempFile(t, `
mappings:
  "MappedSchema": "mapped-subject-value"
`)

	cfg := config.NewDefaultConfig()
	cfg.Naming.NameMappingFile = path

	norm := normalizer.New(cfg)
	kvDet, _ := keyvalue.New(cfg)

	m, err := New(cfg, norm, kvDet, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// This schema is NOT in the mapping file
	schema := &models.GlueSchema{
		Name:         "UnmappedEvent",
		RegistryName: "test-registry",
		DataFormat:   models.SchemaTypeAvro,
		Versions: []models.GlueSchemaVersion{
			{VersionNumber: 1, Definition: `{"type":"record","name":"UnmappedEvent","fields":[{"name":"id","type":"string"}]}`},
		},
	}

	mapping, err := m.MapSchema(context.Background(), schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should use normal topic strategy, not custom-mapping
	if mapping.NamingStrategy == "custom-mapping" {
		t.Error("unmapped schema should not use custom-mapping strategy")
	}
	if mapping.NamingStrategy != "topic" {
		t.Errorf("expected strategy 'topic', got %q", mapping.NamingStrategy)
	}
}

func TestMapSchema_NoMappingFile(t *testing.T) {
	cfg := config.NewDefaultConfig()
	// No NameMappingFile set

	norm := normalizer.New(cfg)
	kvDet, _ := keyvalue.New(cfg)

	m, err := New(cfg, norm, kvDet, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	schema := &models.GlueSchema{
		Name:         "UserEvent",
		RegistryName: "test-registry",
		DataFormat:   models.SchemaTypeAvro,
		Versions: []models.GlueSchemaVersion{
			{VersionNumber: 1, Definition: `{"type":"record","name":"UserEvent","fields":[{"name":"id","type":"string"}]}`},
		},
	}

	mapping, err := m.MapSchema(context.Background(), schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should use normal pipeline
	if mapping.NamingStrategy != "topic" {
		t.Errorf("expected strategy 'topic', got %q", mapping.NamingStrategy)
	}
}

func TestLoadCustomMappings_AllThreeStyles(t *testing.T) {
	path := writeTempFile(t, `
mappings:
  "SimpleSchema": "simple-subject-value"

qualified_mappings:
  "registry-a:QualifiedSchema": "qualified-subject-value"

extended_mappings:
  - source: "ExtendedSchema"
    subject: "extended-subject-value"
    role: "value"
    context: ".ext"
`)

	loaded, err := loadCustomMappings(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 2 simple: SimpleSchema + ExtendedSchema (no colon)
	if len(loaded.simple) != 2 {
		t.Errorf("expected 2 simple mappings, got %d", len(loaded.simple))
	}

	// 1 qualified
	if len(loaded.qualified) != 1 {
		t.Errorf("expected 1 qualified mapping, got %d", len(loaded.qualified))
	}

	if loaded.simple["SimpleSchema"].Subject != "simple-subject-value" {
		t.Errorf("unexpected simple subject: %q", loaded.simple["SimpleSchema"].Subject)
	}
	if loaded.qualified["registry-a:QualifiedSchema"].Subject != "qualified-subject-value" {
		t.Errorf("unexpected qualified subject: %q", loaded.qualified["registry-a:QualifiedSchema"].Subject)
	}
	if loaded.simple["ExtendedSchema"].Subject != "extended-subject-value" {
		t.Errorf("unexpected extended subject: %q", loaded.simple["ExtendedSchema"].Subject)
	}
	if loaded.simple["ExtendedSchema"].Role != "value" {
		t.Errorf("unexpected extended role: %q", loaded.simple["ExtendedSchema"].Role)
	}
	if loaded.simple["ExtendedSchema"].Context != ".ext" {
		t.Errorf("unexpected extended context: %q", loaded.simple["ExtendedSchema"].Context)
	}
}

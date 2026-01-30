package keyvalue

import (
	"testing"

	"github.com/akrishnanDG/glue-to-ccsr/internal/models"
	"github.com/akrishnanDG/glue-to-ccsr/pkg/config"
)

func TestDetector_Detect(t *testing.T) {
	tests := []struct {
		name         string
		schemaName   string
		expectedRole models.SchemaRole
	}{
		// Key patterns
		{"ends with -key", "user-event-key", models.SchemaRoleKey},
		{"ends with _key", "user_event_key", models.SchemaRoleKey},
		{"ends with Key", "UserEventKey", models.SchemaRoleKey},
		{"ends with Id", "usereventId", models.SchemaRoleKey},
		{"ends with ID", "usereventID", models.SchemaRoleKey},
		{"ends with userEventID", "userEventID", models.SchemaRoleKey},
		{"ends with _pk", "user_pk", models.SchemaRoleKey},
		{"ends with -id", "user-id", models.SchemaRoleKey},
		
		// Value patterns
		{"ends with -value", "user-event-value", models.SchemaRoleValue},
		{"ends with _value", "user_event_value", models.SchemaRoleValue},
		{"ends with Value", "UserEventValue", models.SchemaRoleValue},
		{"ends with Event", "UserEvent", models.SchemaRoleValue},
		{"ends with Message", "UserMessage", models.SchemaRoleValue},
		{"ends with Payload", "EventPayload", models.SchemaRoleValue},
		{"ends with Data", "UserData", models.SchemaRoleValue},
		
		// Default (no pattern matches)
		{"no pattern match", "user-schema", models.SchemaRoleValue},
	}

	cfg := config.NewDefaultConfig()
	detector, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create detector: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.Detect("test-registry", tt.schemaName, nil)
			if result.Role != tt.expectedRole {
				t.Errorf("Detect(%q) = %q (reason: %s), expected %q", 
					tt.schemaName, result.Role, result.Reason, tt.expectedRole)
			}
		})
	}
}

func TestDetector_WithUserPatterns(t *testing.T) {
	cfg := config.NewDefaultConfig()
	cfg.KeyValue.KeyRegex = []string{`.*EntityId$`, `.*RecordKey$`}
	cfg.KeyValue.ValueRegex = []string{`.*Envelope$`, `.*Created$`}

	detector, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create detector: %v", err)
	}

	tests := []struct {
		name         string
		schemaName   string
		expectedRole models.SchemaRole
	}{
		{"custom key pattern EntityId", "UserEntityId", models.SchemaRoleKey},
		{"custom key pattern RecordKey", "PaymentRecordKey", models.SchemaRoleKey},
		{"custom value pattern Envelope", "MessageEnvelope", models.SchemaRoleValue},
		{"custom value pattern Created", "OrderCreated", models.SchemaRoleValue},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.Detect("test-registry", tt.schemaName, nil)
			if result.Role != tt.expectedRole {
				t.Errorf("Detect(%q) = %q (reason: %s), expected %q",
					tt.schemaName, result.Role, result.Reason, tt.expectedRole)
			}
		})
	}
}

func TestDetector_DisableBuiltinPatterns(t *testing.T) {
	cfg := config.NewDefaultConfig()
	cfg.KeyValue.DisableBuiltinPatterns = true
	cfg.KeyValue.KeyRegex = []string{`.*MyCustomKey$`}
	cfg.KeyValue.DefaultRole = "value"

	detector, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create detector: %v", err)
	}

	// With builtin disabled, standard patterns shouldn't match
	result := detector.Detect("test-registry", "user-event-key", nil)
	if result.Role != models.SchemaRoleValue {
		t.Errorf("Expected default role 'value' when builtin patterns disabled, got %q", result.Role)
	}

	// Custom pattern should still work
	result = detector.Detect("test-registry", "MySchemaMyCustomKey", nil)
	if result.Role != models.SchemaRoleKey {
		t.Errorf("Expected 'key' for custom pattern, got %q", result.Role)
	}
}

func TestDetector_StructureBasedDetection(t *testing.T) {
	cfg := config.NewDefaultConfig()
	cfg.KeyValue.DisableBuiltinPatterns = true // Disable patterns to test structure

	detector, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create detector: %v", err)
	}

	// Schema with few ID-like fields -> key
	keyParsed := &models.ParsedSchema{
		Fields: []models.Field{
			{Name: "user_id", Type: "string"},
			{Name: "partition_key", Type: "string"},
		},
	}

	result := detector.Detect("test-registry", "unknown-schema", keyParsed)
	if result.Role != models.SchemaRoleKey {
		t.Errorf("Expected key for schema with ID-like fields, got %q (reason: %s)", result.Role, result.Reason)
	}

	// Schema with many fields -> value
	valueParsed := &models.ParsedSchema{
		Fields: []models.Field{
			{Name: "user_id", Type: "string"},
			{Name: "name", Type: "string"},
			{Name: "email", Type: "string"},
			{Name: "address", Type: "string"},
			{Name: "phone", Type: "string"},
			{Name: "created_at", Type: "timestamp"},
		},
	}

	result = detector.Detect("test-registry", "unknown-schema", valueParsed)
	if result.Role != models.SchemaRoleValue {
		t.Errorf("Expected value for schema with many fields, got %q (reason: %s)", result.Role, result.Reason)
	}
}

func TestGetSuffix(t *testing.T) {
	if suffix := GetSuffix(models.SchemaRoleKey); suffix != "-key" {
		t.Errorf("GetSuffix(key) = %q, expected -key", suffix)
	}

	if suffix := GetSuffix(models.SchemaRoleValue); suffix != "-value" {
		t.Errorf("GetSuffix(value) = %q, expected -value", suffix)
	}
}

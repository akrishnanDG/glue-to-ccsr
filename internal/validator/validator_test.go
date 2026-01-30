package validator

import (
	"testing"

	"github.com/akrishnanDG/glue-to-ccsr/internal/models"
	"github.com/akrishnanDG/glue-to-ccsr/pkg/config"
)

func TestValidateSubjectName(t *testing.T) {
	cfg := config.NewDefaultConfig()
	v := New(cfg)

	tests := []struct {
		name    string
		subject string
		wantErr bool
	}{
		{"valid simple name", "user-event-value", false},
		{"valid with dots", "com.example.user-event", false},
		{"valid with underscores", "user_event_value", false},
		{"empty name", "", true},
		{"starts with underscore", "_internal-schema", true},
		{"contains invalid char slash", "user/event", true},
		{"contains invalid char space", "user event", true},
		{"contains invalid char colon", "user:event", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.validateSubjectName(tt.subject)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateSubjectName(%q) error = %v, wantErr %v", tt.subject, err, tt.wantErr)
			}
		})
	}
}

func TestValidateContextName(t *testing.T) {
	cfg := config.NewDefaultConfig()
	v := New(cfg)

	tests := []struct {
		name    string
		context string
		wantErr bool
	}{
		{"valid context", ".payments", false},
		{"valid with underscore", ".payment_context", false},
		{"valid with hyphen", ".payment-context", false},
		{"empty context (default)", "", false},
		{"missing dot prefix", "payments", true},
		{"invalid char in context", ".payment/context", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.validateContextName(tt.context)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateContextName(%q) error = %v, wantErr %v", tt.context, err, tt.wantErr)
			}
		})
	}
}

func TestValidateAll_Collisions(t *testing.T) {
	cfg := config.NewDefaultConfig()
	v := New(cfg)

	mappings := []*models.SchemaMapping{
		{
			SourceRegistry:   "reg1",
			SourceSchemaName: "schema1",
			TargetContext:    ".context",
			TargetSubject:    "same-name-value",
		},
		{
			SourceRegistry:   "reg2",
			SourceSchemaName: "schema2",
			TargetContext:    ".context",
			TargetSubject:    "same-name-value", // Collision!
		},
	}

	result := v.ValidateAll(mappings)
	
	if !result.HasErrors() {
		t.Error("Expected collision error but got none")
	}

	found := false
	for _, err := range result.Errors {
		if err.Message != "" && len(err.Message) > 0 {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to find collision error message")
	}
}

func TestCheckWarnings(t *testing.T) {
	cfg := config.NewDefaultConfig()
	v := New(cfg)

	tests := []struct {
		name           string
		mapping        *models.SchemaMapping
		expectWarnings bool
	}{
		{
			name: "AWS prefix warning",
			mapping: &models.SchemaMapping{
				SourceRegistry:   "test",
				SourceSchemaName: "MSK_PaymentEvent",
				TargetSubject:    "payment-event-value",
			},
			expectWarnings: true,
		},
		{
			name: "Special char warning",
			mapping: &models.SchemaMapping{
				SourceRegistry:   "test",
				SourceSchemaName: "user/event",
				TargetSubject:    "user-event-value",
			},
			expectWarnings: true,
		},
		{
			name: "Version suffix warning",
			mapping: &models.SchemaMapping{
				SourceRegistry:   "test",
				SourceSchemaName: "user-event_v2",
				TargetSubject:    "user-event-value",
			},
			expectWarnings: true,
		},
		{
			name: "Clean schema no warnings",
			mapping: &models.SchemaMapping{
				SourceRegistry:   "test",
				SourceSchemaName: "user-event",
				TargetSubject:    "user-event-value",
			},
			expectWarnings: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warnings := v.checkWarnings(tt.mapping)
			hasWarnings := len(warnings) > 0
			if hasWarnings != tt.expectWarnings {
				t.Errorf("checkWarnings() hasWarnings = %v, expected %v", hasWarnings, tt.expectWarnings)
			}
		})
	}
}

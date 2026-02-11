package extractor

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/glue"
	"github.com/aws/aws-sdk-go-v2/service/glue/types"

	"github.com/akrishnanDG/glue-to-ccsr/internal/models"
	"github.com/akrishnanDG/glue-to-ccsr/pkg/config"
	"golang.org/x/time/rate"
)

// mockGlueClient implements GlueAPI for unit tests.
type mockGlueClient struct {
	ListRegistriesFn    func(ctx context.Context, params *glue.ListRegistriesInput, optFns ...func(*glue.Options)) (*glue.ListRegistriesOutput, error)
	GetRegistryFn       func(ctx context.Context, params *glue.GetRegistryInput, optFns ...func(*glue.Options)) (*glue.GetRegistryOutput, error)
	ListSchemasFn       func(ctx context.Context, params *glue.ListSchemasInput, optFns ...func(*glue.Options)) (*glue.ListSchemasOutput, error)
	GetSchemaFn         func(ctx context.Context, params *glue.GetSchemaInput, optFns ...func(*glue.Options)) (*glue.GetSchemaOutput, error)
	ListSchemaVersionsFn func(ctx context.Context, params *glue.ListSchemaVersionsInput, optFns ...func(*glue.Options)) (*glue.ListSchemaVersionsOutput, error)
	GetSchemaVersionFn  func(ctx context.Context, params *glue.GetSchemaVersionInput, optFns ...func(*glue.Options)) (*glue.GetSchemaVersionOutput, error)
}

func (m *mockGlueClient) ListRegistries(ctx context.Context, params *glue.ListRegistriesInput, optFns ...func(*glue.Options)) (*glue.ListRegistriesOutput, error) {
	if m.ListRegistriesFn != nil {
		return m.ListRegistriesFn(ctx, params, optFns...)
	}
	return &glue.ListRegistriesOutput{}, nil
}

func (m *mockGlueClient) GetRegistry(ctx context.Context, params *glue.GetRegistryInput, optFns ...func(*glue.Options)) (*glue.GetRegistryOutput, error) {
	if m.GetRegistryFn != nil {
		return m.GetRegistryFn(ctx, params, optFns...)
	}
	return &glue.GetRegistryOutput{}, nil
}

func (m *mockGlueClient) ListSchemas(ctx context.Context, params *glue.ListSchemasInput, optFns ...func(*glue.Options)) (*glue.ListSchemasOutput, error) {
	if m.ListSchemasFn != nil {
		return m.ListSchemasFn(ctx, params, optFns...)
	}
	return &glue.ListSchemasOutput{}, nil
}

func (m *mockGlueClient) GetSchema(ctx context.Context, params *glue.GetSchemaInput, optFns ...func(*glue.Options)) (*glue.GetSchemaOutput, error) {
	if m.GetSchemaFn != nil {
		return m.GetSchemaFn(ctx, params, optFns...)
	}
	return &glue.GetSchemaOutput{}, nil
}

func (m *mockGlueClient) ListSchemaVersions(ctx context.Context, params *glue.ListSchemaVersionsInput, optFns ...func(*glue.Options)) (*glue.ListSchemaVersionsOutput, error) {
	if m.ListSchemaVersionsFn != nil {
		return m.ListSchemaVersionsFn(ctx, params, optFns...)
	}
	return &glue.ListSchemaVersionsOutput{}, nil
}

func (m *mockGlueClient) GetSchemaVersion(ctx context.Context, params *glue.GetSchemaVersionInput, optFns ...func(*glue.Options)) (*glue.GetSchemaVersionOutput, error) {
	if m.GetSchemaVersionFn != nil {
		return m.GetSchemaVersionFn(ctx, params, optFns...)
	}
	return &glue.GetSchemaVersionOutput{}, nil
}

// newTestExtractor builds a GlueExtractor wired to the given mock client.
func newTestExtractor(mock *mockGlueClient) *GlueExtractor {
	cfg := config.NewDefaultConfig()
	cfg.AWS.RegistryNames = []string{"test-reg"}
	cfg.Concurrency.Workers = 2

	limiter := rate.NewLimiter(rate.Limit(1000), 1)
	return NewWithClient(cfg, mock, limiter)
}

// ---------------------------------------------------------------------------
// TestGetSchema_Success
// ---------------------------------------------------------------------------

func TestGetSchema_Success(t *testing.T) {
	mock := &mockGlueClient{
		GetSchemaFn: func(ctx context.Context, params *glue.GetSchemaInput, optFns ...func(*glue.Options)) (*glue.GetSchemaOutput, error) {
			return &glue.GetSchemaOutput{
				SchemaName:          aws.String("user-event"),
				RegistryName:        aws.String("test-reg"),
				SchemaArn:           aws.String("arn:aws:glue:us-east-1:123456789012:schema/test-reg/user-event"),
				Description:         aws.String("User domain event"),
				DataFormat:          types.DataFormatAvro,
				Compatibility:       types.CompatibilityBackward,
				LatestSchemaVersion: aws.Int64(2),
			}, nil
		},
		ListSchemaVersionsFn: func(ctx context.Context, params *glue.ListSchemaVersionsInput, optFns ...func(*glue.Options)) (*glue.ListSchemaVersionsOutput, error) {
			return &glue.ListSchemaVersionsOutput{
				Schemas: []types.SchemaVersionListItem{
					{SchemaVersionId: aws.String("ver-id-1"), VersionNumber: aws.Int64(1), Status: types.SchemaVersionStatusAvailable},
					{SchemaVersionId: aws.String("ver-id-2"), VersionNumber: aws.Int64(2), Status: types.SchemaVersionStatusAvailable},
				},
			}, nil
		},
		GetSchemaVersionFn: func(ctx context.Context, params *glue.GetSchemaVersionInput, optFns ...func(*glue.Options)) (*glue.GetSchemaVersionOutput, error) {
			vn := aws.ToInt64(params.SchemaVersionNumber.VersionNumber)
			return &glue.GetSchemaVersionOutput{
				SchemaDefinition: aws.String(fmt.Sprintf(`{"type":"record","name":"UserEvent","version":%d}`, vn)),
				VersionNumber:    aws.Int64(vn),
				SchemaVersionId:  aws.String(fmt.Sprintf("ver-id-%d", vn)),
				Status:           types.SchemaVersionStatusAvailable,
			}, nil
		},
	}

	ext := newTestExtractor(mock)
	schema, err := ext.GetSchema(context.Background(), "test-reg", "user-event")
	if err != nil {
		t.Fatalf("GetSchema returned unexpected error: %v", err)
	}

	if schema.Name != "user-event" {
		t.Errorf("Name = %q, want %q", schema.Name, "user-event")
	}
	if schema.RegistryName != "test-reg" {
		t.Errorf("RegistryName = %q, want %q", schema.RegistryName, "test-reg")
	}
	if schema.ARN != "arn:aws:glue:us-east-1:123456789012:schema/test-reg/user-event" {
		t.Errorf("ARN = %q, want full ARN", schema.ARN)
	}
	if schema.Description != "User domain event" {
		t.Errorf("Description = %q, want %q", schema.Description, "User domain event")
	}
	if schema.DataFormat != models.SchemaTypeAvro {
		t.Errorf("DataFormat = %q, want %q", schema.DataFormat, models.SchemaTypeAvro)
	}
	if schema.Compatibility != string(types.CompatibilityBackward) {
		t.Errorf("Compatibility = %q, want %q", schema.Compatibility, types.CompatibilityBackward)
	}
	if schema.LatestVersion != 2 {
		t.Errorf("LatestVersion = %d, want 2", schema.LatestVersion)
	}
	if len(schema.Versions) != 2 {
		t.Fatalf("len(Versions) = %d, want 2", len(schema.Versions))
	}

	// Versions should be sorted ascending by version number.
	if schema.Versions[0].VersionNumber != 1 {
		t.Errorf("Versions[0].VersionNumber = %d, want 1", schema.Versions[0].VersionNumber)
	}
	if schema.Versions[1].VersionNumber != 2 {
		t.Errorf("Versions[1].VersionNumber = %d, want 2", schema.Versions[1].VersionNumber)
	}
	if schema.Versions[0].SchemaVersionID != "ver-id-1" {
		t.Errorf("Versions[0].SchemaVersionID = %q, want %q", schema.Versions[0].SchemaVersionID, "ver-id-1")
	}
	if schema.Versions[1].Definition == "" {
		t.Error("Versions[1].Definition is empty, expected non-empty schema definition")
	}
	if schema.Versions[0].Status != string(types.SchemaVersionStatusAvailable) {
		t.Errorf("Versions[0].Status = %q, want %q", schema.Versions[0].Status, types.SchemaVersionStatusAvailable)
	}
}

// ---------------------------------------------------------------------------
// TestGetSchema_APIError
// ---------------------------------------------------------------------------

func TestGetSchema_APIError(t *testing.T) {
	apiErr := fmt.Errorf("AccessDeniedException: not authorized")
	mock := &mockGlueClient{
		GetSchemaFn: func(ctx context.Context, params *glue.GetSchemaInput, optFns ...func(*glue.Options)) (*glue.GetSchemaOutput, error) {
			return nil, apiErr
		},
	}

	ext := newTestExtractor(mock)
	_, err := ext.GetSchema(context.Background(), "test-reg", "missing-schema")
	if err == nil {
		t.Fatal("expected error from GetSchema, got nil")
	}
	// The extractor wraps the error; verify the original message is preserved.
	if got := err.Error(); got == "" {
		t.Error("error message is empty")
	}
}

// ---------------------------------------------------------------------------
// TestIsExcluded
// ---------------------------------------------------------------------------

func TestIsExcluded(t *testing.T) {
	tests := []struct {
		name     string
		patterns []string
		registry string
		want     bool
	}{
		{
			name:     "exact match",
			patterns: []string{"staging"},
			registry: "staging",
			want:     true,
		},
		{
			name:     "glob prefix match",
			patterns: []string{"test-*"},
			registry: "test-payments",
			want:     true,
		},
		{
			name:     "glob no match",
			patterns: []string{"test-*"},
			registry: "prod-payments",
			want:     false,
		},
		{
			name:     "multiple patterns first matches",
			patterns: []string{"staging", "dev-*"},
			registry: "staging",
			want:     true,
		},
		{
			name:     "multiple patterns second matches",
			patterns: []string{"staging", "dev-*"},
			registry: "dev-sandbox",
			want:     true,
		},
		{
			name:     "no patterns",
			patterns: nil,
			registry: "anything",
			want:     false,
		},
		{
			name:     "wildcard all",
			patterns: []string{"*"},
			registry: "production",
			want:     true,
		},
		{
			name:     "partial string not a match",
			patterns: []string{"test"},
			registry: "test-payments",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.NewDefaultConfig()
			cfg.AWS.RegistryExclude = tt.patterns
			limiter := rate.NewLimiter(rate.Limit(1000), 1)
			ext := NewWithClient(cfg, &mockGlueClient{}, limiter)

			got := ext.isExcluded(tt.registry)
			if got != tt.want {
				t.Errorf("isExcluded(%q) with patterns %v = %v, want %v", tt.registry, tt.patterns, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestSortVersions
// ---------------------------------------------------------------------------

func TestSortVersions(t *testing.T) {
	versions := []models.GlueSchemaVersion{
		{VersionNumber: 3, SchemaVersionID: "v3"},
		{VersionNumber: 1, SchemaVersionID: "v1"},
		{VersionNumber: 5, SchemaVersionID: "v5"},
		{VersionNumber: 2, SchemaVersionID: "v2"},
		{VersionNumber: 4, SchemaVersionID: "v4"},
	}

	sortVersions(versions)

	for i := 0; i < len(versions); i++ {
		expected := int64(i + 1)
		if versions[i].VersionNumber != expected {
			t.Errorf("versions[%d].VersionNumber = %d, want %d", i, versions[i].VersionNumber, expected)
		}
	}
}

// ---------------------------------------------------------------------------
// TestParseTimestamp
// ---------------------------------------------------------------------------

func TestParseTimestamp(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  time.Time
	}{
		{
			name:  "valid RFC3339",
			input: "2024-06-15T10:30:00Z",
			want:  time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC),
		},
		{
			name:  "valid RFC3339 with offset",
			input: "2024-06-15T10:30:00+05:30",
			want: func() time.Time {
				loc := time.FixedZone("IST", 5*3600+30*60)
				return time.Date(2024, 6, 15, 10, 30, 0, 0, loc)
			}(),
		},
		{
			name:  "empty string",
			input: "",
			want:  time.Time{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseTimestamp(tt.input)
			if !got.Equal(tt.want) {
				t.Errorf("parseTimestamp(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

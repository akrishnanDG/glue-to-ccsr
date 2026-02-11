//go:build integration

package migrator

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/glue"
	gluetypes "github.com/aws/aws-sdk-go-v2/service/glue/types"
	"github.com/akrishnanDG/glue-to-ccsr/internal/extractor"
	"github.com/akrishnanDG/glue-to-ccsr/internal/keyvalue"
	"github.com/akrishnanDG/glue-to-ccsr/internal/loader"
	"github.com/akrishnanDG/glue-to-ccsr/internal/mapper"
	"github.com/akrishnanDG/glue-to-ccsr/internal/normalizer"
	"github.com/akrishnanDG/glue-to-ccsr/internal/validator"
	"github.com/akrishnanDG/glue-to-ccsr/internal/worker"
	"github.com/akrishnanDG/glue-to-ccsr/pkg/config"
	"golang.org/x/time/rate"
)

// mockGlueClient implements extractor.GlueAPI for integration tests.
type mockGlueClient struct {
	schemas map[string]map[string]*mockSchema // registry -> schema name -> data
}

type mockSchema struct {
	definition string
	format     gluetypes.DataFormat
}

func (m *mockGlueClient) ListRegistries(ctx context.Context, params *glue.ListRegistriesInput, optFns ...func(*glue.Options)) (*glue.ListRegistriesOutput, error) {
	var items []gluetypes.RegistryListItem
	seen := make(map[string]bool)
	for reg := range m.schemas {
		if !seen[reg] {
			seen[reg] = true
			items = append(items, gluetypes.RegistryListItem{
				RegistryName: aws.String(reg),
				RegistryArn:  aws.String("arn:aws:glue:us-east-1:123456789:" + reg),
			})
		}
	}
	return &glue.ListRegistriesOutput{Registries: items}, nil
}

func (m *mockGlueClient) GetRegistry(ctx context.Context, params *glue.GetRegistryInput, optFns ...func(*glue.Options)) (*glue.GetRegistryOutput, error) {
	return &glue.GetRegistryOutput{
		RegistryName: params.RegistryId.RegistryName,
		RegistryArn:  aws.String("arn:aws:glue:us-east-1:123456789:" + aws.ToString(params.RegistryId.RegistryName)),
	}, nil
}

func (m *mockGlueClient) ListSchemas(ctx context.Context, params *glue.ListSchemasInput, optFns ...func(*glue.Options)) (*glue.ListSchemasOutput, error) {
	regName := aws.ToString(params.RegistryId.RegistryName)
	var items []gluetypes.SchemaListItem
	if schemas, ok := m.schemas[regName]; ok {
		for name := range schemas {
			items = append(items, gluetypes.SchemaListItem{
				SchemaName:   aws.String(name),
				RegistryName: aws.String(regName),
				SchemaArn:    aws.String("arn:schema:" + name),
			})
		}
	}
	return &glue.ListSchemasOutput{Schemas: items}, nil
}

func (m *mockGlueClient) GetSchema(ctx context.Context, params *glue.GetSchemaInput, optFns ...func(*glue.Options)) (*glue.GetSchemaOutput, error) {
	regName := aws.ToString(params.SchemaId.RegistryName)
	schemaName := aws.ToString(params.SchemaId.SchemaName)
	if schemas, ok := m.schemas[regName]; ok {
		if s, ok := schemas[schemaName]; ok {
			return &glue.GetSchemaOutput{
				SchemaName:          aws.String(schemaName),
				RegistryName:        aws.String(regName),
				DataFormat:          s.format,
				Compatibility:       gluetypes.CompatibilityBackward,
				LatestSchemaVersion: aws.Int64(1),
				SchemaArn:           aws.String("arn:schema:" + schemaName),
			}, nil
		}
	}
	return nil, &gluetypes.EntityNotFoundException{Message: aws.String("schema not found")}
}

func (m *mockGlueClient) ListSchemaVersions(ctx context.Context, params *glue.ListSchemaVersionsInput, optFns ...func(*glue.Options)) (*glue.ListSchemaVersionsOutput, error) {
	return &glue.ListSchemaVersionsOutput{
		Schemas: []gluetypes.SchemaVersionListItem{
			{
				SchemaVersionId: aws.String("ver-001"),
				VersionNumber:   aws.Int64(1),
				Status:          gluetypes.SchemaVersionStatusAvailable,
			},
		},
	}, nil
}

func (m *mockGlueClient) GetSchemaVersion(ctx context.Context, params *glue.GetSchemaVersionInput, optFns ...func(*glue.Options)) (*glue.GetSchemaVersionOutput, error) {
	// Look up the schema definition from the schema ID
	schemaName := aws.ToString(params.SchemaId.SchemaName)
	regName := aws.ToString(params.SchemaId.RegistryName)
	definition := `{"type":"record","name":"Unknown","fields":[{"name":"id","type":"string"}]}`
	if schemas, ok := m.schemas[regName]; ok {
		if s, ok := schemas[schemaName]; ok {
			definition = s.definition
		}
	}
	return &glue.GetSchemaVersionOutput{
		SchemaDefinition: aws.String(definition),
		VersionNumber:    aws.Int64(1),
		SchemaVersionId:  aws.String("ver-001"),
		Status:           gluetypes.SchemaVersionStatusAvailable,
	}, nil
}

type registeredSchema struct {
	Method  string
	Path    string
	Body    map[string]interface{}
}

func TestFullMigrationPipeline(t *testing.T) {
	// Track requests to Confluent Cloud SR
	var mu sync.Mutex
	var registered []registeredSchema

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		if r.Method == "POST" && strings.Contains(r.URL.Path, "/subjects/") {
			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)
			registered = append(registered, registeredSchema{
				Method: r.Method,
				Path:   r.URL.Path,
				Body:   body,
			})
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"id": 1}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	// Set up config
	cfg := config.NewDefaultConfig()
	cfg.AWS.Region = "us-east-1"
	cfg.AWS.RegistryAll = true
	cfg.ConfluentCloud.URL = server.URL
	cfg.ConfluentCloud.APIKey = "test-key"
	cfg.ConfluentCloud.APISecret = "test-secret"
	cfg.Concurrency.Workers = 2
	cfg.Concurrency.RetryAttempts = 0
	cfg.Output.DryRun = false

	// Set up mock Glue client with 2 schemas
	mockClient := &mockGlueClient{
		schemas: map[string]map[string]*mockSchema{
			"test-registry": {
				"UserEvent": {
					definition: `{"type":"record","name":"UserEvent","namespace":"com.example","fields":[{"name":"id","type":"string"},{"name":"name","type":"string"},{"name":"email","type":"string"}]}`,
					format:     gluetypes.DataFormatAvro,
				},
				"OrderEvent": {
					definition: `{"type":"record","name":"OrderEvent","namespace":"com.example","fields":[{"name":"orderId","type":"string"},{"name":"amount","type":"double"},{"name":"status","type":"string"}]}`,
					format:     gluetypes.DataFormatAvro,
				},
			},
		},
	}

	// Build components
	limiter := rate.NewLimiter(rate.Limit(1000), 1)
	ext := extractor.NewWithClient(cfg, mockClient, limiter)

	ldr, err := loader.New(cfg)
	if err != nil {
		t.Fatalf("failed to create loader: %v", err)
	}

	norm := normalizer.New(cfg)
	kvDet, err := keyvalue.New(cfg)
	if err != nil {
		t.Fatalf("failed to create key/value detector: %v", err)
	}

	mpr, err := mapper.New(cfg, norm, kvDet, nil)
	if err != nil {
		t.Fatalf("failed to create mapper: %v", err)
	}

	val := validator.New(cfg)
	pool := worker.NewPool(cfg)

	// Build migrator with injected deps
	m := NewWithDeps(cfg, ext, ldr, mpr, norm, kvDet, val, pool)

	// Run migration
	result, err := m.Run(context.Background())
	if err != nil {
		t.Fatalf("migration failed: %v", err)
	}

	// Verify results
	if result.Successful != 2 {
		t.Errorf("expected 2 successful, got %d", result.Successful)
	}
	if result.Failed != 0 {
		t.Errorf("expected 0 failed, got %d", result.Failed)
	}

	// Verify schemas were registered in Confluent Cloud
	mu.Lock()
	defer mu.Unlock()
	if len(registered) != 2 {
		t.Errorf("expected 2 schema registrations, got %d", len(registered))
	}
}

func TestDryRunProducesReport(t *testing.T) {
	// Server should NOT receive any requests in dry-run mode
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := config.NewDefaultConfig()
	cfg.AWS.Region = "us-east-1"
	cfg.AWS.RegistryAll = true
	cfg.ConfluentCloud.URL = server.URL
	cfg.ConfluentCloud.APIKey = "test-key"
	cfg.ConfluentCloud.APISecret = "test-secret"
	cfg.Output.DryRun = true

	mockClient := &mockGlueClient{
		schemas: map[string]map[string]*mockSchema{
			"test-registry": {
				"TestSchema": {
					definition: `{"type":"record","name":"TestSchema","fields":[{"name":"id","type":"string"}]}`,
					format:     gluetypes.DataFormatAvro,
				},
			},
		},
	}

	limiter := rate.NewLimiter(rate.Limit(1000), 1)
	ext := extractor.NewWithClient(cfg, mockClient, limiter)
	ldr, _ := loader.New(cfg)
	norm := normalizer.New(cfg)
	kvDet, _ := keyvalue.New(cfg)
	mpr, _ := mapper.New(cfg, norm, kvDet, nil)
	val := validator.New(cfg)
	pool := worker.NewPool(cfg)

	m := NewWithDeps(cfg, ext, ldr, mpr, norm, kvDet, val, pool)

	result, err := m.Run(context.Background())
	if err != nil {
		t.Fatalf("dry-run failed: %v", err)
	}

	// Should have a report
	if result.Report == nil {
		t.Error("expected report to be generated")
	}

	// No HTTP requests should have been made
	if requestCount > 0 {
		t.Errorf("expected 0 HTTP requests in dry-run, got %d", requestCount)
	}
}

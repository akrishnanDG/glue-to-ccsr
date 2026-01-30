package graph

import (
	"testing"

	"github.com/akrishnanDG/glue-to-ccsr/internal/models"
)

func TestBuild_SimpleGraph(t *testing.T) {
	schemas := []*models.GlueSchema{
		{
			Name:         "address",
			RegistryName: "shared",
			DataFormat:   models.SchemaTypeAvro,
			Versions: []models.GlueSchemaVersion{
				{VersionNumber: 1, Definition: `{"type":"record","name":"Address","fields":[]}`},
			},
		},
		{
			Name:         "customer",
			RegistryName: "payments",
			DataFormat:   models.SchemaTypeAvro,
			Versions: []models.GlueSchemaVersion{
				{VersionNumber: 1, Definition: `{"type":"record","name":"Customer","fields":[{"name":"addr","type":"string"}]}`},
			},
		},
	}

	graph, err := Build(schemas)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	levels := graph.GetLevels()
	// Both schemas have no dependencies (customer doesn't reference address in this test)
	if len(levels) < 1 {
		t.Error("Expected at least 1 level")
	}

	// Level 0 should have both schemas (no dependencies)
	if len(levels) > 0 {
		if len(levels[0].Schemas) != 2 {
			t.Errorf("Expected 2 schemas in level 0, got %d", len(levels[0].Schemas))
		}
	}
}

func TestBuild_NoDependencies(t *testing.T) {
	schemas := []*models.GlueSchema{
		{
			Name:         "schema1",
			RegistryName: "default",
			DataFormat:   models.SchemaTypeAvro,
			Versions: []models.GlueSchemaVersion{
				{VersionNumber: 1, Definition: `{"type":"record","name":"Schema1","fields":[]}`},
			},
		},
		{
			Name:         "schema2",
			RegistryName: "default",
			DataFormat:   models.SchemaTypeAvro,
			Versions: []models.GlueSchemaVersion{
				{VersionNumber: 1, Definition: `{"type":"record","name":"Schema2","fields":[]}`},
			},
		},
	}

	graph, err := Build(schemas)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	levels := graph.GetLevels()
	if len(levels) != 1 {
		t.Errorf("Expected 1 level for independent schemas, got %d", len(levels))
	}

	if len(levels[0].Schemas) != 2 {
		t.Errorf("Expected 2 schemas in level 0, got %d", len(levels[0].Schemas))
	}
}

func TestSchemaKey(t *testing.T) {
	key := schemaKey("registry", "schema")
	if key != "registry:schema" {
		t.Errorf("schemaKey = %q, expected 'registry:schema'", key)
	}
}

func TestAppendUnique(t *testing.T) {
	slice := []string{"a", "b"}
	
	// Adding new item
	result := appendUnique(slice, "c")
	if len(result) != 3 {
		t.Errorf("Expected 3 items, got %d", len(result))
	}

	// Adding duplicate
	result = appendUnique(result, "b")
	if len(result) != 3 {
		t.Errorf("Expected 3 items (no duplicate), got %d", len(result))
	}
}

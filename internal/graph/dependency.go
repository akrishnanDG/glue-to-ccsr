package graph

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/akrishnanDG/glue-to-ccsr/internal/models"
)

// Level represents a dependency level in the graph
type Level struct {
	Level   int                    `json:"level"`
	Schemas []models.SchemaMapping `json:"schemas"`
}

// DependencyGraph represents the schema dependency graph
type DependencyGraph struct {
	// nodes maps schema key to its parsed schema
	nodes map[string]*models.ParsedSchema
	
	// edges maps schema key to its dependencies (schemas it references)
	edges map[string][]string
	
	// reverseEdges maps schema key to schemas that depend on it
	reverseEdges map[string][]string
	
	// levels stores the topologically sorted levels
	levels []Level
}

// Build builds a dependency graph from the given schemas
func Build(schemas []*models.GlueSchema) (*DependencyGraph, error) {
	g := &DependencyGraph{
		nodes:        make(map[string]*models.ParsedSchema),
		edges:        make(map[string][]string),
		reverseEdges: make(map[string][]string),
	}

	// First pass: add all nodes
	for _, schema := range schemas {
		key := schemaKey(schema.RegistryName, schema.Name)
		parsed, err := parseSchema(schema)
		if err != nil {
			return nil, fmt.Errorf("failed to parse schema %s: %w", key, err)
		}
		g.nodes[key] = parsed
	}

	// Second pass: build edges based on references
	for key, parsed := range g.nodes {
		for _, ref := range parsed.References {
			// Try to resolve the reference to an existing schema
			refKey := g.resolveReference(ref, parsed.GlueSchema.RegistryName)
			if refKey != "" {
				g.edges[key] = append(g.edges[key], refKey)
				g.reverseEdges[refKey] = append(g.reverseEdges[refKey], key)
			}
		}
	}

	// Detect cycles
	if err := g.detectCycles(); err != nil {
		return nil, err
	}

	// Perform topological sort to get levels
	g.levels = g.topologicalSort()

	return g, nil
}

// GetLevels returns the dependency levels for ordered migration
func (g *DependencyGraph) GetLevels() []Level {
	return g.levels
}

// GetDependencies returns the dependencies for a schema
func (g *DependencyGraph) GetDependencies(registryName, schemaName string) []string {
	key := schemaKey(registryName, schemaName)
	return g.edges[key]
}

// GetDependents returns the schemas that depend on the given schema
func (g *DependencyGraph) GetDependents(registryName, schemaName string) []string {
	key := schemaKey(registryName, schemaName)
	return g.reverseEdges[key]
}

func schemaKey(registryName, schemaName string) string {
	return fmt.Sprintf("%s:%s", registryName, schemaName)
}

func (g *DependencyGraph) resolveReference(ref string, currentRegistry string) string {
	// Try exact match first (for cross-registry references)
	if _, exists := g.nodes[ref]; exists {
		return ref
	}

	// Try with current registry
	key := schemaKey(currentRegistry, ref)
	if _, exists := g.nodes[key]; exists {
		return key
	}

	// Try matching just the schema name across all registries
	for nodeKey := range g.nodes {
		parts := strings.SplitN(nodeKey, ":", 2)
		if len(parts) == 2 && parts[1] == ref {
			return nodeKey
		}
	}

	return ""
}

func (g *DependencyGraph) detectCycles() error {
	// Use DFS with coloring to detect cycles
	// 0 = white (unvisited), 1 = gray (in progress), 2 = black (done)
	color := make(map[string]int)
	
	var dfs func(node string, path []string) error
	dfs = func(node string, path []string) error {
		color[node] = 1 // gray
		
		for _, dep := range g.edges[node] {
			if color[dep] == 1 {
				// Found a cycle
				cyclePath := append(path, node, dep)
				return fmt.Errorf("circular dependency detected: %s", strings.Join(cyclePath, " -> "))
			}
			if color[dep] == 0 {
				if err := dfs(dep, append(path, node)); err != nil {
					return err
				}
			}
		}
		
		color[node] = 2 // black
		return nil
	}

	for node := range g.nodes {
		if color[node] == 0 {
			if err := dfs(node, nil); err != nil {
				return err
			}
		}
	}

	return nil
}

func (g *DependencyGraph) topologicalSort() []Level {
	// Calculate in-degree for each node
	inDegree := make(map[string]int)
	for key := range g.nodes {
		inDegree[key] = 0
	}
	for _, deps := range g.edges {
		for _, dep := range deps {
			inDegree[dep]++
		}
	}

	// Start with nodes that have no dependencies
	var levels []Level
	remaining := make(map[string]bool)
	for key := range g.nodes {
		remaining[key] = true
	}

	level := 0
	for len(remaining) > 0 {
		// Find all nodes with in-degree 0 (no remaining dependencies)
		var currentLevel []string
		for key := range remaining {
			if inDegree[key] == 0 {
				currentLevel = append(currentLevel, key)
			}
		}

		if len(currentLevel) == 0 {
			// This shouldn't happen if cycle detection worked
			break
		}

		// Create level with schema mappings
		levelSchemas := make([]models.SchemaMapping, 0, len(currentLevel))
		for _, key := range currentLevel {
			parsed := g.nodes[key]
			mapping := models.SchemaMapping{
				SourceRegistry:   parsed.GlueSchema.RegistryName,
				SourceSchemaName: parsed.GlueSchema.Name,
				SourceVersions:   len(parsed.GlueSchema.Versions),
				References:       g.edges[key],
				DependencyLevel:  level,
				Status:           models.MappingStatusReady,
			}
			levelSchemas = append(levelSchemas, mapping)
			
			// Remove from remaining and update in-degrees
			delete(remaining, key)
			for _, dependentKey := range g.reverseEdges[key] {
				inDegree[dependentKey]--
			}
		}

		levels = append(levels, Level{
			Level:   level,
			Schemas: levelSchemas,
		})
		level++
	}

	return levels
}

func parseSchema(schema *models.GlueSchema) (*models.ParsedSchema, error) {
	parsed := &models.ParsedSchema{
		GlueSchema: schema,
	}

	if len(schema.Versions) == 0 {
		return parsed, nil
	}

	// Parse the latest version to extract metadata
	latestVersion := schema.Versions[len(schema.Versions)-1]
	
	switch schema.DataFormat {
	case models.SchemaTypeAvro:
		if err := parseAvroSchema(latestVersion.Definition, parsed); err != nil {
			return nil, err
		}
	case models.SchemaTypeJSON:
		if err := parseJSONSchema(latestVersion.Definition, parsed); err != nil {
			return nil, err
		}
	case models.SchemaTypeProtobuf:
		if err := parseProtobufSchema(latestVersion.Definition, parsed); err != nil {
			return nil, err
		}
	}

	return parsed, nil
}

func parseAvroSchema(definition string, parsed *models.ParsedSchema) error {
	var avro map[string]interface{}
	if err := json.Unmarshal([]byte(definition), &avro); err != nil {
		return fmt.Errorf("failed to parse Avro schema: %w", err)
	}

	// Extract record name
	if name, ok := avro["name"].(string); ok {
		parsed.RecordName = name
	}

	// Extract namespace
	if ns, ok := avro["namespace"].(string); ok {
		parsed.Namespace = ns
	}

	// Extract documentation
	if doc, ok := avro["doc"].(string); ok {
		parsed.Documentation = doc
	}

	// Extract fields
	if fields, ok := avro["fields"].([]interface{}); ok {
		for _, f := range fields {
			if field, ok := f.(map[string]interface{}); ok {
				fieldModel := models.Field{
					Name: field["name"].(string),
				}
				
				// Parse type
				fieldType := field["type"]
				fieldModel.Type = extractAvroType(fieldType)
				
				// Check if it's a reference to another schema
				refType := extractAvroReference(fieldType)
				if refType != "" {
					parsed.References = appendUnique(parsed.References, refType)
				}
				
				// Extract doc
				if doc, ok := field["doc"].(string); ok {
					fieldModel.Doc = doc
				}
				
				parsed.Fields = append(parsed.Fields, fieldModel)
			}
		}
	}

	return nil
}

func parseJSONSchema(definition string, parsed *models.ParsedSchema) error {
	var jsonSchema map[string]interface{}
	if err := json.Unmarshal([]byte(definition), &jsonSchema); err != nil {
		return fmt.Errorf("failed to parse JSON schema: %w", err)
	}

	// Extract title
	if title, ok := jsonSchema["title"].(string); ok {
		parsed.RecordName = title
	}

	// Extract description
	if desc, ok := jsonSchema["description"].(string); ok {
		parsed.Documentation = desc
	}

	// Extract properties
	if props, ok := jsonSchema["properties"].(map[string]interface{}); ok {
		for name, prop := range props {
			fieldModel := models.Field{
				Name: name,
			}
			
			if propMap, ok := prop.(map[string]interface{}); ok {
				if t, ok := propMap["type"].(string); ok {
					fieldModel.Type = t
				}
				
				// Check for $ref
				if ref, ok := propMap["$ref"].(string); ok {
					parsed.References = appendUnique(parsed.References, ref)
				}
			}
			
			parsed.Fields = append(parsed.Fields, fieldModel)
		}
	}

	// Check required fields
	if required, ok := jsonSchema["required"].([]interface{}); ok {
		requiredMap := make(map[string]bool)
		for _, r := range required {
			if s, ok := r.(string); ok {
				requiredMap[s] = true
			}
		}
		for i := range parsed.Fields {
			if requiredMap[parsed.Fields[i].Name] {
				parsed.Fields[i].IsRequired = true
			}
		}
	}

	return nil
}

func parseProtobufSchema(definition string, parsed *models.ParsedSchema) error {
	// Simple protobuf parsing for message name and imports
	lines := strings.Split(definition, "\n")
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		// Extract package
		if strings.HasPrefix(line, "package ") {
			parsed.Namespace = strings.TrimSuffix(strings.TrimPrefix(line, "package "), ";")
		}
		
		// Extract message name
		if strings.HasPrefix(line, "message ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				parsed.RecordName = parts[1]
			}
		}
		
		// Extract imports (references)
		if strings.HasPrefix(line, "import ") {
			importPath := strings.Trim(strings.TrimSuffix(strings.TrimPrefix(line, "import "), ";"), "\"")
			parsed.References = appendUnique(parsed.References, importPath)
		}
	}

	return nil
}

func extractAvroType(t interface{}) string {
	switch v := t.(type) {
	case string:
		return v
	case map[string]interface{}:
		if typeStr, ok := v["type"].(string); ok {
			return typeStr
		}
	case []interface{}:
		// Union type
		var types []string
		for _, item := range v {
			if s, ok := item.(string); ok {
				types = append(types, s)
			} else if m, ok := item.(map[string]interface{}); ok {
				if typeStr, ok := m["type"].(string); ok {
					types = append(types, typeStr)
				}
			}
		}
		return strings.Join(types, "|")
	}
	return "unknown"
}

func extractAvroReference(t interface{}) string {
	switch v := t.(type) {
	case string:
		// If it's not a primitive type, it might be a reference
		primitives := map[string]bool{
			"null": true, "boolean": true, "int": true, "long": true,
			"float": true, "double": true, "bytes": true, "string": true,
		}
		if !primitives[v] {
			return v
		}
	case map[string]interface{}:
		// Check for named type
		if name, ok := v["name"].(string); ok {
			if _, isPrimitive := v["type"]; !isPrimitive {
				return name
			}
		}
	case []interface{}:
		// Union type - check each element
		for _, item := range v {
			if ref := extractAvroReference(item); ref != "" {
				return ref
			}
		}
	}
	return ""
}

func appendUnique(slice []string, item string) []string {
	for _, s := range slice {
		if s == item {
			return slice
		}
	}
	return append(slice, item)
}

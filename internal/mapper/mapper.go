package mapper

import (
	"context"
	"fmt"
	"strings"
	"text/template"

	"github.com/akrishnanDG/glue-to-ccsr/internal/keyvalue"
	"github.com/akrishnanDG/glue-to-ccsr/internal/llm"
	"github.com/akrishnanDG/glue-to-ccsr/internal/models"
	"github.com/akrishnanDG/glue-to-ccsr/internal/normalizer"
	"github.com/akrishnanDG/glue-to-ccsr/pkg/config"
)

// NomenclatureMapper maps AWS Glue schema names to Confluent Cloud subject names
type NomenclatureMapper struct {
	config          *config.Config
	normalizer      *normalizer.Normalizer
	kvDetector      *keyvalue.Detector
	llmNamer        *llm.Namer
	template        *template.Template
	customMappings  *loadedCustomMappings
	contextMappings map[string]string // registry_name -> context_name
}

// New creates a new NomenclatureMapper
func New(cfg *config.Config, norm *normalizer.Normalizer, kvDet *keyvalue.Detector, llmNmr *llm.Namer) (*NomenclatureMapper, error) {
	m := &NomenclatureMapper{
		config:     cfg,
		normalizer: norm,
		kvDetector: kvDet,
		llmNamer:   llmNmr,
	}

	// Load custom name mappings if specified
	if cfg.Naming.NameMappingFile != "" {
		mappings, err := loadCustomMappings(cfg.Naming.NameMappingFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load custom name mappings: %w", err)
		}
		m.customMappings = mappings
	}

	// Load context mappings if using custom context mapping
	if cfg.Naming.ContextMapping == "custom" && cfg.Naming.ContextMappingFile != "" {
		ctxMappings, err := loadContextMappings(cfg.Naming.ContextMappingFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load context mapping file: %w", err)
		}
		m.contextMappings = ctxMappings
	}

	// Parse custom template if using custom strategy
	if cfg.Naming.SubjectStrategy == "custom" && cfg.Naming.SubjectTemplate != "" {
		tmpl, err := template.New("subject").Parse(cfg.Naming.SubjectTemplate)
		if err == nil {
			m.template = tmpl
		}
	}

	return m, nil
}

// MapAll maps all schemas to Confluent Cloud subjects
func (m *NomenclatureMapper) MapAll(ctx context.Context, schemas []*models.GlueSchema) ([]*models.SchemaMapping, error) {
	var mappings []*models.SchemaMapping

	for _, schema := range schemas {
		mapping, err := m.MapSchema(ctx, schema)
		if err != nil {
			return nil, fmt.Errorf("failed to map schema %s: %w", schema.Name, err)
		}
		mappings = append(mappings, mapping)
	}

	return mappings, nil
}

// MapSchema maps a single schema to a Confluent Cloud subject
func (m *NomenclatureMapper) MapSchema(ctx context.Context, schema *models.GlueSchema) (*models.SchemaMapping, error) {
	mapping := &models.SchemaMapping{
		SourceRegistry:   schema.RegistryName,
		SourceSchemaName: schema.Name,
		SourceVersions:   len(schema.Versions),
		Status:           models.MappingStatusReady,
	}

	// Check custom name mappings first (highest priority)
	if customMapping, found := m.lookupCustomMapping(schema.RegistryName, schema.Name); found {
		mapping.TargetSubject = customMapping.Subject
		mapping.NamingStrategy = "custom-mapping"
		mapping.NamingReason = "Custom name mapping file"
		mapping.Transformations = []string{fmt.Sprintf("custom-mapping: %s -> %s", schema.Name, customMapping.Subject)}

		// Use overridden role if provided, otherwise detect normally
		if customMapping.Role != "" {
			mapping.DetectedRole = models.SchemaRole(customMapping.Role)
		} else {
			parsed := m.parseSchemaMetadata(schema)
			detection := m.kvDetector.Detect(schema.RegistryName, schema.Name, parsed)
			mapping.DetectedRole = detection.Role
		}

		// Use overridden context if provided, otherwise generate normally
		if customMapping.Context != "" {
			mapping.TargetContext = customMapping.Context
		} else {
			mapping.TargetContext = m.generateContext(schema.RegistryName)
		}

		return mapping, nil
	}

	// Parse the schema to extract metadata
	parsed := m.parseSchemaMetadata(schema)

	// Detect key/value role
	detection := m.kvDetector.Detect(schema.RegistryName, schema.Name, parsed)
	mapping.DetectedRole = detection.Role
	mapping.NamingReason = detection.Reason

	// Generate context
	mapping.TargetContext = m.generateContext(schema.RegistryName)

	// Generate subject name based on strategy
	var err error
	mapping.TargetSubject, mapping.NamingStrategy, mapping.Transformations, err = m.generateSubjectName(ctx, schema, parsed, detection.Role)
	if err != nil {
		mapping.Status = models.MappingStatusError
		mapping.Error = err.Error()
		return mapping, nil
	}

	return mapping, nil
}

func (m *NomenclatureMapper) generateContext(registryName string) string {
	switch m.config.Naming.ContextMapping {
	case "registry":
		// Map registry to context
		return "." + registryName
	case "flat":
		// All schemas in default context
		return ""
	case "custom":
		if m.contextMappings != nil {
			if ctx, ok := m.contextMappings[registryName]; ok {
				return "." + ctx
			}
		}
		return "." + registryName
	default:
		return "." + registryName
	}
}

func (m *NomenclatureMapper) generateSubjectName(ctx context.Context, schema *models.GlueSchema, parsed *models.ParsedSchema, role models.SchemaRole) (string, string, []string, error) {
	var baseName string
	var strategy string
	var transformations []string

	switch m.config.Naming.SubjectStrategy {
	case "topic":
		strategy = "topic"
		baseName, transformations = m.topicNameStrategy(schema, role)

	case "record":
		strategy = "record"
		baseName, transformations = m.recordNameStrategy(schema, parsed, role)

	case "llm":
		strategy = "llm"
		var err error
		baseName, transformations, err = m.llmNameStrategy(ctx, schema, parsed, role)
		if err != nil {
			// Fall back to topic strategy
			strategy = "topic (fallback)"
			baseName, transformations = m.topicNameStrategy(schema, role)
		}

	case "custom":
		strategy = "custom"
		var err error
		baseName, transformations, err = m.customNameStrategy(schema, parsed, role)
		if err != nil {
			return "", "", nil, err
		}

	default:
		strategy = "topic"
		baseName, transformations = m.topicNameStrategy(schema, role)
	}

	return baseName, strategy, transformations, nil
}

// topicNameStrategy uses the schema name as the subject base with role suffix
func (m *NomenclatureMapper) topicNameStrategy(schema *models.GlueSchema, role models.SchemaRole) (string, []string) {
	// Normalize the schema name
	normalized, transforms := m.normalizer.Normalize(schema.Name)

	// Strip existing key/value suffixes before adding our own
	normalized = normalizer.StripKeySuffix(normalized)
	normalized = normalizer.StripValueSuffix(normalized)

	// Add role suffix
	suffix := keyvalue.GetSuffix(role)
	result := normalized + suffix

	return result, transforms
}

// recordNameStrategy uses the record name from the schema definition
func (m *NomenclatureMapper) recordNameStrategy(schema *models.GlueSchema, parsed *models.ParsedSchema, role models.SchemaRole) (string, []string) {
	var baseName string
	var transforms []string

	if parsed != nil && parsed.RecordName != "" {
		// Use the record name
		baseName = parsed.RecordName
		
		// Include namespace if present
		if parsed.Namespace != "" {
			baseName = parsed.Namespace + "." + baseName
		}
	} else {
		// Fall back to schema name
		baseName = schema.Name
	}

	// Normalize
	normalized, normTransforms := m.normalizer.Normalize(baseName)
	transforms = append(transforms, normTransforms...)

	// Add role suffix
	suffix := keyvalue.GetSuffix(role)
	result := normalized + suffix

	return result, transforms
}

// llmNameStrategy uses an LLM to suggest the subject name
func (m *NomenclatureMapper) llmNameStrategy(ctx context.Context, schema *models.GlueSchema, parsed *models.ParsedSchema, role models.SchemaRole) (string, []string, error) {
	if m.llmNamer == nil {
		return "", nil, fmt.Errorf("LLM namer not configured")
	}

	suggestion, err := m.llmNamer.SuggestName(ctx, schema, parsed, role)
	if err != nil {
		return "", nil, err
	}

	var transforms []string
	if suggestion.OriginalName != suggestion.SuggestedName {
		transforms = append(transforms, fmt.Sprintf("LLM: %s → %s", suggestion.OriginalName, suggestion.SuggestedName))
	}

	return suggestion.SuggestedName, transforms, nil
}

// customNameStrategy uses a user-defined template
func (m *NomenclatureMapper) customNameStrategy(schema *models.GlueSchema, parsed *models.ParsedSchema, role models.SchemaRole) (string, []string, error) {
	if m.template == nil {
		return "", nil, fmt.Errorf("custom template not configured")
	}

	data := map[string]string{
		"registry":     schema.RegistryName,
		"name":         schema.Name,
		"schema_name":  schema.Name,
		"role":         string(role),
		"suffix":       keyvalue.GetSuffix(role),
	}

	if parsed != nil {
		data["record_name"] = parsed.RecordName
		data["namespace"] = parsed.Namespace
	}

	var buf strings.Builder
	if err := m.template.Execute(&buf, data); err != nil {
		return "", nil, fmt.Errorf("failed to execute template: %w", err)
	}

	result := buf.String()
	
	// Normalize the result
	normalized, transforms := m.normalizer.Normalize(result)

	return normalized, transforms, nil
}

func (m *NomenclatureMapper) parseSchemaMetadata(schema *models.GlueSchema) *models.ParsedSchema {
	parsed := &models.ParsedSchema{
		GlueSchema: schema,
	}

	if len(schema.Versions) == 0 {
		return parsed
	}

	// Parse the latest version
	latestVersion := schema.Versions[len(schema.Versions)-1]
	
	switch schema.DataFormat {
	case models.SchemaTypeAvro:
		m.parseAvroMetadata(latestVersion.Definition, parsed)
	case models.SchemaTypeJSON:
		m.parseJSONMetadata(latestVersion.Definition, parsed)
	case models.SchemaTypeProtobuf:
		m.parseProtobufMetadata(latestVersion.Definition, parsed)
	}

	return parsed
}

func (m *NomenclatureMapper) parseAvroMetadata(definition string, parsed *models.ParsedSchema) {
	// Simple parsing - extract name, namespace, doc
	// This is a simplified version; the full parsing is in graph package
	if strings.Contains(definition, `"name"`) {
		// Extract name
		start := strings.Index(definition, `"name"`)
		if start != -1 {
			rest := definition[start+7:]
			rest = strings.TrimLeft(rest, `: "`)
			end := strings.Index(rest, `"`)
			if end != -1 {
				parsed.RecordName = rest[:end]
			}
		}
	}

	if strings.Contains(definition, `"namespace"`) {
		start := strings.Index(definition, `"namespace"`)
		if start != -1 {
			rest := definition[start+12:]
			rest = strings.TrimLeft(rest, `: "`)
			end := strings.Index(rest, `"`)
			if end != -1 {
				parsed.Namespace = rest[:end]
			}
		}
	}
}

func (m *NomenclatureMapper) parseJSONMetadata(definition string, parsed *models.ParsedSchema) {
	if strings.Contains(definition, `"title"`) {
		start := strings.Index(definition, `"title"`)
		if start != -1 {
			rest := definition[start+8:]
			rest = strings.TrimLeft(rest, `: "`)
			end := strings.Index(rest, `"`)
			if end != -1 {
				parsed.RecordName = rest[:end]
			}
		}
	}
}

func (m *NomenclatureMapper) parseProtobufMetadata(definition string, parsed *models.ParsedSchema) {
	lines := strings.Split(definition, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "message ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				parsed.RecordName = parts[1]
				break
			}
		}
		if strings.HasPrefix(line, "package ") {
			parsed.Namespace = strings.TrimSuffix(strings.TrimPrefix(line, "package "), ";")
		}
	}
}

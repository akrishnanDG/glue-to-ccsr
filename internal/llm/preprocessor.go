package llm

import (
	"encoding/json"
	"strings"

	"github.com/akrishnanDG/glue-to-ccsr/internal/models"
)

// Preprocessor extracts key context from schemas for LLM processing
type Preprocessor struct{}

// SchemaContext represents the preprocessed context sent to the LLM
type SchemaContext struct {
	GlueSchemaName  string   `json:"glue_schema_name"`
	GlueRegistry    string   `json:"glue_registry"`
	SchemaType      string   `json:"schema_type"`
	RecordName      string   `json:"record_name,omitempty"`
	Namespace       string   `json:"namespace,omitempty"`
	Documentation   string   `json:"documentation,omitempty"`
	KeyFields       []string `json:"key_fields,omitempty"`
	FieldCount      int      `json:"field_count"`
	References      []string `json:"references,omitempty"`
}

// FieldSummary represents a summarized field
type FieldSummary struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Required bool   `json:"required,omitempty"`
}

// NewPreprocessor creates a new Preprocessor
func NewPreprocessor() *Preprocessor {
	return &Preprocessor{}
}

// ExtractContext extracts key context from a schema for LLM processing
func (p *Preprocessor) ExtractContext(schema *models.GlueSchema, parsed *models.ParsedSchema) *SchemaContext {
	ctx := &SchemaContext{
		GlueSchemaName: schema.Name,
		GlueRegistry:   schema.RegistryName,
		SchemaType:     string(schema.DataFormat),
	}

	// Add parsed metadata if available
	if parsed != nil {
		ctx.RecordName = parsed.RecordName
		ctx.Namespace = parsed.Namespace
		ctx.Documentation = parsed.Documentation
		ctx.FieldCount = len(parsed.Fields)
		ctx.References = parsed.References

		// Extract key field summaries (top 10)
		for i, field := range parsed.Fields {
			if i >= 10 {
				break
			}
			summary := field.Name
			if field.Type != "" {
				summary += " (" + field.Type + ")"
			}
			ctx.KeyFields = append(ctx.KeyFields, summary)
		}
	}

	// If no parsed data, try to extract from latest version
	if len(schema.Versions) > 0 && ctx.RecordName == "" {
		latestVersion := schema.Versions[len(schema.Versions)-1]
		p.extractFromDefinition(latestVersion.Definition, string(schema.DataFormat), ctx)
	}

	return ctx
}

func (p *Preprocessor) extractFromDefinition(definition string, schemaType string, ctx *SchemaContext) {
	switch schemaType {
	case "AVRO":
		p.extractFromAvro(definition, ctx)
	case "JSON":
		p.extractFromJSON(definition, ctx)
	case "PROTOBUF":
		p.extractFromProtobuf(definition, ctx)
	}
}

func (p *Preprocessor) extractFromAvro(definition string, ctx *SchemaContext) {
	var avro map[string]interface{}
	if err := json.Unmarshal([]byte(definition), &avro); err != nil {
		return
	}

	if name, ok := avro["name"].(string); ok {
		ctx.RecordName = name
	}

	if ns, ok := avro["namespace"].(string); ok {
		ctx.Namespace = ns
	}

	if doc, ok := avro["doc"].(string); ok {
		ctx.Documentation = truncate(doc, 200)
	}

	if fields, ok := avro["fields"].([]interface{}); ok {
		ctx.FieldCount = len(fields)
		
		// Extract top fields
		for i, f := range fields {
			if i >= 10 {
				break
			}
			if field, ok := f.(map[string]interface{}); ok {
				name := ""
				if n, ok := field["name"].(string); ok {
					name = n
				}
				typeStr := extractAvroTypeString(field["type"])
				summary := name
				if typeStr != "" {
					summary += " (" + typeStr + ")"
				}
				ctx.KeyFields = append(ctx.KeyFields, summary)
			}
		}
	}
}

func (p *Preprocessor) extractFromJSON(definition string, ctx *SchemaContext) {
	var jsonSchema map[string]interface{}
	if err := json.Unmarshal([]byte(definition), &jsonSchema); err != nil {
		return
	}

	if title, ok := jsonSchema["title"].(string); ok {
		ctx.RecordName = title
	}

	if desc, ok := jsonSchema["description"].(string); ok {
		ctx.Documentation = truncate(desc, 200)
	}

	if props, ok := jsonSchema["properties"].(map[string]interface{}); ok {
		ctx.FieldCount = len(props)
		
		count := 0
		for name, prop := range props {
			if count >= 10 {
				break
			}
			typeStr := "object"
			if propMap, ok := prop.(map[string]interface{}); ok {
				if t, ok := propMap["type"].(string); ok {
					typeStr = t
				}
			}
			ctx.KeyFields = append(ctx.KeyFields, name+" ("+typeStr+")")
			count++
		}
	}
}

func (p *Preprocessor) extractFromProtobuf(definition string, ctx *SchemaContext) {
	lines := strings.Split(definition, "\n")
	fieldCount := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "package ") {
			ctx.Namespace = strings.TrimSuffix(strings.TrimPrefix(line, "package "), ";")
		}

		if strings.HasPrefix(line, "message ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 && ctx.RecordName == "" {
				ctx.RecordName = parts[1]
			}
		}

		// Count fields (simplified)
		if strings.Contains(line, "=") && !strings.HasPrefix(line, "//") && !strings.HasPrefix(line, "option") {
			fieldCount++
			
			// Extract field name and type
			if len(ctx.KeyFields) < 10 {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					fieldType := parts[0]
					fieldName := parts[1]
					ctx.KeyFields = append(ctx.KeyFields, fieldName+" ("+fieldType+")")
				}
			}
		}
	}

	ctx.FieldCount = fieldCount
}

func extractAvroTypeString(t interface{}) string {
	switch v := t.(type) {
	case string:
		return v
	case map[string]interface{}:
		if typeStr, ok := v["type"].(string); ok {
			return typeStr
		}
		if name, ok := v["name"].(string); ok {
			return name
		}
	case []interface{}:
		// Union type
		var types []string
		for _, item := range v {
			if s, ok := item.(string); ok {
				types = append(types, s)
			}
		}
		if len(types) > 0 {
			return strings.Join(types, "|")
		}
	}
	return "complex"
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

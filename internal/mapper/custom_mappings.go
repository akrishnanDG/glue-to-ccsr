package mapper

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// CustomNameMappingFile represents the structure of a custom name mapping YAML file.
type CustomNameMappingFile struct {
	// Simple mappings: schema name -> subject (matches any registry)
	Mappings map[string]string `yaml:"mappings"`

	// Qualified mappings: registry:schema -> subject (specific registry)
	QualifiedMappings map[string]string `yaml:"qualified_mappings"`

	// Extended mappings with optional role and context overrides
	ExtendedMappings []ExtendedMapping `yaml:"extended_mappings"`
}

// ExtendedMapping represents a mapping with optional role and context overrides.
type ExtendedMapping struct {
	Source  string `yaml:"source"`  // schema name or registry:schema
	Subject string `yaml:"subject"` // target subject name
	Role    string `yaml:"role"`    // optional: key or value
	Context string `yaml:"context"` // optional: target context
}

// ResolvedCustomMapping is the internal representation after loading.
type ResolvedCustomMapping struct {
	Subject string
	Role    string // empty means use auto-detection
	Context string // empty means use default generation
}

// loadedCustomMappings holds all resolved custom mappings for fast lookup.
type loadedCustomMappings struct {
	// Keyed by "registry:schemaName" for qualified lookups
	qualified map[string]*ResolvedCustomMapping
	// Keyed by "schemaName" for simple lookups
	simple map[string]*ResolvedCustomMapping
}

func loadCustomMappings(path string) (*loadedCustomMappings, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read mapping file: %w", err)
	}

	var file CustomNameMappingFile
	if err := yaml.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("failed to parse mapping file: %w", err)
	}

	loaded := &loadedCustomMappings{
		qualified: make(map[string]*ResolvedCustomMapping),
		simple:    make(map[string]*ResolvedCustomMapping),
	}

	// Load simple mappings
	for source, subject := range file.Mappings {
		loaded.simple[source] = &ResolvedCustomMapping{Subject: subject}
	}

	// Load qualified mappings
	for source, subject := range file.QualifiedMappings {
		loaded.qualified[source] = &ResolvedCustomMapping{Subject: subject}
	}

	// Load extended mappings
	for _, ext := range file.ExtendedMappings {
		resolved := &ResolvedCustomMapping{
			Subject: ext.Subject,
			Role:    ext.Role,
			Context: ext.Context,
		}
		if strings.Contains(ext.Source, ":") {
			loaded.qualified[ext.Source] = resolved
		} else {
			loaded.simple[ext.Source] = resolved
		}
	}

	return loaded, nil
}

func loadContextMappings(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read context mapping file: %w", err)
	}

	var mappings map[string]string
	if err := yaml.Unmarshal(data, &mappings); err != nil {
		return nil, fmt.Errorf("failed to parse context mapping file: %w", err)
	}

	return mappings, nil
}

func (m *NomenclatureMapper) lookupCustomMapping(registryName, schemaName string) (*ResolvedCustomMapping, bool) {
	if m.customMappings == nil {
		return nil, false
	}

	// Priority 1: Qualified match (registry:schema)
	qualifiedKey := registryName + ":" + schemaName
	if mapping, ok := m.customMappings.qualified[qualifiedKey]; ok {
		return mapping, true
	}

	// Priority 2: Simple match (schema name only)
	if mapping, ok := m.customMappings.simple[schemaName]; ok {
		return mapping, true
	}

	return nil, false
}

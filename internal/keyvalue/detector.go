package keyvalue

import (
	"os"
	"regexp"
	"strings"

	"github.com/akrishnanDG/glue-to-ccsr/internal/models"
	"github.com/akrishnanDG/glue-to-ccsr/pkg/config"
	"gopkg.in/yaml.v3"
)

// Detector detects whether a schema is a key or value schema
type Detector struct {
	config          *config.Config
	keyPatterns     []*regexp.Regexp
	valuePatterns   []*regexp.Regexp
	overrides       map[string]models.SchemaRole
	registryConfig  map[string]*RegistryKeyValueConfig
}

// RegistryKeyValueConfig holds registry-specific key/value configuration
type RegistryKeyValueConfig struct {
	KeyPatterns   []string `yaml:"key_patterns"`
	ValuePatterns []string `yaml:"value_patterns"`
	DefaultRole   string   `yaml:"default_role"`
}

// RoleOverrideFile represents the structure of the role override file
type RoleOverrideFile struct {
	Overrides     map[string]string                    `yaml:"overrides"`
	KeyPatterns   []string                             `yaml:"key_patterns"`
	ValuePatterns []string                             `yaml:"value_patterns"`
	Registries    map[string]*RegistryKeyValueConfig   `yaml:"registries"`
}

// DetectionResult contains the result of key/value detection
type DetectionResult struct {
	Role   models.SchemaRole
	Reason string
}

// Built-in key patterns
var builtinKeyPatterns = []string{
	`(?i)[-_]key$`,      // ends with -key or _key
	`Key$`,              // ends with Key (camelCase)
	`(?i)[-_]id$`,       // ends with -id or _id
	`Id$`,               // ends with Id (camelCase)
	`ID$`,               // ends with ID (uppercase)
	`(?i)identifier$`,   // ends with Identifier
	`(?i)[-_]pk$`,       // ends with -pk or _pk
	`(?i)primarykey$`,   // ends with PrimaryKey
	`(?i)partitionkey$`, // ends with PartitionKey
}

// Built-in value patterns
var builtinValuePatterns = []string{
	`(?i)[-_]value$`,    // ends with -value or _value
	`Value$`,            // ends with Value (camelCase)
	`(?i)event$`,        // ends with Event
	`(?i)message$`,      // ends with Message
	`(?i)payload$`,      // ends with Payload
	`(?i)data$`,         // ends with Data
	`(?i)record$`,       // ends with Record
}

// New creates a new key/value Detector
func New(cfg *config.Config) (*Detector, error) {
	d := &Detector{
		config:         cfg,
		overrides:      make(map[string]models.SchemaRole),
		registryConfig: make(map[string]*RegistryKeyValueConfig),
	}

	// Load override file if specified
	if cfg.KeyValue.RoleOverrideFile != "" {
		if err := d.loadOverrideFile(cfg.KeyValue.RoleOverrideFile); err != nil {
			return nil, err
		}
	}

	// Compile patterns
	if err := d.compilePatterns(); err != nil {
		return nil, err
	}

	return d, nil
}

func (d *Detector) loadOverrideFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var overrideFile RoleOverrideFile
	if err := yaml.Unmarshal(data, &overrideFile); err != nil {
		return err
	}

	// Load overrides
	for name, role := range overrideFile.Overrides {
		if role == "key" {
			d.overrides[name] = models.SchemaRoleKey
		} else {
			d.overrides[name] = models.SchemaRoleValue
		}
	}

	// Add patterns from file
	d.config.KeyValue.KeyRegex = append(d.config.KeyValue.KeyRegex, overrideFile.KeyPatterns...)
	d.config.KeyValue.ValueRegex = append(d.config.KeyValue.ValueRegex, overrideFile.ValuePatterns...)

	// Load registry-specific config
	d.registryConfig = overrideFile.Registries

	return nil
}

func (d *Detector) compilePatterns() error {
	// Compile key patterns
	if !d.config.KeyValue.DisableBuiltinPatterns {
		for _, pattern := range builtinKeyPatterns {
			re, err := regexp.Compile(pattern)
			if err != nil {
				return err
			}
			d.keyPatterns = append(d.keyPatterns, re)
		}
	}

	for _, pattern := range d.config.KeyValue.KeyRegex {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return err
		}
		d.keyPatterns = append(d.keyPatterns, re)
	}

	// Compile value patterns
	if !d.config.KeyValue.DisableBuiltinPatterns {
		for _, pattern := range builtinValuePatterns {
			re, err := regexp.Compile(pattern)
			if err != nil {
				return err
			}
			d.valuePatterns = append(d.valuePatterns, re)
		}
	}

	for _, pattern := range d.config.KeyValue.ValueRegex {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return err
		}
		d.valuePatterns = append(d.valuePatterns, re)
	}

	return nil
}

// Detect determines if a schema is a key or value schema
func (d *Detector) Detect(registryName, schemaName string, parsed *models.ParsedSchema) DetectionResult {
	// Priority 1: Explicit override
	if role, ok := d.overrides[schemaName]; ok {
		return DetectionResult{Role: role, Reason: "Override file"}
	}

	// Also check with registry prefix
	fullName := registryName + "." + schemaName
	if role, ok := d.overrides[fullName]; ok {
		return DetectionResult{Role: role, Reason: "Override file"}
	}

	// Priority 2: Registry-specific patterns
	if regConfig, ok := d.registryConfig[registryName]; ok {
		for _, pattern := range regConfig.KeyPatterns {
			re, err := regexp.Compile(pattern)
			if err == nil && re.MatchString(schemaName) {
				return DetectionResult{Role: models.SchemaRoleKey, Reason: "Registry pattern: " + pattern}
			}
		}
		for _, pattern := range regConfig.ValuePatterns {
			re, err := regexp.Compile(pattern)
			if err == nil && re.MatchString(schemaName) {
				return DetectionResult{Role: models.SchemaRoleValue, Reason: "Registry pattern: " + pattern}
			}
		}
	}

	// Priority 3: User-provided key patterns
	for i, re := range d.keyPatterns {
		if re.MatchString(schemaName) {
			pattern := "Built-in pattern"
			if !d.config.KeyValue.DisableBuiltinPatterns && i >= len(builtinKeyPatterns) {
				pattern = "User pattern"
			}
			return DetectionResult{Role: models.SchemaRoleKey, Reason: pattern + ": " + re.String()}
		}
	}

	// Priority 4: User-provided value patterns
	for i, re := range d.valuePatterns {
		if re.MatchString(schemaName) {
			pattern := "Built-in pattern"
			if !d.config.KeyValue.DisableBuiltinPatterns && i >= len(builtinValuePatterns) {
				pattern = "User pattern"
			}
			return DetectionResult{Role: models.SchemaRoleValue, Reason: pattern + ": " + re.String()}
		}
	}

	// Priority 5: Also check record name from parsed schema
	if parsed != nil && parsed.RecordName != "" {
		for _, re := range d.keyPatterns {
			if re.MatchString(parsed.RecordName) {
				return DetectionResult{Role: models.SchemaRoleKey, Reason: "Record name pattern: " + re.String()}
			}
		}
		for _, re := range d.valuePatterns {
			if re.MatchString(parsed.RecordName) {
				return DetectionResult{Role: models.SchemaRoleValue, Reason: "Record name pattern: " + re.String()}
			}
		}
	}

	// Priority 6: Structure-based detection
	if parsed != nil {
		result := d.detectByStructure(parsed)
		if result.Role != "" {
			return result
		}
	}

	// Priority 7: Default role
	defaultRole := models.SchemaRoleValue
	if d.config.KeyValue.DefaultRole == "key" {
		defaultRole = models.SchemaRoleKey
	}
	return DetectionResult{Role: defaultRole, Reason: "Default role"}
}

func (d *Detector) detectByStructure(parsed *models.ParsedSchema) DetectionResult {
	// Key schemas typically have 1-3 fields with ID-like names
	if len(parsed.Fields) > 0 && len(parsed.Fields) <= 3 {
		idFieldNames := []string{"id", "key", "uuid", "partition_key", "entity_id", "pk"}
		hasIdField := false
		
		for _, field := range parsed.Fields {
			fieldLower := strings.ToLower(field.Name)
			for _, idName := range idFieldNames {
				if strings.Contains(fieldLower, idName) {
					hasIdField = true
					break
				}
			}
		}
		
		if hasIdField {
			return DetectionResult{Role: models.SchemaRoleKey, Reason: "Structure: few fields with ID-like names"}
		}
	}

	// Value schemas typically have many fields
	if len(parsed.Fields) > 5 {
		return DetectionResult{Role: models.SchemaRoleValue, Reason: "Structure: many fields"}
	}

	return DetectionResult{}
}

// GetSuffix returns the appropriate suffix for the detected role
func GetSuffix(role models.SchemaRole) string {
	if role == models.SchemaRoleKey {
		return "-key"
	}
	return "-value"
}

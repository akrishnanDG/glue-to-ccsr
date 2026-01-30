package validator

import (
	"regexp"
	"strings"

	"github.com/akrishnanDG/glue-to-ccsr/internal/models"
	"github.com/akrishnanDG/glue-to-ccsr/pkg/config"
)

// Validator validates schema mappings before migration
type Validator struct {
	config *config.Config
}

// ValidationResult contains the results of validation
type ValidationResult struct {
	Errors   []models.Error
	Warnings []models.Warning
}

// HasErrors returns true if there are validation errors
func (r *ValidationResult) HasErrors() bool {
	return len(r.Errors) > 0
}

// HasWarnings returns true if there are validation warnings
func (r *ValidationResult) HasWarnings() bool {
	return len(r.Warnings) > 0
}

// New creates a new Validator
func New(cfg *config.Config) *Validator {
	return &Validator{
		config: cfg,
	}
}

// ValidateAll validates all mappings
func (v *Validator) ValidateAll(mappings []*models.SchemaMapping) *ValidationResult {
	result := &ValidationResult{}

	// Track subject names for collision detection
	subjectMap := make(map[string][]string)

	for _, mapping := range mappings {
		// Validate individual mapping
		errs, warns := v.ValidateMapping(mapping)
		result.Errors = append(result.Errors, errs...)
		result.Warnings = append(result.Warnings, warns...)

		// Track for collision detection
		// Format full subject with context (only add prefix if context is not empty)
		fullSubject := mapping.TargetSubject
		if mapping.TargetContext != "" {
			fullSubject = mapping.TargetContext + ":" + mapping.TargetSubject
		}
		sourceKey := mapping.SourceRegistry + "." + mapping.SourceSchemaName
		subjectMap[fullSubject] = append(subjectMap[fullSubject], sourceKey)
	}

	// Check for collisions
	for subject, sources := range subjectMap {
		if len(sources) > 1 {
			result.Errors = append(result.Errors, models.Error{
				Schema:  strings.Join(sources, ", "),
				Message: "Naming collision: multiple schemas map to " + subject,
			})
		}
	}

	return result
}

// ValidateMapping validates a single mapping
func (v *Validator) ValidateMapping(mapping *models.SchemaMapping) ([]models.Error, []models.Warning) {
	var errors []models.Error
	var warnings []models.Warning
	sourceKey := mapping.SourceRegistry + "." + mapping.SourceSchemaName

	// Validate target subject name
	if err := v.validateSubjectName(mapping.TargetSubject); err != nil {
		errors = append(errors, models.Error{
			Schema:  sourceKey,
			Message: err.Error(),
		})
	}

	// Validate context name
	if mapping.TargetContext != "" {
		if err := v.validateContextName(mapping.TargetContext); err != nil {
			errors = append(errors, models.Error{
				Schema:  sourceKey,
				Message: err.Error(),
			})
		}
	}

	// Check for potential issues (warnings)
	warns := v.checkWarnings(mapping)
	warnings = append(warnings, warns...)

	return errors, warnings
}

func (v *Validator) validateSubjectName(subject string) error {
	if subject == "" {
		return &ValidationError{Message: "subject name cannot be empty"}
	}

	// Check length
	if len(subject) > 255 {
		return &ValidationError{Message: "subject name exceeds maximum length of 255 characters"}
	}

	// Check for invalid characters
	// Confluent Cloud subjects allow: alphanumeric, dots, underscores, hyphens
	validPattern := regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)
	if !validPattern.MatchString(subject) {
		return &ValidationError{Message: "subject name contains invalid characters (only alphanumeric, dots, underscores, and hyphens allowed)"}
	}

	// Check for reserved patterns
	if strings.HasPrefix(subject, "_") {
		return &ValidationError{Message: "subject name cannot start with underscore (reserved)"}
	}

	return nil
}

func (v *Validator) validateContextName(context string) error {
	if context == "" {
		return nil // Empty context is valid (default context)
	}

	// Context must start with a dot
	if !strings.HasPrefix(context, ".") {
		return &ValidationError{Message: "context must start with a dot"}
	}

	// Remove the leading dot for validation
	contextName := strings.TrimPrefix(context, ".")

	// Check for invalid characters
	validPattern := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !validPattern.MatchString(contextName) {
		return &ValidationError{Message: "context name contains invalid characters"}
	}

	return nil
}

func (v *Validator) checkWarnings(mapping *models.SchemaMapping) []models.Warning {
	var warnings []models.Warning
	sourceKey := mapping.SourceRegistry + "." + mapping.SourceSchemaName

	// Warn about name changes
	if mapping.SourceSchemaName != mapping.TargetSubject {
		// Check if the change is significant
		normalizedSource := strings.ToLower(strings.ReplaceAll(mapping.SourceSchemaName, "_", "-"))
		normalizedTarget := strings.ToLower(mapping.TargetSubject)
		
		// Remove -key or -value suffix for comparison
		normalizedTarget = strings.TrimSuffix(normalizedTarget, "-key")
		normalizedTarget = strings.TrimSuffix(normalizedTarget, "-value")

		if normalizedSource != normalizedTarget {
			warnings = append(warnings, models.Warning{
				Schema:  sourceKey,
				Message: "Schema name changed significantly: " + mapping.SourceSchemaName + " → " + mapping.TargetSubject,
			})
		}
	}

	// Warn about special characters that were replaced
	specialChars := []string{"/", ":", " ", "\\"}
	for _, char := range specialChars {
		if strings.Contains(mapping.SourceSchemaName, char) {
			warnings = append(warnings, models.Warning{
				Schema:  sourceKey,
				Message: "Name contains special character '" + char + "' which will be replaced",
			})
			break
		}
	}

	// Warn about AWS-specific prefixes
	awsPrefixes := []string{"MSK_", "Glue_", "AWS_"}
	for _, prefix := range awsPrefixes {
		if strings.HasPrefix(mapping.SourceSchemaName, prefix) {
			warnings = append(warnings, models.Warning{
				Schema:  sourceKey,
				Message: "Name has AWS-specific prefix '" + prefix + "' which may be removed",
			})
			break
		}
	}

	// Warn about version suffixes
	versionPatterns := []string{"_v1", "_v2", "-v1", "-v2", "_V1", "_V2"}
	for _, pattern := range versionPatterns {
		if strings.Contains(mapping.SourceSchemaName, pattern) {
			warnings = append(warnings, models.Warning{
				Schema:  sourceKey,
				Message: "Name contains version suffix which may be removed",
			})
			break
		}
	}

	// Warn about unresolved references
	if len(mapping.References) > 0 && v.config.Migration.ReferenceStrategy != "rewrite" {
		warnings = append(warnings, models.Warning{
			Schema:  sourceKey,
			Message: "Schema has references but reference strategy is not 'rewrite'",
		})
	}

	return warnings
}

// ValidationError represents a validation error
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

package normalizer

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/akrishnanDG/glue-to-ccsr/internal/models"
	"github.com/akrishnanDG/glue-to-ccsr/pkg/config"
)

// Normalizer handles name normalization for schema names
type Normalizer struct {
	config *config.Config
}

// New creates a new Normalizer
func New(cfg *config.Config) *Normalizer {
	return &Normalizer{
		config: cfg,
	}
}

// Normalize normalizes a schema name according to the configuration
func (n *Normalizer) Normalize(name string) (string, []string) {
	var transformations []string
	result := name

	// Step 1: Replace invalid characters
	result, invalidTransforms := n.replaceInvalidChars(result)
	transformations = append(transformations, invalidTransforms...)

	// Step 2: Handle dots
	result, dotTransforms := n.handleDots(result)
	transformations = append(transformations, dotTransforms...)

	// Step 3: Normalize case
	result, caseTransforms := n.normalizeCase(result)
	transformations = append(transformations, caseTransforms...)

	return result, transformations
}

// replaceInvalidChars replaces characters not allowed in Confluent Cloud subjects
func (n *Normalizer) replaceInvalidChars(name string) (string, []string) {
	var transformations []string
	result := name
	replacement := n.config.Normalization.InvalidCharReplacement
	if replacement == "" {
		replacement = "-"
	}

	// Characters that need replacement
	invalidChars := map[rune]bool{
		'/': true,
		':': true, // colon is context separator in CC
		' ': true,
		'\\': true,
		'<': true,
		'>': true,
		'"': true,
		'|': true,
		'?': true,
		'*': true,
	}

	var builder strings.Builder
	for _, r := range result {
		if invalidChars[r] {
			builder.WriteString(replacement)
			if len(transformations) == 0 || !strings.Contains(transformations[len(transformations)-1], "invalid") {
				transformations = append(transformations, string(r)+"→"+replacement)
			}
		} else {
			builder.WriteRune(r)
		}
	}

	return builder.String(), transformations
}

// handleDots handles dots according to the configured strategy
func (n *Normalizer) handleDots(name string) (string, []string) {
	var transformations []string
	
	if !strings.Contains(name, ".") {
		return name, transformations
	}

	switch n.config.Normalization.NormalizeDots {
	case "keep":
		// Keep dots as-is
		return name, transformations

	case "replace":
		// Replace dots with the configured replacement
		replacement := n.config.Normalization.DotReplacement
		if replacement == "" {
			replacement = "-"
		}
		result := strings.ReplaceAll(name, ".", replacement)
		transformations = append(transformations, "dots→"+replacement)
		return result, transformations

	case "extract-last":
		// Use only the last segment
		parts := strings.Split(name, ".")
		result := parts[len(parts)-1]
		transformations = append(transformations, "extract-last-segment")
		return result, transformations

	default:
		// Default to replace
		result := strings.ReplaceAll(name, ".", "-")
		transformations = append(transformations, "dots→-")
		return result, transformations
	}
}

// normalizeCase normalizes the case according to the configured strategy
func (n *Normalizer) normalizeCase(name string) (string, []string) {
	var transformations []string

	switch n.config.Normalization.NormalizeCase {
	case "keep":
		// Keep case as-is
		return name, transformations

	case "kebab":
		// Convert to kebab-case
		result := toKebabCase(name)
		if result != name {
			transformations = append(transformations, "case→kebab")
		}
		return result, transformations

	case "snake":
		// Convert to snake_case
		result := toSnakeCase(name)
		if result != name {
			transformations = append(transformations, "case→snake")
		}
		return result, transformations

	case "lower":
		// Just lowercase
		result := strings.ToLower(name)
		if result != name {
			transformations = append(transformations, "case→lower")
		}
		return result, transformations

	default:
		// Default to kebab-case
		result := toKebabCase(name)
		if result != name {
			transformations = append(transformations, "case→kebab")
		}
		return result, transformations
	}
}

// toKebabCase converts a string to kebab-case
func toKebabCase(s string) string {
	// First, handle transitions between cases and separators
	var result strings.Builder
	var prevWasUpper bool
	var prevWasSeparator bool

	for i, r := range s {
		isUpper := unicode.IsUpper(r)
		isSeparator := r == '_' || r == '-' || r == ' '

		if isSeparator {
			if !prevWasSeparator && result.Len() > 0 {
				result.WriteRune('-')
			}
			prevWasSeparator = true
			prevWasUpper = false
			continue
		}

		if isUpper {
			// Add hyphen before uppercase if:
			// - Not at the start
			// - Previous char wasn't uppercase (camelCase transition)
			// - Previous char wasn't a separator
			if i > 0 && !prevWasUpper && !prevWasSeparator {
				result.WriteRune('-')
			}
		}

		result.WriteRune(unicode.ToLower(r))
		prevWasUpper = isUpper
		prevWasSeparator = false
	}

	// Clean up multiple consecutive hyphens
	kebab := result.String()
	for strings.Contains(kebab, "--") {
		kebab = strings.ReplaceAll(kebab, "--", "-")
	}
	kebab = strings.Trim(kebab, "-")

	return kebab
}

// toSnakeCase converts a string to snake_case
func toSnakeCase(s string) string {
	// Similar to kebab but with underscores
	var result strings.Builder
	var prevWasUpper bool
	var prevWasSeparator bool

	for i, r := range s {
		isUpper := unicode.IsUpper(r)
		isSeparator := r == '_' || r == '-' || r == ' '

		if isSeparator {
			if !prevWasSeparator && result.Len() > 0 {
				result.WriteRune('_')
			}
			prevWasSeparator = true
			prevWasUpper = false
			continue
		}

		if isUpper {
			if i > 0 && !prevWasUpper && !prevWasSeparator {
				result.WriteRune('_')
			}
		}

		result.WriteRune(unicode.ToLower(r))
		prevWasUpper = isUpper
		prevWasSeparator = false
	}

	snake := result.String()
	for strings.Contains(snake, "__") {
		snake = strings.ReplaceAll(snake, "__", "_")
	}
	snake = strings.Trim(snake, "_")

	return snake
}

// DetectCollisions detects naming collisions in the mappings
func (n *Normalizer) DetectCollisions(mappings []*models.SchemaMapping) []models.Collision {
	// Map normalized names to source schemas
	normalizedMap := make(map[string][]string)

	for _, m := range mappings {
		// Format full subject with context (only add prefix if context is not empty)
		fullTarget := m.TargetSubject
		if m.TargetContext != "" {
			fullTarget = m.TargetContext + ":" + m.TargetSubject
		}
		sourceKey := m.SourceRegistry + "." + m.SourceSchemaName
		normalizedMap[fullTarget] = append(normalizedMap[fullTarget], sourceKey)
	}

	// Find collisions (multiple sources mapping to same target)
	var collisions []models.Collision
	for normalized, sources := range normalizedMap {
		if len(sources) > 1 {
			collisions = append(collisions, models.Collision{
				NormalizedName: normalized,
				SourceSchemas:  sources,
			})
		}
	}

	return collisions
}

// ResolveCollisions automatically resolves naming collisions based on configured strategy
func (n *Normalizer) ResolveCollisions(mappings []*models.SchemaMapping) []*models.SchemaMapping {
	strategy := n.config.Normalization.CollisionResolution
	if strategy == "" || strategy == "fail" {
		return mappings // No resolution, let validation fail
	}

	// Build map of target names to mappings
	targetMap := make(map[string][]*models.SchemaMapping)
	for _, m := range mappings {
		fullTarget := m.TargetSubject
		if m.TargetContext != "" {
			fullTarget = m.TargetContext + ":" + m.TargetSubject
		}
		targetMap[fullTarget] = append(targetMap[fullTarget], m)
	}

	// Resolve collisions
	resolved := make([]*models.SchemaMapping, 0, len(mappings))
	for _, mappingList := range targetMap {
		if len(mappingList) == 1 {
			// No collision
			resolved = append(resolved, mappingList[0])
		} else {
			// Collision detected, apply resolution strategy
			resolvedMappings := n.applyResolutionStrategy(mappingList, strategy)
			resolved = append(resolved, resolvedMappings...)
		}
	}

	return resolved
}

func (n *Normalizer) applyResolutionStrategy(colliding []*models.SchemaMapping, strategy string) []*models.SchemaMapping {
	switch strategy {
	case "suffix":
		// Add numeric suffix to all but the first
		for i, m := range colliding {
			if i > 0 {
				m.TargetSubject = m.TargetSubject + "-" + string(rune('0'+i))
				m.Transformations = append(m.Transformations, "collision-suffix")
			}
		}
		return colliding

	case "registry-prefix":
		// Add registry name as prefix to all
		for _, m := range colliding {
			m.TargetSubject = m.SourceRegistry + "-" + m.TargetSubject
			m.Transformations = append(m.Transformations, "registry-prefix")
		}
		return colliding

	case "prefer-shorter":
		// Keep the schema with the shorter original name (likely less nested)
		// Sort by original name length
		shortest := colliding[0]
		for _, m := range colliding {
			if len(m.SourceSchemaName) < len(shortest.SourceSchemaName) {
				shortest = m
			}
		}
		return []*models.SchemaMapping{shortest}

	case "skip":
		// Keep only the first, skip others
		if len(colliding) > 0 {
			return []*models.SchemaMapping{colliding[0]}
		}
		return nil

	default:
		// Unknown strategy, keep all (will fail validation)
		return colliding
	}
}

// StripKeySuffix removes key-related suffixes from a name
func StripKeySuffix(name string) string {
	suffixes := []string{"-key", "_key", "Key", "-k", "_k"}
	for _, suffix := range suffixes {
		if strings.HasSuffix(name, suffix) {
			return strings.TrimSuffix(name, suffix)
		}
	}
	return name
}

// StripValueSuffix removes value-related suffixes from a name
func StripValueSuffix(name string) string {
	suffixes := []string{"-value", "_value", "Value", "-v", "_v"}
	for _, suffix := range suffixes {
		if strings.HasSuffix(name, suffix) {
			return strings.TrimSuffix(name, suffix)
		}
	}
	return name
}

// StripIdSuffix removes ID-related suffixes from a name for subject naming
func StripIdSuffix(name string) string {
	// First check for exact ID patterns we want to strip
	patterns := []string{"Id", "ID", "-id", "_id"}
	for _, pattern := range patterns {
		if strings.HasSuffix(name, pattern) {
			return strings.TrimSuffix(name, pattern)
		}
	}
	return name
}

// CleanForSubject cleans a name for use as a Confluent subject
func CleanForSubject(name string) string {
	// Remove disallowed characters
	re := regexp.MustCompile(`[^a-zA-Z0-9._-]`)
	return re.ReplaceAllString(name, "-")
}

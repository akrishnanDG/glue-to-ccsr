package llm

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/akrishnanDG/glue-to-ccsr/internal/models"
	"github.com/akrishnanDG/glue-to-ccsr/pkg/config"
)

// Namer uses LLM to suggest schema names
type Namer struct {
	config       *config.Config
	provider     Provider
	preprocessor *Preprocessor
	cache        *Cache
	callCount    int
	totalCost    float64
}

// NameSuggestion represents an LLM naming suggestion
type NameSuggestion struct {
	OriginalName   string `json:"original_name"`
	SuggestedName  string `json:"suggested_name"`
	IsKeySchema    bool   `json:"is_key_schema"`
	Reasoning      string `json:"reasoning"`
}

// NewNamer creates a new LLM Namer
func NewNamer(cfg *config.Config) (*Namer, error) {
	// Create provider based on configuration
	provider, err := NewProvider(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM provider: %w", err)
	}

	// Create preprocessor
	preprocessor := NewPreprocessor()

	// Create cache
	var cache *Cache
	if cfg.LLM.CacheFile != "" {
		cache, err = NewCache(cfg.LLM.CacheFile)
		if err != nil {
			// Non-fatal - just log and continue without cache
			slog.Warn("failed to load LLM cache", "error", err)
			cache = NewEmptyCache()
		}
	} else {
		cache = NewEmptyCache()
	}

	return &Namer{
		config:       cfg,
		provider:     provider,
		preprocessor: preprocessor,
		cache:        cache,
	}, nil
}

// SuggestName uses the LLM to suggest a subject name
func (n *Namer) SuggestName(ctx context.Context, schema *models.GlueSchema, parsed *models.ParsedSchema, role models.SchemaRole) (*NameSuggestion, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("%s:%s", schema.RegistryName, schema.Name)
	if cached, ok := n.cache.Get(cacheKey); ok {
		return cached, nil
	}

	// Check cost limit
	if n.config.LLM.MaxCost > 0 && n.totalCost >= n.config.LLM.MaxCost {
		return nil, fmt.Errorf("LLM cost limit reached ($%.2f)", n.config.LLM.MaxCost)
	}

	// Preprocess schema to extract context
	schemaContext := n.preprocessor.ExtractContext(schema, parsed)

	// Build prompt
	prompt := n.buildPrompt(schemaContext, role)

	// Call LLM
	response, cost, err := n.provider.Complete(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	n.callCount++
	n.totalCost += cost

	// Parse response
	suggestion, err := n.parseResponse(response, schema.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	// Cache the result
	n.cache.Set(cacheKey, suggestion)

	// Save cache periodically
	if n.callCount%10 == 0 && n.config.LLM.CacheFile != "" {
		n.cache.Save(n.config.LLM.CacheFile)
	}

	return suggestion, nil
}

// GetCallCount returns the number of LLM calls made
func (n *Namer) GetCallCount() int {
	return n.callCount
}

// GetTotalCost returns the total cost of LLM calls
func (n *Namer) GetTotalCost() float64 {
	return n.totalCost
}

// Close saves the cache and cleans up
func (n *Namer) Close() error {
	if n.config.LLM.CacheFile != "" {
		return n.cache.Save(n.config.LLM.CacheFile)
	}
	return nil
}

func (n *Namer) buildPrompt(schemaContext *SchemaContext, role models.SchemaRole) string {
	return fmt.Sprintf(`You are a Confluent Cloud Schema Registry naming expert.

Given information about an AWS Glue schema, suggest an appropriate Confluent Cloud subject name.

## Confluent Subject Naming Conventions
- Use lowercase with hyphens (kebab-case): "payment-transactions"
- Append "-value" for value schemas, "-key" for key schemas
- Be descriptive but concise
- Avoid environment prefixes (prod, dev, staging)
- Avoid version suffixes (v1, v2)
- Avoid AWS-specific prefixes (MSK_, Glue_, etc.)

## Schema Information
Glue Schema Name: %s
Registry: %s
Schema Type: %s
Record Name: %s
Namespace: %s
Documentation: %s
Key Fields: %v
Field Count: %d
Detected Role: %s

## Instructions
1. Analyze the schema name, record name, namespace, and field names
2. The schema has been detected as a %s schema
3. Suggest a clean, descriptive subject name following Confluent conventions
4. Include the appropriate suffix (-%s)

Respond with ONLY the suggested subject name, nothing else.
Example response: payment-transactions-value`,
		schemaContext.GlueSchemaName,
		schemaContext.GlueRegistry,
		schemaContext.SchemaType,
		schemaContext.RecordName,
		schemaContext.Namespace,
		schemaContext.Documentation,
		schemaContext.KeyFields,
		schemaContext.FieldCount,
		role,
		role,
		role,
	)
}

func (n *Namer) parseResponse(response string, originalName string) (*NameSuggestion, error) {
	// Clean up the response
	suggested := response
	suggested = cleanResponse(suggested)

	if suggested == "" {
		return nil, fmt.Errorf("empty response from LLM")
	}

	return &NameSuggestion{
		OriginalName:  originalName,
		SuggestedName: suggested,
		Reasoning:     "LLM suggestion",
	}, nil
}

func cleanResponse(s string) string {
	// Remove common markdown artifacts
	s = trimPrefix(s, "```")
	s = trimSuffix(s, "```")
	s = trimPrefix(s, "`")
	s = trimSuffix(s, "`")
	
	// Trim whitespace
	return trim(s)
}

func trimPrefix(s, prefix string) string {
	for len(s) > 0 && len(s) >= len(prefix) && s[:len(prefix)] == prefix {
		s = s[len(prefix):]
	}
	return s
}

func trimSuffix(s, suffix string) string {
	for len(s) > 0 && len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix {
		s = s[:len(s)-len(suffix)]
	}
	return s
}

func trim(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\n' || s[start] == '\r' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\n' || s[end-1] == '\r' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}

package config

import (
	"fmt"
	"net/url"
	"regexp"
)

// ValidationError represents a configuration validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidationErrors is a collection of validation errors
type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return ""
	}
	msg := "configuration validation failed:\n"
	for _, err := range e {
		msg += fmt.Sprintf("  - %s\n", err.Error())
	}
	return msg
}

// Validate validates the configuration and returns any errors
func (c *Config) Validate() error {
	var errs ValidationErrors

	// Validate AWS configuration
	if c.AWS.Region == "" {
		errs = append(errs, ValidationError{Field: "aws.region", Message: "region is required"})
	}

	if !c.AWS.RegistryAll && len(c.AWS.RegistryNames) == 0 {
		errs = append(errs, ValidationError{
			Field:   "aws.registry_names",
			Message: "either registry_names or registry_all must be specified",
		})
	}

	// Validate Confluent Cloud configuration (skip for dry-run)
	if !c.Output.DryRun {
		if c.ConfluentCloud.URL == "" {
			errs = append(errs, ValidationError{Field: "confluent_cloud.url", Message: "URL is required"})
		} else {
			if _, err := url.Parse(c.ConfluentCloud.URL); err != nil {
				errs = append(errs, ValidationError{Field: "confluent_cloud.url", Message: "invalid URL format"})
			}
		}

		if c.ConfluentCloud.APIKey == "" {
			errs = append(errs, ValidationError{Field: "confluent_cloud.api_key", Message: "API key is required"})
		}

		if c.ConfluentCloud.APISecret == "" {
			errs = append(errs, ValidationError{Field: "confluent_cloud.api_secret", Message: "API secret is required"})
		}
	}

	// Validate naming strategy
	validSubjectStrategies := map[string]bool{"topic": true, "record": true, "llm": true, "custom": true}
	if !validSubjectStrategies[c.Naming.SubjectStrategy] {
		errs = append(errs, ValidationError{
			Field:   "naming.subject_strategy",
			Message: "must be one of: topic, record, llm, custom",
		})
	}

	if c.Naming.SubjectStrategy == "custom" && c.Naming.SubjectTemplate == "" {
		errs = append(errs, ValidationError{
			Field:   "naming.subject_template",
			Message: "template is required when using custom subject strategy",
		})
	}

	validContextMappings := map[string]bool{"registry": true, "flat": true, "custom": true}
	if !validContextMappings[c.Naming.ContextMapping] {
		errs = append(errs, ValidationError{
			Field:   "naming.context_mapping",
			Message: "must be one of: registry, flat, custom",
		})
	}

	// Validate normalization
	validDotStrategies := map[string]bool{"keep": true, "replace": true, "extract-last": true}
	if !validDotStrategies[c.Normalization.NormalizeDots] {
		errs = append(errs, ValidationError{
			Field:   "normalization.normalize_dots",
			Message: "must be one of: keep, replace, extract-last",
		})
	}

	validCaseStrategies := map[string]bool{"keep": true, "kebab": true, "snake": true, "lower": true}
	if !validCaseStrategies[c.Normalization.NormalizeCase] {
		errs = append(errs, ValidationError{
			Field:   "normalization.normalize_case",
			Message: "must be one of: keep, kebab, snake, lower",
		})
	}

	// Validate key/value regex patterns
	for i, pattern := range c.KeyValue.KeyRegex {
		if _, err := regexp.Compile(pattern); err != nil {
			errs = append(errs, ValidationError{
				Field:   fmt.Sprintf("key_value.key_regex[%d]", i),
				Message: fmt.Sprintf("invalid regex pattern: %v", err),
			})
		}
	}

	for i, pattern := range c.KeyValue.ValueRegex {
		if _, err := regexp.Compile(pattern); err != nil {
			errs = append(errs, ValidationError{
				Field:   fmt.Sprintf("key_value.value_regex[%d]", i),
				Message: fmt.Sprintf("invalid regex pattern: %v", err),
			})
		}
	}

	validRoles := map[string]bool{"key": true, "value": true}
	if !validRoles[c.KeyValue.DefaultRole] {
		errs = append(errs, ValidationError{
			Field:   "key_value.default_role",
			Message: "must be one of: key, value",
		})
	}

	// Validate migration configuration
	validVersionStrategies := map[string]bool{"all": true, "latest": true}
	if !validVersionStrategies[c.Migration.VersionStrategy] {
		errs = append(errs, ValidationError{
			Field:   "migration.version_strategy",
			Message: "must be one of: all, latest",
		})
	}

	validReferenceStrategies := map[string]bool{"rewrite": true, "skip": true, "fail": true}
	if !validReferenceStrategies[c.Migration.ReferenceStrategy] {
		errs = append(errs, ValidationError{
			Field:   "migration.reference_strategy",
			Message: "must be one of: rewrite, skip, fail",
		})
	}

	validCrossRegistryRefs := map[string]bool{"resolve": true, "fail": true, "warn": true}
	if !validCrossRegistryRefs[c.Migration.CrossRegistryRefs] {
		errs = append(errs, ValidationError{
			Field:   "migration.cross_registry_refs",
			Message: "must be one of: resolve, fail, warn",
		})
	}

	// Validate LLM configuration if using LLM strategy
	if c.Naming.SubjectStrategy == "llm" {
		validProviders := map[string]bool{"openai": true, "anthropic": true, "bedrock": true, "ollama": true, "local": true}
		if !validProviders[c.LLM.Provider] {
			errs = append(errs, ValidationError{
				Field:   "llm.provider",
				Message: "must be one of: openai, anthropic, bedrock, ollama, local",
			})
		}

		if c.LLM.Model == "" {
			errs = append(errs, ValidationError{Field: "llm.model", Message: "model is required when using LLM strategy"})
		}

		// API key required for cloud providers
		cloudProviders := map[string]bool{"openai": true, "anthropic": true}
		if cloudProviders[c.LLM.Provider] && c.LLM.APIKey == "" {
			errs = append(errs, ValidationError{
				Field:   "llm.api_key",
				Message: "API key is required for cloud LLM providers",
			})
		}

		// Base URL required for local providers
		localProviders := map[string]bool{"ollama": true, "local": true}
		if localProviders[c.LLM.Provider] && c.LLM.BaseURL == "" {
			errs = append(errs, ValidationError{
				Field:   "llm.base_url",
				Message: "base URL is required for local LLM providers",
			})
		}
	}

	// Validate concurrency configuration
	if c.Concurrency.Workers < 1 {
		errs = append(errs, ValidationError{Field: "concurrency.workers", Message: "must be at least 1"})
	}

	if c.Concurrency.BatchSize < 1 {
		errs = append(errs, ValidationError{Field: "concurrency.batch_size", Message: "must be at least 1"})
	}

	if c.Concurrency.RetryAttempts < 0 {
		errs = append(errs, ValidationError{Field: "concurrency.retry_attempts", Message: "cannot be negative"})
	}

	// Validate output configuration
	validFormats := map[string]bool{"table": true, "json": true, "csv": true}
	if !validFormats[c.Output.Format] {
		errs = append(errs, ValidationError{
			Field:   "output.format",
			Message: "must be one of: table, json, csv",
		})
	}

	validLogLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLogLevels[c.Output.LogLevel] {
		errs = append(errs, ValidationError{
			Field:   "output.log_level",
			Message: "must be one of: debug, info, warn, error",
		})
	}

	if len(errs) > 0 {
		return errs
	}

	return nil
}

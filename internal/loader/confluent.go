package loader

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/akrishnanDG/glue-to-ccsr/internal/models"
	"github.com/akrishnanDG/glue-to-ccsr/pkg/config"
	"golang.org/x/time/rate"
)

// ConfluentLoader loads schemas to Confluent Cloud Schema Registry
type ConfluentLoader struct {
	config      *config.Config
	client      *http.Client
	rateLimiter *rate.Limiter
	baseURL     string
}

// New creates a new ConfluentLoader
func New(cfg *config.Config) (*ConfluentLoader, error) {
	baseURL := strings.TrimSuffix(cfg.ConfluentCloud.URL, "/")

	return &ConfluentLoader{
		config:      cfg,
		client:      &http.Client{Timeout: 30 * time.Second},
		rateLimiter: rate.NewLimiter(rate.Limit(cfg.Concurrency.CCRateLimit), 1),
		baseURL:     baseURL,
	}, nil
}

// RegisterSchema registers a schema version in Confluent Cloud
func (l *ConfluentLoader) RegisterSchema(ctx context.Context, mapping *models.SchemaMapping, version *models.GlueSchemaVersion) error {
	if err := l.rateLimiter.Wait(ctx); err != nil {
		return err
	}

	// Build subject name with context
	subject := mapping.TargetSubject
	if mapping.TargetContext != "" {
		subject = mapping.TargetContext + ":" + subject
	}

	// Prepare the schema registration request
	reqBody := SchemaRegistrationRequest{
		Schema:     version.Definition,
		SchemaType: getSchemaType(mapping),
	}

	// Add references if needed
	if len(mapping.References) > 0 && l.config.Migration.ReferenceStrategy == "rewrite" {
		refs, err := l.buildReferences(mapping.References, mapping.TargetContext)
		if err != nil {
			return fmt.Errorf("failed to build references: %w", err)
		}
		reqBody.References = refs
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Make the API call (URL encode subject name)
	encodedSubject := url.PathEscape(subject)
	apiURL := fmt.Sprintf("%s/subjects/%s/versions", l.baseURL, encodedSubject)
	
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(body))
	if err != nil {
		return err
	}

	l.setHeaders(req)

	resp, err := l.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to register schema: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("schema registration failed for subject '%s': %s (status %d)", subject, string(respBody), resp.StatusCode)
	}

	return nil
}

// SetCompatibility sets the compatibility level for a subject
func (l *ConfluentLoader) SetCompatibility(ctx context.Context, subject string, compatibility string) error {
	if err := l.rateLimiter.Wait(ctx); err != nil {
		return err
	}

	reqBody := map[string]string{
		"compatibility": compatibility,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	encodedSubject := url.PathEscape(subject)
	apiURL := fmt.Sprintf("%s/config/%s", l.baseURL, encodedSubject)
	req, err := http.NewRequestWithContext(ctx, "PUT", apiURL, bytes.NewReader(body))
	if err != nil {
		return err
	}

	l.setHeaders(req)

	resp, err := l.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to set compatibility: %s", string(respBody))
	}

	return nil
}

// GetSubjects returns all existing subjects
func (l *ConfluentLoader) GetSubjects(ctx context.Context) ([]string, error) {
	if err := l.rateLimiter.Wait(ctx); err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/subjects", l.baseURL)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	l.setHeaders(req)

	resp, err := l.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get subjects: %s", string(respBody))
	}

	var subjects []string
	if err := json.Unmarshal(respBody, &subjects); err != nil {
		return nil, err
	}

	return subjects, nil
}

// SubjectExists checks if a subject already exists
func (l *ConfluentLoader) SubjectExists(ctx context.Context, subject string) (bool, error) {
	if err := l.rateLimiter.Wait(ctx); err != nil {
		return false, err
	}

	encodedSubject := url.PathEscape(subject)
	apiURL := fmt.Sprintf("%s/subjects/%s/versions", l.baseURL, encodedSubject)
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return false, err
	}

	l.setHeaders(req)

	resp, err := l.client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}

// SetMetadata sets metadata for a subject
func (l *ConfluentLoader) SetMetadata(ctx context.Context, subject string, metadata *models.SubjectMetadata) error {
	if err := l.rateLimiter.Wait(ctx); err != nil {
		return err
	}

	body, err := json.Marshal(metadata)
	if err != nil {
		return err
	}

	encodedSubject := url.PathEscape(subject)
	apiURL := fmt.Sprintf("%s/subjects/%s/metadata", l.baseURL, encodedSubject)
	req, err := http.NewRequestWithContext(ctx, "PUT", apiURL, bytes.NewReader(body))
	if err != nil {
		return err
	}

	l.setHeaders(req)

	resp, err := l.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Metadata endpoint might not exist in all versions
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to set metadata: %s", string(respBody))
	}

	return nil
}

func (l *ConfluentLoader) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/vnd.schemaregistry.v1+json")
	req.SetBasicAuth(l.config.ConfluentCloud.APIKey, l.config.ConfluentCloud.APISecret)
}

func (l *ConfluentLoader) buildReferences(refs []string, context string) ([]models.SchemaReference, error) {
	var result []models.SchemaReference

	for _, ref := range refs {
		// Parse the reference (format: "registry:schema" or just "schema")
		parts := strings.SplitN(ref, ":", 2)
		var schemaName string
		var refContext string

		if len(parts) == 2 {
			refContext = "." + parts[0]
			schemaName = parts[1]
		} else {
			refContext = context
			schemaName = parts[0]
		}

		// Build the subject name for the reference
		// Assuming value schema for now - could be enhanced to detect key schemas
		subject := schemaName + "-value"
		if refContext != "" {
			subject = refContext + ":" + subject
		}

		result = append(result, models.SchemaReference{
			Name:    schemaName,
			Subject: subject,
			Version: 1, // Reference latest version
		})
	}

	return result, nil
}

func getSchemaType(mapping *models.SchemaMapping) string {
	// Get the schema type from the source schema
	// Default to AVRO if not specified
	return "AVRO"
}

// SchemaRegistrationRequest represents a schema registration request
type SchemaRegistrationRequest struct {
	Schema     string                   `json:"schema"`
	SchemaType string                   `json:"schemaType,omitempty"`
	References []models.SchemaReference `json:"references,omitempty"`
}

package loader

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/akrishnanDG/glue-to-ccsr/internal/models"
	"github.com/akrishnanDG/glue-to-ccsr/pkg/config"
)

// newTestLoader creates a ConfluentLoader pointed at the given test server.
func newTestLoader(t *testing.T, serverURL string) *ConfluentLoader {
	t.Helper()
	cfg := config.NewDefaultConfig()
	cfg.ConfluentCloud.URL = serverURL
	cfg.ConfluentCloud.APIKey = "test-key"
	cfg.ConfluentCloud.APISecret = "test-secret"

	loader, err := New(cfg)
	if err != nil {
		t.Fatalf("New() returned unexpected error: %v", err)
	}
	return loader
}

// ---------------------------------------------------------------------------
// TestRegisterSchema_Success
// ---------------------------------------------------------------------------

func TestRegisterSchema_Success(t *testing.T) {
	var capturedReq struct {
		method      string
		path        string
		contentType string
		authHeader  string
		body        []byte
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedReq.method = r.Method
		capturedReq.path = r.URL.Path
		capturedReq.contentType = r.Header.Get("Content-Type")
		capturedReq.authHeader = r.Header.Get("Authorization")
		capturedReq.body, _ = io.ReadAll(r.Body)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":1}`))
	}))
	defer server.Close()

	loader := newTestLoader(t, server.URL)

	mapping := &models.SchemaMapping{
		TargetSubject: "user-event-value",
	}
	version := &models.GlueSchemaVersion{
		Definition: `{"type":"record","name":"UserEvent","fields":[]}`,
	}

	err := loader.RegisterSchema(context.Background(), mapping, version)
	if err != nil {
		t.Fatalf("RegisterSchema returned unexpected error: %v", err)
	}

	// Verify HTTP method.
	if capturedReq.method != "POST" {
		t.Errorf("method = %q, want POST", capturedReq.method)
	}

	// Verify Content-Type header.
	wantCT := "application/vnd.schemaregistry.v1+json"
	if capturedReq.contentType != wantCT {
		t.Errorf("Content-Type = %q, want %q", capturedReq.contentType, wantCT)
	}

	// Verify Basic auth header is present.
	if capturedReq.authHeader == "" {
		t.Error("Authorization header is empty, expected Basic auth")
	}
	if !strings.HasPrefix(capturedReq.authHeader, "Basic ") {
		t.Errorf("Authorization header = %q, want prefix 'Basic '", capturedReq.authHeader)
	}

	// Verify request path.
	wantPath := "/subjects/user-event-value/versions"
	if capturedReq.path != wantPath {
		t.Errorf("path = %q, want %q", capturedReq.path, wantPath)
	}

	// Verify request body contains the schema definition.
	var reqBody SchemaRegistrationRequest
	if err := json.Unmarshal(capturedReq.body, &reqBody); err != nil {
		t.Fatalf("failed to unmarshal request body: %v", err)
	}
	if reqBody.Schema != version.Definition {
		t.Errorf("body schema = %q, want %q", reqBody.Schema, version.Definition)
	}
	if reqBody.SchemaType != "AVRO" {
		t.Errorf("body schemaType = %q, want %q", reqBody.SchemaType, "AVRO")
	}
}

// ---------------------------------------------------------------------------
// TestRegisterSchema_WithContext
// ---------------------------------------------------------------------------

func TestRegisterSchema_WithContext(t *testing.T) {
	var capturedRawPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Use RequestURI to see the raw, percent-encoded path that the loader sent.
		capturedRawPath = r.RequestURI
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":2}`))
	}))
	defer server.Close()

	loader := newTestLoader(t, server.URL)

	mapping := &models.SchemaMapping{
		TargetContext: ".payments",
		TargetSubject: "user-value",
	}
	version := &models.GlueSchemaVersion{
		Definition: `{"type":"record","name":"User","fields":[]}`,
	}

	err := loader.RegisterSchema(context.Background(), mapping, version)
	if err != nil {
		t.Fatalf("RegisterSchema returned unexpected error: %v", err)
	}

	// The subject ".payments:user-value" is passed through url.PathEscape.
	// Colons are valid in URL path segments, so PathEscape leaves them as-is.
	wantRawPath := "/subjects/.payments:user-value/versions"
	if capturedRawPath != wantRawPath {
		t.Errorf("raw path = %q, want %q", capturedRawPath, wantRawPath)
	}
}

// ---------------------------------------------------------------------------
// TestRegisterSchema_ServerError
// ---------------------------------------------------------------------------

func TestRegisterSchema_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte(`{"error_code":409,"message":"Schema already exists"}`))
	}))
	defer server.Close()

	loader := newTestLoader(t, server.URL)

	mapping := &models.SchemaMapping{
		TargetSubject: "user-event-value",
	}
	version := &models.GlueSchemaVersion{
		Definition: `{"type":"record","name":"UserEvent","fields":[]}`,
	}

	err := loader.RegisterSchema(context.Background(), mapping, version)
	if err == nil {
		t.Fatal("expected error from RegisterSchema on 409, got nil")
	}
	if !strings.Contains(err.Error(), "409") {
		t.Errorf("error = %q, expected it to contain status code 409", err.Error())
	}
}

// ---------------------------------------------------------------------------
// TestGetSubjects_Success
// ---------------------------------------------------------------------------

func TestGetSubjects_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/subjects" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`["sub1","sub2"]`))
	}))
	defer server.Close()

	loader := newTestLoader(t, server.URL)

	subjects, err := loader.GetSubjects(context.Background())
	if err != nil {
		t.Fatalf("GetSubjects returned unexpected error: %v", err)
	}

	if len(subjects) != 2 {
		t.Fatalf("len(subjects) = %d, want 2", len(subjects))
	}
	if subjects[0] != "sub1" {
		t.Errorf("subjects[0] = %q, want %q", subjects[0], "sub1")
	}
	if subjects[1] != "sub2" {
		t.Errorf("subjects[1] = %q, want %q", subjects[1], "sub2")
	}
}

// ---------------------------------------------------------------------------
// TestSubjectExists_True
// ---------------------------------------------------------------------------

func TestSubjectExists_True(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[1,2]`))
	}))
	defer server.Close()

	loader := newTestLoader(t, server.URL)

	exists, err := loader.SubjectExists(context.Background(), "user-event-value")
	if err != nil {
		t.Fatalf("SubjectExists returned unexpected error: %v", err)
	}
	if !exists {
		t.Error("SubjectExists = false, want true")
	}
}

// ---------------------------------------------------------------------------
// TestSubjectExists_False
// ---------------------------------------------------------------------------

func TestSubjectExists_False(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error_code":40401,"message":"Subject not found"}`))
	}))
	defer server.Close()

	loader := newTestLoader(t, server.URL)

	exists, err := loader.SubjectExists(context.Background(), "nonexistent-subject")
	if err != nil {
		t.Fatalf("SubjectExists returned unexpected error: %v", err)
	}
	if exists {
		t.Error("SubjectExists = true, want false")
	}
}

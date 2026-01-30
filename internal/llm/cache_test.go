package llm

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCache_GetSet(t *testing.T) {
	cache := NewEmptyCache()

	// Test Set
	suggestion := &NameSuggestion{
		OriginalName:  "MSK_PaymentEvent",
		SuggestedName: "payment-event-value",
		Reasoning:     "Removed AWS prefix",
	}

	cache.Set("test:schema", suggestion)

	// Test Get
	retrieved, ok := cache.Get("test:schema")
	if !ok {
		t.Error("Expected to find cached entry")
	}

	if retrieved.SuggestedName != suggestion.SuggestedName {
		t.Errorf("Retrieved suggestion = %q, expected %q", retrieved.SuggestedName, suggestion.SuggestedName)
	}

	// Test Get non-existent
	_, ok = cache.Get("non-existent")
	if ok {
		t.Error("Expected not to find non-existent entry")
	}
}

func TestCache_SaveLoad(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "cache-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cachePath := filepath.Join(tmpDir, "cache.json")

	// Create and populate cache
	cache := NewEmptyCache()
	cache.Set("test:schema1", &NameSuggestion{
		OriginalName:  "Schema1",
		SuggestedName: "schema1-value",
	})
	cache.Set("test:schema2", &NameSuggestion{
		OriginalName:  "Schema2",
		SuggestedName: "schema2-value",
	})

	// Save
	if err := cache.Save(cachePath); err != nil {
		t.Fatalf("Failed to save cache: %v", err)
	}

	// Load
	loadedCache, err := NewCache(cachePath)
	if err != nil {
		t.Fatalf("Failed to load cache: %v", err)
	}

	// Verify
	if loadedCache.Len() != 2 {
		t.Errorf("Loaded cache has %d entries, expected 2", loadedCache.Len())
	}

	suggestion, ok := loadedCache.Get("test:schema1")
	if !ok {
		t.Error("Expected to find schema1 in loaded cache")
	}
	if suggestion.SuggestedName != "schema1-value" {
		t.Errorf("Loaded suggestion = %q, expected schema1-value", suggestion.SuggestedName)
	}
}

func TestCache_Clear(t *testing.T) {
	cache := NewEmptyCache()
	cache.Set("test:schema", &NameSuggestion{
		OriginalName:  "Schema",
		SuggestedName: "schema-value",
	})

	if cache.Len() != 1 {
		t.Error("Expected 1 entry before clear")
	}

	cache.Clear()

	if cache.Len() != 0 {
		t.Error("Expected 0 entries after clear")
	}
}

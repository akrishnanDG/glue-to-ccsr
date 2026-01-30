package llm

import (
	"encoding/json"
	"os"
	"sync"
	"time"
)

// Cache stores LLM responses to avoid duplicate calls
type Cache struct {
	mu       sync.RWMutex
	entries  map[string]*CacheEntry
	modified bool
}

// CacheEntry represents a cached LLM response
type CacheEntry struct {
	Suggestion *NameSuggestion `json:"suggestion"`
	Model      string          `json:"model"`
	CachedAt   time.Time       `json:"cached_at"`
}

// CacheFile represents the structure of the cache file
type CacheFile struct {
	Version int                     `json:"version"`
	Entries map[string]*CacheEntry  `json:"entries"`
}

// NewCache loads a cache from a file
func NewCache(path string) (*Cache, error) {
	cache := &Cache{
		entries: make(map[string]*CacheEntry),
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cache, nil
		}
		return nil, err
	}

	var file CacheFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, err
	}

	cache.entries = file.Entries
	if cache.entries == nil {
		cache.entries = make(map[string]*CacheEntry)
	}

	return cache, nil
}

// NewEmptyCache creates an empty cache
func NewEmptyCache() *Cache {
	return &Cache{
		entries: make(map[string]*CacheEntry),
	}
}

// Get retrieves a cached suggestion
func (c *Cache) Get(key string) (*NameSuggestion, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[key]
	if !ok {
		return nil, false
	}

	return entry.Suggestion, true
}

// Set stores a suggestion in the cache
func (c *Cache) Set(key string, suggestion *NameSuggestion) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[key] = &CacheEntry{
		Suggestion: suggestion,
		CachedAt:   time.Now(),
	}
	c.modified = true
}

// Save writes the cache to a file
func (c *Cache) Save(path string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.modified {
		return nil
	}

	file := CacheFile{
		Version: 1,
		Entries: c.entries,
	}

	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// Len returns the number of entries in the cache
func (c *Cache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}

// Clear removes all entries from the cache
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]*CacheEntry)
	c.modified = true
}

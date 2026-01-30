package worker

import (
	"encoding/json"
	"os"
	"sync"

	"github.com/akrishnanDG/glue-to-ccsr/internal/models"
)

// CheckpointManager manages checkpoint state for resumable migrations
type CheckpointManager struct {
	path string
	mu   sync.Mutex
}

// NewCheckpointManager creates a new checkpoint manager
func NewCheckpointManager(path string) *CheckpointManager {
	return &CheckpointManager{
		path: path,
	}
}

// Load loads the checkpoint state from disk
func (c *CheckpointManager) Load() (*models.MigrationState, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, err := os.ReadFile(c.path)
	if err != nil {
		return nil, err
	}

	var state models.MigrationState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}

	// Initialize maps if nil
	if state.CompletedSchemas == nil {
		state.CompletedSchemas = make(map[string]models.CompletedSchema)
	}
	if state.FailedSchemas == nil {
		state.FailedSchemas = make(map[string]models.FailedSchema)
	}

	return &state, nil
}

// Save saves the checkpoint state to disk
func (c *CheckpointManager) Save(state *models.MigrationState) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(c.path, data, 0644)
}

// Exists checks if a checkpoint file exists
func (c *CheckpointManager) Exists() bool {
	_, err := os.Stat(c.path)
	return err == nil
}

// Delete removes the checkpoint file
func (c *CheckpointManager) Delete() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.Exists() {
		return nil
	}
	return os.Remove(c.path)
}

package worker

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/akrishnanDG/glue-to-ccsr/internal/models"
)

func TestCheckpoint_SaveAndLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "checkpoint.json")
	cm := NewCheckpointManager(path)

	state := models.NewMigrationState("abc123")
	state.TotalSchemas = 5
	state.CompletedCount = 2
	state.FailedCount = 1
	state.CompletedSchemas["schema1"] = models.CompletedSchema{
		SourceRegistry: "reg1",
		SourceSchema:   "schema1",
		TargetSubject:  "schema1-value",
		Versions:       3,
		CompletedAt:    time.Now().Truncate(time.Second),
	}
	state.FailedSchemas["schema2"] = models.FailedSchema{
		SourceRegistry: "reg1",
		SourceSchema:   "schema2",
		Error:          "connection timeout",
		Attempts:       2,
		LastAttempt:    time.Now().Truncate(time.Second),
	}

	if err := cm.Save(state); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := cm.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded.ConfigHash != "abc123" {
		t.Errorf("ConfigHash = %q, expected %q", loaded.ConfigHash, "abc123")
	}
	if loaded.TotalSchemas != 5 {
		t.Errorf("TotalSchemas = %d, expected 5", loaded.TotalSchemas)
	}
	if loaded.CompletedCount != 2 {
		t.Errorf("CompletedCount = %d, expected 2", loaded.CompletedCount)
	}
	if loaded.FailedCount != 1 {
		t.Errorf("FailedCount = %d, expected 1", loaded.FailedCount)
	}

	cs, ok := loaded.CompletedSchemas["schema1"]
	if !ok {
		t.Fatal("expected CompletedSchemas to contain schema1")
	}
	if cs.TargetSubject != "schema1-value" {
		t.Errorf("CompletedSchemas[schema1].TargetSubject = %q, expected %q", cs.TargetSubject, "schema1-value")
	}

	fs, ok := loaded.FailedSchemas["schema2"]
	if !ok {
		t.Fatal("expected FailedSchemas to contain schema2")
	}
	if fs.Error != "connection timeout" {
		t.Errorf("FailedSchemas[schema2].Error = %q, expected %q", fs.Error, "connection timeout")
	}
}

func TestCheckpoint_Exists(t *testing.T) {
	path := filepath.Join(t.TempDir(), "checkpoint.json")
	cm := NewCheckpointManager(path)

	if cm.Exists() {
		t.Error("Exists() = true before save, expected false")
	}

	state := models.NewMigrationState("hash")
	if err := cm.Save(state); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	if !cm.Exists() {
		t.Error("Exists() = false after save, expected true")
	}
}

func TestCheckpoint_Delete(t *testing.T) {
	path := filepath.Join(t.TempDir(), "checkpoint.json")
	cm := NewCheckpointManager(path)

	state := models.NewMigrationState("hash")
	if err := cm.Save(state); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	if !cm.Exists() {
		t.Fatal("expected checkpoint to exist after save")
	}

	if err := cm.Delete(); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	if cm.Exists() {
		t.Error("Exists() = true after delete, expected false")
	}
}

func TestCheckpoint_LoadNonExistent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nonexistent.json")
	cm := NewCheckpointManager(path)

	_, err := cm.Load()
	if err == nil {
		t.Fatal("expected error loading non-existent checkpoint")
	}
}

func TestCheckpoint_LoadCorruptJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "checkpoint.json")

	if err := os.WriteFile(path, []byte("{{{not valid json"), 0644); err != nil {
		t.Fatalf("failed to write corrupt file: %v", err)
	}

	cm := NewCheckpointManager(path)

	_, err := cm.Load()
	if err == nil {
		t.Fatal("expected error loading corrupt JSON checkpoint")
	}
}

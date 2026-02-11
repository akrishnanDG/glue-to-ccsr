package worker

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/akrishnanDG/glue-to-ccsr/internal/models"
	"github.com/akrishnanDG/glue-to-ccsr/pkg/config"
)

func newTestConfig() *config.Config {
	cfg := config.NewDefaultConfig()
	cfg.Concurrency.Workers = 2
	cfg.Concurrency.RetryAttempts = 0
	cfg.Concurrency.RetryDelay = time.Millisecond
	return cfg
}

func TestNewPool(t *testing.T) {
	cfg := newTestConfig()
	pool := NewPool(cfg)

	if pool.workers != 2 {
		t.Errorf("workers = %d, expected 2", pool.workers)
	}
	if pool.retryAttempts != 0 {
		t.Errorf("retryAttempts = %d, expected 0", pool.retryAttempts)
	}
	if pool.retryDelay != time.Millisecond {
		t.Errorf("retryDelay = %v, expected %v", pool.retryDelay, time.Millisecond)
	}
}

func TestExecute_AllSuccess(t *testing.T) {
	cfg := newTestConfig()
	pool := NewPool(cfg)

	mappings := []models.SchemaMapping{
		{SourceSchemaName: "schema1"},
		{SourceSchemaName: "schema2"},
		{SourceSchemaName: "schema3"},
	}

	work := func(ctx context.Context, mapping models.SchemaMapping) error {
		return nil
	}

	errs := pool.Execute(context.Background(), mappings, work)

	if len(errs) != 3 {
		t.Fatalf("expected 3 error slots, got %d", len(errs))
	}
	for i, err := range errs {
		if err != nil {
			t.Errorf("errs[%d] = %v, expected nil", i, err)
		}
	}
}

func TestExecute_SomeFailures(t *testing.T) {
	cfg := newTestConfig()
	pool := NewPool(cfg)

	mappings := []models.SchemaMapping{
		{SourceSchemaName: "success1"},
		{SourceSchemaName: "fail1"},
		{SourceSchemaName: "success2"},
		{SourceSchemaName: "fail2"},
	}

	failSet := map[string]bool{
		"fail1": true,
		"fail2": true,
	}

	work := func(ctx context.Context, mapping models.SchemaMapping) error {
		if failSet[mapping.SourceSchemaName] {
			return fmt.Errorf("failed: %s", mapping.SourceSchemaName)
		}
		return nil
	}

	errs := pool.Execute(context.Background(), mappings, work)

	if len(errs) != 4 {
		t.Fatalf("expected 4 error slots, got %d", len(errs))
	}

	// success1 at index 0
	if errs[0] != nil {
		t.Errorf("errs[0] = %v, expected nil for success1", errs[0])
	}
	// fail1 at index 1
	if errs[1] == nil {
		t.Error("errs[1] = nil, expected error for fail1")
	}
	// success2 at index 2
	if errs[2] != nil {
		t.Errorf("errs[2] = %v, expected nil for success2", errs[2])
	}
	// fail2 at index 3
	if errs[3] == nil {
		t.Error("errs[3] = nil, expected error for fail2")
	}
}

func TestExecuteWithProgress_CallbackCalled(t *testing.T) {
	cfg := newTestConfig()
	pool := NewPool(cfg)

	mappings := []models.SchemaMapping{
		{SourceSchemaName: "schema1"},
		{SourceSchemaName: "schema2"},
		{SourceSchemaName: "schema3"},
	}

	work := func(ctx context.Context, mapping models.SchemaMapping) error {
		return nil
	}

	var count int64
	progress := func() {
		atomic.AddInt64(&count, 1)
	}

	pool.ExecuteWithProgress(context.Background(), mappings, work, progress)

	got := atomic.LoadInt64(&count)
	if got != int64(len(mappings)) {
		t.Errorf("progress callback called %d times, expected %d", got, len(mappings))
	}
}

func TestExecuteSequential_Order(t *testing.T) {
	cfg := newTestConfig()
	pool := NewPool(cfg)

	mappings := []models.SchemaMapping{
		{SourceSchemaName: "first"},
		{SourceSchemaName: "second"},
		{SourceSchemaName: "third"},
	}

	var mu sync.Mutex
	var order []string

	work := func(ctx context.Context, mapping models.SchemaMapping) error {
		mu.Lock()
		order = append(order, mapping.SourceSchemaName)
		mu.Unlock()
		return nil
	}

	errs := pool.ExecuteSequential(context.Background(), mappings, work)

	if len(errs) != 3 {
		t.Fatalf("expected 3 error slots, got %d", len(errs))
	}
	for i, err := range errs {
		if err != nil {
			t.Errorf("errs[%d] = %v, expected nil", i, err)
		}
	}

	expected := []string{"first", "second", "third"}
	if len(order) != len(expected) {
		t.Fatalf("order has %d entries, expected %d", len(order), len(expected))
	}
	for i, name := range expected {
		if order[i] != name {
			t.Errorf("order[%d] = %q, expected %q", i, order[i], name)
		}
	}
}

func TestExecute_ContextCancellation(t *testing.T) {
	cfg := newTestConfig()
	pool := NewPool(cfg)

	mappings := []models.SchemaMapping{
		{SourceSchemaName: "test"},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	work := func(ctx context.Context, mapping models.SchemaMapping) error {
		return ctx.Err()
	}

	errs := pool.Execute(ctx, mappings, work)

	if len(errs) != 1 {
		t.Fatalf("expected 1 error slot, got %d", len(errs))
	}
	if errs[0] == nil {
		t.Fatal("expected non-nil error for cancelled context")
	}
	if !errors.Is(errs[0], context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", errs[0])
	}
}

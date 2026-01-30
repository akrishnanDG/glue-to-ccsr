package worker

import (
	"context"
	"sync"
	"time"

	"github.com/akrishnanDG/glue-to-ccsr/internal/models"
	"github.com/akrishnanDG/glue-to-ccsr/pkg/config"
	"golang.org/x/sync/errgroup"
)

// Pool manages a pool of workers for concurrent processing
type Pool struct {
	config        *config.Config
	workers       int
	retryAttempts int
	retryDelay    time.Duration
}

// NewPool creates a new worker pool
func NewPool(cfg *config.Config) *Pool {
	return &Pool{
		config:        cfg,
		workers:       cfg.Concurrency.Workers,
		retryAttempts: cfg.Concurrency.RetryAttempts,
		retryDelay:    cfg.Concurrency.RetryDelay,
	}
}

// WorkFunc is the function type for work items
type WorkFunc func(ctx context.Context, mapping models.SchemaMapping) error

// ProgressCallback is called after each item is processed
type ProgressCallback func()

// Execute executes the work function for all mappings using the worker pool
func (p *Pool) Execute(ctx context.Context, mappings []models.SchemaMapping, work WorkFunc) []error {
	return p.ExecuteWithProgress(ctx, mappings, work, nil)
}

// ExecuteWithProgress executes work with progress callback
func (p *Pool) ExecuteWithProgress(ctx context.Context, mappings []models.SchemaMapping, work WorkFunc, progress ProgressCallback) []error {
	errors := make([]error, len(mappings))
	var mu sync.Mutex

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(p.workers)

	for i, mapping := range mappings {
		i, mapping := i, mapping // capture loop variables
		g.Go(func() error {
			err := p.executeWithRetry(ctx, mapping, work)
			mu.Lock()
			errors[i] = err
			if progress != nil {
				progress()
			}
			mu.Unlock()
			return nil // Don't propagate errors to stop other goroutines
		})
	}

	g.Wait()
	return errors
}

func (p *Pool) executeWithRetry(ctx context.Context, mapping models.SchemaMapping, work WorkFunc) error {
	var lastErr error

	for attempt := 0; attempt <= p.retryAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		err := work(ctx, mapping)
		if err == nil {
			return nil
		}

		lastErr = err

		if attempt < p.retryAttempts {
			// Wait before retry with exponential backoff
			delay := p.retryDelay * time.Duration(1<<attempt)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}
	}

	return lastErr
}

// ExecuteSequential executes work items sequentially (for dependency-ordered items)
func (p *Pool) ExecuteSequential(ctx context.Context, mappings []models.SchemaMapping, work WorkFunc) []error {
	errors := make([]error, len(mappings))

	for i, mapping := range mappings {
		select {
		case <-ctx.Done():
			errors[i] = ctx.Err()
			return errors
		default:
		}

		errors[i] = p.executeWithRetry(ctx, mapping, work)
	}

	return errors
}

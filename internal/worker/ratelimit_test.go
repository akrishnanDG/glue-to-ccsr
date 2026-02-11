package worker

import (
	"context"
	"testing"
)

func TestNewRateLimiters(t *testing.T) {
	rl := NewRateLimiters(10, 20, 5)

	if rl == nil {
		t.Fatal("NewRateLimiters returned nil")
	}
	if rl.AWS == nil {
		t.Error("AWS limiter is nil")
	}
	if rl.CC == nil {
		t.Error("CC limiter is nil")
	}
	if rl.LLM == nil {
		t.Error("LLM limiter is nil")
	}
}

func TestWaitAWS_Success(t *testing.T) {
	rl := NewRateLimiters(100, 100, 100)

	err := rl.WaitAWS(context.Background())
	if err != nil {
		t.Errorf("WaitAWS returned error: %v", err)
	}
}

func TestWaitAWS_ContextCancelled(t *testing.T) {
	rl := NewRateLimiters(100, 100, 100)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := rl.WaitAWS(ctx)
	if err == nil {
		t.Error("expected error from WaitAWS with cancelled context")
	}
}

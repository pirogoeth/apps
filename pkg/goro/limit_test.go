package goro

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"sync/atomic"
	"testing"
	"time"
)

func TestLimit(t *testing.T) {
	ctx := context.Background()

	limit := NewLimitGroup(1)
	limit.Add(func(ctx context.Context) error {
		return nil
	})
	err := limit.Run(ctx)
	if err != nil {
		t.Fail()
	}
}

func TestLimitSingleConcurrent(t *testing.T) {
	ctx := context.Background()

	executing := atomic.Bool{}

	nestedWork := func(_ context.Context) error {
		if executing.Load() {
			return fmt.Errorf("concurrent execution detected")
		}

		executing.Store(true)
		defer executing.Store(false)

		sleepTime, err := rand.Int(rand.Reader, big.NewInt(5))
		if err != nil {
			return fmt.Errorf("could not get random int: %w", err)
		}

		time.Sleep(time.Duration(sleepTime.Int64()) * time.Millisecond)
		return nil
	}

	limit := NewLimitGroup(1)
	limit.Add(func(ctx context.Context) error {
		t.Log("run 1")
		return nestedWork(ctx)
	})
	limit.Add(func(ctx context.Context) error {
		t.Log("run 2")
		return nestedWork(ctx)
	})
	limit.Add(func(ctx context.Context) error {
		t.Log("run 3")
		return nestedWork(ctx)
	})

	if err := limit.Run(ctx); err != nil {
		t.Error(err)
	}
}

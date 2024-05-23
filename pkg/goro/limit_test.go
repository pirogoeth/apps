package goro

import (
	"context"
	"testing"
)

func TestLimitOne(t *testing.T) {
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

func TestLimitOrdered(t *testing.T) {
	ctx := context.Background()

	numbers := make(chan int, 2)

	limit := NewLimitGroup(1)
	limit.Add(func(ctx context.Context) error {
		numbers <- 1
		return nil
	})
	limit.Add(func(ctx context.Context) error {
		numbers <- 2
		return nil
	})

	err := limit.Run(ctx)
	if err != nil {
		t.Fail()
	}

	if <-numbers != 1 && <-numbers != 2 {
		t.Error("numbers were not received in order")
	}
}

package goro

import (
	"context"
	"errors"
	"sync"
)

// RaceType is an enum for the type of race to run
type RaceType int

const (
	// RaceAny will return the first result regardless of success/failure
	RaceAny RaceType = iota
	// RaceSuccess will return the first successful result
	RaceSuccess
)

type result[T any] struct {
	value T
	err   error
}

func (r result[T]) IsError() bool {
	return r.err != nil
}

type wrapper[T any] func(context.Context) (T, error)

type RaceGroup[T any] struct {
	raceType RaceType
	fns      []wrapper[T]
}

// NewRaceGroup creates a new RaceGroup
func NewRaceGroup[T any](raceType RaceType, fns ...wrapper[T]) *RaceGroup[T] {
	return &RaceGroup[T]{
		raceType,
		fns,
	}
}

func (rg *RaceGroup[T]) Add(fn wrapper[T]) {
	rg.fns = append(rg.fns, fn)
}

func (rg *RaceGroup[T]) Race(parentCtx context.Context) (T, error) {
	ctx, cancel := context.WithCancel(parentCtx)

	// Create a channel to receive the first result
	resultChan := make(chan result[T], len(rg.fns))
	defer close(resultChan)

	var wg sync.WaitGroup
	// Deferring the wait keeps us from leaking goroutines but slows the return
	defer wg.Wait()

	for _, fn := range rg.fns {
		wg.Add(1)
		go func(ctx context.Context, fn wrapper[T]) {
			defer wg.Done()
			out, err := fn(ctx)
			resultChan <- result[T]{out, err}
		}(ctx, fn)
	}

	errs := make([]error, 0, len(rg.fns))
	resultsCount := 0

	for resultsCount < len(rg.fns) {
		select {
		case result := <-resultChan:
			resultsCount++

			if rg.raceType == RaceSuccess && result.IsError() {
				errs = append(errs, result.err)
				continue
			}

			cancel()
			return result.value, errors.Join(append([]error{result.err}, errs...)...)
		case <-ctx.Done():
		}
	}

	return *new(T), errors.Join(errs...)
}

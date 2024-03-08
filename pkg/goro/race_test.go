package goro

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestRaceOne(t *testing.T) {
	ctx := context.Background()

	race := NewRaceGroup[string](RaceAny)
	race.Add(func(ctx context.Context) (string, error) {
		return "hello", nil
	})
	outcome, err := race.Race(ctx)

	if outcome != "hello" || err != nil {
		t.Fail()
	}
}

func TestRaceMany(t *testing.T) {
	ctx := context.Background()

	race := NewRaceGroup[string](RaceAny)
	race.Add(func(ctx context.Context) (string, error) {
		time.Sleep(2 * time.Second)
		return "hello", nil
	})
	race.Add(func(ctx context.Context) (string, error) {
		return "world", nil
	})
	race.Add(func(ctx context.Context) (string, error) {
		time.Sleep(1 * time.Second)
		return "I lost the race!", nil
	})
	outcome, err := race.Race(ctx)

	if outcome != "world" || err != nil {
		t.Fail()
	}
}

func TestRaceManyEnsureCancelled(t *testing.T) {
	ctx := context.Background()

	cancelled := make(chan bool, 1)
	race := NewRaceGroup[string](RaceAny)
	race.Add(func(ctx context.Context) (string, error) {
		select {
		case <-ctx.Done():
			cancelled <- true
			return "goodbye", nil
		case <-time.After(10 * time.Second):
			return "hello", nil
		}
	})
	race.Add(func(ctx context.Context) (string, error) {
		return "world", nil
	})
	outcome, err := race.Race(ctx)

	wasCancelled := false

	select {
	case wasCancelled = <-cancelled:
	case <-time.After(5 * time.Second):
		t.Errorf("timeout: all goroutines should have been cancelled")
	}

	if outcome != "world" || err != nil || !wasCancelled {
		t.Fail()
	}
}

func TestRaceManyToFirstSuccess(t *testing.T) {
	ctx := context.Background()

	race := NewRaceGroup[string](RaceSuccess)
	race.Add(func(ctx context.Context) (string, error) {
		return "", fmt.Errorf("hello")
	})
	race.Add(func(ctx context.Context) (string, error) {
		time.Sleep(2 * time.Second)
		return "", fmt.Errorf("world")
	})
	race.Add(func(ctx context.Context) (string, error) {
		time.Sleep(3 * time.Second)
		return "success!", nil
	})
	outcome, errs := race.Race(ctx)

	errLines := strings.Split(errs.Error(), "\n")
	if len(errLines) != 2 {
		t.Errorf("expected 2 errors, got %v", len(errLines))
	}

	if !strings.Contains(errs.Error(), "hello") || !strings.Contains(errs.Error(), "world") {
		t.Errorf("expected 'hello' and 'world' in errors, got %v", errs)
	}

	if outcome != "success!" {
		t.Errorf("expected 'success!', got %v", outcome)
	}
}

func TestRaceAllErrorsReturnsAllErrors(t *testing.T) {
	ctx := context.Background()

	race := NewRaceGroup[string](RaceSuccess)
	race.Add(func(ctx context.Context) (string, error) {
		return "", fmt.Errorf("hello")
	})
	race.Add(func(ctx context.Context) (string, error) {
		return "", fmt.Errorf("world")
	})
	race.Add(func(ctx context.Context) (string, error) {
		return "", fmt.Errorf("goodbye")
	})
	_, err := race.Race(
		ctx,
	)

	if err == nil {
		t.Fail()
	}
}

func TestRaceSuccessErrorsOccurFirst(t *testing.T) {
	ctx := context.Background()

	race := NewRaceGroup[string](RaceSuccess)
	race.Add(func(ctx context.Context) (string, error) {
		return "", fmt.Errorf("hello")
	})
	race.Add(func(ctx context.Context) (string, error) {
		time.Sleep(2 * time.Second)
		return "", fmt.Errorf("world")
	})
	race.Add(func(ctx context.Context) (string, error) {
		time.Sleep(1 * time.Second)
		return "success!", nil
	})
	outcome, err := race.Race(ctx)

	if err == nil {
		t.Errorf("expected errors to be returned, not nil")
	}

	if outcome != "success!" {
		t.Errorf("expected 'success!', got %v", outcome)
	}
}

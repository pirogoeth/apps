package ctxtools

import (
	"context"
	"testing"
	"time"
)

func TestBoundedBy(t *testing.T) {
	t.Run("child context is cancelled when upper bound context is cancelled", func(t *testing.T) {
		parentCtx := context.Background()
		upperBoundCtx, upperBoundCancel := context.WithCancel(context.Background())
		defer upperBoundCancel()

		childCtx := BoundedBy(parentCtx, upperBoundCtx)

		// Cancel the upper bound context
		upperBoundCancel()

		// Wait a small amount of time for cancellation to propagate
		time.Sleep(10 * time.Millisecond)

		select {
		case <-childCtx.Done():
			// Expected - child context should be cancelled
		default:
			t.Error("child context should have been cancelled when upper bound context was cancelled")
		}
	})

	t.Run("child context is cancelled when parent context is cancelled", func(t *testing.T) {
		parentCtx, parentCancel := context.WithCancel(context.Background())
		upperBoundCtx := context.Background()

		childCtx := BoundedBy(parentCtx, upperBoundCtx)

		// Cancel the parent context
		parentCancel()

		// Wait a small amount of time for cancellation to propagate
		time.Sleep(10 * time.Millisecond)

		select {
		case <-childCtx.Done():
			// Expected - child context should be cancelled because it inherits from parent
		default:
			t.Error("child context should have been cancelled when parent context was cancelled")
		}

		// Verify that cancelling one parent context does not affect other contexts created with the same upper bound
		newParentCtx := context.Background()
		newChildCtx := BoundedBy(newParentCtx, upperBoundCtx)

		select {
		case <-newChildCtx.Done():
			t.Error("new child context should not have been cancelled")
		default:
			// Expected - new child context should still be active
		}
	})

	t.Run("child context is not cancelled if neither parent nor upper bound context are cancelled", func(t *testing.T) {
		parentCtx := context.Background()
		upperBoundCtx := context.Background()

		childCtx := BoundedBy(parentCtx, upperBoundCtx)

		// Wait a small amount of time to ensure no unexpected cancellations
		time.Sleep(10 * time.Millisecond)

		select {
		case <-childCtx.Done():
			t.Error("child context should not have been cancelled")
		default:
			// Expected - child context should still be active
		}
	})

	t.Run("child context inherits deadline from parent", func(t *testing.T) {
		parentCtx, parentCancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer parentCancel()
		upperBoundCtx := context.Background()

		childCtx := BoundedBy(parentCtx, upperBoundCtx)

		// Wait for parent deadline to expire
		time.Sleep(60 * time.Millisecond)

		select {
		case <-childCtx.Done():
			// Expected - child context should be cancelled due to parent deadline
		default:
			t.Error("child context should have been cancelled when parent deadline expired")
		}
	})

	t.Run("child context is cancelled immediately if upper bound is already cancelled", func(t *testing.T) {
		parentCtx := context.Background()
		upperBoundCtx, upperBoundCancel := context.WithCancel(context.Background())
		upperBoundCancel() // Cancel before creating child

		childCtx := BoundedBy(parentCtx, upperBoundCtx)

		select {
		case <-childCtx.Done():
			// Expected - child context should be cancelled immediately
		default:
			t.Error("child context should have been cancelled immediately")
		}
	})
}

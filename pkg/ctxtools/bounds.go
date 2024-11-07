package ctxtools

import "context"

// BoundedBy creates a copy of `parentCtx` that is effectively time-bounded by
// the context `upperBoundsCtx` and returns the copy.
func BoundedBy(parentCtx, upperBoundCtx context.Context) context.Context {
	childCtx, childCancel := context.WithCancel(parentCtx)
	go func() {
		select {
		case <-childCtx.Done():
			return
		case <-upperBoundCtx.Done():
			childCancel()
		}
	}()

	return childCtx
}

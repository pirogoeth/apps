package goro

import (
	"context"
	"errors"
	"sync"
)

type LimitGroup struct {
	sema *SlottedSemaphore
	fns  []errWrapper
	lock sync.Mutex
}

func NewLimitGroup(limit int) *LimitGroup {
	return &LimitGroup{
		sema: NewSlottedSemaphore(limit),
		fns:  make([]errWrapper, 0),
		lock: sync.Mutex{},
	}
}

func (lg *LimitGroup) Add(fn errWrapper) error {
	if ok := lg.lock.TryLock(); !ok {
		return errors.New("cannot add to a LimitGroup while running")
	}

	lg.fns = append(lg.fns, fn)
	lg.lock.Unlock()

	return nil
}

func (lg *LimitGroup) Run(parentCtx context.Context) error {
	lg.lock.Lock()

	ctx, cancel := context.WithCancel(parentCtx)
	errCh := make(chan error, len(lg.fns))
	wg := sync.WaitGroup{}

	for _, fn := range lg.fns {
		wg.Add(1)
		go func(fn errWrapper) {
			slot := lg.sema.AcquireBlocking()
			defer slot.Release()

			errCh <- fn(ctx)
			wg.Done()
		}(fn)
	}
	wg.Wait()
	close(errCh)

	errs := make([]error, 0)
	for err := range errCh {
		errs = append(errs, err)
	}

	cancel()
	return errors.Join(errs...)
}

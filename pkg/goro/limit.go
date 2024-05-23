package goro

import (
	"context"
	"errors"
	"sync"
)

type LimitGroup struct {
	sema chan struct{}
	fns  []errWrapper
	lock sync.Mutex
}

func NewLimitGroup(limit int) *LimitGroup {
	return &LimitGroup{
		sema: make(chan struct{}, limit),
		fns:  make([]errWrapper, 0),
		lock: sync.Mutex{},
	}
}

func (lg *LimitGroup) Add(fn errWrapper) error {
	if ok := lg.lock.TryLock(); !ok {
		return errors.New("cannot add to a LimitGroup while running")
	}
	defer lg.lock.Unlock()

	lg.fns = append(lg.fns, fn)

	return nil
}

func (lg *LimitGroup) Run(parentCtx context.Context) error {
	lg.lock.Lock()

	ctx, cancel := context.WithCancel(parentCtx)
	errCh := make(chan error, len(lg.fns))
	wg := sync.WaitGroup{}

	for _, fn := range lg.fns {
		lg.sema <- struct{}{}
		wg.Add(1)
		go func(fn errWrapper) {
			defer func() {
				<-lg.sema
			}()
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

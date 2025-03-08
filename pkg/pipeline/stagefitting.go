package pipeline

import (
	"sync/atomic"
)

func NewStageFitting[T any](inlet <-chan T, outlet chan<- T) *StageFitting[T] {
	return &StageFitting[T]{
		inlet:  inlet,
		outlet: outlet,
		done:   new(atomic.Bool),
	}
}

type StageFitting[T any] struct {
	inlet  <-chan T
	outlet chan<- T
	done   *atomic.Bool
}

func (psf *StageFitting[T]) Close() error {
	close(psf.outlet)
	return nil
}

func (psf *StageFitting[T]) Done() bool {
	return psf.done.Load()
}

func (psf *StageFitting[T]) finish() {
	psf.done.Store(true)
}

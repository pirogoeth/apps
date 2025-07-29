package pipeline

import (
	"sync/atomic"
)

func NewStageFitting[Data any](inlet <-chan Data, outlet chan<- Data) *StageFitting[Data] {
	return &StageFitting[Data]{
		inlet:  inlet,
		outlet: outlet,
		done:   new(atomic.Bool),
	}
}

type StageFitting[Data any] struct {
	inlet  <-chan Data
	outlet chan<- Data
	done   *atomic.Bool
}

func (psf *StageFitting[Data]) Close() error {
	close(psf.outlet)
	return nil
}

func (psf *StageFitting[Data]) Done() bool {
	return psf.done.Load()
}

func (psf *StageFitting[Data]) Finish() {
	psf.done.Store(true)
}

func (psf *StageFitting[Data]) Write(items ...Data) {
	for _, item := range items {
		psf.outlet <- item
	}
}

func (psf *StageFitting[Data]) Read() Data {
	return <-psf.inlet
}

func (psf *StageFitting[Data]) Inlet() <-chan Data {
	return psf.inlet
}

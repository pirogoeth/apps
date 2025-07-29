package pipeline

import (
	"context"
	"fmt"
)

type (
	Inlet[T any]  = <-chan T
	Outlet[T any] = chan<- T
)

type StageInit[Deps any, Data any] func(deps Deps, inlet Inlet[Data], outlet Outlet[Data]) Stage

type Stage interface {
	Run(context.Context) error
	Done() bool
	Close() error
}

type Stages []Stage

func (ps Stages) Done() bool {
	for _, stage := range ps {
		if !stage.Done() {
			return false
		}
	}

	return true
}

func (ps Stages) Drain() error {
	return fmt.Errorf("not implemented")
}

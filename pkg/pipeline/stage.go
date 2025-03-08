package pipeline

import (
	"context"
	"fmt"
)

type StageInit[Cfg any, T any] func(cfg Cfg, inlet <-chan T, outlet chan<- T) Stage

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

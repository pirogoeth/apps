package pipeline

import (
	"context"
	"time"

	"github.com/pirogoeth/apps/pkg/errors"
	"github.com/sirupsen/logrus"
)

type Pipeline[T any] struct {
	head   chan<- T
	tail   <-chan T
	stages Stages
}

func (p *Pipeline[T]) Close() error {
	merr := new(errors.MultiError)
	for _, stage := range p.stages {
		if err := stage.Close(); err != nil {
			merr.Add(err)
		}
	}

	return merr
}

func (p *Pipeline[T]) Run(ctx context.Context) error {
	childCtx, childCancel := context.WithCancel(context.Background())
	defer childCancel()

	merr := new(errors.MultiError)

	for _, stage := range p.stages {
		go func() {
			merr.Add(stage.Run(childCtx))
		}()
	}

	for !p.stages.Done() {
		select {
		case completeItem := <-p.tail:
			logrus.Debugf("Received an item out of the pipeline: %#v", completeItem)
		case <-ctx.Done():
			err := p.stages.Drain()
			if err != nil {
				return err
			}

			return nil
		case <-time.After(1 * time.Second):
			continue
		}
	}

	childCancel()
	return nil
}

func NewPipeline[Cfg any, T any](cfg Cfg, stageFns ...StageInit[Cfg, T]) *Pipeline[T] {
	// TODO: are unbuffered channels the right move here?
	stages := make(Stages, 0)
	head := make(chan T)
	source := head
	for _, initFn := range stageFns {
		sink := make(chan T)
		stage := initFn(cfg, source, sink)
		stages = append(stages, stage)
		// We want the sink of the current iteration to be the source of the next iteration, so store sink in source
		source = sink
	}
	tail := source
	// for each pipeline stage, we need to create create a new inlet-outlet channel pair
	return &Pipeline[T]{
		stages: stages,
		head:   head,
		tail:   tail,
	}
}

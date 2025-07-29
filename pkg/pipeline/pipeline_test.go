package pipeline

import (
	"context"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pirogoeth/apps/pkg/logging"
	"github.com/sirupsen/logrus"
)

func TestMain(m *testing.M) {
	logging.Setup()
	logrus.SetFormatter(new(logrus.TextFormatter))
	os.Exit(m.Run())
}

type Deps struct{}

var _ Stage = (*Generator)(nil)

func InitGenerator(deps *Deps, inlet <-chan int, outlet chan<- int) Stage {
	// The initial stage will not use the inlet channel, only generate data to pass to further stages
	return &Generator{
		inlet:  nil,
		outlet: outlet,
		done:   new(atomic.Bool),
	}
}

type Generator struct {
	inlet  <-chan int
	outlet chan<- int
	done   *atomic.Bool
}

func (g *Generator) Done() bool {
	return g.done.Load()
}

func (g *Generator) Close() error {
	close(g.outlet)
	return nil
}

func (g *Generator) Run(ctx context.Context) error {
	outputs := []int{10, 9, 8, 7, 6, 5, 4, 3, 2, 1}
	for !g.done.Load() {
		select {
		case <-time.After(10 * time.Millisecond):
			if len(outputs) > 0 {
				output := outputs[0]
				g.outlet <- output
				outputs = outputs[1:]
			} else {
				g.done.Store(true)
			}
		case <-ctx.Done():
			g.done.Store(true)
		}
	}

	return g.Close()
}

var _ Stage = (*Printer)(nil)

func InitPrinter(deps *Deps, inlet <-chan int, outlet chan<- int) Stage {
	// The initial stage will not use the inlet channel, only generate data to pass to further stages
	return &Printer{
		inlet:  inlet,
		outlet: outlet,
		done:   new(atomic.Bool),
	}
}

type Printer struct {
	inlet  <-chan int
	outlet chan<- int
	done   *atomic.Bool
}

func (p *Printer) Done() bool {
	return p.done.Load()
}

func (p *Printer) Close() error {
	close(p.outlet)
	return nil
}

func (p *Printer) Run(ctx context.Context) error {
	for !p.done.Load() {
		select {
		case value, ok := <-p.inlet:
			if !ok {
				logrus.Debugf("input channel closed, finishing")
				p.done.Store(true)
				break
			}
			logrus.Printf("%d\n", value)
		case <-ctx.Done():
			p.done.Store(true)
		}
	}

	return p.Close()
}

func TestPipelineRunWithManualStages(t *testing.T) {
	var deps *Deps = nil
	p := NewPipeline(deps, InitGenerator, InitPrinter)
	p.Run(context.Background())
}

var _ Stage = (*Frobnicator)(nil)

type Frobnicator struct {
	*StageFitting[string]
}

func (f *Frobnicator) Run(ctx context.Context) error {
	logrus.Debugf("let's frobnicate this thing!")
	f.outlet <- "frobnify the flimflam"
	logrus.Debugf("you have now been frobnicated")
	f.Finish()
	return f.Close()
}

func InitFrobnicator(deps *Deps, inlet <-chan string, outlet chan<- string) Stage {
	return &Frobnicator{
		StageFitting: NewStageFitting(inlet, outlet),
	}
}

func TestPipelineRunWithStageFittings(t *testing.T) {
	p := NewPipeline(nil, InitFrobnicator)
	p.Run(context.Background())
}

var _ Stage = (*Bazinator)(nil)

type Bazinator struct {
	*StageFitting[string]
}

func (b *Bazinator) Run(ctx context.Context) error {
	b.Finish()
	return b.Close()
}

type (
	BazinatorData   = string
	BazinatorInlet  = <-chan string
	BazinatorOutlet = chan<- string
)

func InitBazinator(deps *Deps, inlet BazinatorInlet, outlet BazinatorOutlet) Stage {
	return &Bazinator{
		StageFitting: NewStageFitting(inlet, outlet),
	}
}

func TestPipelineRunWithAliasTypes(t *testing.T) {
	p := NewPipeline(nil, InitBazinator)
	p.Run(context.Background())
}

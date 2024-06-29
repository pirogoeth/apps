package system

import (
	"context"
	"os"
	"os/signal"

	"github.com/sirupsen/logrus"
)

type SignalWaiter struct {
	signals        []os.Signal
	onBeforeCancel func(context.Context) error
}

func NewSignalWaiter(signals ...os.Signal) *SignalWaiter {
	if len(signals) == 0 {
		signals = []os.Signal{os.Interrupt}
	}

	return &SignalWaiter{
		signals: signals,
		onBeforeCancel: func(context.Context) error {
			return nil
		},
	}
}

func (sw *SignalWaiter) OnBeforeCancel(fn func(context.Context) error) {
	sw.onBeforeCancel = fn
}

func (sw *SignalWaiter) Wait(ctx context.Context, canceller context.CancelFunc) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, sw.signals...)

	for {
		select {
		case <-sigCh:
			logrus.Infof("Received signal, shutting down...")
			if err := sw.onBeforeCancel(ctx); err != nil {
				logrus.Errorf("error during onBeforeCancel hook: %s", err.Error())
			}
			logrus.Debugf("Cancelling context...")
			canceller()
		case <-ctx.Done():
			logrus.Infof("Sweet dreams!")
			return
		}
	}
}

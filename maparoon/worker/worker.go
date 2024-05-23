package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/projectdiscovery/goflags"
	naabuResult "github.com/projectdiscovery/naabu/v2/pkg/result"
	naabuRunner "github.com/projectdiscovery/naabu/v2/pkg/runner"
	"github.com/sirupsen/logrus"

	"github.com/pirogoeth/apps/maparoon/client"
	"github.com/pirogoeth/apps/maparoon/database"
	"github.com/pirogoeth/apps/maparoon/types"
	"github.com/pirogoeth/apps/pkg/goro"
)

type worker struct {
	apiClient *client.Client
	cfg       *types.Config
}

func New(apiClient *client.Client, cfg *types.Config) *worker {
	return &worker{
		apiClient: apiClient,
		cfg:       cfg,
	}
}

func (w *worker) Run(ctx context.Context) {
	scanInterval := 5 * time.Second
	for {
		select {
		case <-time.After(scanInterval):
			err := w.doScan(ctx)
			if err != nil {
				logrus.Errorf("encountered error during scan: %s", err.Error())
			}

			scanInterval = w.cfg.Worker.ScanInterval
			logrus.Debugf("Setting next scan interval to %s", scanInterval)
		case <-ctx.Done():
			return
		}
	}
}

func (w *worker) doScan(ctx context.Context) error {
	networks, err := w.apiClient.ListNetworks(ctx)
	if err != nil {
		return err
	}

	lg := goro.NewLimitGroup(w.cfg.Worker.ConcurrentScanLimit)
	for _, network := range networks.Networks {
		// Pop off a goroutine to scan each network, bounded by the worker's concurrent scan limit
		lg.Add(func(ctx context.Context) error {
			return w.doNetworkScan(ctx, network)
		})
	}

	if err = lg.Run(ctx); err != nil {
		return err
	}

	return nil
}

func (w *worker) doNetworkScan(ctx context.Context, network database.Network) error {
	logrus.Debugf("Scanning network %s", network.Name)

	options := naabuRunner.Options{
		Host:       goflags.StringSlice{fmt.Sprintf("%s/%d", network.Address, network.Cidr)},
		ScanType:   "",
		Silent:     true,
		Ping:       true,
		ReversePTR: true,
		Output:     "/dev/null",
		OnResult: func(res *naabuResult.HostResult) {
			logrus.Debugf("Found host %s", res.IP)
			if err := w.receiveHostResult(ctx, network, res); err != nil {
				logrus.Errorf("failed to receive host result: %s", err)
			}
		},
	}

	runner, err := naabuRunner.NewRunner(&options)
	if err != nil {
		return err
	}
	defer runner.Close()

	return runner.RunEnumeration(ctx)
}

func (w *worker) receiveHostResult(ctx context.Context, network database.Network, host *naabuResult.HostResult) error {
	return nil
}

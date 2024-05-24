package worker

import (
	"context"
	"encoding/json"
	"errors"
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

	logrus.Debugf("Found %d networks, only scanning %d concurrently",
		len(networks.Networks),
		w.cfg.Worker.ConcurrentScanLimit,
	)

	lg := goro.NewLimitGroup(w.cfg.Worker.ConcurrentScanLimit)
	for _, network := range networks.Networks {
		lg.Add(func(ctx context.Context) error {
			return w.doNetworkScan(ctx, network)
		})
	}

	// logrus.Debugf("Launching network scanners")

	if err = lg.Run(ctx); err != nil {
		return err
	}

	return nil
}

func (w *worker) doNetworkScan(ctx context.Context, network database.Network) error {
	logrus.Debugf("Scanning network %s", network.Name)

	options := naabuRunner.Options{
		Host:             goflags.StringSlice{fmt.Sprintf("%s/%d", network.Address, network.Cidr)},
		ScanType:         "sc",
		Silent:           true,
		Ping:             true,
		ReversePTR:       true,
		JSON:             true,
		Stream:           true,
		ServiceDiscovery: true,
		Nmap:             true,
		Output:           "/dev/null",
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

func (w *worker) receiveHostResult(ctx context.Context, network database.Network, scanResult *naabuResult.HostResult) error {
	var hostAddress string

	resp, err := w.apiClient.CreateHost(ctx, &database.CreateHostParams{
		Address:   scanResult.IP,
		NetworkID: network.ID,
		Comments:  "",
		Attributes: mustJsonify(attributes{
			"hostname": scanResult.Host,
		}),
	})
	if err != nil {
		if !errors.Is(err, client.ErrAlreadyExists) {
			return err
		}

		hostAddress = scanResult.IP
	} else {
		logrus.Debugf("Created host %#v", resp.Hosts[0])
		hostAddress = resp.Hosts[0].Address
	}

	for _, port := range scanResult.Ports {
		resp, err := w.apiClient.CreateHostPort(ctx, &database.CreateHostPortParams{
			Address:    hostAddress,
			Port:       int64(port.Port),
			Protocol:   port.Protocol.String(),
			Attributes: `{}`,
		})
		if err != nil {
			if !errors.Is(err, client.ErrAlreadyExists) {
				return err
			}
		} else {
			logrus.Debugf("Created hostport %#v", resp.HostPorts[0])
		}
	}

	return nil
}

type attributes map[string]string

func mustJsonify(attrs attributes) string {
	b, err := json.Marshal(attrs)
	if err != nil {
		panic(err)
	}
	return string(b)
}

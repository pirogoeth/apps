package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/projectdiscovery/goflags"
	naabuResult "github.com/projectdiscovery/naabu/v2/pkg/result"
	naabuRunner "github.com/projectdiscovery/naabu/v2/pkg/runner"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sys/unix"

	"github.com/pirogoeth/apps/maparoon/client"
	"github.com/pirogoeth/apps/maparoon/database"
	"github.com/pirogoeth/apps/maparoon/types"
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
			err := w.findAndScanNetworks(ctx)
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

func (w *worker) findAndScanNetworks(ctx context.Context) error {
	networks, err := w.apiClient.ListNetworks(ctx)
	if err != nil {
		return err
	}

	logrus.Debugf("Found %d networks, only scanning %d concurrently",
		len(networks.Networks),
		w.cfg.Worker.ConcurrentNetworkScanLimit,
	)

	eg, _ := errgroup.WithContext(ctx)
	eg.SetLimit(w.cfg.Worker.ConcurrentNetworkScanLimit)
	for _, network := range networks.Networks {
		network := network
		eg.Go(func() error {
			return w.startNetworkScanSingle(ctx, network)
		})
	}

	if err = eg.Wait(); err != nil {
		return err
	}

	return nil
}

func (w *worker) startNetworkScanSingle(ctx context.Context, network database.Network) error {
	logrus.Debugf("Scanning network %s", network.Name)
	defer logrus.Debugf("Finished scanning network %s", network.Name)

	scansDir := path.Join(os.TempDir(), "maparoon")
	if err := os.MkdirAll(scansDir, 0750); err != nil {
		return fmt.Errorf("could not create maparoon temp directory: %w", err)
	}

	scanPipe := path.Join(scansDir, fmt.Sprintf("network-%d.sock", network.ID))
	if err := os.Remove(scanPipe); err != nil {
		logrus.Infof("could not remove existing fifo: %s", err)
	}

	if err := unix.Mkfifo(scanPipe, 0666); err != nil {
		return fmt.Errorf("could not create fifo for nmap scan: %w", err)
	}

	scanDoneCh := make(chan bool)
	procEg, _ := errgroup.WithContext(ctx)
	procEg.Go(func() error {
		return w.processScanResults(ctx, network, scanPipe, scanDoneCh)
	})

	options := naabuRunner.Options{
		Host:       goflags.StringSlice{fmt.Sprintf("%s/%d", network.Address, network.Cidr)},
		ScanType:   "c",
		Silent:     true,
		Ping:       true,
		ReversePTR: true,
		JSON:       true,
		Stream:     false,
		Nmap:       true,
		NmapCLI:    fmt.Sprintf("nmap -oX %s -A -O -sV -v0 --noninteractive", scanPipe),
		Output:     "/dev/null",
		OnResult: func(res *naabuResult.HostResult) {
			logrus.Debugf("Found host %s", res.IP)
			if err := w.saveDiscoveredHost(ctx, network, res); err != nil {
				logrus.Errorf("failed to receive host result: %s", err)
			}
		},
	}

	runner, err := naabuRunner.NewRunner(&options)
	if err != nil {
		return err
	}
	defer runner.Close()

	err = runner.RunEnumeration(ctx)
	if err != nil {
		logrus.Errorf("error while running network enumeration: %s", err)
		return err
	}

	scanDoneCh <- true

	if err = procEg.Wait(); err != nil {
		logrus.Errorf("error while processing scan results: %s", err)
		return err
	}

	return nil
}

func (w *worker) saveDiscoveredHost(ctx context.Context, network database.Network, scanResult *naabuResult.HostResult) error {
	var hostAddress string

	resp, err := w.apiClient.CreateHost(ctx, &database.CreateHostParams{
		Address:   scanResult.IP,
		NetworkID: network.ID,
		Comments:  "",
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
			Address:  hostAddress,
			Port:     int64(port.Port),
			Protocol: port.Protocol.String(),
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

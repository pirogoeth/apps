package worker

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"os"
	"time"

	"github.com/adifire/go-nmap"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"

	"github.com/pirogoeth/apps/maparoon/database"
	"github.com/pirogoeth/apps/maparoon/types"
)

func (w *worker) processScanResults(ctx context.Context, network database.Network, scanPipe string, scanDoneCh chan bool) error {
	defer logrus.Debugf("Finished processing nmap results from %s", scanPipe)

	fd, err := unix.Open(
		scanPipe,
		unix.O_RDONLY|unix.O_EXCL|unix.O_NONBLOCK,
		0660,
	)
	if err != nil {
		return fmt.Errorf("could not open fifo for reading: %w", err)
	}

	f := os.NewFile(uintptr(fd), scanPipe)
	defer f.Close()

	stillReading := true
	contentsBuf := new(bytes.Buffer)
	for stillReading {
		if _, err := contentsBuf.ReadFrom(f); err != nil {
			return fmt.Errorf("could not read from nmap result fifo: %w", err)
		}

		if totalBytes := contentsBuf.Len(); totalBytes > 0 {
			logrus.Debugf("received %d total bytes from nmap result pipe", totalBytes)
		}

		select {
		case <-scanDoneCh:
			stillReading = false
		case <-ctx.Done():
			return fmt.Errorf("context done while reading from nmap result pipe")
		case <-time.After(1 * time.Second):
			continue
		}
	}

	return w.parseNmapResults(ctx, network, contentsBuf)
}

func (w *worker) parseNmapResults(ctx context.Context, network database.Network, resultsBuf *bytes.Buffer) error {
	defer logrus.Debugf("Finished parsing nmap results for network %s", network.Name)

	decoder := xml.NewDecoder(resultsBuf)
	for {
		curTok, _ := decoder.Token()
		if curTok == nil {
			break
		}

		switch eleType := curTok.(type) {
		case xml.StartElement:
			if eleType.Name.Local == "nmaprun" {
				var nmaprun nmap.NmapRun
				if err := decoder.DecodeElement(&nmaprun, &eleType); err != nil {
					logrus.Errorf("could not decode nmaprun element: %s", err)
					return err
				}

				if len(nmaprun.Hosts) > 0 {
					logrus.Debugf("Decoded nmaprun with %d hosts", len(nmaprun.Hosts))
					return w.indexHostScans(ctx, network, nmaprun)
				}

				return nil
			}
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("context done while parsing nmap results")
		case <-time.After(1 * time.Second):
			continue
		}
	}

	return nil
}

func (w *worker) indexHostScans(ctx context.Context, network database.Network, run nmap.NmapRun) error {
	hostScans := make([]types.HostScan, 0, len(run.Hosts))
	for _, host := range run.Hosts {
		hostScans = append(hostScans, types.HostScan{
			Address:     host.Addresses[0].Addr,
			ScanDetails: host,
		})
	}

	resp, err := w.apiClient.CreateHostScans(ctx, types.CreateHostScansRequest{
		HostScans: hostScans,
		NetworkId: network.ID,
	})
	if err != nil {
		logrus.Errorf("could not index host scans for network %s: %s", network.Name, err.Error())
		return err
	}

	logrus.Debugf("host scans response: %#v", resp)

	return nil
}

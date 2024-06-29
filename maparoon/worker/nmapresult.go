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

type nmapScanProcessor struct {
	network          database.Network
	scanPipe         string
	scanDoneCh       chan bool
	nmapHostResultCh chan types.NmapHostScan
}

func (nsp *nmapScanProcessor) ProcessResults(ctx context.Context) error {
	defer logrus.Debugf("Finished processing nmap results from %s", nsp.scanPipe)

	fd, err := unix.Open(
		nsp.scanPipe,
		unix.O_RDONLY|unix.O_EXCL|unix.O_NONBLOCK,
		0660,
	)
	if err != nil {
		return fmt.Errorf("could not open fifo for reading: %w", err)
	}

	f := os.NewFile(uintptr(fd), nsp.scanPipe)
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
		case <-nsp.scanDoneCh:
			stillReading = false
		case <-ctx.Done():
			return fmt.Errorf("context done while reading from nmap result pipe")
		case <-time.After(1 * time.Second):
			continue
		}
	}

	return nsp.parseResults(ctx, contentsBuf)
}

func (nsp *nmapScanProcessor) parseResults(ctx context.Context, resultsBuf *bytes.Buffer) error {
	defer logrus.Debugf("Finished parsing nmap results for network %s", nsp.network.Name)

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
					return nsp.returnScanResults(nmaprun)
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

func (nsp *nmapScanProcessor) returnScanResults(run nmap.NmapRun) error {
	for _, host := range run.Hosts {
		nsp.nmapHostResultCh <- types.NmapHostScan{
			Address: host.Addresses[0].Addr,
			NmapHostScanDocument: types.NmapHostScanDocument{
				FingerprintPorts: unrollOsPortsUsed(host.Os.PortsUsed),
				HostDetails:      host,
				ServicePorts:     unrollServicePorts(host.Ports),
			},
		}
	}

	return nil
}

func unrollScriptDetails(scripts []nmap.Script) map[string]interface{} {
	scriptDetails := make(map[string]interface{})

	for _, script := range scripts {
		scriptLevel := make(map[string]interface{})
		if script.Elements != nil {
			if len(script.Elements) == 1 && script.Elements[0].Key == "" {
				scriptLevel["_output"] = script.Elements[0].Value
				continue
			}

			for _, elem := range script.Elements {
				scriptLevel[elem.Key] = elem.Value
			}
		}

		if script.Tables != nil {
			for idx, table := range script.Tables {
				key := fmt.Sprintf("%d", idx)
				if table.Key != "" {
					key = table.Key
				}
				scriptLevel[key] = recurseTable(table)
			}
		}

		if script.Elements == nil && script.Tables == nil {
			scriptDetails[script.Id] = script.Output
		} else {
			scriptLevel["_output"] = script.Output
			scriptDetails[script.Id] = scriptLevel
		}
	}

	return scriptDetails
}

func recurseTable(table nmap.Table) map[string]interface{} {
	tableMap := make(map[string]interface{})
	if len(table.Elements) > 0 {
		for _, elem := range table.Elements {
			tableMap[elem.Key] = elem.Value
		}
	}

	if len(table.Table) > 0 {
		for idx, subTable := range table.Table {
			key := fmt.Sprintf("%d", idx)
			if subTable.Key != "" {
				key = subTable.Key
			}
			tableMap[key] = recurseTable(subTable)
		}
	}

	return tableMap
}

func unrollOsPortsUsed(portsUsed []nmap.PortUsed) map[string]interface{} {
	usedPorts := make(map[string]interface{})
	for _, port := range portsUsed {
		portDetails := make(map[string]interface{})
		portDetails["state"] = port.State
		portDetails["protocol"] = port.Proto

		usedPorts[fmt.Sprintf("%d", port.PortId)] = portDetails
	}

	return usedPorts
}

func unrollServicePorts(ports []nmap.Port) map[string]interface{} {
	servicePorts := make(map[string]interface{})

	for _, port := range ports {
		portDetails := make(map[string]interface{})
		portDetails["protocol"] = port.Protocol
		portDetails["state"] = port.State
		portDetails["owner"] = port.Owner
		portDetails["service"] = port.Service
		portDetails["scripts"] = unrollScriptDetails(port.Scripts)

		servicePorts[fmt.Sprintf("%d", port.PortId)] = portDetails
	}

	return servicePorts
}

package worker

import (
	"context"
	"fmt"

	"github.com/gosnmp/gosnmp"
	"github.com/pirogoeth/apps/maparoon/snmpsmi"
	"github.com/sirupsen/logrus"
	"github.com/sleepinggenius2/gosmi/types"
	"golang.org/x/sync/errgroup"
)

const (
	// https://www.alvestrand.no/objectid/1.3.6.1.2.1.1.html
	snmpSysDescr    = "1.3.6.1.2.1.1.1"
	snmpSysName     = "1.3.6.1.2.1.1.5"
	snmpSysLocation = "1.3.6.1.2.1.1.6"
	snmpSysServices = "1.3.6.1.2.1.1.7"
)

var (
	defaultRequestOids = []string{
		snmpSysDescr,
		snmpSysName,
		snmpSysLocation,
		snmpSysServices,
	}
)

type SnmpGatherer struct {
	targetQueue       chan string
	targetDataChannel chan SnmpTargetData
}

func NewSnmpGatherer() *SnmpGatherer {
	return &SnmpGatherer{
		targetQueue:       make(chan string),
		targetDataChannel: make(chan SnmpTargetData),
	}
}

func (g *SnmpGatherer) AddTarget(target string) {
	g.targetQueue <- target
}

func (g *SnmpGatherer) ReceiveChannel() <-chan SnmpTargetData {
	return g.targetDataChannel
}

func (g *SnmpGatherer) Run(ctx context.Context) error {
	gathererEg, childCtx := errgroup.WithContext(ctx)
	gathererEg.SetLimit(8)

	keepWaiting := true
	for keepWaiting {
		select {
		case <-ctx.Done():
			logrus.Debugf("SnmpGatherer context done, exiting")
			keepWaiting = false
		case target := <-g.targetQueue:
			gathererEg.Go(func() error {
				return g.gatherSnmpTargetData(childCtx, target)
			})
		}
	}

	return gathererEg.Wait()
}

type SnmpTargetData struct {
	Target string
	Data   *gosnmp.SnmpPacket
}

func (g *SnmpGatherer) gatherSnmpTargetData(ctx context.Context, target string) error {
	snmpClient := &gosnmp.GoSNMP{
		Community: "public",
		Context:   ctx,
		Port:      gosnmp.Default.Port,
		Target:    target,
		Timeout:   gosnmp.Default.Timeout,
		Version:   gosnmp.Version2c,
	}
	if err := snmpClient.Connect(); err != nil {
		return fmt.Errorf("failed to connect to snmp target %s: %w", target, err)
	}
	logrus.Debugf("snmp connected to %s", target)

	snmpData, err := snmpClient.Get(defaultRequestOids)
	if err != nil {
		logrus.Errorf("error while walking snmp target %s: %s", target, err.Error())
		return fmt.Errorf("failed walking snmp target %s: %w", target, err)
	}

	// Use gosmi to rewrite OIDs to human-readable names

	for _, packet := range snmpData.Variables {
		resolvedName, err := snmpsmi.ResolveOID(packet.Name)
		if err != nil {
			logrus.Warnf("could not resolve SNMP OID %s: %s", packet.Name, err.Error())
			continue
		}

		logrus.Debugf("resolved OID %s to %s", packet.Name, resolvedName.Render(types.RenderName))
	}

	logrus.Debugf("gathered %d PDUs from %s", len(snmpData.Variables), target)

	g.targetDataChannel <- SnmpTargetData{
		Target: target,
		Data:   snmpData,
	}

	return nil
}

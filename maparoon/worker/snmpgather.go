package worker

import (
	"context"
	"fmt"
	"strings"

	"github.com/gosnmp/gosnmp"
	"github.com/sirupsen/logrus"
	gosmiTypes "github.com/sleepinggenius2/gosmi/types"
	"golang.org/x/sync/errgroup"

	"github.com/pirogoeth/apps/maparoon/snmpsmi"
	"github.com/pirogoeth/apps/maparoon/types"
)

const (
	// https://www.alvestrand.no/objectid/1.3.6.1.2.1.1.html
	snmpSysDescr    = "1.3.6.1.2.1.1.1.0"
	snmpSysName     = "1.3.6.1.2.1.1.5.0"
	snmpSysLocation = "1.3.6.1.2.1.1.6.0"
	snmpSysServices = "1.3.6.1.2.1.1.7.0"
)

var (
	defaultRequestOids = []string{
		snmpSysDescr,
		snmpSysName,
		snmpSysLocation,
		snmpSysServices,
	}
)

type SnmpGathererOpts struct {
	Community string
}

type SnmpGatherer struct {
	opts              *SnmpGathererOpts
	targetQueue       chan string
	targetDataChannel chan types.SnmpHostScan
}

func NewSnmpGatherer(opts *SnmpGathererOpts) *SnmpGatherer {
	return &SnmpGatherer{
		opts:              opts,
		targetQueue:       make(chan string),
		targetDataChannel: make(chan types.SnmpHostScan),
	}
}

func (g *SnmpGatherer) AddTarget(target string) {
	g.targetQueue <- target
}

func (g *SnmpGatherer) ReceiveChannel() <-chan types.SnmpHostScan {
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

func (g *SnmpGatherer) gatherSnmpTargetData(ctx context.Context, target string) error {
	hostScanResult := types.SnmpHostScan{
		Address: target,
		SnmpHostScanDocument: types.SnmpHostScanDocument{
			Available:    true,
			Measurements: make(map[string]types.SnmpMeasurement, 0),
		},
	}
	defer func() { g.targetDataChannel <- hostScanResult }()

	snmpClient := &gosnmp.GoSNMP{
		Community: g.opts.Community,
		Context:   ctx,
		Port:      gosnmp.Default.Port,
		Target:    target,
		Timeout:   gosnmp.Default.Timeout,
		Version:   gosnmp.Version2c,
	}
	if err := snmpClient.Connect(); err != nil {
		if strings.Contains(err.Error(), "timeout") {
			// No SNMP response in the timeout period, treat as no SNMP service on target
			hostScanResult.Available = false
		} else {
			return fmt.Errorf("failed to connect to snmp target %s: %w", target, err)
		}
	}
	logrus.Debugf("snmp connected to %s", target)

	snmpData, err := snmpClient.Get(defaultRequestOids)
	if err != nil {
		return fmt.Errorf("failed getting default measurements from snmp target %s: %w", target, err)
	}

	for _, packet := range snmpData.Variables {
		resolvedName, err := snmpsmi.ResolveOID(packet.Name)
		if err != nil {
			logrus.Warnf("could not resolve SNMP OID %s: %s", packet.Name, err.Error())
			continue
		}

		logrus.Debugf("resolved OID %s to %s", packet.Name, resolvedName.Render(gosmiTypes.RenderName))

		stringifiedValue := ""
		switch packet.Type {
		case gosnmp.OctetString:
			stringifiedValue = string(packet.Value.([]byte))
		case gosnmp.Integer:
			fallthrough
		case gosnmp.Uinteger32:
			fallthrough
		case gosnmp.Counter32:
			fallthrough
		case gosnmp.Counter64:
			stringifiedValue = gosnmp.ToBigInt(packet.Value).String()
		default:
			stringifiedValue = fmt.Sprintf("%v", packet.Value)
		}

		cleanOid := escapeOid(packet.Name)

		hostScanResult.Measurements[cleanOid] = types.SnmpMeasurement{
			Oid:   packet.Name,
			Name:  resolvedName.Render(gosmiTypes.RenderName),
			Type:  packet.Type.String(),
			Value: stringifiedValue,
		}
	}

	return nil
}

func escapeOid(oid string) string {
	oid = strings.ReplaceAll(oid, ".", "-")
	oid = strings.TrimPrefix(oid, "-")

	return oid
}

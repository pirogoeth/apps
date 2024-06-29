package types

import (
	"github.com/adifire/go-nmap"
	"github.com/pirogoeth/apps/maparoon/database"
)

type NmapHostScanDocument struct {
	FingerprintPorts map[string]interface{} `json:"fingerprint_ports"`
	HostDetails      nmap.Host              `json:"host"`
	ServicePorts     map[string]interface{} `json:"ports"`
}

type NmapHostScan struct {
	NmapHostScanDocument

	Address string `json:"address"`
}

type SnmpMeasurement struct {
	Oid   string `json:"oid"`
	Name  string `json:"name"`
	Type  string `json:"type"`
	Value string `json:"value"`
}

type SnmpHostScanDocument struct {
	Available    bool                       `json:"available"`
	Measurements map[string]SnmpMeasurement `json:"measurements"`
}

type SnmpHostScan struct {
	SnmpHostScanDocument

	Address string `json:"address"`
}

type HostScanDocument struct {
	Address string                `json:"address"`
	Network database.Network      `json:"network"`
	Nmap    *NmapHostScanDocument `json:"nmap"`
	Snmp    *SnmpHostScanDocument `json:"snmp"`
}

type CreateHostScansRequest struct {
	HostScans []*HostScanDocument `json:"host_scans"`
	NetworkId int64               `json:"network_id"`
}

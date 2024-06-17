package types

import (
	"github.com/adifire/go-nmap"
	"github.com/pirogoeth/apps/maparoon/database"
)

type HostScan struct {
	Address          string                 `json:"address"`
	FingerprintPorts map[string]interface{} `json:"fingerprint_ports"`
	HostDetails      nmap.Host              `json:"host"`
	ServicePorts     map[string]interface{} `json:"ports"`
}

type HostScanDocument struct {
	Address          string                 `json:"address"`
	FingerprintPorts map[string]interface{} `json:"fingerprint_ports"`
	HostDetails      nmap.Host              `json:"host"`
	Network          database.Network       `json:"network"`
	ServicePorts     map[string]interface{} `json:"ports"`
}

type CreateHostScansRequest struct {
	HostScans []HostScan `json:"host_scans"`
	NetworkId int64      `json:"network_id"`
}

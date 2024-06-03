package types

import (
	"github.com/adifire/go-nmap"
	"github.com/pirogoeth/apps/maparoon/database"
)

type HostScan struct {
	Address     string    `json:"address"`
	ScanDetails nmap.Host `json:"scan"`
}

type HostScanDocument struct {
	Address     string           `json:"address"`
	Network     database.Network `json:"network"`
	ScanDetails nmap.Host        `json:"scan"`
}

type CreateHostScansRequest struct {
	HostScans []HostScan `json:"host_scans"`
	NetworkId int64      `json:"network_id"`
}

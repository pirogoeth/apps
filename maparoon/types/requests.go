package types

import (
	"github.com/adifire/go-nmap"
	"github.com/pirogoeth/apps/maparoon/database"
)

type HostScan struct {
	Address       string                 `json:"address"`
	HostDetails   nmap.Host              `json:"host"`
	ScriptDetails map[string]interface{} `json:"scripts"`
}

type HostScanDocument struct {
	Address       string                 `json:"address"`
	Network       database.Network       `json:"network"`
	HostDetails   nmap.Host              `json:"host"`
	ScriptDetails map[string]interface{} `json:"scripts"`
}

type CreateHostScansRequest struct {
	HostScans []HostScan `json:"host_scans"`
	NetworkId int64      `json:"network_id"`
}

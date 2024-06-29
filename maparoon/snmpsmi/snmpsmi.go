package snmpsmi

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/sleepinggenius2/gosmi"
	gosmiSmi "github.com/sleepinggenius2/gosmi/smi"
	gosmiTypes "github.com/sleepinggenius2/gosmi/types"

	"github.com/pirogoeth/apps/maparoon/snmpsmi/juniperMibs"
	"github.com/pirogoeth/apps/maparoon/snmpsmi/netSnmpMibs"
)

var globalOidCache = newOidCache()

func Init() {
	gosmi.Init()

	// TODO: Add a way to load custom MIBs via worker configuration
	fses := []gosmiSmi.NamedFS{
		gosmi.NamedFS("Maparoon Embedded MIBS: net-snmp", netSnmpMibs.FS),
		gosmi.NamedFS("Maparoon Embedded MIBS: Juniper", juniperMibs.FS),
	}
	gosmi.SetFS(fses...)

	for _, fs := range fses {
		// Walk the FS and load modules
		mibsDir, err := fs.FS.ReadDir(".")
		if err != nil {
			panic(fmt.Errorf("failed to read embedded mibs directory: %w", err))
		}

		logrus.Debugf("Loading %d MIB modules [%s]...", len(mibsDir), fs.Name)
		for _, file := range mibsDir {
			_, err := gosmi.LoadModule(file.Name())
			if err != nil {
				logrus.Warnf("failed to load module %s: %s, skipping", file.Name(), err)
				continue
			}
		}
	}

	modules := gosmi.GetLoadedModules()
	logrus.Infof("Loaded %d MIB modules", len(modules))
}

func ResolveOID(oid string) (*gosmi.SmiNode, error) {
	if node := globalOidCache.Get(oid); node != nil {
		return node, nil
	}

	node, err := ResolveOIDWithoutCache(oid)
	if err != nil {
		return nil, err
	}

	globalOidCache.Add(oid, node)

	return node, err
}

func ResolveOIDWithoutCache(oid string) (*gosmi.SmiNode, error) {
	node, err := gosmi.GetNodeByOID(gosmiTypes.OidMustFromString(oid))
	if err != nil {
		return nil, err
	}

	return &node, nil
}

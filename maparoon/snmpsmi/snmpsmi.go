package snmpsmi

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/sleepinggenius2/gosmi"
	gosmiTypes "github.com/sleepinggenius2/gosmi/types"

	"github.com/pirogoeth/apps/maparoon/snmpsmi/mibs"
)

func Init() {
	gosmi.Init()

	// TODO: Add a way to load custom MIBs via worker configuration
	gosmi.SetFS(gosmi.NamedFS("Maparoon Embedded MIBS", mibs.Mibs))

	// Walk the FS and load modules
	mibsDir, err := mibs.Mibs.ReadDir(".")
	if err != nil {
		panic(fmt.Errorf("failed to read embedded mibs directory: %w", err))
	}

	logrus.Debugf("Loading %d MIBS", len(mibsDir))
	for _, file := range mibsDir {
		_, err := gosmi.LoadModule(file.Name())
		if err != nil {
			logrus.Warnf("failed to load module %s: %s, skipping", file.Name(), err)
			continue
		}
	}

	modules := gosmi.GetLoadedModules()
	logrus.Infof("Loaded %d MIB modules", len(modules))
}

func ResolveOID(oid string) (*gosmi.SmiNode, error) {
	node, err := gosmi.GetNodeByOID(gosmiTypes.OidMustFromString(oid))
	if err != nil {
		return nil, err
	}

	return &node, nil
}

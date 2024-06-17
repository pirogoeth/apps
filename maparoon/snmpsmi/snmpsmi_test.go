package snmpsmi

import (
	"testing"

	"github.com/pirogoeth/apps/pkg/logging"
)

func TestInit(t *testing.T) {
	logging.Setup()

	Init()
}

func TestResolveOid(t *testing.T) {
	oid := "1.3.6.1.2.1.2.2.1.1"
	expected := "ifIndex"

	value, err := ResolveOID(oid)
	if err != nil {
		t.Error(err)
	}

	if value.Name != expected {
		t.Errorf("expected %s, got %s", expected, value.Name)
	}
}

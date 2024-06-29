package database

import "testing"

func TestNetworkSizeIpv4(t *testing.T) {
	n := Network{
		Address: "10.100.0.1",
		Cidr:    24,
	}

	size, err := n.NetworkSize()
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	if size != 254 {
		t.Errorf("unexpected network size: %d", size)
	}
}

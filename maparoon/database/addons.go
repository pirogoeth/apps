package database

import (
	"fmt"
	"math"
	"net"
)

func (n Network) CidrString() string {
	return fmt.Sprintf("%s/%d", n.Address, n.Cidr)
}

func (n Network) NetworkSize() (int, error) {
	netCidr := n.CidrString()

	_, ipNet, err := net.ParseCIDR(netCidr)
	if err != nil {
		return -1, fmt.Errorf("could not parse network CIDR %s: %w", netCidr, err)
	}

	numOnes, totalBits := ipNet.Mask.Size()
	return int(math.Max(math.Pow(2, float64(totalBits-numOnes))-2, 1.0)), nil
}

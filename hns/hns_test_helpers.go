package hns

import (
	"github.com/Microsoft/hcsshim"
	"github.com/Juniper/contrail-windows-docker-driver/common"
	. "github.com/onsi/gomega"
)

func MockHNSNetwork(netAdapter common.AdapterName, name, subnetCIDR, defaultGW string) string {
	subnets := []hcsshim.Subnet{
		{
			AddressPrefix:  subnetCIDR,
			GatewayAddress: defaultGW,
		},
	}
	netConfig := &hcsshim.HNSNetwork{
		Name:               name,
		Type:               "transparent",
		NetworkAdapterName: string(netAdapter),
		Subnets:            subnets,
	}
	var err error
	netID, err := CreateHNSNetwork(netConfig)
	Expect(err).ToNot(HaveOccurred())
	return netID
}

func MockHNSEndpoint(netID string) string {
	epConfig := &hcsshim.HNSEndpoint{
		VirtualNetwork: netID,
	}
	epID, err := CreateHNSEndpoint(epConfig)
	Expect(err).ToNot(HaveOccurred())
	return epID
}

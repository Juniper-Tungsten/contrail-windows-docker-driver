//
// Copyright (c) 2018 Juniper Networks, Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package helpers

import (
	"github.com/Juniper/contrail-windows-docker-driver/adapters/secondary/local_networking/hns"
	"github.com/Microsoft/hcsshim"
	. "github.com/onsi/gomega"
)

func CreateTestHNSNetwork(netAdapter string, name, subnetCIDR, defaultGW string) string {
	subnets := []hcsshim.Subnet{
		{
			AddressPrefix:  subnetCIDR,
			GatewayAddress: defaultGW,
		},
	}
	netConfig := &hcsshim.HNSNetwork{
		Name:               name,
		Type:               "transparent",
		NetworkAdapterName: netAdapter,
		Subnets:            subnets,
	}
	var err error
	netID, err := hns.CreateHNSNetwork(netConfig)
	Expect(err).ToNot(HaveOccurred())
	return netID
}

func CreateTestHNSEndpoint(netID string) string {
	epConfig := &hcsshim.HNSEndpoint{
		VirtualNetwork: netID,
	}
	epID, err := hns.CreateHNSEndpoint(epConfig)
	Expect(err).ToNot(HaveOccurred())
	return epID
}

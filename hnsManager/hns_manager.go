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

package hnsManager

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Microsoft/hcsshim"
	"github.com/Juniper/contrail-windows-docker-driver/common"
	"github.com/Juniper/contrail-windows-docker-driver/adapters/secondary/hns"
)

// HNSManager manages HNS networks that are used by the driver.
type HNSManager struct {
	// TODO JW-154: store networks here that you know about. If not found here, look in HNS.
	// for now, just look in HNS by name.
}

func contrailHNSNetName(tenant, netName, subnetCIDR string) string {
	return fmt.Sprintf("%s:%s:%s:%s", common.HNSNetworkPrefix, tenant, netName, subnetCIDR)
}

func (m *HNSManager) CreateNetwork(netAdapter common.AdapterName, tenantName, networkName,
	subnetCIDR, defaultGW string) (*hcsshim.HNSNetwork, error) {

	hnsNetName := contrailHNSNetName(tenantName, networkName, subnetCIDR)

	net, err := hns.GetHNSNetworkByName(hnsNetName)
	if net != nil {
		return nil, errors.New("Such HNS network already exists")
	}

	subnets := []hcsshim.Subnet{
		{
			AddressPrefix:  subnetCIDR,
			GatewayAddress: defaultGW,
		},
	}

	configuration := &hcsshim.HNSNetwork{
		Name:               hnsNetName,
		Type:               "transparent",
		NetworkAdapterName: string(netAdapter),
		Subnets:            subnets,
	}

	hnsNetworkID, err := hns.CreateHNSNetwork(configuration)
	if err != nil {
		return nil, err
	}

	hnsNetwork, err := hns.GetHNSNetwork(hnsNetworkID)
	if err != nil {
		return nil, err
	}

	return hnsNetwork, nil
}

func (m *HNSManager) GetNetwork(tenantName, networkName, subnetCIDR string) (*hcsshim.HNSNetwork,
	error) {
	hnsNetName := contrailHNSNetName(tenantName, networkName, subnetCIDR)
	hnsNetwork, err := hns.GetHNSNetworkByName(hnsNetName)
	if err != nil {
		return nil, err
	}
	if hnsNetwork == nil {
		return nil, errors.New("Such HNS network does not exist")
	}
	return hnsNetwork, nil
}

func (m *HNSManager) DeleteNetwork(tenantName, networkName, subnetCIDR string) error {
	hnsNetwork, err := m.GetNetwork(tenantName, networkName, subnetCIDR)
	if err != nil {
		return err
	}
	endpoints, err := hns.ListHNSEndpoints()
	if err != nil {
		return err
	}

	for _, ep := range endpoints {
		if ep.VirtualNetworkName == hnsNetwork.Name {
			return errors.New("Cannot delete network with active endpoints")
		}
	}
	return hns.DeleteHNSNetwork(hnsNetwork.Id)
}

func (m *HNSManager) ListNetworks() ([]hcsshim.HNSNetwork, error) {
	var validNets []hcsshim.HNSNetwork
	nets, err := hns.ListHNSNetworks()
	if err != nil {
		return validNets, err
	}
	for _, net := range nets {
		splitName := strings.Split(net.Name, ":")
		if len(splitName) == 4 {
			if splitName[0] == common.HNSNetworkPrefix {
				validNets = append(validNets, net)
			}
		}
	}
	return validNets, nil
}

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

package hns

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Juniper/contrail-windows-docker-driver/common"
	"github.com/Microsoft/hcsshim"
)

// HNSContrailNetworksRepository handles local HNS networks associated to Contrail subnets.
// It does it by naming the HNS networks in a specific way. The name of HNS network contains
// Contrail FQ (fully qualified) name as well as subnet CIDR. This guarantees 1-to-1 correspondence
// of HNS network with a Contrail subnet. Also, it keeps the driver stateless (relevant state is
// held directly in HNS). An alternative would be to have a local DB (like SQLite) that stores
// associations.
type HNSContrailNetworksRepository struct {
	// physDataplaneNetAdapter is the name of physical dataplane adapter that we should attach our
	// Contrail networks to, e.g. Ethernet0. It is NOT the adapter created by HNS (e.g. "HNS
	// Transparent").
	physDataplaneNetAdapter common.AdapterName
}

func NewHNSContrailNetworksRepository(physDataplaneNetAdapter common.AdapterName) (*HNSContrailNetworksRepository, error) {
	if err := InitRootHNSNetwork(physDataplaneNetAdapter); err != nil {
		return nil, err
	}
	return &HNSContrailNetworksRepository{
		physDataplaneNetAdapter: physDataplaneNetAdapter,
	}, nil
}

func associationNameForHNSNetworkContrailSubnet(tenant, netName, subnetCIDR string) string {
	return fmt.Sprintf("%s:%s:%s:%s", common.HNSNetworkPrefix, tenant, netName, subnetCIDR)
}

func (repo *HNSContrailNetworksRepository) CreateNetwork(tenantName, networkName,
	subnetCIDR, defaultGW string) (*hcsshim.HNSNetwork, error) {

	hnsNetName := associationNameForHNSNetworkContrailSubnet(tenantName, networkName, subnetCIDR)

	net, err := GetHNSNetworkByName(hnsNetName)
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
		NetworkAdapterName: string(repo.physDataplaneNetAdapter),
		Subnets:            subnets,
	}

	hnsNetworkID, err := CreateHNSNetwork(configuration)
	if err != nil {
		return nil, err
	}

	hnsNetwork, err := GetHNSNetwork(hnsNetworkID)
	if err != nil {
		return nil, err
	}

	return hnsNetwork, nil
}

func (repo *HNSContrailNetworksRepository) GetNetwork(tenantName, networkName, subnetCIDR string) (*hcsshim.HNSNetwork,
	error) {
	hnsNetName := associationNameForHNSNetworkContrailSubnet(tenantName, networkName, subnetCIDR)
	hnsNetwork, err := GetHNSNetworkByName(hnsNetName)
	if err != nil {
		return nil, err
	}
	if hnsNetwork == nil {
		return nil, errors.New("Such HNS network does not exist")
	}
	return hnsNetwork, nil
}

func (repo *HNSContrailNetworksRepository) DeleteNetwork(tenantName, networkName, subnetCIDR string) error {
	hnsNetwork, err := repo.GetNetwork(tenantName, networkName, subnetCIDR)
	if err != nil {
		return err
	}
	endpoints, err := ListHNSEndpoints()
	if err != nil {
		return err
	}

	for _, ep := range endpoints {
		if ep.VirtualNetworkName == hnsNetwork.Name {
			return errors.New("Cannot delete network with active endpoints")
		}
	}
	return DeleteHNSNetwork(hnsNetwork.Id)
}

func (repo *HNSContrailNetworksRepository) ListNetworks() ([]hcsshim.HNSNetwork, error) {
	var validNets []hcsshim.HNSNetwork
	nets, err := ListHNSNetworks()
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

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

package hns_contrail

import (
	"errors"
	"strings"

	"github.com/Juniper/contrail-windows-docker-driver/adapters/secondary/hns"
	"github.com/Juniper/contrail-windows-docker-driver/adapters/secondary/local_networking/vmswitch"
	"github.com/Juniper/contrail-windows-docker-driver/core/model"
	"github.com/Microsoft/hcsshim"
)

// HNSContrailNetworksRepository handles local HNS networks associated to Contrail subnets.
// It does it by naming the HNS networks in a specific way. The name of HNS network contains
// Contrail FQ (fully qualified) name as well as subnet CIDR. This guarantees 1-to-1 correspondence
// of HNS network with a Contrail subnet. Also, it keeps the driver stateless (relevant state is
// held directly in HNS, using HNSDBNetworkAssociationMechanism). An alternative would be to have
// a local DB (like SQLite) that stores associations.
type HNSContrailNetworksRepository struct {
	// physDataplaneNetAdapter is the name of physical dataplane adapter that we should attach our
	// Contrail networks to, e.g. Ethernet0. It is NOT the adapter created by HNS (e.g. "HNS
	// Transparent").
	physDataplaneNetAdapter string
	vswitchName             string
	associations            HNSDBNetworkAssociationMechanism
}

func NewHNSContrailNetworksRepository(physDataplaneNetAdapter, vswitchName string) *HNSContrailNetworksRepository {
	return &HNSContrailNetworksRepository{
		physDataplaneNetAdapter: physDataplaneNetAdapter,
		vswitchName:             vswitchName,
		associations:            HNSDBNetworkAssociationMechanism{},
	}
}

func (repo *HNSContrailNetworksRepository) CreateNetwork(dockerNetID string, network *model.Network) error {

	hnsNetName := repo.associations.GenerateName(dockerNetID, network.TenantName, network.NetworkName, network.Subnet.CIDR)

	net, err := hns.GetHNSNetworkByName(hnsNetName)
	if net != nil {
		return errors.New("such HNS network already exists")
	}

	subnets := []hcsshim.Subnet{
		{
			AddressPrefix:  network.Subnet.CIDR,
			GatewayAddress: network.Subnet.DefaultGW,
		},
	}

	configuration := &hcsshim.HNSNetwork{
		Name:               hnsNetName,
		Type:               "transparent",
		NetworkAdapterName: string(repo.physDataplaneNetAdapter),
		Subnets:            subnets,
		DNSServerList:      strings.Join(network.Subnet.DNSServerList, ","),
	}

	if switchState, err := vmswitch.DoesSwitchExist(repo.vswitchName); err != nil {
		panic(err)
	} else if switchState == vmswitch.DELETED || switchState == vmswitch.DELETING {
		panic("VmSwitch doesn't exist")
	}

	_, err = hns.CreateHNSNetwork(configuration)
	if err != nil {
		return err
	}

	return nil
}

func (repo *HNSContrailNetworksRepository) GetNetwork(dockerNetID string) (*model.Network,
	error) {
	hnsNetwork, err := repo.findHNSNetworkByDockerID(dockerNetID)
	if err != nil {
		return nil, err
	}
	_, foundTenantName, foundNetworkName, foundSubnetCIDR :=
		repo.associations.SplitName(hnsNetwork.Name)

	Subnet := model.Subnet{
		CIDR:          foundSubnetCIDR,
		DNSServerList: strings.Split(hnsNetwork.DNSServerList, ","),
	}
	Net := model.Network{
		LocalID:     hnsNetwork.Id,
		TenantName:  foundTenantName,
		NetworkName: foundNetworkName,
		Subnet:      Subnet,
	}
	return &Net, nil
}

func (repo *HNSContrailNetworksRepository) DeleteNetwork(dockerNetID string) error {
	hnsNetwork, err := repo.findHNSNetworkByDockerID(dockerNetID)
	if err != nil {
		return err
	}
	endpoints, err := hns.ListHNSEndpoints()
	if err != nil {
		return err
	}

	for _, ep := range endpoints {
		if ep.VirtualNetworkName == hnsNetwork.Name {
			return errors.New("cannot delete network with active endpoints")
		}
	}
	return hns.DeleteHNSNetwork(hnsNetwork.Id)
}

func (repo *HNSContrailNetworksRepository) ListNetworks() ([]model.Network, error) {
	var ownedNets []model.Network
	hnsNetworks, err := repo.listOwnedHNSNetworks()
	if err != nil {
		return ownedNets, err
	}
	for _, hnsNetwork := range hnsNetworks {
		_, foundTenantName, foundNetworkName, foundSubnetCIDR :=
			repo.associations.SplitName(hnsNetwork.Name)
		Subnet := model.Subnet{
			CIDR:          foundSubnetCIDR,
			DNSServerList: strings.Split(hnsNetwork.DNSServerList, ","),
		}
		net := model.Network{
			LocalID:     hnsNetwork.Id,
			TenantName:  foundTenantName,
			NetworkName: foundNetworkName,
			Subnet:      Subnet,
		}
		ownedNets = append(ownedNets, net)
	}
	return ownedNets, nil
}

func (repo *HNSContrailNetworksRepository) findHNSNetworkByDockerID(dockerNetID string) (*hcsshim.HNSNetwork, error) {
	hnsNetworks, err := repo.listOwnedHNSNetworks()
	if err != nil {
		return nil, err
	}
	for idx, hnsNetwork := range hnsNetworks {
		foundDockerNetID, _, _, _ := repo.associations.SplitName(hnsNetwork.Name)
		if foundDockerNetID == dockerNetID {
			return &hnsNetworks[idx], nil
		}
	}
	return nil, errors.New("could not find HNS network with such docker network ID")
}

func (repo *HNSContrailNetworksRepository) listOwnedHNSNetworks() ([]hcsshim.HNSNetwork, error) {
	var ownedHNSNetworks []hcsshim.HNSNetwork
	hnsNetworks, err := hns.ListHNSNetworks()
	if err != nil {
		return ownedHNSNetworks, err
	}
	for idx, hnsNetwork := range hnsNetworks {
		if repo.associations.IsOwnedByDriver(hnsNetwork.Name) {
			ownedHNSNetworks = append(ownedHNSNetworks, hnsNetworks[idx])
		}
	}
	return ownedHNSNetworks, nil
}

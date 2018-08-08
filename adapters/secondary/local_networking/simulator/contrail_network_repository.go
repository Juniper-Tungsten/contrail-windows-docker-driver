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

package simulator

import (

	// We should rely on some kind of domain objects in the future - not hcsshim structs
	// everywhere.
	"errors"

	"github.com/Juniper/contrail-windows-docker-driver/adapters/secondary/local_networking/hns"
	"github.com/Juniper/contrail-windows-docker-driver/core/model"
	"github.com/Microsoft/hcsshim"
)

type InMemContrailNetworksRepository struct {
	networks     map[string]hcsshim.HNSNetwork
	associations hns.HNSDBNetworkAssociationMechanism
}

func NewInMemContrailNetworksRepository() *InMemContrailNetworksRepository {
	return &InMemContrailNetworksRepository{
		networks:     make(map[string]hcsshim.HNSNetwork),
		associations: hns.HNSDBNetworkAssociationMechanism{},
	}
}

func (repo *InMemContrailNetworksRepository) CreateNetwork(dockerNetID string, net *model.Network) error {
	name := repo.associations.GenerateName(dockerNetID, net.TenantName, net.NetworkName, net.Subnet.CIDR)

	network := hcsshim.HNSNetwork{Name: name}
	repo.networks[dockerNetID] = network

	return nil
}

func (repo *InMemContrailNetworksRepository) GetNetwork(dockerNetID string) (*model.Network, error) {
	if net, exists := repo.networks[dockerNetID]; exists {
		_, foundTenantName, foundNetworkName, foundSubnetCIDR :=
			repo.associations.SplitName(net.Name)
		return &model.Network{
			LocalID:     "123hnsNetID",
			TenantName:  foundTenantName,
			NetworkName: foundNetworkName,
			Subnet: model.Subnet{
				CIDR: foundSubnetCIDR,
			},
		}, nil
	} else {
		return nil, errors.New("network not found")
	}
}

func (repo *InMemContrailNetworksRepository) DeleteNetwork(dockerNetID string) error {
	if _, exists := repo.networks[dockerNetID]; exists {
		delete(repo.networks, dockerNetID)
		return nil
	} else {
		return errors.New("network not found, so couldn't delete it")
	}
}

func (repo *InMemContrailNetworksRepository) ListNetworks() ([]model.Network, error) {
	arr := make([]model.Network, 0, len(repo.networks))
	for _, net := range repo.networks {
		_, foundTenantName, foundNetworkName, foundSubnetCIDR :=
			repo.associations.SplitName(net.Name)
		mnet := model.Network{
			LocalID:     "123hnsNetID",
			TenantName:  foundTenantName,
			NetworkName: foundNetworkName,
			Subnet: model.Subnet{
				CIDR: foundSubnetCIDR,
			},
		}
		arr = append(arr, mnet)
	}
	return arr, nil
}

func (repo *InMemContrailNetworksRepository) FindContrailNetwork(tenantName, networkName,
	subnetCIDR string) (*model.Network, error) {
	for _, net := range repo.networks {
		foundDockerNetID, foundTenantName, foundNetworkName, foundSubnetCIDR :=
			repo.associations.SplitName(net.Name)
		if foundTenantName == tenantName &&
			foundNetworkName == networkName &&
			foundSubnetCIDR == subnetCIDR {
			return repo.GetNetwork(foundDockerNetID)
		}
	}
	return nil, errors.New("network not found")
}

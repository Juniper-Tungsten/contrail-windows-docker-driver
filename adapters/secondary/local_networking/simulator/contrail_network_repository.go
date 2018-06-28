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
	"fmt"

	"github.com/Juniper/contrail-windows-docker-driver/common"
	"github.com/Microsoft/hcsshim"
)

type InMemContrailNetworksRepository struct {
	networks map[string]hcsshim.HNSNetwork
}

func NewInMemContrailNetworksRepository() *InMemContrailNetworksRepository {
	return &InMemContrailNetworksRepository{
		networks: make(map[string]hcsshim.HNSNetwork),
	}
}

func (repo *InMemContrailNetworksRepository) CreateNetwork(tenantName, networkName, subnetCIDR, defaultGW string) (*hcsshim.HNSNetwork, error) {
	// TODO: not sure wheter we actually need such complicated name generation in a
	// simulated repository.
	// TBH, existence of such code in actual repository implementation and a fake suggests
	// that we're looking at some other object. Some kind of naming policy class maybe?
	name := fmt.Sprintf("%s:%s:%s:%s", common.HNSNetworkPrefix, tenantName, networkName, subnetCIDR)

	net := hcsshim.HNSNetwork{Name: name}
	repo.networks[name] = net
	return &net, nil
}

func (repo *InMemContrailNetworksRepository) GetNetwork(tenantName, networkName, subnetCIDR string) (*hcsshim.HNSNetwork, error) {
	nameToLookFor := fmt.Sprintf("%s:%s:%s:%s", common.HNSNetworkPrefix, tenantName, networkName, subnetCIDR)
	if net, exists := repo.networks[nameToLookFor]; exists {
		return &net, nil
	} else {
		return nil, errors.New("network not found")
	}
}

func (repo *InMemContrailNetworksRepository) DeleteNetwork(tenantName, networkName, subnetCIDR string) error {
	nameToLookFor := fmt.Sprintf("%s:%s:%s:%s", common.HNSNetworkPrefix, tenantName, networkName, subnetCIDR)
	if _, exists := repo.networks[nameToLookFor]; exists {
		delete(repo.networks, nameToLookFor)
		return nil
	} else {
		return errors.New("network not found, so couldn't delete it")
	}
}

func (repo *InMemContrailNetworksRepository) ListNetworks() ([]hcsshim.HNSNetwork, error) {
	arr := make([]hcsshim.HNSNetwork, 0, len(repo.networks))
	for _, net := range repo.networks {
		arr = append(arr, net)
	}
	return arr, nil
}

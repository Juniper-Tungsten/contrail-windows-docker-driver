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

package ports

import (
	"github.com/Microsoft/hcsshim"

	contrail "github.com/Juniper/contrail-go-api"
	"github.com/Juniper/contrail-go-api/types"
)

type Agent interface {
	AddPort(vmUUID, vifUUID, ifName, mac, dockerID, ipAddress, vnUUID string) error
	DeletePort(vifUUID string) error
}

type VRouter interface {
	Initialize() error
}

type LocalContrailNetworkRepository interface {
	CreateNetwork(tenantName, networkName, subnetCIDR, defaultGW string) (*hcsshim.HNSNetwork,
		error)
	GetNetwork(tenantName, networkName, subnetCIDR string) (*hcsshim.HNSNetwork,
		error)
	DeleteNetwork(tenantName, networkName, subnetCIDR string) error
	ListNetworks() ([]hcsshim.HNSNetwork, error)
}

type LocalContrailEndpointRepository interface {
	CreateEndpoint(configuration *hcsshim.HNSEndpoint) (string, error)
	GetEndpointByName(name string) (*hcsshim.HNSEndpoint, error)
	DeleteEndpoint(endpointID string) error
}

// TODO: This interface can be simplified
type Controller interface {
	CreateNetworkWithSubnet(tenantName, networkName, subnetCIDR string) (*types.VirtualNetwork, error)
	GetNetworkWithSubnet(tenantName, networkName, subnetCIDR string) (*types.VirtualNetwork, *types.IpamSubnetType, error)

	CreateContainerInSubnet(tenantName, containerID string, network *types.VirtualNetwork, subnet *types.IpamSubnetType) (*ContrailContainer, error)

	// This method is only used by tests; to remove?
	NewProject(domain, tenant string) (*types.Project, error)

	// To remove when refactoring plugin.DeleteEndpoint
	GetNetwork(tenantName, networkName string) (*types.VirtualNetwork, error)
	GetInstance(containerId string) (*types.VirtualMachine, error)
	GetExistingInterface(net *types.VirtualNetwork, tenantName, containerId string) (*types.VirtualMachineInterface, error)
	DeleteElementRecursive(parent contrail.IObject) error
}

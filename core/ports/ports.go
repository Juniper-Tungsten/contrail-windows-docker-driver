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

	"github.com/Juniper/contrail-go-api/types"
	"github.com/Juniper/contrail-windows-docker-driver/core/model"
)

type PortAssociation interface {
	AddPort(vmUUID, vifUUID, ifName, mac, dockerID, ipAddress, vnUUID string) error
	DeletePort(vifUUID string) error
}

type VRouter interface {
	Initialize() error
}

type LocalContrailNetworkRepository interface {
	CreateNetwork(dockerNetID string, network *model.Network) error
	GetNetwork(dockerNetID string) (*model.Network, error)
	DeleteNetwork(dockerNetID string) error
	ListNetworks() ([]model.Network, error)
}

type LocalContrailEndpointRepository interface {
	CreateEndpoint(name string, container *model.Container, network *model.Network) (string, error)
	GetEndpoint(name string) (*hcsshim.HNSEndpoint, error)
	DeleteEndpoint(endpointID string) error
}

type Controller interface {
	CreateNetworkWithSubnet(tenantName, networkName, subnetCIDR string) (*types.VirtualNetwork, error)
	GetNetworkWithSubnet(tenantName, networkName, subnetCIDR string) (*model.Network, error)

	CreateContainerInSubnet(net *model.Network, containerID string) (*model.Container, error)
	DeleteContainer(containerID string) error

	// This method is only used by tests; to remove?
	NewProject(domain, tenant string) (*types.Project, error)

	// To remove when refactoring plugin.DeleteEndpoint
	GetNetwork(tenantName, networkName string) (*types.VirtualNetwork, error)
	GetInstance(containerId string) (*types.VirtualMachine, error)
	GetExistingInterface(net *types.VirtualNetwork, tenantName, containerId string) (*types.VirtualMachineInterface, error)
}

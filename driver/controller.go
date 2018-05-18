//
// Copyright (c) 2017 Juniper Networks, Inc. All Rights Reserved.
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

package driver

import (
	contrail "github.com/Juniper/contrail-go-api"
	"github.com/Juniper/contrail-go-api/types"
)

// TODO: This interface can be simplified
type ControllerPort interface {
	NewProject(domain, tenant string) (*types.Project, error)
	NewDefaultProject(tenant string) (*types.Project, error)

	CreateNetworkWithSubnet(tenantName, networkName, subnetCIDR string) (*types.VirtualNetwork, error)
	GetNetwork(tenantName, networkName string) (*types.VirtualNetwork, error)
	GetIpamSubnet(net *types.VirtualNetwork, CIDR string) (*types.IpamSubnetType, error)
	GetDefaultGatewayIp(subnet *types.IpamSubnetType) (string, error)

	GetOrCreateInstance(vif *types.VirtualMachineInterface, containerId string) (*types.VirtualMachine, error)
	GetInstance(containerId string) (*types.VirtualMachine, error)

	GetExistingInterface(net *types.VirtualNetwork, tenantName, containerId string) (*types.VirtualMachineInterface, error)
	GetOrCreateInterface(net *types.VirtualNetwork, tenantName, containerId string) (*types.VirtualMachineInterface, error)
	GetInterfaceMac(iface *types.VirtualMachineInterface) (string, error)

	GetOrCreateInstanceIp(net *types.VirtualNetwork,
		iface *types.VirtualMachineInterface, subnetUuid string) (*types.InstanceIp, error)

	DeleteElementRecursive(parent contrail.IObject) error
}

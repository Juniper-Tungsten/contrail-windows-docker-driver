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

package controller_rest

import (
	"net"

	contrail "github.com/Juniper/contrail-go-api"
	"github.com/Juniper/contrail-go-api/types"
	"github.com/Juniper/contrail-windows-docker-driver/core/ports"
	log "github.com/sirupsen/logrus"
)

// ControllerAdapter is a facade for complicated ControllerAdapterImpl. It exist, because
// there are lots of tests for ControllerAdapterImpl class, and it would be hard to modify it
// directly without throwing away many tests.
type ControllerAdapter struct {
	controller *ControllerAdapterImpl
}

func newControllerAdapter(controller *ControllerAdapterImpl) *ControllerAdapter {
	return &ControllerAdapter{
		controller: controller,
	}
}

func (c *ControllerAdapter) NewProject(domain, tenant string) (*types.Project, error) {
	return c.controller.NewProject(domain, tenant)
}

func (c *ControllerAdapter) CreateNetworkWithSubnet(tenantName, networkName, subnetCIDR string) (*types.VirtualNetwork, error) {
	return c.controller.CreateNetworkWithSubnet(tenantName, networkName, subnetCIDR)
}

func (c *ControllerAdapter) GetNetwork(tenantName, networkName string) (*types.VirtualNetwork, error) {
	return c.controller.GetNetwork(tenantName, networkName)
}

func (c *ControllerAdapter) GetNetworkWithSubnet(tenantName, networkName, subnetCIDR string) (*types.VirtualNetwork, *types.IpamSubnetType, error) {
	network, err := c.controller.GetNetwork(tenantName, networkName)
	if err != nil {
		return nil, nil, err
	}

	log.Infoln("Got Contrail network", network.GetDisplayName())

	ipamSubnet, err := c.controller.GetIpamSubnet(network, subnetCIDR)
	if err != nil {
		return nil, nil, err
	}

	return network, ipamSubnet, nil
}

func (c *ControllerAdapter) GetDefaultGatewayIp(ipamSubnet *types.IpamSubnetType) (string, error) {
	return c.controller.GetDefaultGatewayIp(ipamSubnet)
}

func (c *ControllerAdapter) CreateContainerInSubnet(tenantName, containerID string,
	network *types.VirtualNetwork, ipamSubnet *types.IpamSubnetType) (*ports.ContrailContainer, error) {

	vif, err := c.controller.GetOrCreateInterface(network, tenantName, containerID)
	if err != nil {
		return nil, err
	}

	vm, err := c.controller.GetOrCreateInstance(vif, containerID)
	if err != nil {
		return nil, err
	}

	ipobj, err := c.controller.GetOrCreateInstanceIp(network, vif, ipamSubnet.SubnetUuid)
	if err != nil {
		return nil, err
	}
	ip := ipobj.GetInstanceIpAddress()
	log.Debugln("Retrieved instance IP:", ip)

	mac, err := c.controller.GetInterfaceMac(vif)
	log.Debugln("Retrieved MAC:", mac)
	if err != nil {
		return nil, err
	}

	container := &ports.ContrailContainer{
		IP:        net.ParseIP(ip),
		PrefixLen: ipamSubnet.Subnet.IpPrefixLen,
		Mac:       mac,
		VmUUID:    vm.GetUuid(),
		VmiUUID:   vif.GetUuid(),
	}

	return container, nil
}

func (c *ControllerAdapter) GetInstance(containerId string) (*types.VirtualMachine, error) {
	return c.controller.GetInstance(containerId)
}

func (c *ControllerAdapter) GetExistingInterface(net *types.VirtualNetwork, tenantName, containerId string) (*types.VirtualMachineInterface, error) {
	return c.controller.GetExistingInterface(net, tenantName, containerId)
}

func (c *ControllerAdapter) DeleteElementRecursive(parent contrail.IObject) error {
	return c.controller.DeleteElementRecursive(parent)
}

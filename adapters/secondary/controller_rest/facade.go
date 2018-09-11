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
	"errors"
	"net"

	"github.com/Juniper/contrail-go-api/types"
	"github.com/Juniper/contrail-windows-docker-driver/core/model"
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

func (c *ControllerAdapter) GetProject(domain, tenant string) (*types.Project, error) {
	return c.controller.GetProject(domain, tenant)
}

func (c *ControllerAdapter) CreateNetworkWithSubnet(tenantName, networkName, subnetCIDR string) (*types.VirtualNetwork, error) {
	return c.controller.CreateNetworkWithSubnet(tenantName, networkName, subnetCIDR)
}

func (c *ControllerAdapter) GetNetwork(tenantName, networkName string) (*types.VirtualNetwork, error) {
	return c.controller.GetNetwork(tenantName, networkName)
}

func (c *ControllerAdapter) GetNetworkWithSubnet(tenantName, networkName, subnetCIDR string) (*model.Network, error) {
	network, ipamSubnet, err := c.controller.GetNetworkWithSubnet(tenantName, networkName, subnetCIDR)
	if err != nil {
		return nil, err
	}

	subnet := model.Subnet{
		CIDR:      c.controller.getCidrFromIpamSubnet(ipamSubnet),
		DefaultGW: ipamSubnet.DefaultGateway,
	}

	net := &model.Network{
		NetworkName: network.GetName(),
		TenantName:  tenantName,
		Subnet:      subnet,
	}

	return net, nil
}

func (c *ControllerAdapter) GetDefaultGatewayIp(ipamSubnet *types.IpamSubnetType) (string, error) {
	return c.controller.GetDefaultGatewayIp(ipamSubnet)
}

func (c *ControllerAdapter) CreateContainerInSubnet(network *model.Network, containerID string) (*model.Container, error) {
	retreivedNetwork, ipamSubnet, err := c.controller.GetNetworkWithSubnet(network.TenantName, network.NetworkName, network.Subnet.CIDR)
	if err != nil {
		return nil, err
	}

	vif, err := c.controller.GetOrCreateInterface(retreivedNetwork, network.TenantName, containerID)
	if err != nil {
		return nil, err
	}

	vm, err := c.controller.GetOrCreateInstance(vif, containerID)
	if err != nil {
		return nil, err
	}

	ipobj, err := c.controller.GetOrCreateInstanceIp(retreivedNetwork, vif, ipamSubnet.SubnetUuid)
	if err != nil {
		return nil, err
	}
	ip := ipobj.GetInstanceIpAddress()
	mac, err := c.controller.GetInterfaceMac(vif)
	if err != nil {
		return nil, err
	}

	gateway := ipamSubnet.DefaultGateway
	if gateway == "" {
		return nil, errors.New("Default GW is empty")
	}

	container := &model.Container{
		IP:        net.ParseIP(ip),
		PrefixLen: ipamSubnet.Subnet.IpPrefixLen,
		Mac:       mac,
		Gateway:   gateway,
		VmUUID:    vm.GetUuid(),
		VmiUUID:   vif.GetUuid(),
		NetUUID:   retreivedNetwork.GetUuid(),
	}

	log.Debugln("Container:", container)
	return container, nil
}

func (c *ControllerAdapter) GetInstance(containerId string) (*types.VirtualMachine, error) {
	return c.controller.GetInstance(containerId)
}

func (c *ControllerAdapter) GetExistingInterface(net *types.VirtualNetwork, tenantName, containerId string) (*types.VirtualMachineInterface, error) {
	return c.controller.GetExistingInterface(net, tenantName, containerId)
}

func (c *ControllerAdapter) DeleteContainer(containerID string) error {
	return c.controller.DeleteContainer(containerID)
}

func (c *ControllerAdapter) GetContainer(containerID string) (*model.Container, error) {

	vm, err := c.controller.GetInstance(containerID)
	if err != nil {
		return nil, err
	}

	vmiRefs, err := vm.GetVirtualMachineInterfaceBackRefs()
	if err != nil {
		return nil, err
	}
	if len(vmiRefs) != 1 {
		return nil, errors.New("For now, only one VMI per endpoint is supported.")
	}
	vmiObj, err := c.controller.ApiClient.FindByUuid("virtual-machine-interface", vmiRefs[0].Uuid)
	if err != nil {
		return nil, err
	}
	vmi := vmiObj.(*types.VirtualMachineInterface)

	iipRefs, err := vmi.GetInstanceIpBackRefs()
	if err != nil {
		return nil, err
	}
	if len(iipRefs) != 1 {
		return nil, errors.New("For now, nly one InstanceIP per endpoint is supported.")
	}
	iipObj, err := c.controller.ApiClient.FindByUuid("instance-ip", iipRefs[0].Uuid)
	if err != nil {
		return nil, err
	}
	iip := iipObj.(*types.InstanceIp)

	vnRefs, err := iip.GetVirtualNetworkRefs()
	if err != nil {
		return nil, err
	}
	if len(vnRefs) != 1 {
		return nil, errors.New("For now, only one virtual network per endpoint is supported.")
	}
	vnObj, err := c.controller.ApiClient.FindByUuid("virtual-network", vnRefs[0].Uuid)
	if err != nil {
		return nil, err
	}
	vn := vnObj.(*types.VirtualNetwork)

	subnet, err := c.findSubnetInNetworkByInstanceIP(iip, vn)
	if err != nil {
		return nil, err
	}

	mac, err := c.controller.GetInterfaceMac(vmi)
	if err != nil {
		return nil, err
	}

	container := &model.Container{
		IP:        net.ParseIP(iip.GetInstanceIpAddress()),
		PrefixLen: subnet.Subnet.IpPrefixLen,
		Mac:       mac,
		Gateway:   subnet.DefaultGateway,
		VmUUID:    vm.GetUuid(),
		VmiUUID:   vmi.GetUuid(),
		NetUUID:   vn.GetUuid(),
	}

	log.Debugln("Container:", container)
	return container, nil
}

func (c *ControllerAdapter) findSubnetInNetworkByInstanceIP(iip *types.InstanceIp, vn *types.VirtualNetwork) (*types.IpamSubnetType, error) {
	subnetUuidToFind := iip.GetSubnetUuid()
	ipamRefs, err := vn.GetNetworkIpamRefs()
	if err != nil {
		return nil, err
	}
	for idx, _ := range ipamRefs {
		subnets := ipamRefs[idx].Attr.(types.VnSubnetsType).IpamSubnets
		for idx, _ := range subnets {
			if subnets[idx].SubnetUuid == subnetUuidToFind {
				return &subnets[idx], nil
			}
		}
	}
	return nil, errors.New("Subnet not found")
}

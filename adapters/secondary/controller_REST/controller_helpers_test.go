// +build unit
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

package controller_rest_test

import (
	"fmt"

	contrail "github.com/Juniper/contrail-go-api"
	"github.com/Juniper/contrail-go-api/config"
	"github.com/Juniper/contrail-go-api/types"
	"github.com/Juniper/contrail-windows-docker-driver/adapters/secondary/controller_rest"
	"github.com/Juniper/contrail-windows-docker-driver/adapters/secondary/controller_rest/api"
	"github.com/Juniper/contrail-windows-docker-driver/common"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"
)

func NewTestClientAndProject(tenant string) (*controller_rest.ControllerAdapterImpl, *types.Project) {
	fakeApiClient := api.NewFakeApiClient()
	c := controller_rest.NewControllerAdapterImpl(fakeApiClient)

	project, err := c.NewProject(common.DomainName, tenant)
	Expect(err).ToNot(HaveOccurred())
	return c, project
}

func CreateTestNetworkWithSubnet(c contrail.ApiClient, netName, subnetCIDR string,
	project *types.Project) *types.VirtualNetwork {
	netUUID, err := config.CreateNetworkWithSubnet(c, project.GetUuid(), netName, subnetCIDR)
	Expect(err).ToNot(HaveOccurred())
	Expect(netUUID).ToNot(BeNil())
	testNetwork, err := types.VirtualNetworkByUuid(c, netUUID)
	Expect(err).ToNot(HaveOccurred())
	Expect(testNetwork).ToNot(BeNil())
	return testNetwork
}

func CreateTestNetwork(c contrail.ApiClient, netName string,
	project *types.Project) *types.VirtualNetwork {
	netUUID, err := config.CreateNetwork(c, project.GetUuid(), netName)
	Expect(err).ToNot(HaveOccurred())
	Expect(netUUID).ToNot(BeNil())
	testNetwork, err := types.VirtualNetworkByUuid(c, netUUID)
	Expect(err).ToNot(HaveOccurred())
	Expect(testNetwork).ToNot(BeNil())
	return testNetwork
}

func RemoveTestSecurityGroup(c contrail.ApiClient, groupName string,
	project *types.Project) {
	secGroupFqName := fmt.Sprintf("%s:%s:default", common.DomainName, tenantName)
	secGroup, err := types.SecurityGroupByName(c, secGroupFqName)
	err = c.Delete(secGroup)
	Expect(err).ToNot(HaveOccurred())
}

func AddSubnetWithDefaultGateway(c contrail.ApiClient, subnetPrefix, defaultGW string,
	subnetMask int, testNetwork *types.VirtualNetwork) {
	subnet := &types.IpamSubnetType{
		Subnet:         &types.SubnetType{IpPrefix: subnetPrefix, IpPrefixLen: subnetMask},
		DefaultGateway: defaultGW,
	}

	var ipamSubnets types.VnSubnetsType
	ipamSubnets.AddIpamSubnets(subnet)

	ipam, err := c.FindByName("network-ipam",
		"default-domain:default-project:default-network-ipam")
	Expect(err).ToNot(HaveOccurred())
	err = testNetwork.AddNetworkIpam(ipam.(*types.NetworkIpam), ipamSubnets)
	Expect(err).ToNot(HaveOccurred())

	err = c.Update(testNetwork)
	Expect(err).ToNot(HaveOccurred())
}

func CreateTestInstance(c contrail.ApiClient, vif *types.VirtualMachineInterface,
	containerID string) *types.VirtualMachine {
	testInstance := new(types.VirtualMachine)
	testInstance.SetName(containerID)
	err := c.Create(testInstance)
	Expect(err).ToNot(HaveOccurred())

	createdInstance, err := c.FindByName("virtual-machine", containerID)
	Expect(err).ToNot(HaveOccurred())

	err = vif.AddVirtualMachine(createdInstance.(*types.VirtualMachine))
	Expect(err).ToNot(HaveOccurred())
	err = c.Update(vif)
	Expect(err).ToNot(HaveOccurred())

	return testInstance
}

func CreateMockedInterface(c contrail.ApiClient, net *types.VirtualNetwork, tenantName,
	containerId string) *types.VirtualMachineInterface {
	iface := new(types.VirtualMachineInterface)

	iface.SetFQName("project", []string{common.DomainName, tenantName, containerId})

	err := iface.AddVirtualNetwork(net)
	Expect(err).ToNot(HaveOccurred())
	err = c.Create(iface)
	Expect(err).ToNot(HaveOccurred())
	return iface
}

func AddMacToInterface(c contrail.ApiClient, ifaceMac string,
	iface *types.VirtualMachineInterface) {
	macs := new(types.MacAddressesType)
	macs.AddMacAddress(ifaceMac)
	iface.SetVirtualMachineInterfaceMacAddresses(macs)
	err := c.Update(iface)
	Expect(err).ToNot(HaveOccurred())
}

func CreateTestInstanceIP(c contrail.ApiClient, tenantName string,
	iface *types.VirtualMachineInterface,
	net *types.VirtualNetwork) *types.InstanceIp {
	instIP := &types.InstanceIp{}
	instIP.SetName(iface.GetName())
	err := instIP.AddVirtualNetwork(net)
	Expect(err).ToNot(HaveOccurred())
	err = instIP.AddVirtualMachineInterface(iface)
	Expect(err).ToNot(HaveOccurred())
	err = c.Create(instIP)
	Expect(err).ToNot(HaveOccurred())

	allocatedIP, err := types.InstanceIpByUuid(c, instIP.GetUuid())
	Expect(err).ToNot(HaveOccurred())
	return allocatedIP
}

func ForceDeleteProject(c *controller_rest.ControllerAdapterImpl, tenant string) {
	projToDelete, _ := c.ApiClient.FindByName("project", fmt.Sprintf("%s:%s", common.DomainName,
		tenant))
	if projToDelete != nil {
		c.DeleteElementRecursive(projToDelete)
	}
}

func CleanupLingeringVM(c *controller_rest.ControllerAdapterImpl, containerID string) {
	instance, err := types.VirtualMachineByName(c.ApiClient, containerID)
	if err == nil {
		log.Debugln("Cleaning up lingering test vm", instance.GetUuid())
		c.DeleteElementRecursive(instance)
	}
}

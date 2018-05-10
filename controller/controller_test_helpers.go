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

package controller

import (
	"fmt"

	contrail "github.com/Juniper/contrail-go-api"
	"github.com/Juniper/contrail-go-api/config"
	"github.com/Juniper/contrail-go-api/mocks"
	"github.com/Juniper/contrail-go-api/types"
	"github.com/Juniper/contrail-windows-docker-driver/adapters/secondary/auth"
	"github.com/Juniper/contrail-windows-docker-driver/common"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"
)

func TestKeystoneParams() *auth.KeystoneParams {
	keys := &auth.KeystoneParams{}
	keys.LoadFromEnvironment()
	// try using env variables first, and if they aren't set, use the hardcoded values.
	if keys.Os_auth_url != "" {
		log.Warn("OS_AUTH_URL is SET, will use env variables for Keystone auth during testing")
		return keys
	} else {
		return &auth.KeystoneParams{
			Os_auth_url:    "http://10.7.0.54:5000/v2.0",
			Os_username:    "admin",
			Os_tenant_name: "admin",
			Os_password:    "secret123",
			Os_token:       "",
		}
	}
}

func NewMockedClientAndProject(tenant string) (*Controller, *types.Project) {
	c := &Controller{}
	mockedApiClient := new(mocks.ApiClient)
	mockedApiClient.Init()
	c.ApiClient = mockedApiClient

	project := new(types.Project)
	project.SetFQName("domain", []string{common.DomainName, tenant})
	err := c.ApiClient.Create(project)
	Expect(err).ToNot(HaveOccurred())
	return c, project
}

func NewClientAndProject(tenant, controllerAddr string, controllerPort int) (*Controller,
	*types.Project) {
	keys := TestKeystoneParams()
	keystone, err := auth.NewKeystoneAuth(keys)
	Expect(err).ToNot(HaveOccurred())

	err = keystone.Authenticate()
	Expect(err).ToNot(HaveOccurred())

	c, err := NewController(controllerAddr, controllerPort, keystone)
	Expect(err).ToNot(HaveOccurred())

	ForceDeleteProject(c, tenant)

	project := new(types.Project)
	project.SetFQName("domain", []string{common.DomainName, tenant})
	Expect(err).ToNot(HaveOccurred())
	err = c.ApiClient.Create(project)
	Expect(err).ToNot(HaveOccurred())
	return c, project
}

func CreateMockedNetworkWithSubnet(c contrail.ApiClient, netName, subnetCIDR string,
	project *types.Project) *types.VirtualNetwork {
	netUUID, err := config.CreateNetworkWithSubnet(c, project.GetUuid(), netName, subnetCIDR)
	Expect(err).ToNot(HaveOccurred())
	Expect(netUUID).ToNot(BeNil())
	testNetwork, err := types.VirtualNetworkByUuid(c, netUUID)
	Expect(err).ToNot(HaveOccurred())
	Expect(testNetwork).ToNot(BeNil())
	return testNetwork
}

func CreateMockedNetwork(c contrail.ApiClient, netName string,
	project *types.Project) *types.VirtualNetwork {
	netUUID, err := config.CreateNetwork(c, project.GetUuid(), netName)
	Expect(err).ToNot(HaveOccurred())
	Expect(netUUID).ToNot(BeNil())
	testNetwork, err := types.VirtualNetworkByUuid(c, netUUID)
	Expect(err).ToNot(HaveOccurred())
	Expect(testNetwork).ToNot(BeNil())
	return testNetwork
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

func CreateMockedInstance(c contrail.ApiClient, vif *types.VirtualMachineInterface,
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

func CreateMockedInstanceIP(c contrail.ApiClient, tenantName string,
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

func ForceDeleteProject(c *Controller, tenant string) {
	projToDelete, _ := c.ApiClient.FindByName("project", fmt.Sprintf("%s:%s", common.DomainName,
		tenant))
	if projToDelete != nil {
		c.DeleteElementRecursive(projToDelete)
	}
}

func CleanupLingeringVM(c *Controller, containerID string) {
	instance, err := types.VirtualMachineByName(c.ApiClient, containerID)
	if err == nil {
		log.Debugln("Cleaning up lingering test vm", instance.GetUuid())
		c.DeleteElementRecursive(instance)
	}
}

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

// +build unit

package controller_rest_test

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	contrail "github.com/Juniper/contrail-go-api"
	"github.com/Juniper/contrail-go-api/config"
	"github.com/Juniper/contrail-go-api/types"
	"github.com/Juniper/contrail-windows-docker-driver/adapters/secondary/controller_rest"
	"github.com/Juniper/contrail-windows-docker-driver/adapters/secondary/controller_rest/api"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"
)

func NewTestClientAndProject(tenant string) (*controller_rest.ControllerAdapterImpl, *types.Project) {
	fakeApiClient := api.NewFakeApiClient()
	c := controller_rest.NewControllerAdapterImpl(fakeApiClient)
	project := NewTestProject(c, controller_rest.DomainName, tenant)
	return c, project
}

func NewTestProject(c *controller_rest.ControllerAdapterImpl, domainName, tenantName string) *types.Project {
	testProject, err := c.NewProject(domainName, tenantName)
	Expect(err).ToNot(HaveOccurred())
	return testProject
}

func CreateTestProject(c contrail.ApiClient, domainName, tenantName string) *types.Project {
	project := new(types.Project)
	project.SetFQName("domain", []string{domainName, tenantName})
	err := c.Create(project)
	Expect(err).ToNot(HaveOccurred())
	Expect(project).ToNot(BeNil())
	return project
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

func CreateTestIPAMWithDNS(c contrail.ApiClient, project *types.Project, ipamName, dnsType string) *types.NetworkIpam {
	ipam := new(types.NetworkIpam)
	ipam.SetFQName("project", []string{controller_rest.DomainName, project.GetName(), ipamName})
	ipamType := &types.IpamType{
		IpamDnsMethod: dnsType,
	}
	ipam.SetNetworkIpamMgmt(ipamType)
	err := c.Create(ipam)
	Expect(err).ToNot(HaveOccurred())
	ipamFqName := fmt.Sprintf("%s:%s:%s", controller_rest.DomainName, project.GetName(), ipamName)
	createdIpam, err := types.NetworkIpamByName(c, ipamFqName)
	Expect(err).ToNot(HaveOccurred())
	return createdIpam
}

func AddTenantDNSAddresses(c contrail.ApiClient, testIPAM *types.NetworkIpam, dnsList []string) {
	ipamType := testIPAM.GetNetworkIpamMgmt()
	ipAddrType := &types.IpAddressesType{}
	for _, dns := range dnsList {
		ipAddrType.AddIpAddress(dns)
	}
	ipamDnsAddr := &types.IpamDnsAddressType{
		TenantDnsServerAddress: ipAddrType,
	}
	ipamType.IpamDnsServer = ipamDnsAddr
	testIPAM.SetNetworkIpamMgmt(&ipamType)
	c.Update(testIPAM)
}

func RemoveTestSecurityGroup(c contrail.ApiClient, groupName string,
	project *types.Project) {
	secGroupFqName := fmt.Sprintf("%s:%s:%s", controller_rest.DomainName, tenantName,
		controller_rest.DefaultSecurityGroup)
	secGroup, err := types.SecurityGroupByName(c, secGroupFqName)
	err = c.Delete(secGroup)
	Expect(err).ToNot(HaveOccurred())
}

func AddSubnetToNetwork(c contrail.ApiClient, testNetwork *types.VirtualNetwork, testIPAM *types.NetworkIpam,
	subnetPrefix, defaultGW, dnsServer string, subnetMask int, enableDHCP bool) *types.IpamSubnetType {
	subnet := &types.IpamSubnetType{
		Subnet:           &types.SubnetType{IpPrefix: subnetPrefix, IpPrefixLen: subnetMask},
		DefaultGateway:   defaultGW,
		DnsServerAddress: dnsServer,
		EnableDhcp:       enableDHCP,
	}

	var ipamSubnets types.VnSubnetsType
	ipamSubnets.AddIpamSubnets(subnet)

	err := testNetwork.AddNetworkIpam(testIPAM, ipamSubnets)
	Expect(err).ToNot(HaveOccurred())

	err = c.Update(testNetwork)
	Expect(err).ToNot(HaveOccurred())
	return subnet
}

func AddSubnetWithDefaultGateway(c contrail.ApiClient, subnetPrefix, defaultGW string, subnetMask int,
	testNetwork *types.VirtualNetwork) {
	ipamFqName := fmt.Sprintf("%s:%s:%s", controller_rest.DomainName, controller_rest.DefaultProject,
		controller_rest.DefaultIPAM)
	ipam, err := types.NetworkIpamByName(c, ipamFqName)
	Expect(err).ToNot(HaveOccurred())
	AddSubnetToNetwork(c, testNetwork, ipam, subnetPrefix, defaultGW, "", subnetMask, false)
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

	iface.SetFQName("project", []string{controller_rest.DomainName, tenantName, containerId})

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
	projToDelete, _ := c.ApiClient.FindByName("project", fmt.Sprintf("%s:%s", controller_rest.DomainName,
		tenant))
	if projToDelete != nil {
		ForceDeleteElementRecursive(c, projToDelete)
	}
}

func CleanupLingeringVM(c *controller_rest.ControllerAdapterImpl, containerID string) {
	instance, err := types.VirtualMachineByName(c.ApiClient, containerID)
	if err == nil {
		log.Debugln("Cleaning up lingering test vm", instance.GetUuid())
		ForceDeleteElementRecursive(c, instance)
	}
}

// This regex finds all strings like:
// `virtual-network/23e300f4-ab1a-4d97-a1d9-9ed69b601e17`
var resourceUriRegexp = regexp.MustCompile(
	"([a-z-]*\\/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})")

func ForceDeleteElementRecursive(c *controller_rest.ControllerAdapterImpl, parent contrail.IObject) error {
	log.Debugln("Deleting", parent.GetType(), parent.GetUuid())
	for err := c.ApiClient.Delete(parent); err != nil; err = c.ApiClient.Delete(parent) {
		// TODO: when fixing this method, consider using c.is404() method.
		if strings.Contains(err.Error(), "404 Resource") {
			log.Errorln("Failed to delete Contrail resource", err.Error())
			break
		} else if strings.Contains(err.Error(), "409 Conflict") {
			msg := err.Error()
			// example error message when object has children:
			// `409 Conflict: Delete when children still present:
			// ['http://10.7.0.54:8082/virtual-network/23e300f4-ab1a-4d97-a1d9-9ed69b601e17']`
			found := resourceUriRegexp.FindAll([]byte(msg), -1)

			for _, f := range found {
				split := strings.Split(string(f), "/")
				typename := split[0]
				UUID := split[1]
				var child contrail.IObject
				child, err = c.ApiClient.FindByUuid(typename, UUID)
				if err != nil {
					return err
				}
				if child == nil {
					return errors.New("Child object is nil")
				}
				err = ForceDeleteElementRecursive(c, child)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

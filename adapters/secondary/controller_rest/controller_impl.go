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
	"fmt"
	"net/http"
	"strconv"
	"strings"

	contrail "github.com/Juniper/contrail-go-api"
	"github.com/Juniper/contrail-go-api/config"
	"github.com/Juniper/contrail-go-api/types"
	log "github.com/sirupsen/logrus"
)

const (
	// Default resources in Contrail
	DomainName           = "default-domain"
	DefaultProject       = "default-project"
	DefaultSecurityGroup = "default"
	DefaultIPAM          = "default-network-ipam"
	// Admin project in Contrail
	AdminProject = "admin"
)

type ControllerAdapterImpl struct {
	ApiClient contrail.ApiClient
}

// TODO: this factory function is public only because we have a bunch of tests for
// ControllerAdapterImpl methods. We should instead test via the Facade - ControllerAdapter.
// I decided against rewriting all the tests for now, but it would be an improvement.
// Right now, ControllerAdapterImpl test the implementation, and not the behaviour.
func NewControllerAdapterImpl(apiClient contrail.ApiClient) *ControllerAdapterImpl {
	client := &ControllerAdapterImpl{ApiClient: apiClient}
	return client
}

func (c *ControllerAdapterImpl) NewProject(domain, tenant string) (*types.Project, error) {
	project := new(types.Project)
	project.SetFQName("domain", []string{domain, tenant})
	if err := c.ApiClient.Create(project); err != nil {
		return nil, err
	}

	// Create security group and network ipam as soon as project is created.
	// This reflects contrail orchestrator plugins' behaviour.
	secGroup, err := c.createSecurityGroup(domain, tenant, DefaultSecurityGroup)
	if err != nil {
		if warn := c.ApiClient.Delete(project); warn != nil {
			log.Warnf("Failed to delete project %s after failed default security group creation: %v", tenant, warn)
		}
		return nil, err
	}
	if _, err := c.createNetworkIPAM(domain, tenant, DefaultIPAM, "default-dns-server"); err != nil {
		if warn := c.ApiClient.Delete(secGroup); warn != nil {
			log.Warnf("Failed to delete default security group after failed default IPAM creation: %v", warn)
		}
		if warn := c.ApiClient.Delete(project); warn != nil {
			log.Warnf("Failed to delete project %s after failed default IPAM creation: %v", tenant, warn)
		}
		return nil, err
	}

	return project, nil
}

func (c *ControllerAdapterImpl) GetOrCreateProject(domain, tenant string) (*types.Project, error) {
	project, err := c.GetProject(domain, tenant)
	if err == nil && project != nil {
		return project, nil
	}
	return c.NewProject(domain, tenant)
}

func (c *ControllerAdapterImpl) GetProject(domain, tenant string) (*types.Project, error) {
	projectFQName := fmt.Sprintf("%s:%s", domain, tenant)
	project, err := types.ProjectByName(c.ApiClient, projectFQName)
	if err != nil {
		return nil, err
	}
	return project, nil
}

func (c *ControllerAdapterImpl) createSecurityGroup(domain, tenant, name string) (*types.SecurityGroup, error) {
	secgroup := new(types.SecurityGroup)
	secgroup.SetFQName("project", []string{domain, tenant, name})
	if err := c.ApiClient.Create(secgroup); err != nil {
		return nil, err
	}
	return secgroup, nil
}

func (c *ControllerAdapterImpl) createNetworkIPAM(domain, tenant, name, ipamDnsMethod string) (*types.NetworkIpam, error) {
	ipam := new(types.NetworkIpam)
	ipamType := &types.IpamType{
		IpamDnsMethod: ipamDnsMethod}
	ipam.SetNetworkIpamMgmt(ipamType)
	ipam.SetFQName("project", []string{domain, tenant, name})
	if err := c.ApiClient.Create(ipam); err != nil {
		return nil, err
	}
	return ipam, nil
}

func (c *ControllerAdapterImpl) CreateNetworkWithSubnet(tenantName, networkName, subnetCIDR string) (*types.VirtualNetwork, error) {
	net, err := c.GetNetwork(tenantName, networkName)
	if err == nil {
		showDetails := true
		details, err := config.NetworkShow(c.ApiClient, net.GetUuid(), showDetails)
		if err != nil {
			return nil, err
		}
		for _, existingCIDR := range details.Subnets {
			if subnetCIDR == existingCIDR {
				return net, nil
			}
		}
		return nil, errors.New("such network already has a different subnet")
	}
	projectFQName := fmt.Sprintf("%s:%s", DomainName, tenantName)
	project, err := c.ApiClient.FindByName("project", projectFQName)
	if err != nil {
		return nil, err
	}

	netUUID, err := config.CreateNetworkWithSubnet(c.ApiClient, project.GetUuid(), networkName, subnetCIDR)
	if err != nil {
		return nil, err
	}

	net, err = types.VirtualNetworkByUuid(c.ApiClient, netUUID)
	if err != nil {
		return nil, err
	}
	return net, nil
}

func (c *ControllerAdapterImpl) GetNetwork(tenantName, networkName string) (*types.VirtualNetwork,
	error) {
	name := fmt.Sprintf("%s:%s:%s", DomainName, tenantName, networkName)
	net, err := types.VirtualNetworkByName(c.ApiClient, name)
	if err != nil {
		log.Errorf("Failed to get virtual network %s by name: %v", name, err)
		return nil, err
	}
	return net, nil
}

func (c *ControllerAdapterImpl) GetNetworkWithSubnet(tenantName, networkName, subnetCIDR string) (
	*types.VirtualNetwork, *types.IpamSubnetType, error) {
	network, err := c.GetNetwork(tenantName, networkName)
	if err != nil {
		return nil, nil, err
	}

	log.Debugln("Got Contrail network", network.GetName())

	ipamSubnet, err := c.GetIpamSubnet(network, subnetCIDR)
	if err != nil {
		return nil, nil, err
	}
	return network, ipamSubnet, nil
}

// GetIpamSubnet returns IPAM subnet of specified virtual network with specified CIDR.
// If virtual network has only one subnet, CIDR is ignored.
func (c *ControllerAdapterImpl) GetIpamSubnet(net *types.VirtualNetwork, CIDR string) (
	*types.IpamSubnetType, error) {

	if strings.HasPrefix(CIDR, "0.0.0.0") {
		// this means that the user didn't provide a subnet
		CIDR = ""
	}

	ipamReferences, err := net.GetNetworkIpamRefs()
	if err != nil {
		log.Errorf("Failed to get ipam references: %v", err)
		return nil, err
	}

	var allIpamSubnets []types.IpamSubnetType
	for _, ref := range ipamReferences {
		attribute := ref.Attr
		ipamSubnets := attribute.(types.VnSubnetsType).IpamSubnets
		for _, ipamSubnet := range ipamSubnets {
			allIpamSubnets = append(allIpamSubnets, ipamSubnet)
		}
	}

	if len(allIpamSubnets) == 0 {
		err = errors.New("No Ipam subnets found")
		log.Error(err)
		return nil, err
	}

	if CIDR == "" {
		if len(allIpamSubnets) > 1 {
			err = errors.New("Didn't specify subnet CIDR and there are multiple Contrail subnets")
			log.Error(err)
			return nil, err
		}
		// return the one and only subnet
		return &allIpamSubnets[0], nil
	}

	// there are multiple subnets to choose from
	for _, ipam := range allIpamSubnets {

		thisCIDR := fmt.Sprintf("%s/%v", ipam.Subnet.IpPrefix,
			ipam.Subnet.IpPrefixLen)

		if thisCIDR == CIDR {
			return &ipam, nil
		}
	}

	err = errors.New("Subnet with specified CIDR not found")
	log.Error(err)
	return nil, err
}

func (c *ControllerAdapterImpl) GetDefaultGatewayIp(subnet *types.IpamSubnetType) (string, error) {
	gw := subnet.DefaultGateway
	if gw == "" {
		err := errors.New("Default GW is empty")
		log.Error(err)
		return "", err
	}
	return gw, nil
}

func (c *ControllerAdapterImpl) GetDNSAddresses(net *types.VirtualNetwork, subnet *types.IpamSubnetType) ([]string, error) {

	// if DHCP is enabled there is no point in setting DNS field in HNS Network as DNS configuration will be
	// passed through DHCP, however, for now, agent doesn't answer DNS requests if DHCP is not enabled in network.
	// if subnet.EnableDhcp {
	// 	return []string{}, fmt.Errorf("DHCP is enabled in subnet %s, DNS will not be set", subnet.SubnetUuid)
	// }

	ipam, err := c.getIPAMForSubnet(net, subnet)
	if err != nil {
		return []string{}, err
	}
	ipamType := ipam.GetNetworkIpamMgmt()
	switch ipamType.IpamDnsMethod {
	case "tenant-dns-server":
		if ipamType.IpamDnsServer != nil && ipamType.IpamDnsServer.TenantDnsServerAddress != nil {
			return ipamType.IpamDnsServer.TenantDnsServerAddress.IpAddress, nil
		}
		return []string{}, nil
	case "virtual-dns-server":
		return []string{subnet.DnsServerAddress}, nil
	// Default mode means that default gateway is set as DNS address
	case "default-dns-server":
		return []string{subnet.DefaultGateway}, nil
	case "none":
		return []string{}, nil
	// IpamDnsMethod might not be specified and in that case it will be treated as "none" DNS mode
	default:
		return []string{}, fmt.Errorf("Not supported DNS method %s", ipamType.IpamDnsMethod)
	}
}

func (c *ControllerAdapterImpl) getIPAMForSubnet(net *types.VirtualNetwork, subnet *types.IpamSubnetType) (
	*types.NetworkIpam, error) {
	ipamReferences, err := net.GetNetworkIpamRefs()
	if err != nil {
		log.Errorf("Failed to get ipam references: %v", err)
		return nil, err
	}
	for _, ref := range ipamReferences {
		attribute := ref.Attr
		to := ref.To
		ipamSubnets := attribute.(types.VnSubnetsType).IpamSubnets
		for _, ipamSubnet := range ipamSubnets {
			if ipamSubnet.SubnetUuid == subnet.SubnetUuid {
				return types.NetworkIpamByName(c.ApiClient, strings.Join(to, ":"))
			}
		}
	}
	return nil, fmt.Errorf("Failed to get NetworkIpam for given subnet %s in network %s", subnet.SubnetUuid, net.GetUuid())
}

func (c *ControllerAdapterImpl) getCidrFromIpamSubnet(ipam *types.IpamSubnetType) string {
	return fmt.Sprintf("%s/%v", ipam.Subnet.IpPrefix, ipam.Subnet.IpPrefixLen)
}

func (c *ControllerAdapterImpl) GetOrCreateInstance(vif *types.VirtualMachineInterface, containerId string) (
	*types.VirtualMachine, error) {
	instance, err := c.GetInstance(containerId)
	if err == nil {
		return instance, nil
	} else if !c.isResourceNotFound(err) {
		log.Errorf("Failed to get instance: %v", err)
		return nil, err
	}

	instance = new(types.VirtualMachine)
	instance.SetName(containerId)
	err = c.ApiClient.Create(instance)
	if err != nil {
		log.Errorf("Failed to create instance: %v", err)
		return nil, err
	}

	createdInstance, err := types.VirtualMachineByName(c.ApiClient, containerId)
	if err != nil {
		log.Errorf("Failed to retreive instance %s by name: %v", containerId, err)
		return nil, err
	}
	log.Infoln("Created instance: ", createdInstance.GetFQName())

	err = vif.AddVirtualMachine(createdInstance)
	if err != nil {
		log.Errorf("Failed to add instance to vif")
		return nil, err
	}
	err = c.ApiClient.Update(vif)
	if err != nil {
		log.Errorf("Failed to update vif")
		return nil, err
	}

	return createdInstance, nil
}

func (c *ControllerAdapterImpl) GetInstance(containerId string) (
	*types.VirtualMachine, error) {
	return types.VirtualMachineByName(c.ApiClient, containerId)
}

func (c *ControllerAdapterImpl) GetExistingInterface(net *types.VirtualNetwork, tenantName,
	containerId string) (*types.VirtualMachineInterface, error) {

	fqName := fmt.Sprintf("%s:%s:%s", DomainName, tenantName, containerId)
	iface, err := types.VirtualMachineInterfaceByName(c.ApiClient, fqName)
	if err != nil {
		return nil, err
	}
	if iface != nil {
		return iface, nil
	}

	log.Errorf("Failed to get interface which does not exist")
	return nil, errors.New("Interface does not exist")
}

func (c *ControllerAdapterImpl) GetOrCreateInterface(net *types.VirtualNetwork, tenantName,
	containerId string) (*types.VirtualMachineInterface, error) {

	fqName := fmt.Sprintf("%s:%s:%s", DomainName, tenantName, containerId)
	iface, err := types.VirtualMachineInterfaceByName(c.ApiClient, fqName)
	if err == nil && iface != nil {
		return iface, nil
	}

	iface = new(types.VirtualMachineInterface)
	iface.SetFQName("project", []string{DomainName, tenantName, containerId})
	err = iface.AddVirtualNetwork(net)
	if err != nil {
		log.Errorf("Failed to add network to interface: %v", err)
		return nil, err
	}

	iface.SetPortSecurityEnabled(true)
	err = c.assignDefaultSecurityGroup(iface, tenantName)
	if err != nil {
		log.Errorf("Failed to add security group to interface: %v", err)
		return nil, err
	}

	err = c.ApiClient.Create(iface)
	if err != nil {
		log.Errorf("Failed to create interface: %v", err)
		return nil, err
	}

	createdIface, err := types.VirtualMachineInterfaceByName(c.ApiClient, fqName)
	if err != nil {
		log.Errorf("Failed to retrieve vmi %s by name: %v", fqName, err)
		return nil, err
	}
	log.Infoln("Created instance: ", createdIface.GetFQName())
	return createdIface, nil
}

func (c *ControllerAdapterImpl) assignDefaultSecurityGroup(iface *types.VirtualMachineInterface, tenantName string) error {
	secGroupFqName := fmt.Sprintf("%s:%s:%s", DomainName, tenantName, DefaultSecurityGroup)
	secGroup, err := types.SecurityGroupByName(c.ApiClient, secGroupFqName)
	if err != nil || secGroup == nil {
		return fmt.Errorf("Failed to retrieve security group %s by name: %v", secGroupFqName, err)

	}
	return iface.AddSecurityGroup(secGroup)
}

func (c *ControllerAdapterImpl) GetInterfaceMac(iface *types.VirtualMachineInterface) (string, error) {
	macs := iface.GetVirtualMachineInterfaceMacAddresses()
	if len(macs.MacAddress) == 0 {
		err := errors.New("Empty MAC list")
		log.Error(err)
		return "", err
	}
	return macs.MacAddress[0], nil
}

func (c *ControllerAdapterImpl) GetOrCreateInstanceIp(net *types.VirtualNetwork,
	iface *types.VirtualMachineInterface, subnetUuid, ip string) (*types.InstanceIp, error) {
	instIP, err := types.InstanceIpByName(c.ApiClient, iface.GetName())

	if err == nil && instIP != nil {
		ipCtrl := instIP.GetInstanceIpAddress()

		if len(ip) > 0 && ip != ipCtrl {
			return nil, fmt.Errorf("InstanceIp already exists with IP %s different than given %s", ipCtrl, ip)
		}

		return instIP, nil
	}

	instIP = &types.InstanceIp{}
	instIP.SetName(iface.GetName())
	instIP.SetSubnetUuid(subnetUuid)
	if len(ip) > 0 {
		instIP.SetInstanceIpAddress(ip)
	}

	err = instIP.AddVirtualNetwork(net)
	if err != nil {
		log.Errorf("Failed to add network to instanceIP object: %v", err)
		return nil, err
	}
	err = instIP.AddVirtualMachineInterface(iface)
	if err != nil {
		log.Errorf("Failed to add vmi to instanceIP object: %v", err)
		return nil, err
	}
	err = c.ApiClient.Create(instIP)
	if err != nil {
		log.Errorf("Failed to instanceIP: %v", err)
		return nil, err
	}

	allocatedIP, err := types.InstanceIpByUuid(c.ApiClient, instIP.GetUuid())
	if err != nil {
		log.Errorf("Failed to retreive instanceIP object %s by name: %v", instIP.GetUuid(), err)
		return nil, err
	}
	if len(ip) > 0 && allocatedIP.GetInstanceIpAddress() != ip {
		return nil, fmt.Errorf("Created instanceIp has different ip %s than provided %s", allocatedIP.GetInstanceIpAddress(), ip)
	}

	return allocatedIP, nil
}

func (c *ControllerAdapterImpl) DeleteContainer(containerID string) error {
	log.Debugln("Starting delete procedure of container and related resources", containerID)
	container, err := types.VirtualMachineByName(c.ApiClient, containerID)
	if err != nil {
		return err
	}

	if err := c.removeVMIBackRefsOf(container); err != nil {
		return err
	}

	log.Debugln("Deleting virtual-machine", container.GetName(), container.GetUuid())
	return c.ApiClient.DeleteByUuid("virtual-machine", container.GetUuid())
}

func (c *ControllerAdapterImpl) removeVMIBackRefsOf(container *types.VirtualMachine) error {
	vmiRefs, err := container.GetVirtualMachineInterfaceBackRefs()
	if err != nil {
		return err
	}
	if len(vmiRefs) > 0 {
		log.Debugln("Container has virtual-machine-interface refs")
	}
	for _, vmiRef := range vmiRefs {
		vmi, err := c.ApiClient.FindByUuid("virtual-machine-interface", vmiRef.Uuid)
		if err != nil {
			return err
		}
		if err := c.removeInstanceIPBackRefsOf(vmi.(*types.VirtualMachineInterface)); err != nil {
			return err
		}
		log.Debugln("Deleting virtual-machine-interface", vmi.GetUuid())
		err = c.ApiClient.DeleteByUuid("virtual-machine-interface", vmi.GetUuid())
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *ControllerAdapterImpl) removeInstanceIPBackRefsOf(vmi *types.VirtualMachineInterface) error {
	iipRefs, err := vmi.GetInstanceIpBackRefs()
	if err != nil {
		return err
	}
	if len(iipRefs) > 0 {
		log.Debugln("virtual-machine-interaface has instance-ip refs")
	}
	for _, iipRef := range iipRefs {
		log.Debugln("Deleting instance-ip", iipRef.Uuid)
		if err := c.ApiClient.DeleteByUuid("instance-ip", iipRef.Uuid); err != nil {
			return err
		}
	}
	return nil
}

func (c *ControllerAdapterImpl) isResourceNotFound(err error) bool {
	return strings.HasPrefix(err.Error(), strconv.Itoa(http.StatusNotFound))
}

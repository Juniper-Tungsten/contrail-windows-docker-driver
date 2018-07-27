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
	"regexp"
	"strings"
	"strconv"

	contrail "github.com/Juniper/contrail-go-api"
	"github.com/Juniper/contrail-go-api/config"
	"github.com/Juniper/contrail-go-api/types"
	"github.com/Juniper/contrail-windows-docker-driver/common"
	log "github.com/sirupsen/logrus"
)

type Info struct {
}

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

// TODO: this method is only used by tests - it can probably be removed from ControllerAdapterImpl
// entirely and moved to helpers.
func (c *ControllerAdapterImpl) NewProject(domain, tenant string) (*types.Project, error) {
	project := new(types.Project)
	project.SetFQName("domain", []string{domain, tenant})
	if err := c.ApiClient.Create(project); err != nil {
		return nil, err
	}

	// Create security group as soon as project is created. This mimics contrail API server
	// behaviuor. We can do it here, because NewProject method is used only in tests (see
	// method comment).
	if _, err := c.createSecurityGroup(domain, tenant); err != nil {
		return nil, err
	}

	return project, nil
}

func (c *ControllerAdapterImpl) createSecurityGroup(domain, tenant string) (*types.SecurityGroup, error) {
	secgroup := new(types.SecurityGroup)
	secgroup.SetFQName("project", []string{domain, tenant, "default"})
	if err := c.ApiClient.Create(secgroup); err != nil {
		return nil, err
	}
	return secgroup, nil
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
	projectFQName := fmt.Sprintf("%s:%s", common.DomainName, tenantName)
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
	name := fmt.Sprintf("%s:%s:%s", common.DomainName, tenantName, networkName)
	net, err := types.VirtualNetworkByName(c.ApiClient, name)
	if err != nil {
		log.Errorf("Failed to get virtual network %s by name: %v", name, err)
		return nil, err
	}
	return net, nil
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

	fqName := fmt.Sprintf("%s:%s:%s", common.DomainName, tenantName, containerId)
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

	fqName := fmt.Sprintf("%s:%s:%s", common.DomainName, tenantName, containerId)
	iface, err := types.VirtualMachineInterfaceByName(c.ApiClient, fqName)
	if err == nil && iface != nil {
		return iface, nil
	}

	iface = new(types.VirtualMachineInterface)
	iface.SetFQName("project", []string{common.DomainName, tenantName, containerId})
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
	secGroupFqName := fmt.Sprintf("%s:%s:default", common.DomainName, tenantName)
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
	iface *types.VirtualMachineInterface, subnetUuid string) (*types.InstanceIp, error) {
	instIp, err := types.InstanceIpByName(c.ApiClient, iface.GetName())
	if err == nil && instIp != nil {
		return instIp, nil
	}

	instIp = &types.InstanceIp{}
	instIp.SetName(iface.GetName())
	instIp.SetSubnetUuid(subnetUuid)

	err = instIp.AddVirtualNetwork(net)
	if err != nil {
		log.Errorf("Failed to add network to instanceIP object: %v", err)
		return nil, err
	}
	err = instIp.AddVirtualMachineInterface(iface)
	if err != nil {
		log.Errorf("Failed to add vmi to instanceIP object: %v", err)
		return nil, err
	}
	err = c.ApiClient.Create(instIp)
	if err != nil {
		log.Errorf("Failed to instanceIP: %v", err)
		return nil, err
	}

	allocatedIP, err := types.InstanceIpByUuid(c.ApiClient, instIp.GetUuid())
	if err != nil {
		log.Errorf("Failed to retreive instanceIP object %s by name: %v", instIp.GetUuid(), err)
		return nil, err
	}
	return allocatedIP, nil
}

func (c *ControllerAdapterImpl) DeleteElementRecursive(parent contrail.IObject) error {
	log.Debugln("Deleting", parent.GetType(), parent.GetUuid())
	for err := c.ApiClient.Delete(parent); err != nil; err = c.ApiClient.Delete(parent) {
		if c.isResourceNotFound(err) {
			log.Errorln("Failed to delete Contrail resource", err.Error())
			break
		} else if strings.Contains(err.Error(), "409 Conflict") {
			msg := err.Error()
			// example error message when object has children:
			// `409 Conflict: Delete when children still present:
			// ['http://10.7.0.54:8082/virtual-network/23e300f4-ab1a-4d97-a1d9-9ed69b601e17']`

			// This regex finds all strings like:
			// `virtual-network/23e300f4-ab1a-4d97-a1d9-9ed69b601e17`
			var re *regexp.Regexp
			re, err = regexp.Compile(
				"([a-z-]*\\/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})")
			if err != nil {
				return err
			}
			found := re.FindAll([]byte(msg), -1)

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
				err = c.DeleteElementRecursive(child)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (c *ControllerAdapterImpl) isResourceNotFound(err error) bool {
	return strings.HasPrefix(err.Error(), strconv.Itoa(http.StatusNotFound))
}

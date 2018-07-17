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

// Implemented according to
// https://github.com/docker/libnetwork/blob/master/docs/remote.md

package driver_core

import (
	"errors"
	"fmt"
	"strings"

	// TODO: this import should be removed when making Controller port smaller

	// TODO: this import should be removed

	"github.com/Juniper/contrail-windows-docker-driver/core/model"
	"github.com/Juniper/contrail-windows-docker-driver/core/ports"
	log "github.com/sirupsen/logrus"
)

type ContrailDriverCore struct {
	vrouter ports.VRouter
	// TODO: all these fields below should be made private as we remove the need for them in
	// driver package.
	Controller                 ports.Controller
	Agent                      ports.Agent
	LocalContrailNetworksRepo  ports.LocalContrailNetworkRepository
	LocalContrailEndpointsRepo ports.LocalContrailEndpointRepository
}

func NewContrailDriverCore(vr ports.VRouter, c ports.Controller, a ports.Agent,
	nr ports.LocalContrailNetworkRepository,
	er ports.LocalContrailEndpointRepository) (*ContrailDriverCore, error) {
	core := ContrailDriverCore{
		vrouter:    vr,
		Controller: c,
		Agent:      a,
		LocalContrailNetworksRepo:  nr,
		LocalContrailEndpointsRepo: er,
	}
	if err := core.initializeAdapters(); err != nil {
		return nil, err
	}
	return &core, nil
}

func (core *ContrailDriverCore) initializeAdapters() error {
	return core.vrouter.Initialize()
}

func (core *ContrailDriverCore) CreateNetwork(dockerNetID, tenantName, networkName, subnetCIDR string) error {
	network, ipamSubnet, err := core.Controller.GetNetworkWithSubnet(tenantName, networkName, subnetCIDR)
	if err != nil {
		return err
	}
	if network == nil {
		return errors.New("Retrieved Contrail network is nil")
	}

	log.Debugln("Got Contrail network", network.GetDisplayName())

	gateway := ipamSubnet.DefaultGateway
	if gateway == "" {
		// TODO: this fails in unit tests using contrail-go-api mock. So either:
		// * fix contrail-go-api mock to return *some* default GW
		// * or maybe we should keep going if default GW is empty, as a user may wish not
		//   to specify it. (this is what we do now, and just generate a warning)
		log.Warn("Default GW is empty")
	}

	return core.LocalContrailNetworksRepo.CreateNetwork(dockerNetID, tenantName, networkName, subnetCIDR,
		gateway)
}

func (core *ContrailDriverCore) DeleteNetwork(dockerNetID string) error {
	return core.LocalContrailNetworksRepo.DeleteNetwork(dockerNetID)
}

func (core *ContrailDriverCore) CreateEndpoint(dockerNetID, endpointID string) (*model.Container, error) {

	network, err := core.LocalContrailNetworksRepo.GetNetwork(dockerNetID)
	if err != nil {
		return nil, err
	}

	// WORKAROUND: We need to retreive Container ID here and use it instead of EndpointID as
	// argument to GetOrCreateInstance(). EndpointID is equiv to interface, but in Contrail,
	// we have a "VirtualMachine" in data model. A single VM can be connected to two or more
	// overlay networks, but when we use EndpointID, this won't work. We need something like:
	// containerID := req.Options["vmname"]
	containerID := endpointID

	container, err := core.Controller.CreateContainerInSubnet(network, containerID)
	if err != nil {
		return nil, err
	}

	localEndpoint, err := core.createContainerEndpointInLocalNetwork(container, network, endpointID)
	if err != nil {
		return nil, err
	}

	go func() {
		err := core.associatePort(container, localEndpoint)
		if err != nil {
			log.Error(err.Error())
		}
	}()
	return container, nil
}

func (core *ContrailDriverCore) createContainerEndpointInLocalNetwork(container *model.Container, network *model.Network, name string) (*model.LocalEndpoint, error) {

	hnsEndpointID, err := core.LocalContrailEndpointsRepo.CreateEndpoint(name, container, network)
	if err != nil {
		return nil, err
	}

	// TODO: this can be refactored into some nice mechanism.
	ifName := core.generateFriendlyName(hnsEndpointID)

	ep := &model.LocalEndpoint{
		IfName: ifName,
		Name:   hnsEndpointID,
	}
	return ep, nil
}

func (core *ContrailDriverCore) associatePort(container *model.Container, ep *model.LocalEndpoint) error {
	return core.Agent.AddPort(container.VmUUID, container.VmiUUID, ep.IfName,
		container.Mac, ep.Name, container.IP.String(), container.NetUUID)
}

func (core *ContrailDriverCore) DeleteEndpoint(dockerNetID, endpointID string) error {

	network, err := core.LocalContrailNetworksRepo.GetNetwork(dockerNetID)
	if err != nil {
		return err
	}

	contrailNetwork, err := core.Controller.GetNetwork(network.TenantName, network.NetworkName)
	if err != nil {
		return err
	}
	log.Infoln("Retrieved Contrail network:", contrailNetwork.GetUuid())

	// WORKAROUND: We need to retreive Container ID here and use it instead of EndpointID as
	// argument to GetOrCreateInstance(). EndpointID is equiv to interface, but in Contrail,
	// we have a "VirtualMachine" in data model. A single VM can be connected to two or more
	// overlay networks, but when we use EndpointID, this won't work. We need something like:
	// containerID := req.Options["vmname"]
	containerID := endpointID

	contrailVif, err := core.Controller.GetExistingInterface(contrailNetwork,
		network.TenantName, containerID)
	if err != nil {
		return err
	}

	vifUUID := contrailVif.GetUuid()
	if err != nil {
		log.Warn("When handling DeleteEndpoint, interface wasn't found")
	} else {
		go func() {
			err := core.Agent.DeletePort(vifUUID)
			if err != nil {
				log.Error(err.Error())
			}
		}()
	}

	err = core.Controller.DeleteContainer(containerID)
	if err != nil {
		log.Warn("When handling DeleteEndpoint, failed to remove Contrail vm instance")
	}

	return core.LocalContrailEndpointsRepo.DeleteEndpoint(endpointID)
}

func (core *ContrailDriverCore) generateFriendlyName(hnsEndpointID string) string {
	// Here's how the Forwarding Extension (kernel) can identify interfaces based on their
	// friendly names.
	// Windows Containers have NIC names like "NIC ID abcdef", where abcdef are the first 6 chars
	// of their HNS endpoint ID.
	// Hyper-V Containers have NIC names consisting of two uuids, probably representing utitlity
	// VM's interface and endpoint's interface:
	// "227301f6-bee9-4ae2-8a93-5e900cde3f47--910c5490-bff8-45e3-a2a0-0114ed9903e0"
	// The second UUID (after the "--") is the HNS endpoints ID.

	// For now, we will always send the name in the Windows Containers format, because it probably
	// has enough information to recognize it in kernel (6 first chars of UUID should be enough):
	containerNicID := strings.Split(hnsEndpointID, "-")[0]
	return fmt.Sprintf("Container NIC %s", containerNicID)
}

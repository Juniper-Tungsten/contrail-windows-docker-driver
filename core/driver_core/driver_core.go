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

	"github.com/Juniper/contrail-windows-docker-driver/core/model"
	"github.com/Juniper/contrail-windows-docker-driver/core/ports"
	log "github.com/sirupsen/logrus"
)

type ContrailDriverCore struct {
	vrouter         ports.VRouter
	controller      ports.Controller
	portAssociation ports.PortAssociation
	// TODO: all these fields below should be made private eventually
	LocalContrailNetworksRepo  ports.LocalContrailNetworkRepository
	LocalContrailEndpointsRepo ports.LocalContrailEndpointRepository
}

func NewContrailDriverCore(vr ports.VRouter, c ports.Controller, a ports.PortAssociation,
	nr ports.LocalContrailNetworkRepository,
	er ports.LocalContrailEndpointRepository) (*ContrailDriverCore, error) {
	core := ContrailDriverCore{
		vrouter:                    vr,
		controller:                 c,
		portAssociation:            a,
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
	network, err := core.controller.GetNetworkWithSubnet(tenantName, networkName, subnetCIDR)
	if err != nil {
		return err
	}
	if network == nil {
		return errors.New("Retrieved Contrail network is nil")
	}

	return core.LocalContrailNetworksRepo.CreateNetwork(dockerNetID, network)
}

func (core *ContrailDriverCore) DeleteNetwork(dockerNetID string) error {
	return core.LocalContrailNetworksRepo.DeleteNetwork(dockerNetID)
}

func (core *ContrailDriverCore) CreateEndpoint(dockerNetID, endpointID string) (*model.Container, error) {

	containerID := core.getIdOfContainerWithEndpoint(endpointID)

	network, err := core.LocalContrailNetworksRepo.GetNetwork(dockerNetID)
	if err != nil {
		return nil, err
	}

	container, err := core.controller.CreateContainerInSubnet(network, containerID)
	if err != nil {
		return nil, err
	}

	localEndpoint, err := core.createContainerEndpointInLocalNetwork(container, network, endpointID)
	if err != nil {
		return nil, err
	}

	core.associatePort(container, localEndpoint)

	return container, nil
}

func (core *ContrailDriverCore) DeleteEndpoint(dockerNetID, endpointID string) error {

	containerID := core.getIdOfContainerWithEndpoint(endpointID)

	container, err := core.controller.GetContainer(endpointID)
	if err != nil {
		return err
	}

	core.disassociatePort(container)

	err = core.controller.DeleteContainer(containerID)
	if err != nil {
		log.Warn("When handling DeleteEndpoint, failed to remove Contrail vm instance. Continuing.")
	}

	return core.LocalContrailEndpointsRepo.DeleteEndpoint(endpointID)
}

func (core *ContrailDriverCore) getIdOfContainerWithEndpoint(endpointID string) string {
	// WORKAROUND:
	// At the time of handling CreateEndpoint request, docker container doesn't exist yet, and
	// there is no way of knowing what will docker container ID be. CNM spec gives us only endpoint
	// ID in the request.
	// This is problematic, because in Contrail, a single VM (or container) can have multiple
	// endpoints.
	// For now, the workaround is to just assume that container ID is equal to endpoind ID.
	// A slightly better solution would be to pass an additional parameter, like `-opt vmname=1234`
	// to `docker run` command. Then, we could use `containerID := req.Options["vmname"]`. However,
	// `docker run` doesn't support custom options.
	// Another solution would be to investigate newer HNS APIs, because they introduce some kind of
	// how interface adding feature.
	containerID := endpointID
	return containerID
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

func (core *ContrailDriverCore) associatePort(container *model.Container, ep *model.LocalEndpoint) {
	// WORKAROUND:
	// Normally, we would wait for result of AddPort request, but we would wait forever,
	// because no container would be created until we return from CreateEndpoint request.
	// This is because docker daemon first waits for plugins, and only after it receives
	// responses, creates the container.
	// The second part of this workaround consists of polling for interface to appear
	// in the OS upon receiving the AddPort request in vRouter Agent code.
	// See contrail-controller/src/vnsw/agent/oper/windows/interface_params.cc,
	// function GetVmInterfaceLuidFromName.
	go func() {
		err := core.portAssociation.AddPort(container.VmUUID, container.VmiUUID, ep.IfName,
			container.Mac, ep.Name, container.IP.String(), container.NetUUID)
		if err != nil {
			log.Error(err.Error())
		}
		log.Debugln("core.associatePort done for", container, ep)
	}()
}

func (core *ContrailDriverCore) disassociatePort(container *model.Container) {
	// WORKAROUND:
	// Refer to comment for associatePort.
	go func() {
		err := core.portAssociation.DeletePort(container.VmiUUID)
		if err != nil {
			log.Error(err.Error())
		}
		log.Debugln("core.disassociatePort done for", container)
	}()
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

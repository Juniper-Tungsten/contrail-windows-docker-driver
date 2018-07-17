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
	"time"

	// TODO: this import should be removed when making Controller port smaller
	"github.com/Juniper/contrail-go-api/types"

	// TODO: this import should be removed
	"github.com/Microsoft/hcsshim"

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
	// TODO: get rid of this sleep workaround asap
	hnsEndpointWaitingTime time.Duration
}

func NewContrailDriverCore(vr ports.VRouter, c ports.Controller, a ports.Agent,
	nr ports.LocalContrailNetworkRepository,
	er ports.LocalContrailEndpointRepository,
	hnsEndpointWaitingTime time.Duration) (*ContrailDriverCore, error) {
	core := ContrailDriverCore{
		vrouter:    vr,
		Controller: c,
		Agent:      a,
		LocalContrailNetworksRepo:  nr,
		LocalContrailEndpointsRepo: er,
		hnsEndpointWaitingTime:     hnsEndpointWaitingTime,
	}
	if err := core.initializeAdapters(); err != nil {
		return nil, err
	}
	return &core, nil
}

func (core *ContrailDriverCore) initializeAdapters() error {
	return core.vrouter.Initialize()
}

func (core *ContrailDriverCore) CreateNetwork(tenantName, networkName, subnetCIDR string) error {
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

	_, err = core.LocalContrailNetworksRepo.CreateNetwork(tenantName, networkName, subnetCIDR,
		gateway)

	return err
}

func (core *ContrailDriverCore) DeleteNetwork(tenantName, networkName, subnetCIDR string) error {
	return core.LocalContrailNetworksRepo.DeleteNetwork(tenantName, networkName, subnetCIDR)
}

func (core *ContrailDriverCore) CreateEndpoint(tenantName, networkName, subnetCIDR, endpointID string) (*ports.ContrailContainer, error) {

	// WORKAROUND: We need to retreive Container ID here and use it instead of EndpointID as
	// argument to GetOrCreateInstance(). EndpointID is equiv to interface, but in Contrail,
	// we have a "VirtualMachine" in data model. A single VM can be connected to two or more
	// overlay networks, but when we use EndpointID, this won't work. We need something like:
	// containerID := req.Options["vmname"]
	containerID := endpointID

	network, ipamSubnet, err := core.Controller.GetNetworkWithSubnet(tenantName, networkName, subnetCIDR)
	if err != nil {
		return nil, err
	}

	log.Infoln(core.Controller.CreateContainerInSubnet(tenantName, containerID, network, ipamSubnet))

	container, err := core.Controller.CreateContainerInSubnet(tenantName, containerID, network, ipamSubnet)
	if err != nil {
		return nil, err
	}

	// contrail MACs are like 11:22:aa:bb:cc:dd
	// HNS needs MACs like 11-22-AA-BB-CC-DD
	formattedMac := strings.Replace(strings.ToUpper(container.Mac), ":", "-", -1)

	hnsNet, err := core.LocalContrailNetworksRepo.GetNetwork(tenantName, networkName, subnetCIDR)
	if err != nil {
		return nil, err
	}

	gateway := ipamSubnet.DefaultGateway
	if gateway == "" {
		return nil, errors.New("Default GW is empty")
	}

	// TODO: We should remove references to hcsshim here - probably by adding a new struct
	// to core/ports/models.go
	hnsEndpointConfig := &hcsshim.HNSEndpoint{
		VirtualNetworkName: hnsNet.Name,
		Name:               endpointID,
		IPAddress:          container.IP,
		MacAddress:         formattedMac,
		GatewayAddress:     gateway,
	}

	hnsEndpointID, err := core.LocalContrailEndpointsRepo.CreateEndpoint(hnsEndpointConfig)
	if err != nil {
		return nil, err
	}

	// TODO: this can be refactored into some nice mechanism.
	ifName := core.generateFriendlyName(hnsEndpointID)

	go func() {
		// WORKAROUND: Temporary workaround for HNS issue.
		// Due to the bug in Microsoft HNS, Docker Driver has to wait for some time until endpoint
		// is ready to handle ARP requests. Unfortunately it seems that HNS doesn't have api
		// to verify if endpoint setup is done
		time.Sleep(core.hnsEndpointWaitingTime)
		err := core.Agent.AddPort(container.VmUUID, container.VmiUUID, ifName,
			container.Mac, containerID, container.IP.String(), network.GetUuid())
		if err != nil {
			log.Error(err.Error())
		}
	}()
	return container, nil
}

func (core *ContrailDriverCore) GetContrailSubnetCIDR(ipam *types.IpamSubnetType) string {
	// TODO: this should be probably moved to Controller when we make Controller port smaller.
	return fmt.Sprintf("%s/%v", ipam.Subnet.IpPrefix, ipam.Subnet.IpPrefixLen)
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
